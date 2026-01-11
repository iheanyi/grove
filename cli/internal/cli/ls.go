package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/iheanyi/grove/internal/discovery"
	"github.com/iheanyi/grove/internal/github"
	"github.com/iheanyi/grove/internal/registry"
	"github.com/iheanyi/grove/internal/styles"
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
	lsCmd.Flags().Bool("fast", false, "Skip activity detection (deprecated, now default behavior)")
	lsCmd.Flags().Bool("detect-activity", false, "Detect Claude, VS Code, and git status (slower)")
	lsCmd.Flags().Bool("full", false, "Show full info including GitHub PR/CI/review status (implies --detect-activity)")
	lsCmd.Flags().StringSlice("tag", nil, "Filter by tag (can be specified multiple times, uses OR logic)")
	lsCmd.Flags().String("group", "mainRepo", "Group by: mainRepo (default), activity, status, none")
}

func runLs(cmd *cobra.Command, args []string) error {
	outputJSON, _ := cmd.Flags().GetBool("json")
	onlyRunning, _ := cmd.Flags().GetBool("running")
	onlyServers, _ := cmd.Flags().GetBool("servers")
	onlyActive, _ := cmd.Flags().GetBool("active")
	showAll, _ := cmd.Flags().GetBool("all")
	detectActivity, _ := cmd.Flags().GetBool("detect-activity")
	fullMode, _ := cmd.Flags().GetBool("full")
	tagFilters, _ := cmd.Flags().GetStringSlice("tag")
	groupBy, _ := cmd.Flags().GetString("group")
	_ = showAll // Reserved for future use

	// --full implies --detect-activity (need activity data for full output)
	if fullMode {
		detectActivity = true
	}

	// Fast mode is now the default - activity detection only when explicitly requested
	fastMode := !detectActivity

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
			Tags:      server.Tags,
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
		// Tag filtering (OR logic - match any of the specified tags)
		if len(tagFilters) > 0 {
			hasMatchingTag := false
			for _, filterTag := range tagFilters {
				for _, viewTag := range view.Tags {
					if viewTag == filterTag {
						hasMatchingTag = true
						break
					}
				}
				if hasMatchingTag {
					break
				}
			}
			if !hasMatchingTag {
				continue
			}
		}
		filtered = append(filtered, view)
	}

	// Sort: running servers first, then by name (stable sort order)
	sort.Slice(filtered, func(i, j int) bool {
		// Running servers come first
		iRunning := filtered[i].Server != nil && filtered[i].Server.IsRunning()
		jRunning := filtered[j].Server != nil && filtered[j].Server.IsRunning()
		if iRunning != jRunning {
			return iRunning
		}
		// Then sort by name
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
		return outputJSONFormatNew(filtered, reg.GetProxy(), fullMode, githubInfoMap, groupBy)
	}

	return outputTableFormatNew(filtered, reg.GetProxy(), fullMode, githubInfoMap, groupBy)
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
	Tags      []string
}

