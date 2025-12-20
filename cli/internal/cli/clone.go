package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var cloneCmd = &cobra.Command{
	Use:   "clone <repo-url> [directory]",
	Short: "Clone a repository set up for worktree workflow",
	Long: `Clone a repository and set it up for worktree-based development.

The repo is cloned as a bare repository with a main worktree, enabling
efficient creation of multiple worktrees without duplicating object storage.

Structure created:
  <directory>/
  ├── .bare/              # Bare git repository (object storage)
  ├── main/               # Main branch worktree
  └── <feature-branch>/   # Additional worktrees (created with 'grove new')

Examples:
  grove clone https://github.com/user/repo.git
  grove clone git@github.com:user/repo.git myproject
  grove clone https://github.com/user/repo.git --branch develop`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runClone,
}

func init() {
	cloneCmd.Flags().StringP("branch", "b", "", "Branch to checkout in main worktree (default: repo's default branch)")
	cloneCmd.Flags().Bool("no-checkout", false, "Don't checkout files after cloning")
}

func runClone(cmd *cobra.Command, args []string) error {
	repoURL := args[0]
	branch, _ := cmd.Flags().GetString("branch")
	noCheckout, _ := cmd.Flags().GetBool("no-checkout")

	// Determine directory name
	var directory string
	if len(args) > 1 {
		directory = args[1]
	} else {
		// Extract repo name from URL
		directory = extractRepoName(repoURL)
	}

	// Get absolute path
	absDir, err := filepath.Abs(directory)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	// Check if directory already exists
	if _, err := os.Stat(absDir); err == nil {
		return fmt.Errorf("directory already exists: %s", absDir)
	}

	fmt.Printf("Cloning %s into worktree structure...\n", repoURL)
	fmt.Printf("Location: %s\n\n", absDir)

	// Create the parent directory
	if err := os.MkdirAll(absDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Clone as bare repository into .bare
	bareDir := filepath.Join(absDir, ".bare")
	fmt.Println("Step 1/3: Cloning bare repository...")

	cloneArgs := []string{"clone", "--bare", repoURL, bareDir}
	gitClone := exec.Command("git", cloneArgs...)
	gitClone.Stdout = os.Stdout
	gitClone.Stderr = os.Stderr

	if err := gitClone.Run(); err != nil {
		// Cleanup on failure
		os.RemoveAll(absDir)
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	// Create .git file that points to .bare
	fmt.Println("Step 2/3: Setting up git directory reference...")
	gitFile := filepath.Join(absDir, ".git")
	gitContent := fmt.Sprintf("gitdir: .bare\n")
	if err := os.WriteFile(gitFile, []byte(gitContent), 0644); err != nil {
		os.RemoveAll(absDir)
		return fmt.Errorf("failed to create .git file: %w", err)
	}

	// Detect the default branch if not specified
	if branch == "" {
		detected, err := detectRemoteDefaultBranch(bareDir)
		if err == nil && detected != "" {
			branch = detected
		} else {
			branch = "main" // Fallback
		}
	}

	// Create the main worktree
	fmt.Printf("Step 3/3: Creating main worktree (%s)...\n", branch)
	mainWorktree := filepath.Join(absDir, "main")

	worktreeArgs := []string{"worktree", "add", mainWorktree, branch}
	if noCheckout {
		worktreeArgs = append(worktreeArgs, "--no-checkout")
	}

	gitWorktree := exec.Command("git", worktreeArgs...)
	gitWorktree.Dir = bareDir
	gitWorktree.Stdout = os.Stdout
	gitWorktree.Stderr = os.Stderr

	if err := gitWorktree.Run(); err != nil {
		// Don't fail completely - the bare clone succeeded
		fmt.Printf("Warning: failed to create main worktree: %v\n", err)
		fmt.Println("You can create it manually with: git worktree add main <branch>")
	}

	fmt.Println()
	fmt.Println("✓ Repository cloned successfully!")
	fmt.Println()
	fmt.Println("Structure created:")
	fmt.Printf("  %s/\n", directory)
	fmt.Println("  ├── .bare/     # Bare git repository")
	fmt.Println("  ├── .git       # Git directory reference")
	fmt.Printf("  └── main/      # %s branch worktree\n", branch)
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Printf("  cd %s/main\n", absDir)
	fmt.Println("  grove start npm run dev     # Start dev server")
	fmt.Println("  grove new feature-branch    # Create a new worktree")

	return nil
}

// extractRepoName extracts the repository name from a git URL
func extractRepoName(url string) string {
	// Remove trailing .git if present
	name := strings.TrimSuffix(url, ".git")

	// Handle HTTPS URLs: https://github.com/user/repo.git
	if strings.HasPrefix(name, "https://") || strings.HasPrefix(name, "http://") {
		parts := strings.Split(name, "/")
		if len(parts) > 0 {
			return parts[len(parts)-1]
		}
	}

	// Handle SSH URLs: git@github.com:user/repo.git
	if strings.Contains(name, ":") {
		parts := strings.Split(name, ":")
		if len(parts) > 1 {
			pathParts := strings.Split(parts[len(parts)-1], "/")
			if len(pathParts) > 0 {
				return pathParts[len(pathParts)-1]
			}
		}
	}

	// Fallback: use the last path component
	return filepath.Base(name)
}

// detectRemoteDefaultBranch detects the default branch from a bare repository
func detectRemoteDefaultBranch(bareDir string) (string, error) {
	// Try to get the default branch from HEAD
	cmd := exec.Command("git", "symbolic-ref", "HEAD")
	cmd.Dir = bareDir
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	// Output format: refs/heads/main
	ref := strings.TrimSpace(string(output))
	parts := strings.Split(ref, "/")
	if len(parts) >= 3 {
		return parts[len(parts)-1], nil
	}

	return "", fmt.Errorf("could not parse default branch")
}
