package cli

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/iheanyi/grove/internal/port"
	"github.com/iheanyi/grove/internal/registry"
	"github.com/iheanyi/grove/internal/worktree"
	"github.com/spf13/cobra"
)

var attachCmd = &cobra.Command{
	Use:   "attach <port>",
	Short: "Attach to an already running dev server",
	Long: `Attach to a dev server that is already running on the specified port.

This is useful when:
- You started a server outside of grove (e.g., directly with npm run dev)
- You want to add a running server to the grove proxy

The server will be registered and routed through the proxy like normal.

Examples:
  grove attach 3000                    # Attach to server on port 3000
  grove attach 3000 --name my-server   # Use custom name
  grove attach 8080 --url /api         # Only route /api paths`,
	Args: cobra.ExactArgs(1),
	RunE: runAttach,
}

func init() {
	attachCmd.Flags().StringP("name", "n", "", "Custom name for the server (default: worktree name)")
	attachCmd.Flags().String("url", "", "Only route requests matching this path prefix")
	attachCmd.Flags().Int("pid", 0, "Specify the PID of the running process (for tracking)")
}

func runAttach(cmd *cobra.Command, args []string) error {
	portStr := args[0]
	customName, _ := cmd.Flags().GetString("name")
	urlPrefix, _ := cmd.Flags().GetString("url")
	pid, _ := cmd.Flags().GetInt("pid")

	// Parse port
	portNum, err := strconv.Atoi(portStr)
	if err != nil {
		return fmt.Errorf("invalid port number: %s", portStr)
	}

	// Validate port range
	if portNum < 1 || portNum > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}

	// Check if something is actually listening on the port
	if !port.IsListening(portNum) {
		return fmt.Errorf("no server found listening on port %d", portNum)
	}

	// Determine server name
	name := customName
	if name == "" {
		wt, err := worktree.Detect()
		if err != nil {
			return fmt.Errorf("failed to detect worktree (use --name to specify): %w", err)
		}
		name = wt.Name
	}

	// Load registry
	reg, err := registry.Load()
	if err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}

	// Check if already registered
	if existing, ok := reg.Get(name); ok {
		if existing.Port == portNum {
			fmt.Printf("Server '%s' is already registered on port %d\n", name, portNum)
			return nil
		}
		// Only block if the server is actually running
		if existing.IsRunning() {
			return fmt.Errorf("server '%s' is already running on port %d (stop it first or use a different name)", name, existing.Port)
		}
		// Server exists but is stopped - we can overwrite it
		fmt.Printf("Note: Overwriting stopped server '%s' (was on port %d)\n", name, existing.Port)
	}

	// Check if port is already registered by another RUNNING server
	for _, s := range reg.List() {
		if s.Port == portNum && s.Name != name && s.IsRunning() {
			return fmt.Errorf("port %d is already in use by running server '%s'", portNum, s.Name)
		}
	}

	// Get worktree info for path and branch
	var path, branch string
	wt, err := worktree.Detect()
	if err == nil {
		path = wt.Path
		branch = wt.Branch
	}

	// Try to find the PID if not specified
	if pid == 0 {
		pid = findPIDOnPort(portNum)
	}

	// Create server entry
	server := &registry.Server{
		Name:      name,
		Port:      portNum,
		Path:      path,
		Branch:    branch,
		Status:    registry.StatusRunning,
		PID:       pid,
		URL:       cfg.ServerURL(name, portNum),
		StartedAt: time.Now(),
		Health:    registry.HealthUnknown,
	}

	// Store URL prefix info in command for reference
	if urlPrefix != "" {
		server.Command = []string{"[attached]", fmt.Sprintf("prefix:%s", urlPrefix)}
	} else {
		server.Command = []string{"[attached]"}
	}

	// Register the server
	if err := reg.Set(server); err != nil {
		return fmt.Errorf("failed to register server: %w", err)
	}

	if err := reg.Save(); err != nil {
		return fmt.Errorf("failed to save registry: %w", err)
	}

	fmt.Printf("✓ Attached to server on port %d\n", portNum)
	fmt.Printf("  Name:   %s\n", name)
	fmt.Printf("  URL:    %s\n", server.URL)
	if urlPrefix != "" {
		fmt.Printf("  Prefix: %s\n", urlPrefix)
	}
	if pid > 0 {
		fmt.Printf("  PID:    %d\n", pid)
	} else {
		fmt.Println("  PID:    unknown (server won't be tracked for lifecycle)")
	}

	// Check if proxy is running (only relevant in subdomain mode)
	if cfg.IsSubdomainMode() {
		proxy := reg.GetProxy()
		if !proxy.IsRunning() || !isProcessRunning(proxy.PID) {
			fmt.Println()
			fmt.Println("Note: The proxy is not running. Start it with: grove proxy start")
		}
	}

	return nil
}

