package cli

import (
	"fmt"
	"os"
	"syscall"
	"time"

	"github.com/iheanyi/wt/internal/project"
	"github.com/iheanyi/wt/internal/registry"
	"github.com/iheanyi/wt/internal/worktree"
	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:   "stop [name]",
	Short: "Stop a dev server",
	Long: `Stop a dev server for the current worktree or a named worktree.

If no name is provided, stops the server for the current worktree.

Examples:
  wt stop              # Stop server for current worktree
  wt stop feature-auth # Stop server by name`,
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
			return fmt.Errorf("failed to detect worktree: %w", err)
		}
		name = wt.Name
	}

	return stopServer(reg, name, timeout)
}

func stopServer(reg *registry.Registry, name string, timeout time.Duration) error {
	server, ok := reg.Get(name)
	if !ok {
		return fmt.Errorf("no server registered for '%s'", name)
	}

	if !server.IsRunning() {
		return fmt.Errorf("server '%s' is not running", name)
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

	// Find the process
	process, err := os.FindProcess(server.PID)
	if err != nil {
		// Process doesn't exist, just update registry
		server.Status = registry.StatusStopped
		server.PID = 0
		server.StoppedAt = time.Now()
		reg.Set(server)
		// Reload proxy to remove stale route (only in subdomain mode)
		if cfg.IsSubdomainMode() {
			if err := ReloadProxy(); err != nil {
				fmt.Printf("Warning: failed to reload proxy: %v\n", err)
			}
		}
		fmt.Println("Server process not found, marking as stopped")
		return nil
	}

	// Send SIGTERM for graceful shutdown
	server.Status = registry.StatusStopping
	reg.Set(server)

	if err := process.Signal(syscall.SIGTERM); err != nil {
		// Process might already be dead
		server.Status = registry.StatusStopped
		server.PID = 0
		server.StoppedAt = time.Now()
		reg.Set(server)
		// Reload proxy to remove stale route (only in subdomain mode)
		if cfg.IsSubdomainMode() {
			if err := ReloadProxy(); err != nil {
				fmt.Printf("Warning: failed to reload proxy: %v\n", err)
			}
		}
		fmt.Println("Server stopped")
		return nil
	}

	// Wait for process to exit
	done := make(chan error, 1)
	go func() {
		_, err := process.Wait()
		done <- err
	}()

	select {
	case <-done:
		// Process exited gracefully
	case <-time.After(timeout):
		// Timeout, force kill
		fmt.Println("Timeout waiting for graceful shutdown, sending SIGKILL...")
		process.Signal(syscall.SIGKILL)
		<-done
	}

	// Update registry
	server.Status = registry.StatusStopped
	server.PID = 0
	server.StoppedAt = time.Now()
	reg.Set(server)

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
		return fmt.Errorf("no server registered for '%s'", name)
	}

	if !server.IsRunning() {
		return fmt.Errorf("server '%s' is not running", name)
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

	// Find the process
	process, err := os.FindProcess(server.PID)
	if err != nil {
		// Process doesn't exist, just update registry
		server.Status = registry.StatusStopped
		server.PID = 0
		server.StoppedAt = time.Now()
		reg.Set(server)
		fmt.Printf("Server '%s' process not found, marking as stopped\n", name)
		return nil
	}

	// Send SIGTERM for graceful shutdown
	server.Status = registry.StatusStopping
	reg.Set(server)

	if err := process.Signal(syscall.SIGTERM); err != nil {
		// Process might already be dead
		server.Status = registry.StatusStopped
		server.PID = 0
		server.StoppedAt = time.Now()
		reg.Set(server)
		fmt.Printf("Server '%s' stopped\n", name)
		return nil
	}

	// Wait for process to exit
	done := make(chan error, 1)
	go func() {
		_, err := process.Wait()
		done <- err
	}()

	select {
	case <-done:
		// Process exited gracefully
	case <-time.After(timeout):
		// Timeout, force kill
		fmt.Printf("Timeout waiting for '%s' graceful shutdown, sending SIGKILL...\n", name)
		process.Signal(syscall.SIGKILL)
		<-done
	}

	// Update registry
	server.Status = registry.StatusStopped
	server.PID = 0
	server.StoppedAt = time.Now()
	reg.Set(server)

	fmt.Printf("Server '%s' stopped\n", name)
	return nil
}
