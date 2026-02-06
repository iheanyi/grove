package cli

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/charmbracelet/x/ansi"
	"github.com/iheanyi/grove/internal/registry"
	"github.com/iheanyi/grove/internal/styles"
	"github.com/spf13/cobra"
)

var adoptCmd = &cobra.Command{
	Use:   "adopt",
	Short: "Adopt running dev servers into grove",
	Long: `Detect running dev servers and adopt them into grove's registry.

This command scans for running processes that look like dev servers,
matches them to registered worktrees by their working directory,
and updates the registry with the actual running ports.

Examples:
  grove adopt              # Detect and adopt running servers
  grove adopt --dry-run    # Show what would be adopted without making changes
  grove adopt --all        # Also show servers that couldn't be matched`,
	RunE: runAdopt,
}

func init() {
	adoptCmd.Flags().Bool("dry-run", false, "Show what would be adopted without making changes")
	adoptCmd.Flags().Bool("all", false, "Show all detected servers, including unmatched ones")
	adoptCmd.GroupID = "server"
	rootCmd.AddCommand(adoptCmd)
}

// detectedServer represents a running process that looks like a dev server
type detectedServer struct {
	PID     int
	Port    int
	Command string
	WorkDir string
	Type    string // "rails", "node", "python", etc.
}

// Allowlist patterns for dev server processes
var devServerPatterns = []struct {
	pattern *regexp.Regexp
	name    string
}{
	// Ruby/Rails
	{regexp.MustCompile(`puma`), "rails"},
	{regexp.MustCompile(`unicorn`), "rails"},
	{regexp.MustCompile(`thin\s+start`), "rails"},
	{regexp.MustCompile(`falcon`), "rails"},
	{regexp.MustCompile(`rails\s+s(erver)?`), "rails"},

	// Python
	{regexp.MustCompile(`gunicorn`), "python"},
	{regexp.MustCompile(`uvicorn`), "python"},
	{regexp.MustCompile(`daphne`), "python"},
	{regexp.MustCompile(`python.*manage\.py\s+runserver`), "python"},
	{regexp.MustCompile(`flask\s+run`), "python"},

	// Node.js - be more specific to avoid matching bundlers
	{regexp.MustCompile(`node.*astro\s+dev`), "node"},
	{regexp.MustCompile(`node.*next\s+dev`), "node"},
	{regexp.MustCompile(`node.*vite`), "node"},
	{regexp.MustCompile(`node.*webpack-dev-server`), "node"},
	{regexp.MustCompile(`node.*webpack\s+serve`), "node"},
	{regexp.MustCompile(`node.*react-scripts\s+start`), "node"},
	{regexp.MustCompile(`node.*nuxt\s+dev`), "node"},
	{regexp.MustCompile(`node.*svelte-kit\s+dev`), "node"},
	{regexp.MustCompile(`node.*remix\s+dev`), "node"},
	{regexp.MustCompile(`npm\s+run\s+(dev|start|serve)`), "node"},
	{regexp.MustCompile(`yarn\s+(dev|start|serve)`), "node"},
	{regexp.MustCompile(`pnpm\s+(dev|start|serve)`), "node"},
	{regexp.MustCompile(`bun\s+(dev|start|serve|run\s+dev)`), "node"},

	// PHP
	{regexp.MustCompile(`php\s+-S`), "php"},
	{regexp.MustCompile(`php\s+artisan\s+serve`), "php"},

	// Go
	{regexp.MustCompile(`go\s+run\s+.*main\.go`), "go"},
	{regexp.MustCompile(`air`), "go"}, // air is a popular Go live-reload tool

	// Generic
	{regexp.MustCompile(`bin/dev`), "generic"},
	{regexp.MustCompile(`foreman\s+start`), "generic"},
	{regexp.MustCompile(`overmind\s+start`), "generic"},
}

// Blocklist patterns - exclude even if they match allowlist
var excludePatterns = []*regexp.Regexp{
	regexp.MustCompile(`rubocop`),
	regexp.MustCompile(`solargraph`),
	regexp.MustCompile(`language-server`),
	regexp.MustCompile(`eslint_d`),
	regexp.MustCompile(`prettier`),
	regexp.MustCompile(`typescript-language-server`),
	regexp.MustCompile(`gopls`),
	regexp.MustCompile(`rust-analyzer`),
	regexp.MustCompile(`pyright`),
	regexp.MustCompile(`pylsp`),
	// esbuild in watch mode is a bundler, not a server
	regexp.MustCompile(`esbuild.*--watch`),
}

