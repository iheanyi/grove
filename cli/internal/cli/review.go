package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/iheanyi/grove/internal/discovery"
	"github.com/iheanyi/grove/internal/registry"
	"github.com/iheanyi/grove/pkg/browser"
	"github.com/spf13/cobra"
)

var reviewCmd = &cobra.Command{
	Use:   "review",
	Short: "Show review queue of workspaces with changes",
	Long: `Show workspaces with uncommitted changes or recent commits not on main.

Displays a review queue with:
- Workspace name
- Task summary (from Tasuku, Beads, or last commit)
- File changes (+/- lines, file count)
- Server URL (if running)

Interactive menu allows opening workspaces in browser or viewing diffs.

Examples:
  grove review              # Interactive review queue
  grove review --json       # Output as JSON (for tooling)`,
	RunE: runReview,
}

func init() {
	reviewCmd.Flags().Bool("json", false, "Output as JSON")
	rootCmd.AddCommand(reviewCmd)
}

// ReviewItem represents a workspace ready for review
type ReviewItem struct {
	Name         string `json:"name"`
	Path         string `json:"path"`
	Branch       string `json:"branch"`
	TaskSummary  string `json:"task_summary,omitempty"`
	FilesChanged int    `json:"files_changed"`
	LinesAdded   int    `json:"lines_added"`
	LinesRemoved int    `json:"lines_removed"`
	ServerURL    string `json:"server_url,omitempty"`
	IsRunning    bool   `json:"is_running"`
	HasUnpushed  bool   `json:"has_unpushed"`
	IsDirty      bool   `json:"is_dirty"`
}

func runReview(cmd *cobra.Command, args []string) error {
	jsonOutput, _ := cmd.Flags().GetBool("json")

	// Load registry
	reg, err := registry.Load()
	if err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}

	// Cleanup stale entries first
	if _, err := reg.Cleanup(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to cleanup stale entries: %v\n", err)
	}

	// Get all workspaces with changes
	items := collectReviewItems(reg)

	if len(items) == 0 {
		if jsonOutput {
			fmt.Println("[]")
		} else {
			fmt.Println("No workspaces with changes found.")
			fmt.Println("\nAll worktrees are clean and up-to-date with their remote branches.")
		}
		return nil
	}

	if jsonOutput {
		return outputReviewJSON(items)
	}

	return runReviewInteractive(items)
}

// collectReviewItems gathers all workspaces that have changes
func collectReviewItems(reg *registry.Registry) []*ReviewItem {
	var items []*ReviewItem

	workspaces := reg.ListWorkspaces()

	for _, ws := range workspaces {
		// Skip if path doesn't exist
		if _, err := os.Stat(ws.Path); os.IsNotExist(err) {
			continue
		}

		// Check if workspace has changes worth reviewing
		isDirty := checkGitDirty(ws.Path)
		hasUnpushed := checkUnpushedCommits(ws.Path)

		if !isDirty && !hasUnpushed {
			continue
		}

		item := &ReviewItem{
			Name:        ws.Name,
			Path:        ws.Path,
			Branch:      ws.Branch,
			IsDirty:     isDirty,
			HasUnpushed: hasUnpushed,
		}

		// Get diff stats
		added, removed, files := getGitDiffStats(ws.Path)
		item.LinesAdded = added
		item.LinesRemoved = removed
		item.FilesChanged = files

		// Get task summary from beads if available
		item.TaskSummary = getTaskSummary(ws.Path)

		// Get server info
		if ws.Server != nil && ws.IsRunning() {
			item.ServerURL = ws.GetURL()
			item.IsRunning = true
		}

		items = append(items, item)
	}

	// Sort by name
	sort.Slice(items, func(i, j int) bool {
		return items[i].Name < items[j].Name
	})

	return items
}

