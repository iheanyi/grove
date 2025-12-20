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

	"github.com/iheanyi/grove/internal/config"
	"github.com/iheanyi/grove/internal/registry"
	"github.com/spf13/cobra"
)

var proxyCmd = &cobra.Command{
	Use:   "proxy",
	Short: "Manage the reverse proxy daemon",
	Long: `Manage the reverse proxy daemon that handles routing to your dev servers.

The proxy provides:
- Clean URLs like https://feature-branch.localhost
- Wildcard subdomains for multi-tenant apps
- Automatic HTTPS with local certificates

Examples:
  grove proxy start   # Start the proxy daemon
  grove proxy stop    # Stop the proxy daemon
  grove proxy status  # Check proxy status
  grove proxy routes  # List all registered routes`,
}

var proxyStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the reverse proxy daemon",
	RunE:  runProxyStart,
}

var proxyStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the reverse proxy daemon",
	RunE:  runProxyStop,
}

var proxyStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check proxy status",
	RunE:  runProxyStatus,
}

var proxyRoutesCmd = &cobra.Command{
	Use:   "routes",
	Short: "List all registered routes",
	RunE:  runProxyRoutes,
}

func init() {
	proxyCmd.AddCommand(proxyStartCmd)
	proxyCmd.AddCommand(proxyStopCmd)
	proxyCmd.AddCommand(proxyStatusCmd)
	proxyCmd.AddCommand(proxyRoutesCmd)

	proxyStartCmd.Flags().BoolP("foreground", "f", false, "Run in foreground")
}

func runProxyStart(cmd *cobra.Command, args []string) error {
	// Warn if in port mode
	if !cfg.IsSubdomainMode() {
		fmt.Println("Note: URL mode is set to 'port'. The proxy is only needed for 'subdomain' mode.")
		fmt.Println("To use subdomain mode, set 'url_mode: subdomain' in ~/.config/wt/config.yaml")
		fmt.Println()
	}

	foreground, _ := cmd.Flags().GetBool("foreground")

	// Load registry to check if already running
	reg, err := registry.Load()
	if err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}

	proxy := reg.GetProxy()
	if proxy.IsRunning() && isProcessRunning(proxy.PID) {
		return fmt.Errorf("proxy is already running (PID: %d)\nUse 'grove proxy stop' to stop it first", proxy.PID)
	}

	fmt.Printf("Starting proxy on :%d/:%d...\n", cfg.ProxyHTTPPort, cfg.ProxyHTTPSPort)

	if foreground {
		return runProxyForeground(reg)
	}

	return runProxyDaemon(reg)
}

func runProxyForeground(reg *registry.Registry) error {
	// Generate Caddyfile
	caddyfilePath, err := generateCaddyfile(reg)
	if err != nil {
		return fmt.Errorf("failed to generate Caddyfile: %w", err)
	}

	// Find caddy binary
	caddyPath, err := exec.LookPath("caddy")
	if err != nil {
		return fmt.Errorf("caddy not found in PATH. Install with: brew install caddy")
	}

	// Start caddy
	cmd := exec.Command(caddyPath, "run", "--config", caddyfilePath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start caddy: %w", err)
	}

	// Update registry
	proxy := &registry.ProxyInfo{
		PID:       cmd.Process.Pid,
		StartedAt: time.Now(),
		HTTPPort:  cfg.ProxyHTTPPort,
		HTTPSPort: cfg.ProxyHTTPSPort,
	}
	reg.UpdateProxy(proxy)

	fmt.Printf("Proxy running (PID: %d)\n", proxy.PID)
	fmt.Println("Press Ctrl+C to stop...")

	// Handle signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait for caddy to exit or signal
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-sigChan:
		fmt.Println("\nStopping proxy...")
		cmd.Process.Signal(syscall.SIGTERM)
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			cmd.Process.Kill()
		}
	case err := <-done:
		if err != nil {
			return fmt.Errorf("caddy exited with error: %w", err)
		}
	}

	proxy.PID = 0
	reg.UpdateProxy(proxy)

	return nil
}

func generateCaddyfile(reg *registry.Registry) (string, error) {
	caddyfilePath := filepath.Join(config.ConfigDir(), "Caddyfile")

	var sb strings.Builder

	// Global options
	sb.WriteString("{\n")
	sb.WriteString("\tlocal_certs\n")
	sb.WriteString("\tauto_https disable_redirects\n")
	sb.WriteString("}\n\n")

	// Reload registry to get latest data
	freshReg, err := registry.Load()
	if err != nil {
		fmt.Printf("Warning: failed to reload registry: %v\n", err)
	} else {
		reg = freshReg
	}

	// Get all servers (both running and stopped - for routing)
	servers := reg.List()

	if len(servers) == 0 {
		// Default fallback when no servers
		sb.WriteString(fmt.Sprintf("https://*.%s {\n", cfg.TLD))
		sb.WriteString("\trespond \"No server registered for this domain\" 503\n")
		sb.WriteString("}\n")
	} else {
		// Generate route for each server
		for _, server := range servers {
			// Main domain
			sb.WriteString(fmt.Sprintf("https://%s.%s {\n", server.Name, cfg.TLD))
			sb.WriteString(fmt.Sprintf("\treverse_proxy localhost:%d\n", server.Port))
			sb.WriteString("}\n\n")

			// Wildcard subdomains
			sb.WriteString(fmt.Sprintf("https://*.%s.%s {\n", server.Name, cfg.TLD))
			sb.WriteString(fmt.Sprintf("\treverse_proxy localhost:%d\n", server.Port))
			sb.WriteString("}\n\n")
		}
	}

	if err := os.WriteFile(caddyfilePath, []byte(sb.String()), 0644); err != nil {
		return "", fmt.Errorf("failed to write Caddyfile: %w", err)
	}

	return caddyfilePath, nil
}