func outputJSONFormatNew(views []*WorktreeView, proxy *registry.ProxyInfo, fullMode bool, githubInfoMap map[string]*github.BranchInfo, groupBy string) error {
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
		Tags      []string        `json:"tags,omitempty"`
		Group     string          `json:"group,omitempty"`
		GitHub    *jsonGitHubInfo `json:"github,omitempty"`
	}

	type output struct {
		Worktrees []*jsonWorktreeView `json:"worktrees"`
		Proxy     *jsonProxy          `json:"proxy,omitempty"`
		URLMode   string              `json:"url_mode"`
		GroupBy   string              `json:"group_by,omitempty"`
	}

	out := output{
		Worktrees: make([]*jsonWorktreeView, 0, len(views)),
		URLMode:   string(cfg.URLMode),
		GroupBy:   groupBy,
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
			Tags:      view.Tags,
			Group:     getGroupForView(view, groupBy),
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

func outputTableFormatNew(views []*WorktreeView, proxy *registry.ProxyInfo, fullMode bool, githubInfoMap map[string]*github.BranchInfo, groupBy string) error {
	if len(views) == 0 {
		fmt.Println("No worktrees discovered")
		fmt.Println("\nUse 'grove discover' to scan for git worktrees, or 'grove start <command>' to start a server")
		return nil
	}

	// Group views if grouping is enabled
	if groupBy != "none" && groupBy != "" {
		groups := groupViewsMap(views, groupBy)
		groupOrder := getGroupOrder(groupBy, groups)

		for _, groupName := range groupOrder {
			groupViews := groups[groupName]
			if len(groupViews) == 0 {
				continue
			}

			// Print group header
			fmt.Printf("\n=== %s ===\n", strings.ToUpper(groupName))
			printViewsTable(groupViews, fullMode, githubInfoMap)
		}
	} else {
		// No grouping, print flat list
		printViewsTable(views, fullMode, githubInfoMap)
	}

	// Legend
	fmt.Println()
	if fullMode {
		fmt.Println("Legend: running  stopped  Claude  clean  dirty")
		fmt.Println("PR: open/draft/merged/closed  CI: success  failure  pending")
		fmt.Println("Review: approved/changes/pending")
	} else {
		fmt.Println("Legend: running  stopped  Claude  VS Code  clean  dirty")
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

// printViewsTable prints a table of views
func printViewsTable(views []*WorktreeView, fullMode bool, githubInfoMap map[string]*github.BranchInfo) {
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
	headerStyle := styles.HeaderStyle
	cellStyle := styles.CellStyle

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

// getGroupForView returns the group name for a view based on the groupBy strategy
func getGroupForView(view *WorktreeView, groupBy string) string {
	switch groupBy {
	case "mainRepo":
		if view.MainRepo != "" {
			return filepath.Base(view.MainRepo)
		}
		return "unknown"
	case "activity":
		// Active: has running server, Claude, or VSCode
		if view.Server != nil && view.Server.IsRunning() {
			return "active"
		}
		if view.HasClaude || view.HasVSCode {
			return "active"
		}
		// Recent: has git changes (dirty)
		if view.GitDirty {
			return "recent"
		}
		// Stale: no activity
		return "stale"
	case "status":
		if view.Server == nil {
			return "no-server"
		}
		switch view.Server.Status {
		case registry.StatusRunning, registry.StatusStarting:
			return "running"
		case registry.StatusStopped, registry.StatusStopping:
			return "stopped"
		case registry.StatusCrashed:
			return "error"
		default:
			return "stopped"
		}
	case "none":
		return ""
	default:
		return ""
	}
}

// groupViewsMap groups views by the specified groupBy strategy
func groupViewsMap(views []*WorktreeView, groupBy string) map[string][]*WorktreeView {
	groups := make(map[string][]*WorktreeView)
	for _, view := range views {
		group := getGroupForView(view, groupBy)
		groups[group] = append(groups[group], view)
	}
	return groups
}

// getGroupOrder returns the ordered list of group names for display
func getGroupOrder(groupBy string, groups map[string][]*WorktreeView) []string {
	switch groupBy {
	case "activity":
		// Fixed order: active, recent, stale
		order := []string{}
		for _, g := range []string{"active", "recent", "stale"} {
			if _, ok := groups[g]; ok {
				order = append(order, g)
			}
		}
		return order
	case "status":
		// Fixed order: running, stopped, error, no-server
		order := []string{}
		for _, g := range []string{"running", "stopped", "error", "no-server"} {
			if _, ok := groups[g]; ok {
				order = append(order, g)
			}
		}
		return order
	case "mainRepo":
		// Sort alphabetically
		order := make([]string, 0, len(groups))
		for g := range groups {
			order = append(order, g)
		}
		sort.Strings(order)
		return order
	default:
		return []string{""}
	}
}
