package cli

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/iheanyi/grove/internal/registry"
	"github.com/iheanyi/grove/internal/worktree"
	"github.com/spf13/cobra"
)

var pruneCmd = &cobra.Command{
	Use:   "prune",
	Short: "Remove stale worktrees and stopped servers",
	Long: `Remove stale worktrees and stopped servers from grove.

By default, shows all prunable items and prompts for confirmation.
Use flags to control what gets pruned.

What can be pruned:
  - Stopped servers: Registry entries for servers that aren't running
  - Merged worktrees: Git worktrees whose branches have been merged
  - Orphaned entries: Registry entries for paths that no longer exist

Examples:
  grove prune                # Interactive - show all and prompt
  grove prune --stopped      # Only prune stopped servers
  grove prune --merged       # Only prune merged git worktrees
  grove prune --orphaned     # Only prune orphaned registry entries
  grove prune --all          # Prune everything without prompting per-category
  grove prune --dry-run      # Show what would be pruned`,
	RunE: runPrune,
}

func init() {
	pruneCmd.Flags().Bool("dry-run", false, "Show what would be pruned without making changes")
	pruneCmd.Flags().Bool("force", false, "Skip confirmation prompts")
	pruneCmd.Flags().Bool("stopped", false, "Prune stopped servers from registry")
	pruneCmd.Flags().Bool("merged", false, "Prune git worktrees with merged branches")
	pruneCmd.Flags().Bool("orphaned", false, "Prune registry entries for non-existent paths")
	pruneCmd.Flags().Bool("all", false, "Prune everything (stopped + merged + orphaned)")
	rootCmd.AddCommand(pruneCmd)
}

type pruneResult struct {
	stoppedServers  []string
	mergedWorktrees []worktreeEntry
	orphanedEntries []string
}

type worktreeEntry struct {
	Path   string
	Branch string
	Name   string
}

