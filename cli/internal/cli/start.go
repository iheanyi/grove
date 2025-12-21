package cli

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/iheanyi/grove/internal/port"
	"github.com/iheanyi/grove/internal/project"
	"github.com/iheanyi/grove/internal/registry"
	"github.com/iheanyi/grove/internal/worktree"
	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start [command...]",
	Short: "Start a dev server for the current worktree",
	Long: `Start a dev server for the current worktree.

If a .grove.yaml file exists and defines a command, it will be used by default.
Otherwise, you must provide a command.

Examples:
  grove start                  # Use command from .grove.yaml
  grove start bin/dev          # Start with specific command
  grove start rails s          # Start Rails server
  grove start npm run dev      # Start npm dev server`,
	RunE: runStart,
}

func init() {
	startCmd.Flags().IntP("port", "p", 0, "Override port allocation")
	startCmd.Flags().BoolP("foreground", "f", false, "Run in foreground (don't daemonize)")
}

func runStart(cmd *cobra.Command, args []string) error {
	// Detect worktree
	wt, err := worktree.Detect()
	if err != nil {
		return fmt.Errorf("failed to detect worktree: %w", err)
	}

	// Load project config if exists
	projConfig, _ := project.Load(wt.Path)

	// Determine command to run
	var command []string
	if len(args) > 0 {
		command = args
	} else if projConfig != nil && projConfig.Command != "" {
		command = []string{projConfig.Command}
	} else {
		return fmt.Errorf("no command specified and no .grove.yaml found\nUsage: grove start <command>")
	}

	// Load registry
	reg, err := registry.Load()
	if err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}

	// Check if already running
	if existing, ok := reg.Get(wt.Name); ok && existing.IsRunning() {
		return fmt.Errorf("server '%s' is already running at %s (port %d)\nUse 'grove stop' to stop it first, or 'grove restart' to restart",
			wt.Name, existing.URL, existing.Port)
	}

	// Allocate port
	portFlag, _ := cmd.Flags().GetInt("port")
	var serverPort int

	if portFlag > 0 {
		serverPort = portFlag
	} else if projConfig != nil && projConfig.Port > 0 {
		serverPort = projConfig.Port
	} else if existing, ok := reg.Get(wt.Name); ok && existing.Port > 0 {
		// Reuse existing port from stopped server
		serverPort = existing.Port
	} else {
		allocator := port.NewAllocator(cfg.PortMin, cfg.PortMax)
		serverPort, err = allocator.AllocateWithFallback(wt.Name, reg.GetUsedPorts())
		if err != nil {
			return fmt.Errorf("failed to allocate port: %w", err)
		}
	}

	// Check if port is available
	if !port.IsAvailable(serverPort) {
		return fmt.Errorf("port %d is already in use", serverPort)
	}

	// Build URL based on configured mode
	url := cfg.ServerURL(wt.Name, serverPort)

	// Run before_start hooks
	if projConfig != nil && len(projConfig.Hooks.BeforeStart) > 0 {
		fmt.Println("Running before_start hooks...")
		for _, hook := range projConfig.Hooks.BeforeStart {
			if err := runHook(hook, wt.Path); err != nil {
				return fmt.Errorf("before_start hook failed: %w", err)
			}
		}
	}

	// Create log file
	logDir := filepath.Join(cfg.LogDir)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}
	logFile := filepath.Join(logDir, fmt.Sprintf("%s.log", wt.Name))

	foreground, _ := cmd.Flags().GetBool("foreground")

	fmt.Printf("Starting server for '%s' on port %d...\n", wt.Name, serverPort)

	// Create server entry
	server := &registry.Server{
		Name:      wt.Name,
		Port:      serverPort,
		Command:   command,
		Path:      wt.Path,
		URL:       url,
		Status:    registry.StatusStarting,
		Health:    registry.HealthUnknown,
		StartedAt: time.Now(),
		Branch:    wt.Branch,
		LogFile:   logFile,
	}

	if foreground {
		// Run in foreground
		return runForeground(server, reg, projConfig)
	}

	// Run as daemon
	return runDaemon(server, reg, projConfig)
}