// findPIDOnPort attempts to find the PID of a process listening on the given port.
// Uses lsof on macOS/Linux which is reliable and commonly available on dev machines.
func findPIDOnPort(targetPort int) int {
	// lsof -ti:PORT returns PIDs listening on that port (one per line)
	// -t = terse output (PIDs only)
	// -i:PORT = filter by port
	// -sTCP:LISTEN = only show listening sockets (not established connections)
	cmd := exec.Command("lsof", "-ti", fmt.Sprintf(":%d", targetPort), "-sTCP:LISTEN")
	output, err := cmd.Output()
	if err != nil {
		// lsof failed or not found - try without -sTCP:LISTEN for older versions
		cmd = exec.Command("lsof", "-ti", fmt.Sprintf(":%d", targetPort))
		output, err = cmd.Output()
		if err != nil {
			return 0
		}
	}

	// Parse output - may have multiple PIDs (parent/child, SO_REUSEPORT)
	// Take the first one, which is typically the main/parent process
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 0 || lines[0] == "" {
		return 0
	}

	pid, err := strconv.Atoi(strings.TrimSpace(lines[0]))
	if err != nil {
		return 0
	}

	return pid
}

// DetachCmd removes a server from tracking without stopping it
var detachCmd = &cobra.Command{
	Use:   "detach [name...]",
	Short: "Detach one or more servers from grove tracking",
	Long: `Detach servers from grove tracking without stopping the processes.

This removes the servers from the registry and proxy but leaves
the actual processes running.

Examples:
  grove detach                  # Detach current worktree's server
  grove detach my-server        # Detach named server
  grove detach server1 server2  # Detach multiple servers at once`,
	RunE: runDetach,
}

func runDetach(cmd *cobra.Command, args []string) error {
	// Determine server names
	var names []string
	if len(args) > 0 {
		names = args
	} else {
		wt, err := worktree.Detect()
		if err != nil {
			return fmt.Errorf("failed to detect worktree: %w", err)
		}
		names = []string{wt.Name}
	}

	// Load registry once
	reg, err := registry.Load()
	if err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}

	// Process all names
	var detachedServers []string
	var detachedWorktrees []string
	var runningPIDs []struct {
		name string
		pid  int
	}
	var notFound []string

	for _, name := range names {
		// Check if registered as a server
		server, isServer := reg.Get(name)
		_, isWorktree := reg.GetWorktree(name)

		if !isServer && !isWorktree {
			notFound = append(notFound, name)
			continue
		}

		// Remove from servers if present (without saving yet)
		if isServer {
			reg.RemoveWithoutSave(name)
			detachedServers = append(detachedServers, name)
			if server.IsRunning() {
				runningPIDs = append(runningPIDs, struct {
					name string
					pid  int
				}{name, server.PID})
			}
		}

		// Remove from worktrees if present (without saving yet)
		if isWorktree {
			reg.RemoveWorktreeWithoutSave(name)
			detachedWorktrees = append(detachedWorktrees, name)
		}
	}

	// Save once after all removals
	if len(detachedServers) > 0 || len(detachedWorktrees) > 0 {
		if err := reg.Save(); err != nil {
			return fmt.Errorf("failed to save registry: %w", err)
		}
	}

	// Print results
	for _, name := range detachedServers {
		fmt.Printf("✓ Detached server '%s'\n", name)
	}
	for _, name := range detachedWorktrees {
		if !contains(detachedServers, name) {
			fmt.Printf("✓ Removed worktree '%s' from registry\n", name)
		}
	}
	for _, rp := range runningPIDs {
		fmt.Printf("  Note: Process for '%s' (PID %d) is still running\n", rp.name, rp.pid)
	}
	for _, name := range notFound {
		fmt.Printf("⚠ '%s' is not registered as a server or worktree\n", name)
	}

	if len(runningPIDs) > 0 {
		fmt.Println("  Use 'kill' to stop running processes if needed")
	}

	return nil
}

// contains checks if a string is in a slice
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