func runPrune(cmd *cobra.Command, args []string) error {
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	force, _ := cmd.Flags().GetBool("force")
	pruneStopped, _ := cmd.Flags().GetBool("stopped")
	pruneMerged, _ := cmd.Flags().GetBool("merged")
	pruneOrphaned, _ := cmd.Flags().GetBool("orphaned")
	pruneAll, _ := cmd.Flags().GetBool("all")

	// If --all, enable everything
	if pruneAll {
		pruneStopped = true
		pruneMerged = true
		pruneOrphaned = true
	}

	// If no specific flags, show everything (interactive mode)
	interactive := !pruneStopped && !pruneMerged && !pruneOrphaned

	// Load registry
	reg, err := registry.Load()
	if err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}

	result := pruneResult{}

	// Find stopped servers
	if pruneStopped || interactive {
		for _, server := range reg.List() {
			if !server.IsRunning() {
				result.stoppedServers = append(result.stoppedServers, server.Name)
			}
		}
	}

	// Find orphaned entries (paths that don't exist)
	if pruneOrphaned || interactive {
		for _, server := range reg.List() {
			if _, err := os.Stat(server.Path); os.IsNotExist(err) {
				// Don't double-count if already in stopped list
				found := false
				for _, s := range result.stoppedServers {
					if s == server.Name {
						found = true
						break
					}
				}
				if !found {
					result.orphanedEntries = append(result.orphanedEntries, server.Name)
				}
			}
		}
		for _, wt := range reg.ListWorktrees() {
			if _, err := os.Stat(wt.Path); os.IsNotExist(err) {
				result.orphanedEntries = append(result.orphanedEntries, wt.Name)
			}
		}
	}

	// Find merged worktrees (only if we're in a git repo)
	if pruneMerged || interactive {
		currentWt, err := worktree.Detect()
		if err == nil {
			mainRepoPath := currentWt.Path
			if currentWt.IsWorktree && currentWt.MainWorktreePath != "" {
				mainRepoPath = currentWt.MainWorktreePath
			}

			defaultBranch, err := detectDefaultBranch(mainRepoPath)
			if err == nil {
				worktrees, err := listWorktrees(mainRepoPath)
				if err == nil {
					for _, wt := range worktrees {
						if wt.Path == mainRepoPath || wt.Branch == defaultBranch {
							continue
						}
						isMerged, err := isBranchMerged(mainRepoPath, wt.Branch, defaultBranch)
						if err == nil && isMerged {
							result.mergedWorktrees = append(result.mergedWorktrees, wt)
						}
					}
				}
			}
		}
	}

	// Check if anything to prune
	totalItems := len(result.stoppedServers) + len(result.mergedWorktrees) + len(result.orphanedEntries)
	if totalItems == 0 {
		fmt.Println("Nothing to prune - everything is clean!")
		return nil
	}

	// Display what can be pruned
	fmt.Println("=== Prunable Items ===")
	fmt.Println()

	if len(result.stoppedServers) > 0 {
		fmt.Printf("Stopped servers (%d):\n", len(result.stoppedServers))
		for _, name := range result.stoppedServers {
			server, _ := reg.Get(name)
			path := ""
			if server != nil {
				path = server.Path
			}
			fmt.Printf("  • %s\n    %s\n", name, shortenPath(path))
		}
		fmt.Println()
	}

	if len(result.orphanedEntries) > 0 {
		fmt.Printf("Orphaned entries (%d) - paths no longer exist:\n", len(result.orphanedEntries))
		for _, name := range result.orphanedEntries {
			fmt.Printf("  • %s\n", name)
		}
		fmt.Println()
	}

	if len(result.mergedWorktrees) > 0 {
		fmt.Printf("Merged worktrees (%d):\n", len(result.mergedWorktrees))
		for _, wt := range result.mergedWorktrees {
			fmt.Printf("  • %s (branch: %s)\n    %s\n", wt.Name, wt.Branch, shortenPath(wt.Path))
		}
		fmt.Println()
	}

	if dryRun {
		fmt.Println("--dry-run specified, no changes made.")
		return nil
	}

	// In interactive mode, prompt for each category
	if interactive && !force {
		if len(result.stoppedServers) > 0 {
			if confirm(fmt.Sprintf("Remove %d stopped server(s) from registry?", len(result.stoppedServers))) {
				pruneStopped = true
			}
		}
		if len(result.orphanedEntries) > 0 {
			if confirm(fmt.Sprintf("Remove %d orphaned entry/entries?", len(result.orphanedEntries))) {
				pruneOrphaned = true
			}
		}
		if len(result.mergedWorktrees) > 0 {
			if confirm(fmt.Sprintf("Remove %d merged worktree(s)?", len(result.mergedWorktrees))) {
				pruneMerged = true
			}
		}
	} else if !force {
		// Non-interactive mode with specific flags - single confirmation
		if !confirm(fmt.Sprintf("Prune %d item(s)?", totalItems)) {
			fmt.Println("Canceled")
			return nil
		}
	}

	// Execute pruning
	pruned := 0

	// Prune stopped servers
	if pruneStopped && len(result.stoppedServers) > 0 {
		fmt.Println("Removing stopped servers...")
		for _, name := range result.stoppedServers {
			if err := reg.Remove(name); err != nil {
				fmt.Printf("  ✗ %s: %v\n", name, err)
			} else {
				fmt.Printf("  ✓ %s\n", name)
				pruned++
			}
		}
	}

	// Prune orphaned entries
	if pruneOrphaned && len(result.orphanedEntries) > 0 {
		fmt.Println("Removing orphaned entries...")
		for _, name := range result.orphanedEntries {
			var errs []string
			if err := reg.Remove(name); err != nil {
				errs = append(errs, fmt.Sprintf("server: %v", err))
			}
			if err := reg.RemoveWorktree(name); err != nil {
				errs = append(errs, fmt.Sprintf("worktree: %v", err))
			}
			if len(errs) > 0 {
				fmt.Printf("  ✗ %s: %s\n", name, strings.Join(errs, ", "))
			} else {
				fmt.Printf("  ✓ %s\n", name)
				pruned++
			}
		}
	}

	// Prune merged worktrees
	if pruneMerged && len(result.mergedWorktrees) > 0 {
		fmt.Println("Removing merged worktrees...")
		currentWt, _ := worktree.Detect()
		mainRepoPath := currentWt.Path
		if currentWt.IsWorktree && currentWt.MainWorktreePath != "" {
			mainRepoPath = currentWt.MainWorktreePath
		}

		for _, wt := range result.mergedWorktrees {
			fmt.Printf("  Removing %s... ", wt.Name)

			// Remove git worktree
			gitCmd := exec.Command("git", "worktree", "remove", wt.Path, "--force")
			gitCmd.Dir = mainRepoPath
			if err := gitCmd.Run(); err != nil {
				fmt.Printf("FAILED: %v\n", err)
				continue
			}

			// Delete the local branch (ignore error - branch might already be deleted)
			branchCmd := exec.Command("git", "branch", "-D", wt.Branch)
			branchCmd.Dir = mainRepoPath
			if err := branchCmd.Run(); err != nil {
				// Not fatal - branch may have been deleted already
				fmt.Printf("(branch already deleted) ")
			}

			// Remove from registry
			if err := reg.Remove(wt.Name); err != nil {
				fmt.Printf("warning: failed to remove server entry: %v\n", err)
			}
			if err := reg.RemoveWorktree(wt.Name); err != nil {
				fmt.Printf("warning: failed to remove worktree entry: %v\n", err)
			}

			fmt.Println("OK")
			pruned++
		}

		// Clean up git worktree metadata
		pruneMetadataCmd := exec.Command("git", "worktree", "prune")
		pruneMetadataCmd.Dir = mainRepoPath
		if err := pruneMetadataCmd.Run(); err != nil {
			fmt.Printf("Warning: failed to prune git worktree metadata: %v\n", err)
		}
	}

	fmt.Printf("\nPruned %d item(s)\n", pruned)
	return nil
}

