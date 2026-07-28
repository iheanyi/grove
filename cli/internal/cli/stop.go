package cli

import (
	"fmt"
	"os"
	"syscall"
	"time"

	"github.com/iheanyi/grove/internal/project"
	"github.com/iheanyi/grove/internal/registry"
	"github.com/iheanyi/grove/internal/worktree"
	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:   "stop [name]",
	Short: "Stop a dev server",
	Long: `Stop a dev server for the current worktree or a named worktree.

If no name is provided, stops the server for the current worktree.

Examples:
  grove stop              # Stop server for current worktree
  grove stop feature-auth # Stop server by name`,
	RunE: runStop,
}

func init() {
	stopCmd.Flags().Bool("all", false, "Stop all running servers")
	stopCmd.Flags().DurationP("timeout", "t", 10*time.Second, "Timeout for graceful shutdown")
}

func runStop(cmd *cobra.Command, args []string) error {
	stopAll, _ := cmd.Flags().GetBool("all")
	timeout, _ := cmd.Flags().GetDuration("timeout")

	// Load registry
	reg, err := registry.Load()
	if err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}

	if stopAll {
		return stopAllServers(reg, timeout)
	}

	// Determine which server to stop
	var name string
	if len(args) > 0 {
		name = args[0]
	} else {
		// Use current worktree
		wt, err := worktree.Detect()
		if err != nil {
			return fmt.Errorf("failed to detect worktree: %w\nRun this command from inside a worktree, pass a server name, or use 'grove discover --register' from your repo root", err)
		}
		name = wt.Name
	}

	return stopServer(reg, name, timeout)
}

func stopServer(reg *registry.Registry, name string, timeout time.Duration) error {
	server, ok := reg.Get(name)
	if !ok {
		return fmt.Errorf("no server registered for '%s'. Run 'grove discover --register' from the repo, or 'cd <path> && grove start' from the worktree", name)
	}

	if !server.IsRunning() {
		return fmt.Errorf("server '%s' is not running. Nothing to stop. Run 'cd %s && grove start' to start it", name, server.Path)
	}

	fmt.Printf("Stopping server '%s' (PID: %d)...\n", name, server.PID)

	// Load project config for hooks
	projConfig, _ := project.Load(server.Path)

	// Run before_stop hooks
	if projConfig != nil && len(projConfig.Hooks.BeforeStop) > 0 {
		fmt.Println("Running before_stop hooks...")
		for _, hook := range projConfig.Hooks.BeforeStop {
			if err := runHook(hook, server.Path); err != nil {
				fmt.Printf("Warning: before_stop hook failed: %v\n", err)
			}
		}
	}

	// Send SIGTERM for graceful shutdown
	server.Status = registry.StatusStopping
	if err := reg.Set(server); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to update registry: %v\n", err)
	}

	if err := signalServerPID(server.PID, syscall.SIGTERM); err != nil {
		// Process might already be dead
		server.Status = registry.StatusStopped
		server.PID = 0
		server.StoppedAt = time.Now()
		if err := reg.Set(server); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to update registry: %v\n", err)
		}
		// Reload proxy to remove stale route (only in subdomain mode)
		if cfg.IsSubdomainMode() {
			if err := ReloadProxy(); err != nil {
				fmt.Printf("Warning: failed to reload proxy: %v\n", err)
			}
		}
		fmt.Println("Server stopped")
		return nil
	}

	if !waitForServerPIDExit(server.PID, timeout) {
		// Timeout, force kill
		fmt.Println("Timeout waiting for graceful shutdown, sending SIGKILL...")
		if err := signalServerPID(server.PID, syscall.SIGKILL); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to send SIGKILL: %v\n", err)
		}
		if !waitForServerPIDExit(server.PID, 5*time.Second) {
			fmt.Fprintf(os.Stderr, "Warning: server process group %d is still running\n", server.PID)
		}
	}

	// Update registry
	server.Status = registry.StatusStopped
	server.PID = 0
	server.StoppedAt = time.Now()
	if err := reg.Set(server); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to update registry: %v\n", err)
	}

	// Reload proxy to remove route (only in subdomain mode)
	if cfg.IsSubdomainMode() {
		if err := ReloadProxy(); err != nil {
			fmt.Printf("Warning: failed to reload proxy: %v\n", err)
		}
	}

	fmt.Println("Server stopped")
	return nil
}

func stopAllServers(reg *registry.Registry, timeout time.Duration) error {
	running := reg.ListRunning()
	if len(running) == 0 {
		fmt.Println("No servers running")
		return nil
	}

	fmt.Printf("Stopping %d server(s)...\n", len(running))

	var lastErr error
	for _, server := range running {
		if err := stopServerNoReload(reg, server.Name, timeout); err != nil {
			fmt.Printf("Error stopping '%s': %v\n", server.Name, err)
			lastErr = err
		}
	}

	// Reload proxy once after all servers are stopped (only in subdomain mode)
	if cfg.IsSubdomainMode() {
		if err := ReloadProxy(); err != nil {
			fmt.Printf("Warning: failed to reload proxy: %v\n", err)
		}
	}

	return lastErr
}

// stopServerNoReload stops a server without reloading the proxy (used by stopAllServers)
func stopServerNoReload(reg *registry.Registry, name string, timeout time.Duration) error {
	server, ok := reg.Get(name)
	if !ok {
		return fmt.Errorf("no server registered for '%s'. Run 'grove discover --register' from the repo, or 'cd <path> && grove start' from the worktree", name)
	}

	if !server.IsRunning() {
		return fmt.Errorf("server '%s' is not running. Nothing to stop. Run 'cd %s && grove start' to start it", name, server.Path)
	}

	fmt.Printf("Stopping server '%s' (PID: %d)...\n", name, server.PID)

	// Load project config for hooks
	projConfig, _ := project.Load(server.Path)

	// Run before_stop hooks
	if projConfig != nil && len(projConfig.Hooks.BeforeStop) > 0 {
		fmt.Println("Running before_stop hooks...")
		for _, hook := range projConfig.Hooks.BeforeStop {
			if err := runHook(hook, server.Path); err != nil {
				fmt.Printf("Warning: before_stop hook failed: %v\n", err)
			}
		}
	}

	// Send SIGTERM for graceful shutdown
	server.Status = registry.StatusStopping
	if err := reg.Set(server); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to update registry: %v\n", err)
	}

	if err := signalServerPID(server.PID, syscall.SIGTERM); err != nil {
		// Process might already be dead
		server.Status = registry.StatusStopped
		server.PID = 0
		server.StoppedAt = time.Now()
		if err := reg.Set(server); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to update registry: %v\n", err)
		}
		fmt.Printf("Server '%s' stopped\n", name)
		return nil
	}

	if !waitForServerPIDExit(server.PID, timeout) {
		// Timeout, force kill
		fmt.Printf("Timeout waiting for '%s' graceful shutdown, sending SIGKILL...\n", name)
		if err := signalServerPID(server.PID, syscall.SIGKILL); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to send SIGKILL: %v\n", err)
		}
		if !waitForServerPIDExit(server.PID, 5*time.Second) {
			fmt.Fprintf(os.Stderr, "Warning: server process group %d is still running\n", server.PID)
		}
	}

	// Update registry
	server.Status = registry.StatusStopped
	server.PID = 0
	server.StoppedAt = time.Now()
	if err := reg.Set(server); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to update registry: %v\n", err)
	}

	fmt.Printf("Server '%s' stopped\n", name)
	return nil
}
