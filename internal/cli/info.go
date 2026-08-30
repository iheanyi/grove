package cli

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/iheanyi/grove/internal/config"
	"github.com/iheanyi/grove/internal/registry"
	"github.com/iheanyi/grove/internal/worktree"
	"github.com/spf13/cobra"
)

var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "Show comprehensive information about the current project",
	Long: `Show comprehensive information about the current project including:

- Git repository details
- Current worktree and branch
- All worktrees with their status
- Running servers
- Configuration paths

This provides a complete overview of the project state.`,
	RunE: runInfo,
}

func init() {
	infoCmd.Flags().Bool("json", false, "Output as JSON")
}

func runInfo(cmd *cobra.Command, args []string) error {
	jsonOutput, _ := cmd.Flags().GetBool("json")

	// Detect current worktree
	wt, err := worktree.Detect()
	if err != nil {
		return fmt.Errorf("failed to detect worktree: %w", err)
	}

	// Load registry
	reg, err := registry.Load()
	if err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}

	if jsonOutput {
		return outputInfoJSON(wt, reg)
	}

	return outputInfoText(wt, reg)
}

func outputInfoText(wt *worktree.Info, reg *registry.Registry) error {
	// Header
	fmt.Println("╭─────────────────────────────────────────────────────────────╮")
	fmt.Println("│                    grove project info                        │")
	fmt.Println("╰─────────────────────────────────────────────────────────────╯")
	fmt.Println()

	// Current location
	fmt.Println("CURRENT WORKTREE")
	fmt.Printf("  Name:      %s\n", wt.Name)
	fmt.Printf("  Branch:    %s\n", wt.Branch)
	fmt.Printf("  Path:      %s\n", wt.Path)
	if wt.IsWorktree {
		fmt.Printf("  Type:      worktree (linked to %s)\n", wt.MainWorktreePath)
	} else {
		fmt.Printf("  Type:      main repository\n")
	}

	// Current server status
	if server, ok := reg.Get(wt.Name); ok {
		fmt.Println()
		fmt.Println("CURRENT SERVER")
		fmt.Printf("  Status:    %s\n", formatServerStatus(string(server.Status)))
		fmt.Printf("  URL:       %s\n", server.URL)
		fmt.Printf("  Port:      %d\n", server.Port)
		if server.IsRunning() {
			fmt.Printf("  PID:       %d\n", server.PID)
			fmt.Printf("  Uptime:    %s\n", server.UptimeString())
		}
	} else {
		fmt.Println()
		fmt.Println("CURRENT SERVER")
		fmt.Println("  No server registered for this worktree")
		fmt.Println("  Use 'grove start <command>' to start one")
	}

	// All worktrees
	mainRepo := wt.Path
	if wt.IsWorktree && wt.MainWorktreePath != "" {
		mainRepo = wt.MainWorktreePath
	}

	worktrees, err := listAllWorktrees(mainRepo)
	if err == nil && len(worktrees) > 1 {
		fmt.Println()
		fmt.Println("ALL WORKTREES")
		for _, entry := range worktrees {
			indicator := "  "
			if entry.Path == wt.Path {
				indicator = "→ "
			}

			// Check if this worktree has a running server
			serverStatus := ""
			wtName := filepath.Base(entry.Path)
			if server, ok := reg.Get(wtName); ok {
				if server.IsRunning() {
					serverStatus = fmt.Sprintf(" ● %s", server.URL)
				} else {
					serverStatus = " ○ (stopped)"
				}
			}

			fmt.Printf("%s%-25s %s%s\n", indicator, entry.Name, entry.Branch, serverStatus)
		}
	}

	// All running servers
	running := reg.ListRunning()
	if len(running) > 0 {
		fmt.Println()
		fmt.Println("RUNNING SERVERS")
		for _, server := range running {
			fmt.Printf("  ● %-20s %s (port %d)\n", server.Name, server.URL, server.Port)
		}
	}

	// Proxy status
	proxy := reg.GetProxy()
	fmt.Println()
	fmt.Println("PROXY")
	if proxy.IsRunning() && isProcessRunning(proxy.PID) {
		fmt.Printf("  Status:    running (PID %d)\n", proxy.PID)
		fmt.Printf("  HTTP:      :%d\n", cfg.ProxyHTTPPort)
		fmt.Printf("  HTTPS:     :%d\n", cfg.ProxyHTTPSPort)
	} else {
		fmt.Println("  Status:    stopped")
		fmt.Println("  Start:     grove proxy start")
	}

	// Configuration
	fmt.Println()
	fmt.Println("CONFIGURATION")
	fmt.Printf("  TLD:       %s\n", cfg.TLD)
	fmt.Printf("  Config:    %s/config.yaml\n", config.ConfigDir())
	fmt.Printf("  Registry:  %s/registry.json\n", config.ConfigDir())

	fmt.Println()

	return nil
}

