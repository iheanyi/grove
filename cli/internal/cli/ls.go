package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/iheanyi/wt/internal/registry"
	"github.com/spf13/cobra"
)

var lsCmd = &cobra.Command{
	Use:     "ls",
	Aliases: []string{"list"},
	Short:   "List all registered servers",
	Long: `List all registered servers and their status.

Examples:
  wt ls           # List all servers
  wt ls --json    # Output as JSON (for MCP/tooling)
  wt ls --running # Only show running servers`,
	RunE: runLs,
}

func init() {
	lsCmd.Flags().Bool("json", false, "Output as JSON")
	lsCmd.Flags().Bool("running", false, "Only show running servers")
}

func runLs(cmd *cobra.Command, args []string) error {
	outputJSON, _ := cmd.Flags().GetBool("json")
	onlyRunning, _ := cmd.Flags().GetBool("running")

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

	if outputJSON {
		return outputJSONFormat(servers, reg.GetProxy())
	}

	return outputTableFormat(servers, reg.GetProxy())
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
