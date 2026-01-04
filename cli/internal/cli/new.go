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

When a directory conflict occurs, you'll be prompted with options to resolve it.

Examples:
  grove new feature-auth              # Create worktree from main/master
  grove new feature-auth develop      # Create worktree from develop branch
  grove new bugfix-123 v1.0.0         # Create worktree from v1.0.0 tag
  grove new feature-auth --dir ~/worktrees  # Override worktree location
  grove new feature-auth --name myapp-auth  # Custom worktree name
  grove new feature-auth --track      # Force tracking existing remote branch
  grove new feature-auth --no-track   # Force creating new branch (ignore remote)
  grove new --pick                    # Pick from available remote branches
  grove new --pick --filter feat      # Pick from remote branches matching 'feat'`,
	Args: cobra.RangeArgs(0, 2),
	RunE: runNew,
}

func init() {
	newCmd.Flags().String("dir", "", "Override worktree parent directory")
	newCmd.Flags().String("name", "", "Override worktree name (for resolving conflicts)")
	newCmd.Flags().Bool("track", false, "Force tracking existing remote branch without prompt")
	newCmd.Flags().Bool("no-track", false, "Force creating new branch even if remote exists")
	newCmd.Flags().Bool("pick", false, "Interactively pick from remote branches")
	newCmd.Flags().String("filter", "", "Filter remote branches by pattern (used with --pick)")
}

func runNew(cmd *cobra.Command, args []string) error {
	// Get flags
	pickMode, _ := cmd.Flags().GetBool("pick")
	filterPattern, _ := cmd.Flags().GetString("filter")
	forceTrack, _ := cmd.Flags().GetBool("track")
	forceNoTrack, _ := cmd.Flags().GetBool("no-track")

	var branchName string

	// Handle --pick mode
	if pickMode {
		if len(args) > 0 {
			return fmt.Errorf("cannot specify branch name with --pick flag")
		}

		// Detect current repo first
		wt, err := worktree.Detect()
		if err != nil {
			return fmt.Errorf("failed to detect git repository: %w", err)
		}
		mainRepoPath := wt.Path
		if wt.IsWorktree && wt.MainWorktreePath != "" {
			mainRepoPath = wt.MainWorktreePath
		}

		// Fetch latest remote branches
		fmt.Println("Fetching remote branches...")
		if err := fetchRemote(mainRepoPath); err != nil {
			fmt.Printf("Warning: could not fetch remote: %v\n", err)
		}

		// Get available remote branches
		branches, err := listRemoteBranches(mainRepoPath, filterPattern)
		if err != nil {
			return fmt.Errorf("failed to list remote branches: %w", err)
		}

		if len(branches) == 0 {
			if filterPattern != "" {
				return fmt.Errorf("no remote branches matching '%s'", filterPattern)
			}
			return fmt.Errorf("no remote branches found")
		}

		// Present picker
		selected, err := pickBranch(branches)
		if err != nil {
			return err
		}

		branchName = selected
		forceTrack = true // When picking a remote branch, always track it
	} else {
		if len(args) < 1 {
			return fmt.Errorf("branch name required (or use --pick to select from remote branches)")
		}
		branchName = args[0]
	}

	// Validate branch name
	if strings.TrimSpace(branchName) == "" {
		return fmt.Errorf("branch name cannot be empty")
	}

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
	nameOverride, _ := cmd.Flags().GetString("name")

	// Allow custom name override
	effectiveBranchName := branchName
	if nameOverride != "" {
		effectiveBranchName = nameOverride
	}

	if dirOverride != "" {
		// Flag override: use <dir>/<project>/<branch>
		expandedDir := expandPath(dirOverride)
		worktreePath = filepath.Join(expandedDir, repoName, effectiveBranchName)
		worktreeName = fmt.Sprintf("%s-%s", repoName, effectiveBranchName)
	} else if cfg.WorktreesDir != "" {
		// Centralized worktrees: use <worktrees_dir>/<project>/<branch>
		expandedDir := expandPath(cfg.WorktreesDir)
		worktreePath = filepath.Join(expandedDir, repoName, effectiveBranchName)
		worktreeName = fmt.Sprintf("%s-%s", repoName, effectiveBranchName)
	} else {
		// Default: sibling directory to main repo
		worktreeName = fmt.Sprintf("%s-%s", repoName, effectiveBranchName)
		parentDir := filepath.Dir(mainRepoPath)
		worktreePath = filepath.Join(parentDir, worktreeName)
	}

	// Check if worktree path already exists and prompt for resolution
	if _, err := os.Stat(worktreePath); err == nil {
		newPath, newName, err := handleCollision(branchName, worktreePath, worktreeName, repoName, mainRepoPath)
		if err != nil {
			return err
		}
		worktreePath = newPath
		worktreeName = newName
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

// fetchRemote fetches the latest from origin
func fetchRemote(repoPath string) error {
	cmd := exec.Command("git", "fetch", "origin", "--prune")
	cmd.Dir = repoPath
	return cmd.Run()
}

// listRemoteBranches returns a list of remote branches, optionally filtered
func listRemoteBranches(repoPath, filter string) ([]string, error) {
	cmd := exec.Command("git", "branch", "-r", "--format=%(refname:short)")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var branches []string
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")

	// Get list of existing local worktree branches to exclude
	existingBranches := getExistingWorktreeBranches(repoPath)

	for _, line := range lines {
		branch := strings.TrimSpace(line)
		if branch == "" {
			continue
		}

		// Skip HEAD pointer
		if strings.Contains(branch, "HEAD") {
			continue
		}

		// Remove origin/ prefix for display
		shortName := strings.TrimPrefix(branch, "origin/")

		// Skip if worktree already exists for this branch
		if existingBranches[shortName] {
			continue
		}

		// Apply filter if provided
		if filter != "" && !strings.Contains(strings.ToLower(shortName), strings.ToLower(filter)) {
			continue
		}

		branches = append(branches, shortName)
	}

	return branches, nil
}

// getExistingWorktreeBranches returns a map of branch names that already have worktrees
func getExistingWorktreeBranches(repoPath string) map[string]bool {
	result := make(map[string]bool)

	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return result
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "branch ") {
			branchRef := strings.TrimPrefix(line, "branch ")
			// Extract branch name from refs/heads/branch-name
			parts := strings.Split(branchRef, "/")
			if len(parts) >= 3 {
				branchName := strings.Join(parts[2:], "/")
				result[branchName] = true
			}
		}
	}

	return result
}

