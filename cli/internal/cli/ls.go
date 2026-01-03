package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/iheanyi/grove/internal/discovery"
	"github.com/iheanyi/grove/internal/github"
	"github.com/iheanyi/grove/internal/registry"
	"github.com/iheanyi/grove/internal/worktree"
	"github.com/spf13/cobra"
)

var lsCmd = &cobra.Command{
	Use:     "ls",
	Aliases: []string{"list"},
	Short:   "List all registered servers and discovered worktrees",
	Long: `List all registered servers and discovered worktrees with their status.

Examples:
  grove ls                      # List all discovered worktrees (grouped by mainRepo)
  grove ls --json               # Output as JSON (for MCP/tooling)
  grove ls --servers            # Only show worktrees with servers
  grove ls --active             # Only show worktrees with any activity
  grove ls --tag frontend       # Filter by tag
  grove ls --group activity     # Group by: active, recent, stale
  grove ls --group status       # Group by: running, stopped, error
  grove ls --group none         # No grouping (flat list)
  grove ls --full               # Show GitHub info (PR, CI, review status)
  grove ls --all                # Show all discovered worktrees (default)`,
	RunE: runLs,
}

func init() {
	lsCmd.Flags().Bool("json", false, "Output as JSON")
	lsCmd.Flags().Bool("servers", false, "Only show worktrees with servers")
	lsCmd.Flags().Bool("active", false, "Only show worktrees with any activity")
	lsCmd.Flags().Bool("all", false, "Show all discovered worktrees (default)")
	lsCmd.Flags().Bool("running", false, "Only show running servers (deprecated, use --servers)")
	lsCmd.Flags().Bool("fast", false, "Skip activity detection (Claude, VSCode, git status) for faster output")
	lsCmd.Flags().Bool("full", false, "Show full info including GitHub PR/CI/review status")
}

func runLs(cmd *cobra.Command, args []string) error {
	outputJSON, _ := cmd.Flags().GetBool("json")
	onlyRunning, _ := cmd.Flags().GetBool("running")
	onlyServers, _ := cmd.Flags().GetBool("servers")
	onlyActive, _ := cmd.Flags().GetBool("active")
	showAll, _ := cmd.Flags().GetBool("all")
	fastMode, _ := cmd.Flags().GetBool("fast")
	fullMode, _ := cmd.Flags().GetBool("full")
	_ = showAll // Reserved for future use

	// Backward compatibility: --running implies --servers
	if onlyRunning {
		onlyServers = true
	}

	// Load registry
	reg, err := registry.Load()
	if err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}

	// Cleanup stale entries first (non-critical, continue on error)
	if _, err := reg.Cleanup(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to cleanup stale entries: %v\n", err)
	}

	// Auto-discover worktrees from current repo (fast operation)
	if !fastMode {
		autoDiscoverCurrentRepo(reg)
	}

	// Update worktree activities (non-critical, continue on error)
	// Skip in fast mode - this is the slow part (ps, lsof, git status for each worktree)
	if !fastMode {
		if err := reg.UpdateWorktreeActivities(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to update worktree activities: %v\n", err)
		}
	}

	// Build combined view
	views := make(map[string]*WorktreeView)

	// Add all registered servers
	for _, server := range reg.List() {
		// Try to get main_repo from worktree registry
		var mainRepo string
		if wt, exists := reg.GetWorktree(server.Name); exists {
			mainRepo = wt.MainRepo
		}
		views[server.Name] = &WorktreeView{
			Name:      server.Name,
			Path:      server.Path,
			Branch:    server.Branch,
			MainRepo:  mainRepo,
			Server:    server,
			HasServer: true,
		}
	}

	// Add/merge discovered worktrees
	for _, wt := range reg.ListWorktrees() {
		if view, exists := views[wt.Name]; exists {
			// Merge with existing server entry
			view.HasClaude = wt.HasClaude
			view.HasVSCode = wt.HasVSCode
			view.GitDirty = wt.GitDirty
			view.MainRepo = wt.MainRepo
		} else {
			// New worktree without server
			views[wt.Name] = &WorktreeView{
				Name:      wt.Name,
				Path:      wt.Path,
				Branch:    wt.Branch,
				MainRepo:  wt.MainRepo,
				HasServer: false,
				HasClaude: wt.HasClaude,
				HasVSCode: wt.HasVSCode,
				GitDirty:  wt.GitDirty,
			}
		}
	}

	// Filter based on flags
	var filtered []*WorktreeView
	for _, view := range views {
		if onlyServers && !view.HasServer {
			continue
		}
		if onlyRunning && (view.Server == nil || !view.Server.IsRunning()) {
			continue
		}
		if onlyActive && !view.HasServer && !view.HasClaude && !view.HasVSCode && !view.GitDirty {
			continue
		}
		filtered = append(filtered, view)
	}

	// Sort by name
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Name < filtered[j].Name
	})

	// Fetch GitHub info for all worktrees if --full is set
	var githubInfoMap map[string]*github.BranchInfo
	if fullMode {
		branches := make([]string, 0, len(filtered))
		for _, view := range filtered {
			if view.Branch != "" {
				branches = append(branches, view.Branch)
			}
		}
		githubInfoMap = github.GetBranchInfoBatch(branches)
	}

	if outputJSON {
		return outputJSONFormatNew(filtered, reg.GetProxy(), fullMode, githubInfoMap)
	}

	return outputTableFormatNew(filtered, reg.GetProxy(), fullMode, githubInfoMap)
}