// listWorktrees returns all worktrees in the repository
func listWorktrees(mainRepoPath string) ([]worktreeEntry, error) {
	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	cmd.Dir = mainRepoPath
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var worktrees []worktreeEntry
	var current worktreeEntry

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "worktree ") {
			if current.Path != "" {
				current.Name = filepath.Base(current.Path)
				worktrees = append(worktrees, current)
			}
			current = worktreeEntry{
				Path: strings.TrimPrefix(line, "worktree "),
			}
		} else if strings.HasPrefix(line, "branch ") {
			branchRef := strings.TrimPrefix(line, "branch ")
			parts := strings.Split(branchRef, "/")
			if len(parts) >= 3 {
				current.Branch = strings.Join(parts[2:], "/")
			}
		}
	}

	if current.Path != "" {
		current.Name = filepath.Base(current.Path)
		worktrees = append(worktrees, current)
	}

	return worktrees, nil
}

// isBranchMerged checks if a branch has been merged into the base branch
func isBranchMerged(repoPath, branch, baseBranch string) (bool, error) {
	cmd := exec.Command("git", "branch", "--merged", baseBranch)
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		branchName := strings.TrimSpace(strings.TrimPrefix(line, "*"))
		if branchName == branch {
			return true, nil
		}
	}

	return false, nil
}

// confirm prompts the user for yes/no confirmation
func confirm(prompt string) bool {
	fmt.Printf("%s [y/N]: ", prompt)
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes"
}

// shortenPath replaces home directory with ~
func shortenPath(path string) string {
	if home, err := os.UserHomeDir(); err == nil {
		if strings.HasPrefix(path, home) {
			return "~" + strings.TrimPrefix(path, home)
		}
	}
	return path
}
