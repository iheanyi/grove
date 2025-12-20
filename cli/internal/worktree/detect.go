package worktree

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Info contains information about the current worktree/repository
type Info struct {
	// Name is the sanitized name suitable for URLs (e.g., "feature-auth")
	Name string

	// Branch is the current branch name (e.g., "feature/auth")
	Branch string

	// Path is the absolute path to the worktree root
	Path string

	// IsWorktree indicates if this is a linked worktree (vs main working tree)
	IsWorktree bool

	// MainWorktreePath is the path to the main worktree (if this is a linked worktree)
	MainWorktreePath string
}

// Detect detects the current git worktree/repository information
func Detect() (*Info, error) {
	return DetectAt(".")
}

// DetectAt detects git worktree/repository information at the given path
func DetectAt(path string) (*Info, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Use git commands for better worktree support
	// Get the top-level directory of the worktree
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = absPath
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("not a git repository: %w", err)
	}
	wtPath := strings.TrimSpace(string(output))

	// Get current branch name
	cmd = exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = absPath
	output, err = cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get branch: %w", err)
	}
	branch := strings.TrimSpace(string(output))

	// Handle detached HEAD state
	if branch == "HEAD" {
		// Try to get a more descriptive name
		cmd = exec.Command("git", "describe", "--tags", "--always")
		cmd.Dir = absPath
		output, err = cmd.Output()
		if err == nil {
			branch = strings.TrimSpace(string(output))
		}
	}

	// Check if this is a linked worktree
	isWorktree, mainPath := detectLinkedWorktree(wtPath)

	info := &Info{
		Name:             Sanitize(branch),
		Branch:           branch,
		Path:             wtPath,
		IsWorktree:       isWorktree,
		MainWorktreePath: mainPath,
	}

	return info, nil
}

// detectLinkedWorktree checks if the path is a linked worktree and returns the main worktree path
func detectLinkedWorktree(path string) (bool, string) {
	gitDir := filepath.Join(path, ".git")

	// Check if .git is a file (indicates linked worktree) or directory (main worktree)
	info, err := os.Stat(gitDir)
	if err != nil {
		return false, ""
	}

	if info.IsDir() {
		// This is the main working tree
		return false, ""
	}

	// .git is a file, read it to find the actual git dir
	content, err := os.ReadFile(gitDir)
	if err != nil {
		return false, ""
	}

	// Format: "gitdir: /path/to/main/.git/worktrees/worktree-name"
	line := strings.TrimSpace(string(content))
	if !strings.HasPrefix(line, "gitdir: ") {
		return false, ""
	}

	gitDirPath := strings.TrimPrefix(line, "gitdir: ")

	// Extract main worktree path (go up from .git/worktrees/name to main repo)
	// Path looks like: /main/repo/.git/worktrees/worktree-name
	if strings.Contains(gitDirPath, "/.git/worktrees/") {
		parts := strings.Split(gitDirPath, "/.git/worktrees/")
		if len(parts) > 0 {
			return true, parts[0]
		}
	}

	return true, ""
}

// FindProjectRoot walks up the directory tree to find the project root
// (directory containing .wt.yaml, .git, or common project markers)
func FindProjectRoot(startPath string) (string, error) {
	absPath, err := filepath.Abs(startPath)
	if err != nil {
		return "", err
	}

	current := absPath
	for {
		// Check for common project root markers
		markers := []string{".wt.yaml", ".git", "go.mod", "package.json", "Gemfile", "Cargo.toml"}
		for _, marker := range markers {
			if _, err := os.Stat(filepath.Join(current, marker)); err == nil {
				return current, nil
			}
		}

		// Move up one directory
		parent := filepath.Dir(current)
		if parent == current {
			// Reached root, no project found
			return absPath, nil
		}
		current = parent
	}
}
