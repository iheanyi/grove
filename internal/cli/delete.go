package cli

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/iheanyi/grove/internal/config"
	"github.com/iheanyi/grove/internal/registry"
	"github.com/iheanyi/grove/internal/worktree"
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Safely remove a git worktree",
	Long: `Safely remove a git worktree and clean up associated resources.

This command performs the following steps:
1. Checks for uncommitted changes (warns if present)
2. Stops any running server for the worktree
3. Removes the worktree using 'git worktree remove'
4. Removes the worktree from the registry
5. Deletes associated log files

Examples:
  grove delete feature-auth         # Delete with safety prompts
  grove delete feature-auth --force # Skip confirmation prompts
  grove delete feature-auth --dry-run # Show what would be deleted`,
	Args: cobra.ExactArgs(1),
	RunE: runDelete,
}

func init() {
	deleteCmd.Flags().Bool("force", false, "Skip confirmation prompts and force deletion")
	deleteCmd.Flags().Bool("dry-run", false, "Show what would be deleted without making changes")
}

func runDelete(cmd *cobra.Command, args []string) error {
	name := args[0]
	force, _ := cmd.Flags().GetBool("force")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	// Load registry
	reg, err := registry.Load()
	if err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}

	// Find the worktree path - check registry first, then git worktree list
	var worktreePath string
	var mainRepoPath string

	// Check if we have a server registered for this name
	if server, ok := reg.Get(name); ok {
		worktreePath = server.Path
	}

	// Check registry worktrees
	if worktreePath == "" {
		if wt, ok := reg.GetWorktree(name); ok {
			worktreePath = wt.Path
			mainRepoPath = wt.MainRepo
		}
	}

	// If not in registry, try to find via git worktree list
	if worktreePath == "" {
		// Detect current repo to find worktrees
		currentWt, err := worktree.Detect()
		if err != nil {
			return fmt.Errorf("failed to detect git repository: %w", err)
		}

		mainRepoPath = currentWt.Path
		if currentWt.IsWorktree && currentWt.MainWorktreePath != "" {
			mainRepoPath = currentWt.MainWorktreePath
		}

		// Search for the worktree
		worktreePath, err = findWorktree(mainRepoPath, name)
		if err != nil {
			return fmt.Errorf("worktree '%s' not found", name)
		}
	}

	// Get main repo path if we don't have it
	if mainRepoPath == "" {
		wtInfo, err := worktree.DetectAt(worktreePath)
		if err != nil {
			return fmt.Errorf("failed to detect worktree info: %w", err)
		}
		mainRepoPath = wtInfo.Path
		if wtInfo.IsWorktree && wtInfo.MainWorktreePath != "" {
			mainRepoPath = wtInfo.MainWorktreePath
		}
	}

	// Check if trying to delete the main worktree
	if worktreePath == mainRepoPath {
		return fmt.Errorf("cannot delete the main worktree; use 'rm -rf' to remove the entire repository")
	}

	fmt.Printf("Worktree: %s\n", name)
	fmt.Printf("Path: %s\n", worktreePath)
	fmt.Println()

	// Safety checks
	var warnings []string

	// Check for uncommitted changes
	hasChanges, err := checkUncommittedChanges(worktreePath)
	if err != nil {
		fmt.Printf("Warning: could not check for uncommitted changes: %v\n", err)
	} else if hasChanges {
		warnings = append(warnings, "Worktree has uncommitted changes")
	}

	// Check if server is running
	var serverRunning bool
	if server, ok := reg.Get(name); ok && server.IsRunning() {
		serverRunning = true
		warnings = append(warnings, fmt.Sprintf("Server is running (PID: %d)", server.PID))
	}

	// Check for log files
	logPath := getLogPath(name)
	hasLogs := false
	if _, err := os.Stat(logPath); err == nil {
		hasLogs = true
	}

	// Display warnings
	if len(warnings) > 0 {
		fmt.Println("Warnings:")
		for _, w := range warnings {
			fmt.Printf("  - %s\n", w)
		}
		fmt.Println()
	}

	// Show what will be deleted
	fmt.Println("This will:")
	if serverRunning {
		fmt.Println("  - Stop the running server")
	}
	fmt.Printf("  - Remove worktree at %s\n", worktreePath)
	fmt.Println("  - Remove from registry")
	if hasLogs {
		fmt.Printf("  - Delete log file: %s\n", logPath)
	}
	fmt.Println()

	if dryRun {
		fmt.Println("(Dry run - no changes made)")
		return nil
	}

	// Confirm deletion
	if !force {
		if len(warnings) > 0 {
			fmt.Print("There are warnings. Are you sure you want to continue? [y/N]: ")
		} else {
			fmt.Print("Proceed with deletion? [y/N]: ")
		}

		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read input: %w", err)
		}

		response = strings.ToLower(strings.TrimSpace(response))
		if response != "y" && response != "yes" {
			fmt.Println("Canceled")
			return nil
		}
		fmt.Println()
	}

	// Stop server if running
	if serverRunning {
		fmt.Print("Stopping server... ")
		if err := stopServer(reg, name, 10*time.Second); err != nil {
			if !force {
				return fmt.Errorf("failed to stop server: %w (use --force to continue anyway)", err)
			}
			fmt.Printf("Warning: %v\n", err)
		} else {
			fmt.Println("done")
		}
	}

	// Remove worktree using git
	fmt.Print("Removing worktree... ")
	gitArgs := []string{"worktree", "remove", worktreePath}
	if force {
		gitArgs = append(gitArgs, "--force")
	}
	gitCmd := exec.Command("git", gitArgs...)
	gitCmd.Dir = mainRepoPath
	if output, err := gitCmd.CombinedOutput(); err != nil {
		if !force {
			return fmt.Errorf("failed to remove worktree: %s", strings.TrimSpace(string(output)))
		}
		fmt.Printf("Warning: %s\n", strings.TrimSpace(string(output)))
	} else {
		fmt.Println("done")
	}

	// Remove from registry (both server and worktree entries)
	fmt.Print("Updating registry... ")
	if _, ok := reg.Get(name); ok {
		if err := reg.Remove(name); err != nil {
			fmt.Printf("Warning: failed to remove server from registry: %v\n", err)
		}
	}
	if _, ok := reg.GetWorktree(name); ok {
		if err := reg.RemoveWorktree(name); err != nil {
			fmt.Printf("Warning: failed to remove worktree from registry: %v\n", err)
		}
	}
	fmt.Println("done")

	// Delete log files
	if hasLogs {
		fmt.Print("Deleting log files... ")
		if err := os.Remove(logPath); err != nil {
			fmt.Printf("Warning: %v\n", err)
		} else {
			fmt.Println("done")
		}
	}

	// Clean up worktree metadata
	fmt.Print("Cleaning up git worktree metadata... ")
	pruneCmd := exec.Command("git", "worktree", "prune")
	pruneCmd.Dir = mainRepoPath
	if err := pruneCmd.Run(); err != nil {
		fmt.Printf("Warning: %v\n", err)
	} else {
		fmt.Println("done")
	}

	fmt.Printf("\nSuccessfully deleted worktree '%s'\n", name)

	return nil
}

// checkUncommittedChanges checks if a worktree has uncommitted changes
func checkUncommittedChanges(path string) (bool, error) {
	cmd := exec.Command("git", "-C", path, "status", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}
	return len(strings.TrimSpace(string(output))) > 0, nil
}

// getLogPath returns the path to the log file for a server
func getLogPath(name string) string {
	logDir := filepath.Join(config.ConfigDir(), "logs")
	return filepath.Join(logDir, name+".log")
}
