package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/iheanyi/grove/internal/github"
	"github.com/iheanyi/grove/internal/registry"
	"github.com/spf13/cobra"
)

var lsCmd = &cobra.Command{
	Use:     "ls",
	Aliases: []string{"list"},
	Short:   "List all registered servers",
	Long: `List all registered servers and their status.

Examples:
  grove ls           # List all servers
  grove ls --full    # Include CI status and PR links (requires gh CLI)
  grove ls --json    # Output as JSON (for MCP/tooling)
  grove ls --running # Only show running servers`,
	RunE: runLs,
}

func init() {
	lsCmd.Flags().Bool("json", false, "Output as JSON")
	lsCmd.Flags().Bool("running", false, "Only show running servers")
	lsCmd.Flags().Bool("full", false, "Show full info including CI status and PR links")
}

func runLs(cmd *cobra.Command, args []string) error {
	outputJSON, _ := cmd.Flags().GetBool("json")
	onlyRunning, _ := cmd.Flags().GetBool("running")
	showFull, _ := cmd.Flags().GetBool("full")

	// Load registry
	reg, err := registry.Load()
	if err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}

	// Cleanup stale entries first
	reg.Cleanup()

	// Get servers
	var servers []*registry.Server
	if onlyRunning {
		servers = reg.ListRunning()
	} else {
		servers = reg.List()
	}

	// Sort by name
	sort.Slice(servers, func(i, j int) bool {
		return servers[i].Name < servers[j].Name
	})

	// Fetch GitHub info if --full flag is set
	var ghInfo map[string]*github.BranchInfo
	if showFull && len(servers) > 0 {
		branches := make([]string, 0, len(servers))
		for _, s := range servers {
			if s.Branch != "" {
				branches = append(branches, s.Branch)
			}
		}
		ghInfo = github.GetBranchInfoBatch(branches)
	}

	if outputJSON {
		return outputJSONFormat(servers, reg.GetProxy(), ghInfo)
	}

	return outputTableFormat(servers, reg.GetProxy(), showFull, ghInfo)
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
	Branch     string `json:"branch,omitempty"`
	Uptime     string `json:"uptime,omitempty"`
	PID        int    `json:"pid,omitempty"`
	LogFile    string `json:"log_file,omitempty"`
	CI         string `json:"ci,omitempty"`
	PRNumber   int    `json:"pr_number,omitempty"`
	PRURL      string `json:"pr_url,omitempty"`
}

type jsonProxy struct {
	Status    string `json:"status"`
	HTTPPort  int    `json:"http_port,omitempty"`
	HTTPSPort int    `json:"https_port,omitempty"`
	PID       int    `json:"pid,omitempty"`
}

func outputJSONFormat(servers []*registry.Server, proxy *registry.ProxyInfo, ghInfo map[string]*github.BranchInfo) error {
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
			Branch:  s.Branch,
			Uptime:  s.UptimeString(),
			PID:     s.PID,
			LogFile: s.LogFile,
		}

		// Only include subdomains in subdomain mode
		if cfg.IsSubdomainMode() {
			js.Subdomains = cfg.SubdomainURL(s.Name)
		}

		// Add GitHub info if available
		if ghInfo != nil && s.Branch != "" {
			if info := ghInfo[s.Branch]; info != nil {
				if info.CI != nil {
					js.CI = info.CI.State
				}
				if info.PR != nil {
					js.PRNumber = info.PR.Number
					js.PRURL = info.PR.URL
				}
			}
		}

		output.Servers = append(output.Servers, js)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(output)
}

func outputTableFormat(servers []*registry.Server, proxy *registry.ProxyInfo, showFull bool, ghInfo map[string]*github.BranchInfo) error {
	if len(servers) == 0 {
		fmt.Println("No servers registered")
		fmt.Println("\nUse 'grove start <command>' to start a server")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	// Header - include CI and PR columns when showing full info
	if showFull {
		fmt.Fprintln(w, "NAME\tURL\tPORT\tSTATUS\tCI\tPR\tUPTIME")
		fmt.Fprintln(w, strings.Repeat("-", 100))
	} else {
		fmt.Fprintln(w, "NAME\tURL\tPORT\tSTATUS\tUPTIME")
		fmt.Fprintln(w, strings.Repeat("-", 80))
	}

	for _, s := range servers {
		status := formatStatus(s.Status)
		uptime := s.UptimeString()
		if uptime == "" || !s.IsRunning() {
			uptime = "-"
		}

		// Generate URL based on current mode
		url := cfg.ServerURL(s.Name, s.Port)

		if showFull {
			ciStatus := "-"
			prInfo := "-"

			if ghInfo != nil && s.Branch != "" {
				if info := ghInfo[s.Branch]; info != nil {
					if info.CI != nil {
						ciStatus = github.FormatCIStatus(info.CI)
					}
					if info.PR != nil {
						prInfo = github.FormatPRInfo(info.PR)
					}
				}
			}

			fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%s\t%s\t%s\n",
				s.Name,
				url,
				s.Port,
				status,
				ciStatus,
				prInfo,
				uptime,
			)
		} else {
			fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%s\n",
				s.Name,
				url,
				s.Port,
				status,
				uptime,
			)
		}
	}

	w.Flush()

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
