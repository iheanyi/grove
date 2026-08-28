package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/iheanyi/grove/internal/registry"
	"github.com/iheanyi/grove/internal/worktree"
	"github.com/spf13/cobra"
)

var cdCmd = &cobra.Command{
	Use:   "cd <name>",
	Short: "Output the path to a worktree (for shell integration)",
	Long: `Output the path to a worktree for use in shell functions.

This command prints ONLY the path to stdout, making it suitable for shell integration.
It looks up the worktree by name in the registry first, then falls back to git worktree list.

Shell integration example (add to .bashrc/.zshrc):

  grovecd() { cd "$(grove cd "$@")" && grove start 2>/dev/null || true; }

Examples:
  grove cd feature-auth     # Output path to feature-auth worktree
  cd "$(grove cd myapp)"    # Change to myapp worktree directory`,
	Args: cobra.ExactArgs(1),
	RunE: runCd,
}

func runCd(cmd *cobra.Command, args []string) error {
	name := args[0]

	// First, try to find in registry (servers)
	reg, err := registry.Load()
	if err == nil {
		if server, ok := reg.Get(name); ok {
			if _, err := os.Stat(server.Path); err == nil {
				fmt.Println(server.Path)
				return nil
			}
		}

		// Try worktrees in registry
		if wt, ok := reg.GetWorktree(name); ok {
			if _, err := os.Stat(wt.Path); err == nil {
				fmt.Println(wt.Path)
				return nil
			}
		}
	}

	// Fallback: try to find via git worktree list
	path, err := findWorktreeByName(name)
	if err != nil {
		return fmt.Errorf("worktree '%s' not found", name)
	}

	fmt.Println(path)
	return nil
}

// findWorktreeByName searches for a worktree by name using git worktree list
func findWorktreeByName(name string) (string, error) {
	// First, try to detect current repo to get main repo path
	currentWt, err := worktree.Detect()
	if err == nil {
		// Determine the main repository path
		mainRepoPath := currentWt.Path
		if currentWt.IsWorktree && currentWt.MainWorktreePath != "" {
			mainRepoPath = currentWt.MainWorktreePath
		}

		// Search worktrees from current repo
		path, err := searchWorktreesFromRepo(mainRepoPath, name)
		if err == nil {
			return path, nil
		}
	}

	// If not in a git repo or not found, try searching from registry worktrees
	reg, err := registry.Load()
	if err != nil {
		return "", fmt.Errorf("not found")
	}

	// Get a list of unique main repo paths from worktrees
	seenRepos := make(map[string]bool)
	for _, wt := range reg.ListWorktrees() {
		mainRepo := wt.MainRepo
		if mainRepo == "" {
			mainRepo = wt.Path
		}
		if !seenRepos[mainRepo] {
			seenRepos[mainRepo] = true
			path, err := searchWorktreesFromRepo(mainRepo, name)
			if err == nil {
				return path, nil
			}
		}
	}

	// Also check server paths for main repos
	for _, server := range reg.List() {
		if server.Path != "" && !seenRepos[server.Path] {
			seenRepos[server.Path] = true
			path, err := searchWorktreesFromRepo(server.Path, name)
			if err == nil {
				return path, nil
			}
		}
	}

	return "", fmt.Errorf("worktree '%s' not found", name)
}

// searchWorktreesFromRepo searches for a worktree by name within a git repository
func searchWorktreesFromRepo(repoPath string, name string) (string, error) {
	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(output), "\n")
	var currentPath string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "worktree ") {
			currentPath = strings.TrimPrefix(line, "worktree ")

			// Check if the base name matches
			baseName := filepath.Base(currentPath)
			if baseName == name {
				// Verify the path exists
				if _, err := os.Stat(currentPath); err == nil {
					return currentPath, nil
				}
			}
		}
	}

	return "", fmt.Errorf("not found in repo")
}
