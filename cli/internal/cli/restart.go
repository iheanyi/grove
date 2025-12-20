package cli

import (
	"fmt"
	"time"

	"github.com/iheanyi/wt/internal/registry"
	"github.com/iheanyi/wt/internal/worktree"
	"github.com/spf13/cobra"
)

var restartCmd = &cobra.Command{
	Use:   "restart [name]",
	Short: "Restart a dev server",
	Long: `Restart a dev server for the current worktree or a named worktree.

This is equivalent to running 'wt stop' followed by 'wt start'.

Examples:
  wt restart              # Restart server for current worktree
  wt restart feature-auth # Restart server by name`,
	RunE: runRestart,
}

func init() {
	restartCmd.Flags().DurationP("timeout", "t", 10*time.Second, "Timeout for graceful shutdown")
}

func runRestart(cmd *cobra.Command, args []string) error {
	timeout, _ := cmd.Flags().GetDuration("timeout")

	// Load registry
	reg, err := registry.Load()
	if err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}

	// Determine which server to restart
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

	// Get server info
	server, ok := reg.Get(name)
	if !ok {
		return fmt.Errorf("no server registered for '%s'\nUse 'wt start <command>' to start a new server", name)
	}

	if !server.IsRunning() {
		return fmt.Errorf("server '%s' is not running\nUse 'wt start' to start it", name)
	}

	// Remember the command for restart
	command := server.Command

	// Stop the server
	fmt.Println("Stopping server...")
	if err := stopServer(reg, name, timeout); err != nil {
		return fmt.Errorf("failed to stop server: %w", err)
	}

	// Wait a moment for port to be released
	time.Sleep(500 * time.Millisecond)

	// Start the server with the same command
	fmt.Println("Starting server...")
	return runStart(cmd, command)
}