func runProxyDaemon(reg *registry.Registry) error {
	// Start as a background process
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable: %w", err)
	}

	cmd := exec.Command(executable, "proxy", "start", "--foreground")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	// Redirect output to log file
	logFile, err := os.OpenFile(
		config.ConfigDir()+"/proxy.log",
		os.O_CREATE|os.O_WRONLY|os.O_APPEND,
		0644,
	)
	if err != nil {
		return fmt.Errorf("failed to open proxy log: %w", err)
	}
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	if err := cmd.Start(); err != nil {
		logFile.Close()
		return fmt.Errorf("failed to start proxy: %w", err)
	}

	// Detach
	cmd.Process.Release()
	logFile.Close()

	// Update registry
	proxy := &registry.ProxyInfo{
		PID:       cmd.Process.Pid,
		StartedAt: time.Now(),
		HTTPPort:  cfg.ProxyHTTPPort,
		HTTPSPort: cfg.ProxyHTTPSPort,
	}
	reg.UpdateProxy(proxy)

	fmt.Printf("Proxy started (PID: %d)\n", proxy.PID)
	fmt.Printf("Logs: %s/proxy.log\n", config.ConfigDir())

	return nil
}

func runProxyStop(cmd *cobra.Command, args []string) error {
	// Load registry
	reg, err := registry.Load()
	if err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}

	proxy := reg.GetProxy()
	if !proxy.IsRunning() {
		fmt.Println("Proxy is not running")
		return nil
	}

	fmt.Printf("Stopping proxy (PID: %d)...\n", proxy.PID)

	// Find process
	process, err := os.FindProcess(proxy.PID)
	if err != nil {
		// Process doesn't exist
		proxy.PID = 0
		reg.UpdateProxy(proxy)
		fmt.Println("Proxy process not found, marking as stopped")
		return nil
	}

	// Send SIGTERM
	if err := process.Signal(syscall.SIGTERM); err != nil {
		proxy.PID = 0
		reg.UpdateProxy(proxy)
		fmt.Println("Proxy stopped")
		return nil
	}

	// Wait for exit
	done := make(chan error, 1)
	go func() {
		_, err := process.Wait()
		done <- err
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		process.Signal(syscall.SIGKILL)
		<-done
	}

	proxy.PID = 0
	reg.UpdateProxy(proxy)

	fmt.Println("Proxy stopped")
	return nil
}

func runProxyStatus(cmd *cobra.Command, args []string) error {
	// Load registry
	reg, err := registry.Load()
	if err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}

	proxy := reg.GetProxy()

	if proxy.IsRunning() && isProcessRunning(proxy.PID) {
		fmt.Printf("Status:     running\n")
		fmt.Printf("PID:        %d\n", proxy.PID)
		fmt.Printf("HTTP Port:  %d\n", proxy.HTTPPort)
		fmt.Printf("HTTPS Port: %d\n", proxy.HTTPSPort)
		fmt.Printf("Started At: %s\n", proxy.StartedAt.Format("2006-01-02 15:04:05"))
	} else {
		fmt.Println("Status: stopped")
		fmt.Println("\nUse 'grove proxy start' to start the proxy")
	}

	return nil
}

func runProxyRoutes(cmd *cobra.Command, args []string) error {
	// Load registry
	reg, err := registry.Load()
	if err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}

	servers := reg.ListRunning()
	if len(servers) == 0 {
		fmt.Println("No routes registered")
		fmt.Println("\nStart a server with 'grove start' to register routes")
		return nil
	}

	fmt.Println("Registered Routes:")
	fmt.Println()

	for _, s := range servers {
		fmt.Printf("  %s.%s -> localhost:%d\n", s.Name, cfg.TLD, s.Port)
		fmt.Printf("  *.%s.%s -> localhost:%d\n", s.Name, cfg.TLD, s.Port)
		fmt.Println()
	}

	return nil
}

func isProcessRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// ReloadProxy regenerates the Caddyfile and reloads Caddy to pick up new routes.
// This should be called whenever servers are started or stopped.
func ReloadProxy() error {
	// Load registry to check if proxy is running
	reg, err := registry.Load()
	if err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}

	proxy := reg.GetProxy()
	if !proxy.IsRunning() || !isProcessRunning(proxy.PID) {
		// Proxy not running, nothing to reload
		return nil
	}

	// Regenerate Caddyfile with current servers
	caddyfilePath, err := generateCaddyfile(reg)
	if err != nil {
		return fmt.Errorf("failed to generate Caddyfile: %w", err)
	}

	// Find caddy binary
	caddyPath, err := exec.LookPath("caddy")
	if err != nil {
		return fmt.Errorf("caddy not found in PATH: %w", err)
	}

	// Reload Caddy with new config
	cmd := exec.Command(caddyPath, "reload", "--config", caddyfilePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to reload caddy: %w\nOutput: %s", err, string(output))
	}

	return nil
}
