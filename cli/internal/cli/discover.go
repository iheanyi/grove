package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/iheanyi/grove/internal/port"
	"github.com/iheanyi/grove/internal/registry"
	"github.com/iheanyi/grove/internal/worktree"
	"github.com/spf13/cobra"
)

var discoverCmd = &cobra.Command{
	Use:   "discover [path]",
	Short: "Discover git worktrees in a directory",
	Long: `Scan a directory for git repositories and worktrees.

By default, scans the current directory and its subdirectories (1 level deep).
Use --depth to scan deeper, or --recursive for unlimited depth.

Examples:
  grove discover                    # Scan current directory
  grove discover ~/development      # Scan specific directory
  grove discover --depth 2          # Scan 2 levels deep
  grove discover --register         # Register all discovered worktrees
  grove discover --register --start # Register and start all with default command`,
	RunE: runDiscover,
}

func init() {
	discoverCmd.Flags().IntP("depth", "d", 1, "How many directory levels to scan (0 = current only)")
	discoverCmd.Flags().BoolP("recursive", "r", false, "Scan recursively (unlimited depth)")
	discoverCmd.Flags().Bool("register", false, "Register all discovered worktrees")
	discoverCmd.Flags().Bool("start", false, "Start all discovered worktrees (implies --register)")
	discoverCmd.Flags().StringP("command", "c", "", "Command to use when starting (default: from .grove.yaml or prompt)")
	rootCmd.AddCommand(discoverCmd)
}

type discoveredWorktree struct {
	Path       string
	Name       string
	Branch     string
	IsWorktree bool // true if linked worktree, false if main repo
	HasConfig  bool // true if .grove.yaml exists
	Registered bool // true if already in registry
	Running    bool // true if currently running
	Port       int  // allocated port if registered
}

func runDiscover(cmd *cobra.Command, args []string) error {
	scanPath := "."
	if len(args) > 0 {
		scanPath = args[0]
	}

	absPath, err := filepath.Abs(scanPath)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	depth, _ := cmd.Flags().GetInt("depth")
	recursive, _ := cmd.Flags().GetBool("recursive")
	register, _ := cmd.Flags().GetBool("register")
	start, _ := cmd.Flags().GetBool("start")
	command, _ := cmd.Flags().GetString("command")

	if start {
		register = true
	}

	if recursive {
		depth = -1 // unlimited
	}

	fmt.Printf("Scanning %s for git repositories...\n\n", absPath)

	// Load registry to check existing entries
	reg, err := registry.Load()
	if err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}

	// Discover worktrees
	discovered := discoverWorktrees(absPath, depth, reg)

	if len(discovered) == 0 {
		fmt.Println("No git repositories found.")
		return nil
	}

	// Display results
	fmt.Printf("Found %d git repositories:\n\n", len(discovered))
	fmt.Printf("%-35s %-25s %-10s %-6s %s\n", "NAME", "BRANCH", "STATUS", "PORT", "PATH")
	fmt.Println(strings.Repeat("-", 120))

	for _, wt := range discovered {
		status := "new"
		if wt.Running {
			status = "● running"
		} else if wt.Registered {
			status = "○ stopped"
		}

		portStr := "-"
		if wt.Port > 0 {
			portStr = fmt.Sprintf("%d", wt.Port)
		}

		configMarker := ""
		if wt.HasConfig {
			configMarker = "*"
		}

		// Show relative path from base
		relPath := wt.Path
		if rel, err := filepath.Rel(absPath, wt.Path); err == nil {
			relPath = rel
		}

		fmt.Printf("%-35s %-25s %-10s %-6s %s\n",
			truncate(wt.Name, 35),
			truncate(wt.Branch+configMarker, 25),
			status,
			portStr,
			truncate(relPath, 40),
		)
	}

	fmt.Println()

	// Count new ones
	newCount := 0
	for _, wt := range discovered {
		if !wt.Registered {
			newCount++
		}
	}

	if newCount == 0 {
		fmt.Println("All discovered repositories are already registered.")
		return nil
	}

	fmt.Printf("Found %d new repositories.\n", newCount)

	if !register {
		fmt.Println("\nRun with --register to add them to wt, or --register --start to also start them.")
		return nil
	}

	// Register new worktrees
	fmt.Println("\nRegistering new repositories...")

	allocator := port.NewAllocator(cfg.PortMin, cfg.PortMax)

	for _, wt := range discovered {
		if wt.Registered {
			continue
		}

		// Allocate port
		serverPort, err := allocator.AllocateWithFallback(wt.Name, reg.GetUsedPorts())
		if err != nil {
			fmt.Printf("  ✗ %s: failed to allocate port: %v\n", wt.Name, err)
			continue
		}

		// Determine command
		cmdToUse := command
		if cmdToUse == "" && wt.HasConfig {
			// Try to load from .grove.yaml
			// For now, just note it has a config
			cmdToUse = "" // Will be loaded when starting
		}

		// Create server entry
		server := &registry.Server{
			Name:   wt.Name,
			Port:   serverPort,
			Path:   wt.Path,
			URL:    fmt.Sprintf("https://%s.%s", wt.Name, cfg.TLD),
			Status: registry.StatusStopped,
			Branch: wt.Branch,
		}

		if err := reg.Set(server); err != nil {
			fmt.Printf("  ✗ %s: failed to register: %v\n", wt.Name, err)
			continue
		}

		fmt.Printf("  ✓ %s (port %d)\n", wt.Name, serverPort)

		if start && cmdToUse != "" {
			// Start the server
			fmt.Printf("    Starting with: %s\n", cmdToUse)
			startCmd := exec.Command("grove", "start", cmdToUse)
			startCmd.Dir = wt.Path
			if err := startCmd.Run(); err != nil {
				fmt.Printf("    ✗ Failed to start: %v\n", err)
			}
		}
	}

	fmt.Println("\nDone!")
	return nil
}

