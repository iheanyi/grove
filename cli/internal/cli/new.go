package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/iheanyi/grove/internal/worktree"
	"github.com/spf13/cobra"
)

var newCmd = &cobra.Command{
	Use:   "new <branch-name> [base-branch]",
	Short: "Create a new git worktree with the given branch name",
	Long: `Create a new git worktree with the given branch name.

The worktree is created in a sibling directory to the current repository,
using the pattern: <repo-name>-<branch-name>

If base-branch is not specified, it defaults to 'main' or 'master' (auto-detected).

Examples:
  grove new feature-auth              # Create worktree from main/master
  grove new feature-auth develop      # Create worktree from develop branch
  grove new bugfix-123 v1.0.0         # Create worktree from v1.0.0 tag`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runNew,
}

func runNew(cmd *cobra.Command, args []string) error {
	branchName := args[0]

	// Validate branch name
	if strings.TrimSpace(branchName) == "" {
		return fmt.Errorf("branch name cannot be empty")
	}

	// Detect current worktree/repo
	wt, err := worktree.Detect()
	if err != nil {
		return fmt.Errorf("failed to detect git repository: %w", err)
	}

	// Determine the main repository path
	mainRepoPath := wt.Path
	if wt.IsWorktree && wt.MainWorktreePath != "" {
		mainRepoPath = wt.MainWorktreePath
	}

	// Determine base branch
	baseBranch := "main"
	if len(args) > 1 {
		baseBranch = args[1]
	} else {
		// Auto-detect main or master
		detected, err := detectDefaultBranch(mainRepoPath)
		if err == nil && detected != "" {
			baseBranch = detected
		}
	}

	// Verify base branch exists
	if err := verifyRefExists(mainRepoPath, baseBranch); err != nil {
		return fmt.Errorf("base branch '%s' does not exist: %w", baseBranch, err)
	}

	// Determine repository name from the main repo path
	repoName := filepath.Base(mainRepoPath)

	// Create worktree directory name
	worktreeName := fmt.Sprintf("%s-%s", repoName, branchName)

	// Determine parent directory (sibling to main repo)
	parentDir := filepath.Dir(mainRepoPath)
	worktreePath := filepath.Join(parentDir, worktreeName)

	// Check if worktree path already exists
	if _, err := os.Stat(worktreePath); err == nil {
		return fmt.Errorf("worktree directory already exists: %s", worktreePath)
	}

	// Create the worktree
	fmt.Printf("Creating worktree '%s' from '%s'...\n", branchName, baseBranch)
	fmt.Printf("Location: %s\n", worktreePath)

	gitCmd := exec.Command("git", "worktree", "add", "-b", branchName, worktreePath, baseBranch)
	gitCmd.Dir = mainRepoPath
	gitCmd.Stdout = os.Stdout
	gitCmd.Stderr = os.Stderr

	if err := gitCmd.Run(); err != nil {
		return fmt.Errorf("failed to create worktree: %w", err)
	}

	fmt.Printf("\nWorktree created successfully!\n")
	fmt.Printf("Branch: %s\n", branchName)
	fmt.Printf("Path: %s\n", worktreePath)
	fmt.Printf("\nTo switch to this worktree:\n")
	fmt.Printf("  cd %s\n", worktreePath)
	fmt.Printf("  # or use: grove switch %s\n", worktreeName)

	return nil
}

// detectDefaultBranch attempts to detect the default branch (main or master)
func detectDefaultBranch(repoPath string) (string, error) {
	// Try to get the default branch from remote
	cmd := exec.Command("git", "symbolic-ref", "refs/remotes/origin/HEAD")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err == nil {
		// Output format: refs/remotes/origin/main
		ref := strings.TrimSpace(string(output))
		parts := strings.Split(ref, "/")
		if len(parts) > 0 {
			return parts[len(parts)-1], nil
		}
	}

	// Fallback: check if main exists, otherwise use master
	for _, branch := range []string{"main", "master"} {
		cmd = exec.Command("git", "rev-parse", "--verify", branch)
		cmd.Dir = repoPath
		if err := cmd.Run(); err == nil {
			return branch, nil
		}
	}

	return "", fmt.Errorf("could not detect default branch")
}

// verifyRefExists checks if a git ref (branch, tag, commit) exists
func verifyRefExists(repoPath, ref string) error {
	cmd := exec.Command("git", "rev-parse", "--verify", ref)
	cmd.Dir = repoPath
	return cmd.Run()
}