func runAdopt(cmd *cobra.Command, args []string) error {
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	showAll, _ := cmd.Flags().GetBool("all")

	// Load registry
	reg, err := registry.Load()
	if err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}

	// Detect running servers
	servers, err := detectRunningServers()
	if err != nil {
		return fmt.Errorf("failed to detect servers: %w", err)
	}

	if len(servers) == 0 {
		fmt.Println("No running dev servers detected.")
		return nil
	}

	// Match servers to worktrees
	type matchedServer struct {
		server    detectedServer
		worktree  string
		oldPort   int
		isRunning bool
	}

	matchedMap := make(map[string]matchedServer) // keyed by worktree name
	var unmatched []matchedServer

	for _, srv := range servers {
		// Find matching worktree by path
		found := false
		for _, wt := range reg.ListWorktrees() {
			if wt.Path == srv.WorkDir {
				// Check if already registered as a server
				existingServer, hasServer := reg.Get(wt.Name)
				oldPort := 0
				isRunning := false
				if hasServer {
					oldPort = existingServer.Port
					isRunning = existingServer.IsRunning()
				}

				// Only keep one server per worktree - prefer lower port (main server)
				if existing, exists := matchedMap[wt.Name]; exists {
					// Keep the one with the lower port (typically the main server)
					if srv.Port < existing.server.Port {
						matchedMap[wt.Name] = matchedServer{srv, wt.Name, oldPort, isRunning}
					}
				} else {
					matchedMap[wt.Name] = matchedServer{srv, wt.Name, oldPort, isRunning}
				}
				found = true
				break
			}
		}

		if !found {
			unmatched = append(unmatched, matchedServer{srv, "", 0, false})
		}
	}

	// Convert map to slice
	var matched []matchedServer
	for _, m := range matchedMap {
		matched = append(matched, m)
	}

	// Display results
	if len(matched) > 0 {
		fmt.Printf("Found %d running dev servers matching registered worktrees:\n\n", len(matched))
		fmt.Printf("%-*s %-8s %-8s %-*s %s\n",
			styles.ColWidthWorktree, "WORKTREE", "PORT", "OLD",
			styles.ColWidthType, "TYPE", "STATUS")
		fmt.Println(strings.Repeat("-", styles.SeparatorMedium))

		for _, m := range matched {
			status := "new"
			if m.isRunning && m.oldPort == m.server.Port {
				status = "already adopted"
			} else if m.oldPort > 0 {
				status = fmt.Sprintf("port change (%d→%d)", m.oldPort, m.server.Port)
			}

			oldPortStr := "-"
			if m.oldPort > 0 {
				oldPortStr = fmt.Sprintf("%d", m.oldPort)
			}

			fmt.Printf("%-*s %-8d %-8s %-*s %s\n",
				styles.ColWidthWorktree, ansi.Truncate(m.worktree, styles.ColWidthWorktree, styles.TruncateTail),
				m.server.Port,
				oldPortStr,
				styles.ColWidthType, m.server.Type,
				status,
			)
		}
	}

	if showAll && len(unmatched) > 0 {
		fmt.Printf("\n\nFound %d running servers without matching worktrees:\n\n", len(unmatched))
		fmt.Printf("%-8s %-*s %-*s\n", "PORT", styles.ColWidthType, "TYPE", styles.ColWidthWorkDir, "WORKING DIRECTORY")
		fmt.Println(strings.Repeat("-", styles.SeparatorShort))

		for _, u := range unmatched {
			fmt.Printf("%-8d %-*s %-*s\n",
				u.server.Port,
				styles.ColWidthType, u.server.Type,
				styles.ColWidthWorkDir, ansi.Truncate(u.server.WorkDir, styles.ColWidthWorkDir, styles.TruncateTail),
			)
		}
		fmt.Println("\nTip: Run 'grove discover <path> --register' to register these directories first.")
	}

	if len(matched) == 0 {
		fmt.Println("No running servers matched registered worktrees.")
		if !showAll && len(unmatched) > 0 {
			fmt.Printf("\n%d servers found but not matched. Use --all to see them.\n", len(unmatched))
		}
		return nil
	}

	if dryRun {
		fmt.Println("\n--dry-run specified, no changes made.")
		return nil
	}

	// Adopt the servers
	fmt.Println("\nAdopting servers...")
	adopted := 0

	for _, m := range matched {
		// Skip if already adopted with same port
		if m.isRunning && m.oldPort == m.server.Port {
			continue
		}

		// Get or create server entry
		server, exists := reg.Get(m.worktree)
		if !exists {
			// Create new server entry
			wt, _ := reg.GetWorktree(m.worktree)
			server = &registry.Server{
				Name:   m.worktree,
				Path:   wt.Path,
				Branch: wt.Branch,
			}
		}

		// Update with detected info
		server.Port = m.server.Port
		server.PID = m.server.PID
		server.Status = registry.StatusRunning
		server.URL = cfg.ServerURL(server.Name, server.Port)

		if err := reg.Set(server); err != nil {
			fmt.Printf("  ✗ %s: %v\n", m.worktree, err)
			continue
		}

		fmt.Printf("  ✓ %s (port %d)\n", m.worktree, m.server.Port)
		adopted++
	}

	fmt.Printf("\nAdopted %d servers.\n", adopted)
	return nil
}