type jsonProxy struct {
	Status    string `json:"status"`
	HTTPPort  int    `json:"http_port,omitempty"`
	HTTPSPort int    `json:"https_port,omitempty"`
	PID       int    `json:"pid,omitempty"`
}

func formatStatus(status registry.ServerStatus) string {
	switch status {
	case registry.StatusRunning:
		return "● running"
	case registry.StatusStopped:
		return "○ stopped"
	case registry.StatusStarting:
		return "◐ starting"
	case registry.StatusStopping:
		return "◑ stopping"
	case registry.StatusCrashed:
		return "✗ crashed"
	default:
		return string(status)
	}
}

// WorktreeView represents a combined view of server and worktree data
type WorktreeView struct {
	Name      string
	Path      string
	Branch    string
	MainRepo  string
	Server    *registry.Server
	HasServer bool
	HasClaude bool
	HasVSCode bool
	GitDirty  bool
}

func outputJSONFormatNew(views []*WorktreeView, proxy *registry.ProxyInfo, fullMode bool, githubInfoMap map[string]*github.BranchInfo) error {
	type jsonGitHubInfo struct {
		PRNumber     int    `json:"pr_number,omitempty"`
		PRStatus     string `json:"pr_status,omitempty"`
		PRURL        string `json:"pr_url,omitempty"`
		CIStatus     string `json:"ci_status,omitempty"`
		ReviewStatus string `json:"review_status,omitempty"`
	}

	type jsonWorktreeView struct {
		Name      string          `json:"name"`
		Path      string          `json:"path"`
		Branch    string          `json:"branch,omitempty"`
		MainRepo  string          `json:"main_repo,omitempty"`
		URL       string          `json:"url,omitempty"`
		Port      int             `json:"port,omitempty"`
		Status    string          `json:"status,omitempty"`
		HasServer bool            `json:"has_server"`
		HasClaude bool            `json:"has_claude"`
		HasVSCode bool            `json:"has_vscode"`
		GitDirty  bool            `json:"git_dirty"`
		PID       int             `json:"pid,omitempty"`
		Uptime    string          `json:"uptime,omitempty"`
		LogFile   string          `json:"log_file,omitempty"`
		GitHub    *jsonGitHubInfo `json:"github,omitempty"`
	}

	type output struct {
		Worktrees []*jsonWorktreeView `json:"worktrees"`
		Proxy     *jsonProxy          `json:"proxy,omitempty"`
		URLMode   string              `json:"url_mode"`
	}

	out := output{
		Worktrees: make([]*jsonWorktreeView, 0, len(views)),
		URLMode:   string(cfg.URLMode),
	}

	// Only include proxy info if in subdomain mode
	if cfg.IsSubdomainMode() {
		out.Proxy = &jsonProxy{
			HTTPPort:  proxy.HTTPPort,
			HTTPSPort: proxy.HTTPSPort,
			PID:       proxy.PID,
		}
		if proxy.IsRunning() {
			out.Proxy.Status = "running"
		} else {
			out.Proxy.Status = "stopped"
		}
	}

	for _, view := range views {
		jv := &jsonWorktreeView{
			Name:      view.Name,
			Path:      view.Path,
			Branch:    view.Branch,
			MainRepo:  view.MainRepo,
			HasServer: view.HasServer,
			HasClaude: view.HasClaude,
			HasVSCode: view.HasVSCode,
			GitDirty:  view.GitDirty,
		}

		if view.Server != nil {
			jv.URL = cfg.ServerURL(view.Server.Name, view.Server.Port)
			jv.Port = view.Server.Port
			jv.Status = string(view.Server.Status)
			jv.PID = view.Server.PID
			jv.Uptime = view.Server.UptimeString()
			jv.LogFile = view.Server.LogFile
		}

		// Add GitHub info if --full is set
		if fullMode && view.Branch != "" {
			if info, ok := githubInfoMap[view.Branch]; ok && info != nil {
				ghInfo := &jsonGitHubInfo{}
				if info.PR != nil {
					ghInfo.PRNumber = info.PR.Number
					ghInfo.PRStatus = github.FormatPRStatus(info.PR)
					ghInfo.PRURL = info.PR.URL
					ghInfo.ReviewStatus = github.FormatReviewStatus(info.PR)
				}
				if info.CI != nil {
					ghInfo.CIStatus = info.CI.State
				}
				// Only include if we have some GitHub info
				if ghInfo.PRNumber > 0 || ghInfo.CIStatus != "" {
					jv.GitHub = ghInfo
				}
			}
		}

		out.Worktrees = append(out.Worktrees, jv)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func outputTableFormatNew(views []*WorktreeView, proxy *registry.ProxyInfo, fullMode bool, githubInfoMap map[string]*github.BranchInfo) error {
	if len(views) == 0 {
		fmt.Println("No worktrees discovered")
		fmt.Println("\nUse 'grove discover' to scan for git worktrees, or 'grove start <command>' to start a server")
		return nil
	}

	// Build table rows
	var rows [][]string
	for _, view := range views {
		// Server status with emoji
		status := "○"
		port := "-"
		if view.Server != nil {
			if view.Server.IsRunning() {
				status = "●"
			}
			port = fmt.Sprintf("%d", view.Server.Port)
		}

		// Claude status
		claudeStatus := "-"
		if view.HasClaude {
			claudeStatus = "🤖"
		}

		// VS Code status
		vscodeStatus := "-"
		if view.HasVSCode {
			vscodeStatus = "💻"
		}

		// Git status
		gitStatus := "✓"
		if view.GitDirty {
			gitStatus = "📝"
		}

		// Shorten path for display
		displayPath := view.Path
		if homeDir, err := os.UserHomeDir(); err == nil {
			if strings.HasPrefix(view.Path, homeDir) {
				displayPath = "~" + strings.TrimPrefix(view.Path, homeDir)
			}
		}

		if fullMode {
			// Full mode: include GitHub info columns
			prStatus := "-"
			ciStatus := "-"
			reviewStatus := "-"

			if view.Branch != "" {
				if info, ok := githubInfoMap[view.Branch]; ok && info != nil {
					if info.PR != nil {
						prStatus = github.FormatPRStatus(info.PR)
						reviewStatus = github.FormatReviewStatus(info.PR)
					}
					if info.CI != nil {
						ciStatus = github.FormatCIStatus(info.CI)
					}
				}
			}

			rows = append(rows, []string{
				view.Name,
				status,
				port,
				prStatus,
				ciStatus,
				reviewStatus,
				claudeStatus,
				gitStatus,
			})
		} else {
			rows = append(rows, []string{
				view.Name,
				status,
				port,
				claudeStatus,
				vscodeStatus,
				gitStatus,
				displayPath,
			})
		}
	}

	// Style definitions
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("252")).PaddingRight(2)
	cellStyle := lipgloss.NewStyle().PaddingRight(2)

	var t *table.Table
	if fullMode {
		// Full mode table with GitHub columns
		t = table.New().
			Border(lipgloss.NormalBorder()).
			BorderRow(false).
			BorderColumn(false).
			BorderTop(false).
			BorderBottom(false).
			BorderLeft(false).
			BorderRight(false).
			Headers("NAME", "SERVER", "PORT", "PR", "CI", "REVIEW", "CLAUDE", "GIT").
			Rows(rows...).
			StyleFunc(func(row, col int) lipgloss.Style {
				if row == table.HeaderRow {
					return headerStyle
				}
				return cellStyle
			})
	} else {
		// Default table
		t = table.New().
			Border(lipgloss.NormalBorder()).
			BorderRow(false).
			BorderColumn(false).
			BorderTop(false).
			BorderBottom(false).
			BorderLeft(false).
			BorderRight(false).
			Headers("NAME", "STATUS", "PORT", "CLAUDE", "VSCODE", "GIT", "PATH").
			Rows(rows...).
			StyleFunc(func(row, col int) lipgloss.Style {
				if row == table.HeaderRow {
					return headerStyle
				}
				return cellStyle
			})
	}

	fmt.Println(t)

	// Legend
	fmt.Println()
	if fullMode {
		fmt.Println("Legend: ● running  ○ stopped  🤖 Claude  ✓ clean  📝 dirty")
		fmt.Println("PR: open/draft/merged/closed  CI: ✓ success  ✗ failure  ◐ pending")
		fmt.Println("Review: approved/changes/pending")
	} else {
		fmt.Println("Legend: ● running  ○ stopped  🤖 Claude  💻 VS Code  ✓ clean  📝 dirty")
	}

	// Proxy status (only relevant in subdomain mode)
	fmt.Println()
	if cfg.IsSubdomainMode() {
		if proxy.IsRunning() {
			fmt.Printf("Proxy: running on :%d/:%d (PID: %d)\n",
				proxy.HTTPPort, proxy.HTTPSPort, proxy.PID)
		} else {
			fmt.Println("Proxy: not running (use 'grove proxy start' to start)")
		}
	} else {
		fmt.Printf("URL mode: port (access servers directly via http://localhost:PORT)\n")
	}

	return nil
}

// autoDiscoverCurrentRepo discovers worktrees from the current git repo and registers them.
// This is a fast operation that only runs `git worktree list` for the current repo.
func autoDiscoverCurrentRepo(reg *registry.Registry) {
	// Try to detect current worktree
	wt, err := worktree.Detect()
	if err != nil {
		// Not in a git repo, skip
		return
	}

	// Discover all worktrees for this repo
	worktrees, err := discovery.Discover(wt.Path)
	if err != nil {
		return
	}

	// Register any new worktrees
	for _, discovered := range worktrees {
		if _, exists := reg.GetWorktree(discovered.Name); !exists {
			// New worktree, register it (non-critical, continue on error)
			if err := reg.SetWorktree(discovered); err != nil {
				// Silently continue - auto-discovery is best-effort
				continue
			}
		}
	}
}