func runForeground(server *registry.Server, reg *registry.Registry, projConfig *project.Config) error {
	// Build command
	cmdName := server.Command[0]
	cmdArgs := server.Command[1:]

	execCmd := exec.Command(cmdName, cmdArgs...)
	execCmd.Dir = server.Path
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr
	execCmd.Stdin = os.Stdin

	// Set environment
	execCmd.Env = append(os.Environ(),
		fmt.Sprintf("PORT=%d", server.Port),
	)

	// Inject GROVE_URL (or custom var name from config)
	urlVarName := "GROVE_URL"
	if projConfig != nil && projConfig.URLVar != "" {
		urlVarName = projConfig.URLVar
	}
	execCmd.Env = append(execCmd.Env, fmt.Sprintf("%s=%s", urlVarName, server.URL))

	// Add project-specific env vars
	if projConfig != nil {
		for k, v := range projConfig.Env {
			execCmd.Env = append(execCmd.Env, fmt.Sprintf("%s=%s", k, v))
		}
	}

	// Handle signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start process
	if err := execCmd.Start(); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}

	server.PID = execCmd.Process.Pid
	server.Status = registry.StatusRunning

	// Save to registry
	if err := reg.Set(server); err != nil {
		execCmd.Process.Kill()
		return fmt.Errorf("failed to save to registry: %w", err)
	}

	// Reload proxy to pick up new route (only in subdomain mode)
	if cfg.IsSubdomainMode() {
		if err := ReloadProxy(); err != nil {
			fmt.Printf("Warning: failed to reload proxy: %v\n", err)
			fmt.Println("Run 'grove proxy stop && grove proxy start' to update routes manually")
		}
	}

	fmt.Printf("Server running at: %s\n", server.URL)
	if cfg.IsSubdomainMode() {
		fmt.Printf("Subdomains available: %s\n", cfg.SubdomainURL(server.Name))
	}
	fmt.Printf("PID: %d\n", server.PID)
	fmt.Println("Press Ctrl+C to stop...")

	// Wait for signal or process exit
	done := make(chan error, 1)
	go func() {
		done <- execCmd.Wait()
	}()

	select {
	case <-sigChan:
		fmt.Println("\nStopping server...")
		execCmd.Process.Signal(syscall.SIGTERM)

		// Wait a bit for graceful shutdown
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			execCmd.Process.Kill()
		}
	case err := <-done:
		if err != nil {
			server.Status = registry.StatusCrashed
		} else {
			server.Status = registry.StatusStopped
		}
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

	// Run after_stop hooks
	if projConfig != nil && len(projConfig.Hooks.BeforeStop) > 0 {
		for _, hook := range projConfig.Hooks.BeforeStop {
			runHook(hook, server.Path)
		}
	}

	return nil
}

func runDaemon(server *registry.Server, reg *registry.Registry, projConfig *project.Config) error {
	// Open log file
	logFile, err := os.OpenFile(server.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	// Use nohup approach: wrap the command in a shell that uses tail -f /dev/null
	// to keep stdin open forever. This prevents processes like esbuild --watch
	// from exiting due to closed stdin.
	shellCmd := fmt.Sprintf("tail -f /dev/null | exec %s", shellQuoteArgs(server.Command))

	execCmd := exec.Command("/bin/sh", "-c", shellCmd)
	execCmd.Dir = server.Path
	execCmd.Stdout = logFile
	execCmd.Stderr = logFile

	// Set environment
	execCmd.Env = append(os.Environ(),
		fmt.Sprintf("PORT=%d", server.Port),
	)

	// Inject GROVE_URL (or custom var name from config)
	urlVarName := "GROVE_URL"
	if projConfig != nil && projConfig.URLVar != "" {
		urlVarName = projConfig.URLVar
	}
	execCmd.Env = append(execCmd.Env, fmt.Sprintf("%s=%s", urlVarName, server.URL))

	// Add project-specific env vars
	if projConfig != nil {
		for k, v := range projConfig.Env {
			execCmd.Env = append(execCmd.Env, fmt.Sprintf("%s=%s", k, v))
		}
	}

	// Start as a new process group so it survives parent exit
	execCmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	// Start process
	if err := execCmd.Start(); err != nil {
		logFile.Close()
		return fmt.Errorf("failed to start server: %w", err)
	}

	server.PID = execCmd.Process.Pid
	server.Status = registry.StatusRunning

	// Save to registry
	if err := reg.Set(server); err != nil {
		execCmd.Process.Kill()
		logFile.Close()
		return fmt.Errorf("failed to save to registry: %w", err)
	}

	// Detach from process - the process will continue running
	execCmd.Process.Release()
	logFile.Close()

	// Reload proxy to pick up new route (only in subdomain mode)
	if cfg.IsSubdomainMode() {
		if err := ReloadProxy(); err != nil {
			fmt.Printf("Warning: failed to reload proxy: %v\n", err)
			fmt.Println("Run 'grove proxy stop && grove proxy start' to update routes manually")
		}
	}

	fmt.Printf("Server running at: %s\n", server.URL)
	if cfg.IsSubdomainMode() {
		fmt.Printf("Subdomains available: %s\n", cfg.SubdomainURL(server.Name))
	}
	fmt.Printf("PID: %d\n", server.PID)
	fmt.Printf("Logs: %s\n", server.LogFile)

	// Run after_start hooks
	if projConfig != nil && len(projConfig.Hooks.AfterStart) > 0 {
		fmt.Println("Running after_start hooks...")
		for _, hook := range projConfig.Hooks.AfterStart {
			if err := runHook(hook, server.Path); err != nil {
				fmt.Printf("Warning: after_start hook failed: %v\n", err)
			}
		}
	}

	return nil
}

// shellQuoteArgs quotes arguments for safe shell execution
func shellQuoteArgs(args []string) string {
	quoted := make([]string, len(args))
	for i, arg := range args {
		// Simple quoting - wrap in single quotes and escape any single quotes
		escaped := strings.ReplaceAll(arg, "'", "'\\''")
		quoted[i] = "'" + escaped + "'"
	}
	return strings.Join(quoted, " ")
}

func runHook(hook string, dir string) error {
	cmd := exec.Command("sh", "-c", hook)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
