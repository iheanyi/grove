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

var newCmd = &cobra.Command{
	Use:   "new <branch-name> [base-branch]",
	Short: "Create a new git worktree with the given branch name",
	Long: `Create a new git worktree with the given branch name.

By default, the worktree is created in a sibling directory to the current repository,
using the pattern: <repo-name>-<branch-name>

If worktrees_dir is configured in ~/.config/grove/config.yaml, worktrees are created
in a centralized location: <worktrees_dir>/<project>/<branch>

If base-branch is not specified, it defaults to 'main' or 'master' (auto-detected).

Examples:
  grove new feature-auth              # Create worktree from main/master
  grove new feature-auth develop      # Create worktree from develop branch
  grove new bugfix-123 v1.0.0         # Create worktree from v1.0.0 tag
  grove new feature-auth --dir ~/worktrees  # Override worktree location
  grove new feature-auth --track      # Force tracking existing remote branch
  grove new feature-auth --no-track   # Force creating new branch (ignore remote)`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runNew,
}

func init() {
	newCmd.Flags().String("dir", "", "Override worktree parent directory")
	newCmd.Flags().Bool("track", false, "Force tracking existing remote branch without prompt")
	newCmd.Flags().Bool("no-track", false, "Force creating new branch even if remote exists")
}

func runNew(cmd *cobra.Command, args []string) error {
	branchName := args[0]

	// Validate branch name
	if strings.TrimSpace(branchName) == "" {
		return fmt.Errorf("branch name cannot be empty")
	}

	// Get flags
	forceTrack, _ := cmd.Flags().GetBool("track")
	forceNoTrack, _ := cmd.Flags().GetBool("no-track")

	// Validate conflicting flags
	if forceTrack && forceNoTrack {
		return fmt.Errorf("cannot use both --track and --no-track")
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

	// Check if the branch exists on remote (unless --no-track is set)
	trackRemote := false
	if !forceNoTrack {
		remoteBranchExists := remoteBranchExists(mainRepoPath, branchName)
		if remoteBranchExists {
			if forceTrack {
				// User explicitly wants to track remote
				trackRemote = true
			} else {
				// Prompt user
				fmt.Printf("\nBranch '%s' exists on remote (origin/%s).\n", branchName, branchName)
				trackRemote = promptYesNo("Track existing remote branch?", true)
				fmt.Println()
			}
		}
	}

	// If not tracking remote, verify base branch exists
	if !trackRemote {
		if err := verifyRefExists(mainRepoPath, baseBranch); err != nil {
			return fmt.Errorf("base branch '%s' does not exist: %w", baseBranch, err)
		}
	}

	// Determine repository name from the main repo path
	repoName := filepath.Base(mainRepoPath)

	// Determine worktree path based on config/flags
	var worktreePath string
	var worktreeName string

	dirOverride, _ := cmd.Flags().GetString("dir")

	if dirOverride != "" {
		// Flag override: use <dir>/<project>/<branch>
		expandedDir := expandPath(dirOverride)
		worktreePath = filepath.Join(expandedDir, repoName, branchName)
		worktreeName = fmt.Sprintf("%s-%s", repoName, branchName)
	} else if cfg.WorktreesDir != "" {
		// Centralized worktrees: use <worktrees_dir>/<project>/<branch>
		expandedDir := expandPath(cfg.WorktreesDir)
		worktreePath = filepath.Join(expandedDir, repoName, branchName)
		worktreeName = fmt.Sprintf("%s-%s", repoName, branchName)
	} else {
		// Default: sibling directory to main repo
		worktreeName = fmt.Sprintf("%s-%s", repoName, branchName)
		parentDir := filepath.Dir(mainRepoPath)
		worktreePath = filepath.Join(parentDir, worktreeName)
	}

	// Check if worktree path already exists
	if _, err := os.Stat(worktreePath); err == nil {
		return fmt.Errorf("worktree directory already exists: %s", worktreePath)
	}

	// Ensure parent directory exists for centralized worktrees
	parentDir := filepath.Dir(worktreePath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	// Create the worktree
	var gitCmd *exec.Cmd
	if trackRemote {
		// Track existing remote branch
		fmt.Printf("Creating worktree tracking 'origin/%s'...\n", branchName)
		fmt.Printf("Location: %s\n", worktreePath)
		gitCmd = exec.Command("git", "worktree", "add", worktreePath, "origin/"+branchName)
	} else {
		// Create new branch from base
		fmt.Printf("Creating worktree '%s' from '%s'...\n", branchName, baseBranch)
		fmt.Printf("Location: %s\n", worktreePath)
		gitCmd = exec.Command("git", "worktree", "add", "-b", branchName, worktreePath, baseBranch)
	}
	gitCmd.Dir = mainRepoPath
	gitCmd.Stdout = os.Stdout
	gitCmd.Stderr = os.Stderr

	if err := gitCmd.Run(); err != nil {
		return fmt.Errorf("failed to create worktree: %w", err)
	}

	fmt.Printf("\nWorktree created successfully!\n")
	fmt.Printf("Branch: %s\n", branchName)
	if trackRemote {
		fmt.Printf("Tracking: origin/%s\n", branchName)
	}
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

// expandPath expands ~ to the home directory
func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}

// remoteBranchExists checks if a branch exists on origin
func remoteBranchExists(repoPath, branchName string) bool {
	cmd := exec.Command("git", "rev-parse", "--verify", "origin/"+branchName)
	cmd.Dir = repoPath
	return cmd.Run() == nil
}

// promptYesNo prompts the user with a yes/no question
// defaultYes determines the default answer when user just presses Enter
func promptYesNo(question string, defaultYes bool) bool {
	reader := bufio.NewReader(os.Stdin)

	prompt := question
	if defaultYes {
		prompt += " [Y/n]: "
	} else {
		prompt += " [y/N]: "
	}
	fmt.Print(prompt)

	input, err := reader.ReadString('\n')
	if err != nil {
		return defaultYes
	}

	input = strings.TrimSpace(strings.ToLower(input))
	if input == "" {
		return defaultYes
	}

	return input == "y" || input == "yes"
}
