package cli

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/iheanyi/grove/internal/worktree"
	"github.com/spf13/cobra"
)

var pruneCmd = &cobra.Command{
	Use:   "prune",
	Short: "List and remove stale worktrees",
	Long: `List and remove stale worktrees.

This command identifies worktrees whose branches have been merged into the main branch
and prompts for confirmation before deletion.

Examples:
  grove prune              # Interactive prune with confirmation
  grove prune --dry-run    # Show what would be deleted
  grove prune --force      # Delete without confirmation`,
	RunE: runPrune,
}

func init() {
	pruneCmd.Flags().Bool("dry-run", false, "Show what would be deleted without actually deleting")
	pruneCmd.Flags().Bool("force", false, "Skip confirmation and delete immediately")
}

type worktreeEntry struct {
	Path   string
	Branch string
	Name   string
}

func runPrune(cmd *cobra.Command, args []string) error {
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	force, _ := cmd.Flags().GetBool("force")

	// Detect current worktree to find the main repo
	currentWt, err := worktree.Detect()
	if err != nil {
		return fmt.Errorf("failed to detect git repository: %w", err)
	}

	// Determine the main repository path
	mainRepoPath := currentWt.Path
	if currentWt.IsWorktree && currentWt.MainWorktreePath != "" {
		mainRepoPath = currentWt.MainWorktreePath
	}

	// Detect default branch
	defaultBranch, err := detectDefaultBranch(mainRepoPath)
	if err != nil {
		return fmt.Errorf("failed to detect default branch: %w", err)
	}

	fmt.Printf("Scanning worktrees (base branch: %s)...\n\n", defaultBranch)

	// List all worktrees
	worktrees, err := listWorktrees(mainRepoPath)
	if err != nil {
		return fmt.Errorf("failed to list worktrees: %w", err)
	}

	if len(worktrees) == 0 {
		fmt.Println("No worktrees found")
		return nil
	}

	// Find stale worktrees (merged branches)
	staleWorktrees := []worktreeEntry{}
	for _, wt := range worktrees {
		// Skip the main worktree
		if wt.Path == mainRepoPath {
			continue
		}

		// Skip if branch is the default branch
		if wt.Branch == defaultBranch {
			continue
		}

		// Check if branch is merged
		isMerged, err := isBranchMerged(mainRepoPath, wt.Branch, defaultBranch)
		if err != nil {
			fmt.Printf("Warning: could not check merge status for %s: %v\n", wt.Branch, err)
			continue
		}

		if isMerged {
			staleWorktrees = append(staleWorktrees, wt)
		}
	}

	if len(staleWorktrees) == 0 {
		fmt.Println("No stale worktrees found (all branches are unmerged or active)")
		return nil
	}

	// Display stale worktrees
	fmt.Printf("Found %d stale worktree(s):\n\n", len(staleWorktrees))
	for i, wt := range staleWorktrees {
		fmt.Printf("%d. %s\n", i+1, wt.Name)
		fmt.Printf("   Branch: %s (merged)\n", wt.Branch)
		fmt.Printf("   Path:   %s\n", wt.Path)
		fmt.Println()
	}

	if dryRun {
		fmt.Println("(Dry run - no changes made)")
		return nil
	}

	// Confirm deletion
	if !force {
		fmt.Printf("Delete these %d worktree(s)? [y/N]: ", len(staleWorktrees))
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
	}

	// Delete worktrees
	fmt.Println("\nDeleting worktrees...")
	for _, wt := range staleWorktrees {
		fmt.Printf("Removing %s... ", wt.Name)

		// Remove worktree using git worktree remove
		gitCmd := exec.Command("git", "worktree", "remove", wt.Path, "--force")
		gitCmd.Dir = mainRepoPath
		if err := gitCmd.Run(); err != nil {
			fmt.Printf("FAILED: %v\n", err)
			continue
		}

		// Delete the local branch
		branchCmd := exec.Command("git", "branch", "-D", wt.Branch)
		branchCmd.Dir = mainRepoPath
		if err := branchCmd.Run(); err != nil {
			fmt.Printf("OK (worktree removed, branch deletion failed: %v)\n", err)
		} else {
			fmt.Println("OK")
		}
	}

	// Clean up any stale worktree metadata
	fmt.Println("\nCleaning up worktree metadata...")
	pruneMetadataCmd := exec.Command("git", "worktree", "prune")
	pruneMetadataCmd.Dir = mainRepoPath
	if err := pruneMetadataCmd.Run(); err != nil {
		fmt.Printf("Warning: failed to prune metadata: %v\n", err)
	}

	fmt.Printf("\nSuccessfully pruned %d worktree(s)\n", len(staleWorktrees))

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
			// New worktree entry
			if current.Path != "" {
				current.Name = filepath.Base(current.Path)
				worktrees = append(worktrees, current)
			}
			current = worktreeEntry{
				Path: strings.TrimPrefix(line, "worktree "),
			}
		} else if strings.HasPrefix(line, "branch ") {
			branchRef := strings.TrimPrefix(line, "branch ")
			// Extract branch name from refs/heads/branch-name
			parts := strings.Split(branchRef, "/")
			if len(parts) >= 3 {
				current.Branch = strings.Join(parts[2:], "/")
			}
		}
	}

	// Add the last entry
	if current.Path != "" {
		current.Name = filepath.Base(current.Path)
		worktrees = append(worktrees, current)
	}

	return worktrees, nil
}

// isBranchMerged checks if a branch has been merged into the base branch
func isBranchMerged(repoPath, branch, baseBranch string) (bool, error) {
	// Use git branch --merged to check if the branch is merged
	cmd := exec.Command("git", "branch", "--merged", baseBranch)
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}

	// Parse the output to see if our branch is in the list
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		// Remove leading * and whitespace
		branchName := strings.TrimSpace(strings.TrimPrefix(line, "*"))
		if branchName == branch {
			return true, nil
		}
	}

	return false, nil
}
