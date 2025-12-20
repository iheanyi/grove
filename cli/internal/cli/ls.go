package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/iheanyi/grove/internal/registry"
	"github.com/spf13/cobra"
)

var lsCmd = &cobra.Command{
	Use:     "ls",
	Aliases: []string{"list"},
	Short:   "List all registered servers and discovered worktrees",
	Long: `List all registered servers and discovered worktrees with their status.

Examples:
  wt ls            # List all discovered worktrees
  wt ls --json     # Output as JSON (for MCP/tooling)
  wt ls --servers  # Only show worktrees with servers (old behavior)
  wt ls --active   # Only show worktrees with any activity
  wt ls --all      # Show all discovered worktrees (default)`,
	RunE: runLs,
}

func init() {
	lsCmd.Flags().Bool("json", false, "Output as JSON")
	lsCmd.Flags().Bool("servers", false, "Only show worktrees with servers")
	lsCmd.Flags().Bool("active", false, "Only show worktrees with any activity")
	lsCmd.Flags().Bool("all", false, "Show all discovered worktrees (default)")
	lsCmd.Flags().Bool("running", false, "Only show running servers (deprecated, use --servers)")
}

func runLs(cmd *cobra.Command, args []string) error {
	outputJSON, _ := cmd.Flags().GetBool("json")
	onlyRunning, _ := cmd.Flags().GetBool("running")
	onlyServers, _ := cmd.Flags().GetBool("servers")
	onlyActive, _ := cmd.Flags().GetBool("active")
	showAll, _ := cmd.Flags().GetBool("all")
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

	// Cleanup stale entries first
	reg.Cleanup()

	// Update worktree activities
	reg.UpdateWorktreeActivities()

	// Build combined view
	views := make(map[string]*WorktreeView)

	// Add all registered servers
	for _, server := range reg.List() {
		views[server.Name] = &WorktreeView{
			Name:      server.Name,
			Path:      server.Path,
			Branch:    server.Branch,
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
		} else {
			// New worktree without server
			views[wt.Name] = &WorktreeView{
				Name:      wt.Name,
				Path:      wt.Path,
				Branch:    wt.Branch,
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

	if outputJSON {
		return outputJSONFormatNew(filtered, reg.GetProxy())
	}

	return outputTableFormatNew(filtered, reg.GetProxy())
}

type jsonOutput struct {
	Servers []*jsonServer `json:"servers"`
	Proxy   *jsonProxy    `json:"proxy"`
	URLMode string        `json:"url_mode"`
}

type jsonServer struct {
	Name       string `json:"name"`
	URL        string `json:"url"`
	Subdomains string `json:"subdomains,omitempty"`
	Port       int    `json:"port"`
	Status     string `json:"status"`
	Health     string `json:"health,omitempty"`
	Path       string `json:"path"`
	Uptime     string `json:"uptime,omitempty"`
	PID        int    `json:"pid,omitempty"`
	LogFile    string `json:"log_file,omitempty"`
}

type jsonProxy struct {
	Status    string `json:"status"`
	HTTPPort  int    `json:"http_port,omitempty"`
	HTTPSPort int    `json:"https_port,omitempty"`
	PID       int    `json:"pid,omitempty"`
}

func outputJSONFormat(servers []*registry.Server, proxy *registry.ProxyInfo) error {
	output := jsonOutput{
		Servers: make([]*jsonServer, 0, len(servers)),
		URLMode: string(cfg.URLMode),
	}

	// Only include proxy info if in subdomain mode
	if cfg.IsSubdomainMode() {
		output.Proxy = &jsonProxy{
			HTTPPort:  proxy.HTTPPort,
			HTTPSPort: proxy.HTTPSPort,
			PID:       proxy.PID,
		}
		if proxy.IsRunning() {
			output.Proxy.Status = "running"
		} else {
			output.Proxy.Status = "stopped"
		}
	}

	for _, s := range servers {
		// Generate URL based on current mode (in case registry has old URLs)
		url := cfg.ServerURL(s.Name, s.Port)

		js := &jsonServer{
			Name:    s.Name,
			URL:     url,
			Port:    s.Port,
			Status:  string(s.Status),
			Health:  string(s.Health),
			Path:    s.Path,
			Uptime:  s.UptimeString(),
			PID:     s.PID,
			LogFile: s.LogFile,
		}

		// Only include subdomains in subdomain mode
		if cfg.IsSubdomainMode() {
			js.Subdomains = cfg.SubdomainURL(s.Name)
		}

		output.Servers = append(output.Servers, js)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(output)
}

func outputTableFormat(servers []*registry.Server, proxy *registry.ProxyInfo) error {
	if len(servers) == 0 {
		fmt.Println("No servers registered")
		fmt.Println("\nUse 'wt start <command>' to start a server")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	// Header
	fmt.Fprintln(w, "NAME\tURL\tPORT\tSTATUS\tUPTIME")
	fmt.Fprintln(w, strings.Repeat("-", 80))

	for _, s := range servers {
		status := formatStatus(s.Status)
		uptime := s.UptimeString()
		if uptime == "" || !s.IsRunning() {
			uptime = "-"
		}

		// Generate URL based on current mode
		url := cfg.ServerURL(s.Name, s.Port)

		fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%s\n",
			s.Name,
			url,
			s.Port,
			status,
			uptime,
		)
	}

	w.Flush()

	// Proxy status (only relevant in subdomain mode)
	fmt.Println()
	if cfg.IsSubdomainMode() {
		if proxy.IsRunning() {
			fmt.Printf("Proxy: running on :%d/:%d (PID: %d)\n",
				proxy.HTTPPort, proxy.HTTPSPort, proxy.PID)
		} else {
			fmt.Println("Proxy: not running (use 'wt proxy start' to start)")
		}
	} else {
		fmt.Printf("URL mode: port (access servers directly via http://localhost:PORT)\n")
	}

	return nil
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
	Server    *registry.Server
	HasServer bool
	HasClaude bool
	HasVSCode bool
	GitDirty  bool
}

func outputJSONFormatNew(views []*WorktreeView, proxy *registry.ProxyInfo) error {
	type jsonWorktreeView struct {
		Name      string `json:"name"`
		Path      string `json:"path"`
		Branch    string `json:"branch,omitempty"`
		URL       string `json:"url,omitempty"`
		Port      int    `json:"port,omitempty"`
		Status    string `json:"status,omitempty"`
		HasServer bool   `json:"has_server"`
		HasClaude bool   `json:"has_claude"`
		HasVSCode bool   `json:"has_vscode"`
		GitDirty  bool   `json:"git_dirty"`
		PID       int    `json:"pid,omitempty"`
		Uptime    string `json:"uptime,omitempty"`
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
		}

		out.Worktrees = append(out.Worktrees, jv)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func outputTableFormatNew(views []*WorktreeView, proxy *registry.ProxyInfo) error {
	if len(views) == 0 {
		fmt.Println("No worktrees discovered")
		fmt.Println("\nUse 'wt discover' to scan for git worktrees, or 'wt start <command>' to start a server")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	// Header - updated format with activity indicators
	fmt.Fprintln(w, "NAME\tSERVER\tCLAUDE\tVSCODE\tGIT\tPATH")
	fmt.Fprintln(w, strings.Repeat("-", 100))

	for _, view := range views {
		// Server status
		serverStatus := "○ -"
		if view.Server != nil {
			if view.Server.IsRunning() {
				serverStatus = fmt.Sprintf("● :%d", view.Server.Port)
			} else {
				serverStatus = fmt.Sprintf("○ :%d", view.Server.Port)
			}
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

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			view.Name,
			serverStatus,
			claudeStatus,
			vscodeStatus,
			gitStatus,
			displayPath,
		)
	}

	w.Flush()

	// Legend
	fmt.Println()
	fmt.Println("Legend:")
	fmt.Println("  SERVER: ● = running, ○ = stopped, - = no server")
	fmt.Println("  CLAUDE: 🤖 = active")
	fmt.Println("  VSCODE: 💻 = active")
	fmt.Println("  GIT: ✓ = clean, 📝 = dirty")

	// Proxy status (only relevant in subdomain mode)
	fmt.Println()
	if cfg.IsSubdomainMode() {
		if proxy.IsRunning() {
			fmt.Printf("Proxy: running on :%d/:%d (PID: %d)\n",
				proxy.HTTPPort, proxy.HTTPSPort, proxy.PID)
		} else {
			fmt.Println("Proxy: not running (use 'wt proxy start' to start)")
		}
	} else {
		fmt.Printf("URL mode: port (access servers directly via http://localhost:PORT)\n")
	}

	return nil
}