// handleCollision prompts the user to resolve a worktree path collision
func handleCollision(branchName, existingPath, existingName, repoName, mainRepoPath string) (string, string, error) {
	fmt.Printf("\n⚠️  Directory conflict: %s already exists\n\n", existingPath)
	fmt.Println("Options:")
	fmt.Printf("  1. Use different name (e.g., grove new %s --name %s-v2)\n", branchName, branchName)
	fmt.Printf("  2. Use different directory (e.g., grove new %s --dir ~/worktrees-alt)\n", branchName)
	fmt.Println("  3. Cancel")
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("Choose option [1-3]: ")
		input, err := reader.ReadString('\n')
		if err != nil {
			return "", "", fmt.Errorf("failed to read input: %w", err)
		}

		input = strings.TrimSpace(input)
		switch input {
		case "1":
			// Prompt for new name
			fmt.Print("Enter new name: ")
			newName, err := reader.ReadString('\n')
			if err != nil {
				return "", "", fmt.Errorf("failed to read input: %w", err)
			}
			newName = strings.TrimSpace(newName)
			if newName == "" {
				fmt.Println("Name cannot be empty")
				continue
			}

			// Calculate new path with the new name
			worktreeName := fmt.Sprintf("%s-%s", repoName, newName)
			parentDir := filepath.Dir(mainRepoPath)
			worktreePath := filepath.Join(parentDir, worktreeName)

			// Check if this new path also exists
			if _, err := os.Stat(worktreePath); err == nil {
				fmt.Printf("Directory %s also exists. Try a different name.\n", worktreePath)
				continue
			}

			return worktreePath, worktreeName, nil

		case "2":
			// Prompt for new directory
			fmt.Print("Enter new directory path: ")
			newDir, err := reader.ReadString('\n')
			if err != nil {
				return "", "", fmt.Errorf("failed to read input: %w", err)
			}
			newDir = strings.TrimSpace(newDir)
			if newDir == "" {
				fmt.Println("Directory cannot be empty")
				continue
			}

			// Calculate new path with the new directory
			expandedDir := expandPath(newDir)
			worktreePath := filepath.Join(expandedDir, repoName, branchName)
			worktreeName := fmt.Sprintf("%s-%s", repoName, branchName)

			// Check if this new path also exists
			if _, err := os.Stat(worktreePath); err == nil {
				fmt.Printf("Directory %s also exists. Try a different location.\n", worktreePath)
				continue
			}

			return worktreePath, worktreeName, nil

		case "3", "q", "quit", "":
			return "", "", fmt.Errorf("cancelled")

		default:
			fmt.Println("Please enter 1, 2, or 3")
		}
	}
}

// pickBranch presents an interactive picker for selecting a branch
func pickBranch(branches []string) (string, error) {
	if len(branches) == 0 {
		return "", fmt.Errorf("no branches to pick from")
	}

	// Show numbered list
	fmt.Printf("\nAvailable remote branches (%d):\n", len(branches))
	fmt.Println(strings.Repeat("-", 40))

	// Limit display for very long lists
	displayLimit := 25
	for i, branch := range branches {
		if i >= displayLimit {
			fmt.Printf("  ... and %d more (use --filter to narrow down)\n", len(branches)-displayLimit)
			break
		}
		fmt.Printf("  %2d) %s\n", i+1, branch)
	}
	fmt.Println()

	// Prompt for selection
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("Select branch number (or 'q' to quit): ")
		input, err := reader.ReadString('\n')
		if err != nil {
			return "", fmt.Errorf("failed to read input: %w", err)
		}

		input = strings.TrimSpace(input)
		if input == "q" || input == "quit" {
			return "", fmt.Errorf("selection cancelled")
		}

		// Parse number
		var num int
		if _, err := fmt.Sscanf(input, "%d", &num); err != nil {
			fmt.Println("Please enter a valid number")
			continue
		}

		if num < 1 || num > len(branches) {
			fmt.Printf("Please enter a number between 1 and %d\n", len(branches))
			continue
		}

		return branches[num-1], nil
	}
}