// checkGitDirty checks if the worktree has uncommitted changes
func checkGitDirty(path string) bool {
	cmd := exec.Command("git", "-C", path, "status", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return len(strings.TrimSpace(string(output))) > 0
}

// checkUnpushedCommits checks if there are commits not on the remote
func checkUnpushedCommits(path string) bool {
	// Check if we have an upstream branch
	cmd := exec.Command("git", "-C", path, "rev-parse", "--abbrev-ref", "@{upstream}")
	if err := cmd.Run(); err != nil {
		// No upstream branch, check for commits ahead of main/master
		for _, base := range []string{"origin/main", "origin/master"} {
			cmd := exec.Command("git", "-C", path, "rev-list", "--count", base+"..HEAD")
			output, err := cmd.Output()
			if err == nil {
				count, _ := strconv.Atoi(strings.TrimSpace(string(output)))
				if count > 0 {
					return true
				}
			}
		}
		return false
	}

	// Has upstream, check for commits ahead
	cmd = exec.Command("git", "-C", path, "rev-list", "--count", "@{upstream}..HEAD")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	count, _ := strconv.Atoi(strings.TrimSpace(string(output)))
	return count > 0
}

// getGitDiffStats returns lines added, removed, and file count
func getGitDiffStats(path string) (added, removed, files int) {
	// Get stats for staged and unstaged changes combined
	cmd := exec.Command("git", "-C", path, "diff", "--stat", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		// Try without HEAD (for new repos)
		cmd = exec.Command("git", "-C", path, "diff", "--stat")
		output, _ = cmd.Output()
	}

	return parseDiffStats(string(output))
}

// parseDiffStats parses the output of git diff --stat
func parseDiffStats(output string) (added, removed, files int) {
	lines := strings.Split(output, "\n")
	if len(lines) == 0 {
		return 0, 0, 0
	}

	// The summary line is typically the last non-empty line
	// Format: " N files changed, M insertions(+), P deletions(-)"
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		// Check if this is a summary line
		if strings.Contains(line, "changed") {
			// Parse file count
			if match := regexp.MustCompile(`(\d+) files? changed`).FindStringSubmatch(line); len(match) > 1 {
				files, _ = strconv.Atoi(match[1])
			}
			// Parse insertions
			if match := regexp.MustCompile(`(\d+) insertions?`).FindStringSubmatch(line); len(match) > 1 {
				added, _ = strconv.Atoi(match[1])
			}
			// Parse deletions
			if match := regexp.MustCompile(`(\d+) deletions?`).FindStringSubmatch(line); len(match) > 1 {
				removed, _ = strconv.Atoi(match[1])
			}
			break
		}
	}

	return added, removed, files
}

// getTaskSummary tries to get a task summary from Tasuku, Beads, or recent commit
func getTaskSummary(path string) string {
	// Try Tasuku first (.tasuku/tasks/)
	if taskID, taskDesc := discovery.GetActiveTask(path); taskID != "" {
		summary := taskID
		if taskDesc != "" {
			summary = taskDesc
		}
		if len(summary) > 60 {
			summary = summary[:57] + "..."
		}
		return summary
	}

	// Fall back to Beads (.beads/issues/) for backwards compatibility
	beadsPath := filepath.Join(path, ".beads", "issues")
	if info, err := os.Stat(beadsPath); err == nil && info.IsDir() {
		if summary := findBeadsTask(beadsPath); summary != "" {
			return summary
		}
	}

	// Fall back to last commit message
	cmd := exec.Command("git", "-C", path, "log", "-1", "--format=%s")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	msg := strings.TrimSpace(string(output))
	// Truncate if too long
	if len(msg) > 60 {
		msg = msg[:57] + "..."
	}
	return msg
}

// findBeadsTask looks for an in_progress issue in beads
func findBeadsTask(beadsPath string) string {
	entries, err := os.ReadDir(beadsPath)
	if err != nil {
		return ""
	}

	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		content, err := os.ReadFile(filepath.Join(beadsPath, entry.Name()))
		if err != nil {
			continue
		}

		// Check if issue is in_progress
		if !strings.Contains(string(content), "status: in_progress") {
			continue
		}

		// Extract title from first heading
		lines := strings.Split(string(content), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "# ") {
				title := strings.TrimPrefix(line, "# ")
				if len(title) > 50 {
					title = title[:47] + "..."
				}
				return title
			}
			// Also try title from frontmatter
			if strings.HasPrefix(line, "title:") {
				title := strings.TrimSpace(strings.TrimPrefix(line, "title:"))
				title = strings.Trim(title, "\"'")
				if len(title) > 50 {
					title = title[:47] + "..."
				}
				return title
			}
		}
	}

	return ""
}

func outputReviewJSON(items []*ReviewItem) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(items)
}