// detectRunningServers finds processes that look like dev servers
func detectRunningServers() ([]detectedServer, error) {
	// Use lsof to find listening TCP connections on dev ports (3000-49151)
	// We exclude ephemeral ports (49152-65535) which are typically background tools
	cmd := exec.Command("lsof", "-iTCP", "-sTCP:LISTEN", "-P", "-n")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run lsof: %w", err)
	}

	var servers []detectedServer
	seen := make(map[string]bool) // Dedupe by workdir+port

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		// Parse lsof output
		// Format: COMMAND PID USER FD TYPE DEVICE SIZE/OFF NODE NAME (STATE)
		// Example: ruby 3101 iheanyi 7u IPv4 0x... 0t0 TCP 127.0.0.1:3179 (LISTEN)
		fields := strings.Fields(line)
		if len(fields) < 10 {
			continue
		}

		processName := fields[0]
		pidStr := fields[1]

		// Skip header line
		if processName == "COMMAND" {
			continue
		}

		// Skip non-dev processes by name
		if !isDevProcessName(processName) {
			continue
		}

		// Extract port from the NAME field (field 9, like "127.0.0.1:3000" or "[::1]:3000")
		nameField := fields[8] // 0-indexed, so field 9 is index 8
		port := extractPort(nameField)
		if port == 0 {
			continue
		}

		// Filter to dev port range (3000-49151)
		if port < 3000 || port > 49151 {
			continue
		}

		pid, _ := strconv.Atoi(pidStr)

		// Get full command and working directory
		fullCmd := getProcessCommand(pid)
		workDir := getProcessWorkDir(pid)

		if workDir == "" || workDir == "/" {
			continue
		}

		// Check against allowlist
		serverType := matchDevServer(fullCmd)
		if serverType == "" {
			continue
		}

		// Check against blocklist
		if isExcluded(fullCmd) {
			continue
		}

		// Dedupe
		key := fmt.Sprintf("%s:%d", workDir, port)
		if seen[key] {
			continue
		}
		seen[key] = true

		servers = append(servers, detectedServer{
			PID:     pid,
			Port:    port,
			Command: fullCmd,
			WorkDir: workDir,
			Type:    serverType,
		})
	}

	return servers, nil
}

// isDevProcessName does a quick filter on process name
func isDevProcessName(name string) bool {
	devNames := []string{"ruby", "node", "python", "python3", "php", "go", "java", "deno", "bun"}
	name = strings.ToLower(name)
	for _, dev := range devNames {
		if name == dev {
			return true
		}
	}
	return false
}

// extractPort gets the port number from lsof NAME field
func extractPort(nameField string) int {
	// Format is typically "localhost:3000" or "*:3000" or "[::1]:3000"
	idx := strings.LastIndex(nameField, ":")
	if idx == -1 {
		return 0
	}
	portStr := nameField[idx+1:]
	port, _ := strconv.Atoi(portStr)
	return port
}

// getProcessCommand gets the full command line for a process
func getProcessCommand(pid int) string {
	cmd := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "command=")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

// getProcessWorkDir gets the working directory for a process
func getProcessWorkDir(pid int) string {
	cmd := exec.Command("lsof", "-p", strconv.Itoa(pid))
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "cwd") {
			fields := strings.Fields(line)
			if len(fields) >= 9 {
				// The path is the last field
				return fields[len(fields)-1]
			}
		}
	}
	return ""
}

// matchDevServer checks if command matches any dev server pattern
func matchDevServer(command string) string {
	for _, p := range devServerPatterns {
		if p.pattern.MatchString(command) {
			return p.name
		}
	}
	return ""
}

// isExcluded checks if command matches any exclude pattern
func isExcluded(command string) bool {
	for _, p := range excludePatterns {
		if p.MatchString(command) {
			return true
		}
	}
	return false
}