func discoverWorktrees(basePath string, maxDepth int, reg *registry.Registry) []discoveredWorktree {
	var discovered []discoveredWorktree
	seen := make(map[string]bool)

	var scan func(path string, depth int)
	scan = func(path string, depth int) {
		if maxDepth >= 0 && depth > maxDepth {
			return
		}

		// Check if this is a git repository
		gitPath := filepath.Join(path, ".git")
		info, err := os.Stat(gitPath)
		if err == nil {
			// Found a git repo
			wt := analyzeGitRepo(path, info.IsDir(), reg)
			if wt != nil && !seen[wt.Path] {
				seen[wt.Path] = true
				discovered = append(discovered, *wt)

				// If it's a main repo, also check for linked worktrees
				if info.IsDir() {
					linkedWorktrees := findLinkedWorktrees(path)
					for _, linked := range linkedWorktrees {
						if !seen[linked.Path] {
							seen[linked.Path] = true
							// Check registry status for linked worktree
							if server, ok := reg.Get(linked.Name); ok {
								linked.Registered = true
								linked.Running = server.IsRunning()
								linked.Port = server.Port
							}
							discovered = append(discovered, linked)
						}
					}
				}
			}

			// Don't descend into git repos
			return
		}

		// Not a git repo, scan subdirectories
		entries, err := os.ReadDir(path)
		if err != nil {
			return
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			// Skip hidden directories and common non-project dirs
			name := entry.Name()
			if strings.HasPrefix(name, ".") ||
				name == "node_modules" ||
				name == "vendor" ||
				name == "__pycache__" ||
				name == "venv" ||
				name == ".venv" {
				continue
			}

			scan(filepath.Join(path, name), depth+1)
		}
	}

	scan(basePath, 0)
	return discovered
}

func analyzeGitRepo(path string, isMainRepo bool, reg *registry.Registry) *discoveredWorktree {
	// Get worktree info
	wt, err := worktree.DetectAt(path)
	if err != nil {
		return nil
	}

	// For main repos (not linked worktrees), use directory name as the server name
	// This avoids conflicts when multiple projects are on the same branch (e.g., "main")
	name := wt.Name
	if !wt.IsWorktree {
		dirName := filepath.Base(path)
		name = worktree.Sanitize(dirName)
	}

	discovered := &discoveredWorktree{
		Path:       wt.Path,
		Name:       name,
		Branch:     wt.Branch,
		IsWorktree: wt.IsWorktree,
		HasConfig:  fileExists(filepath.Join(path, ".grove.yaml")),
	}

	// Check if already registered
	if server, ok := reg.Get(name); ok {
		discovered.Registered = true
		discovered.Running = server.IsRunning()
		discovered.Port = server.Port
	}

	return discovered
}

func findLinkedWorktrees(mainRepoPath string) []discoveredWorktree {
	var worktrees []discoveredWorktree

	// Use git worktree list to find linked worktrees
	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	cmd.Dir = mainRepoPath
	output, err := cmd.Output()
	if err != nil {
		return worktrees
	}

	lines := strings.Split(string(output), "\n")
	var currentPath, currentBranch string

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "worktree ") {
			currentPath = strings.TrimPrefix(line, "worktree ")
		} else if strings.HasPrefix(line, "branch ") {
			currentBranch = strings.TrimPrefix(line, "branch refs/heads/")
		} else if line == "" && currentPath != "" {
			// End of entry
			if currentPath != mainRepoPath {
				// This is a linked worktree
				name := worktree.Sanitize(currentBranch)
				worktrees = append(worktrees, discoveredWorktree{
					Path:       currentPath,
					Name:       name,
					Branch:     currentBranch,
					IsWorktree: true,
					HasConfig:  fileExists(filepath.Join(currentPath, ".grove.yaml")),
				})
			}
			currentPath = ""
			currentBranch = ""
		}
	}

	return worktrees
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