func runReviewInteractive(items []*ReviewItem) error {
	// Styles
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	nameStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14"))
	urlStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	statsStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	// Print header
	fmt.Println()
	fmt.Println(headerStyle.Render("Review Queue"))
	fmt.Println()

	// Print items
	for i, item := range items {
		// Number and name
		fmt.Printf("%s. %s\n",
			dimStyle.Render(fmt.Sprintf("%d", i+1)),
			nameStyle.Render(item.Name))

		// Task summary
		if item.TaskSummary != "" {
			fmt.Printf("   Task: %s\n", item.TaskSummary)
		}

		// Changes
		changeStr := formatChanges(item.LinesAdded, item.LinesRemoved, item.FilesChanged)
		if changeStr != "" {
			fmt.Printf("   Changes: %s\n", statsStyle.Render(changeStr))
		}

		// Status indicators
		var statusParts []string
		if item.IsDirty {
			statusParts = append(statusParts, "uncommitted changes")
		}
		if item.HasUnpushed {
			statusParts = append(statusParts, "unpushed commits")
		}
		if len(statusParts) > 0 {
			fmt.Printf("   Status: %s\n", dimStyle.Render(strings.Join(statusParts, ", ")))
		}

		// Server URL
		if item.IsRunning {
			fmt.Printf("   URL: %s\n", urlStyle.Render(item.ServerURL))
		} else {
			fmt.Printf("   URL: %s\n", dimStyle.Render("(server not running)"))
		}

		fmt.Println()
	}

	// Print actions
	fmt.Println(dimStyle.Render("─────────────────────────────────────────────────────────"))
	fmt.Println()
	fmt.Println("Actions:")
	fmt.Printf("  [1-%d] Open in browser\n", len(items))
	fmt.Println("  [a]   Open all")
	fmt.Println("  [d]   Show diff (enter number after)")
	fmt.Println("  [q]   Quit")
	fmt.Println()

	// Interactive loop
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("Choice: ")
		input, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		input = strings.TrimSpace(strings.ToLower(input))

		if input == "q" || input == "quit" {
			return nil
		}

		if input == "a" || input == "all" {
			// Open all running servers
			opened := 0
			for _, item := range items {
				if item.IsRunning && item.ServerURL != "" {
					if err := browser.Open(item.ServerURL); err == nil {
						opened++
					}
				}
			}
			if opened == 0 {
				fmt.Println("No running servers to open.")
			} else {
				fmt.Printf("Opened %d servers in browser.\n", opened)
			}
			continue
		}

		if strings.HasPrefix(input, "d") || strings.HasPrefix(input, "diff") {
			// Show diff for specified item
			numStr := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(input, "diff"), "d"))
			if numStr == "" {
				fmt.Print("Enter number to show diff: ")
				numStr, _ = reader.ReadString('\n')
				numStr = strings.TrimSpace(numStr)
			}

			num, err := strconv.Atoi(numStr)
			if err != nil || num < 1 || num > len(items) {
				fmt.Printf("Invalid number. Enter 1-%d\n", len(items))
				continue
			}

			item := items[num-1]
			showDiff(item.Path)
			continue
		}

		// Try to parse as number
		num, err := strconv.Atoi(input)
		if err != nil || num < 1 || num > len(items) {
			fmt.Printf("Invalid choice. Enter 1-%d, 'a', 'd', or 'q'\n", len(items))
			continue
		}

		item := items[num-1]
		if !item.IsRunning {
			fmt.Printf("Server for '%s' is not running. Start it with 'grove start' in that directory.\n", item.Name)
			continue
		}

		fmt.Printf("Opening %s...\n", item.ServerURL)
		if err := browser.Open(item.ServerURL); err != nil {
			fmt.Printf("Failed to open browser: %v\n", err)
		}
	}
}

// formatChanges formats the change statistics
func formatChanges(added, removed, files int) string {
	if files == 0 && added == 0 && removed == 0 {
		return ""
	}

	parts := []string{}
	if added > 0 {
		parts = append(parts, fmt.Sprintf("+%d", added))
	}
	if removed > 0 {
		parts = append(parts, fmt.Sprintf("-%d", removed))
	}
	if files > 0 {
		filesWord := "files"
		if files == 1 {
			filesWord = "file"
		}
		parts = append(parts, fmt.Sprintf("(%d %s)", files, filesWord))
	}

	return strings.Join(parts, " ")
}

// showDiff displays the git diff for a workspace
func showDiff(path string) {
	cmd := exec.Command("git", "-C", path, "diff", "--color=always", "HEAD")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	_ = cmd.Run()
}
