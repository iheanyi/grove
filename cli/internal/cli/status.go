package cli

import (
	"fmt"

	"github.com/iheanyi/grove/internal/port"
	"github.com/iheanyi/grove/internal/registry"
	"github.com/iheanyi/grove/internal/worktree"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status [name]",
	Short: "Show status of a server",
	Long: `Show detailed status of the current worktree's server or a named server.

Examples:
  grove status              # Show status for current worktree
  grove status feature-auth # Show status for named server`,
	RunE: runStatus,
}

func runStatus(cmd *cobra.Command, args []string) error {
	// Load registry
	reg, err := registry.Load()
	if err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}

	// Determine which server
	var name string
	if len(args) > 0 {
		name = args[0]
	} else {
		// Use current worktree
		wt, err := worktree.Detect()
		if err != nil {
			return fmt.Errorf("failed to detect worktree: %w", err)
		}
		name = wt.Name
	}

	server, ok := reg.Get(name)
	if !ok {
		fmt.Printf("Server '%s' is not registered\n", name)
		fmt.Println("\nUse 'grove start <command>' to start a server")
		return nil
	}

	// Display status
	fmt.Printf("Name:        %s\n", server.Name)
	fmt.Printf("Status:      %s\n", formatStatus(server.Status))
	fmt.Printf("URL:         %s\n", server.URL)
	if cfg.IsSubdomainMode() {
		fmt.Printf("Subdomains:  %s\n", cfg.SubdomainURL(server.Name))
	}
	fmt.Printf("Port:        %d\n", server.Port)
	fmt.Printf("Path:        %s\n", server.Path)

	if server.Branch != "" {
		fmt.Printf("Branch:      %s\n", server.Branch)
	}

	if server.IsRunning() {
		fmt.Printf("PID:         %d\n", server.PID)
		fmt.Printf("Uptime:      %s\n", server.UptimeString())

		// Check if port is actually listening
		if port.IsListening(server.Port) {
			fmt.Printf("Port Status: listening\n")
		} else {
			fmt.Printf("Port Status: not listening (server may still be starting)\n")
		}
	}

	if server.Health != "" && server.Health != registry.HealthUnknown {
		fmt.Printf("Health:      %s\n", server.Health)
	}

	if server.LogFile != "" {
		fmt.Printf("Log File:    %s\n", server.LogFile)
	}

	if !server.StartedAt.IsZero() {
		fmt.Printf("Started At:  %s\n", server.StartedAt.Format("2006-01-02 15:04:05"))
	}

	if !server.StoppedAt.IsZero() && !server.IsRunning() {
		fmt.Printf("Stopped At:  %s\n", server.StoppedAt.Format("2006-01-02 15:04:05"))
	}

	return nil
}