func outputInfoJSON(wt *worktree.Info, reg *registry.Registry) error {
	// For JSON output, build a structured response
	type WorktreeInfo struct {
		Name          string `json:"name"`
		Branch        string `json:"branch"`
		Path          string `json:"path"`
		IsCurrent     bool   `json:"is_current"`
		ServerURL     string `json:"server_url,omitempty"`
		ServerRunning bool   `json:"server_running"`
	}

	type InfoOutput struct {
		CurrentWorktree struct {
			Name       string `json:"name"`
			Branch     string `json:"branch"`
			Path       string `json:"path"`
			IsWorktree bool   `json:"is_worktree"`
			MainRepo   string `json:"main_repo,omitempty"`
		} `json:"current_worktree"`
		Worktrees      []WorktreeInfo `json:"worktrees"`
		RunningServers int            `json:"running_servers"`
		ProxyRunning   bool           `json:"proxy_running"`
	}

	output := InfoOutput{}
	output.CurrentWorktree.Name = wt.Name
	output.CurrentWorktree.Branch = wt.Branch
	output.CurrentWorktree.Path = wt.Path
	output.CurrentWorktree.IsWorktree = wt.IsWorktree
	if wt.IsWorktree {
		output.CurrentWorktree.MainRepo = wt.MainWorktreePath
	}

	output.RunningServers = len(reg.ListRunning())
	output.ProxyRunning = reg.GetProxy().IsRunning() && isProcessRunning(reg.GetProxy().PID)

	// List worktrees
	mainRepo := wt.Path
	if wt.IsWorktree && wt.MainWorktreePath != "" {
		mainRepo = wt.MainWorktreePath
	}

	worktrees, _ := listAllWorktrees(mainRepo)
	for _, entry := range worktrees {
		wtName := filepath.Base(entry.Path)
		wtInfo := WorktreeInfo{
			Name:      entry.Name,
			Branch:    entry.Branch,
			Path:      entry.Path,
			IsCurrent: entry.Path == wt.Path,
		}
		if server, ok := reg.Get(wtName); ok {
			wtInfo.ServerURL = server.URL
			wtInfo.ServerRunning = server.IsRunning()
		}
		output.Worktrees = append(output.Worktrees, wtInfo)
	}

	// Output JSON - use encoding/json
	jsonBytes, err := marshalJSON(output)
	if err != nil {
		return err
	}
	fmt.Println(string(jsonBytes))
	return nil
}

// marshalJSON marshals to indented JSON
func marshalJSON(v interface{}) ([]byte, error) {
	return json.MarshalIndent(v, "", "  ")
}

// worktreeListEntry represents a worktree from git worktree list
type worktreeListEntry struct {
	Name   string
	Path   string
	Branch string
}

// listAllWorktrees lists all worktrees for a repository
func listAllWorktrees(mainRepoPath string) ([]worktreeListEntry, error) {
	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	cmd.Dir = mainRepoPath
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var worktrees []worktreeListEntry
	var current worktreeListEntry

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "worktree ") {
			if current.Path != "" {
				current.Name = filepath.Base(current.Path)
				worktrees = append(worktrees, current)
			}
			current = worktreeListEntry{
				Path: strings.TrimPrefix(line, "worktree "),
			}
		} else if strings.HasPrefix(line, "branch ") {
			branchRef := strings.TrimPrefix(line, "branch ")
			parts := strings.Split(branchRef, "/")
			if len(parts) >= 3 {
				current.Branch = strings.Join(parts[2:], "/")
			}
		} else if line == "detached" {
			current.Branch = "(detached)"
		}
	}

	// Add the last entry
	if current.Path != "" {
		current.Name = filepath.Base(current.Path)
		worktrees = append(worktrees, current)
	}

	return worktrees, nil
}

// formatServerStatus formats a server status with color indicators
func formatServerStatus(status string) string {
	switch status {
	case "running":
		return "● running"
	case "stopped":
		return "○ stopped"
	case "crashed":
		return "✗ crashed"
	default:
		return status
	}
}
