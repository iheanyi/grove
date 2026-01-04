package discovery

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// AgentInfo represents an active AI agent/assistant session
type AgentInfo struct {
	Type      string    `json:"type"`       // "claude", "cursor", "copilot", etc.
	PID       int       `json:"pid"`        // Process ID
	Path      string    `json:"path"`       // Working directory
	StartTime time.Time `json:"start_time"` // When the process started
	Command   string    `json:"command"`    // Full command line
}

// Worktree represents a discovered git worktree
type Worktree struct {
	Name         string    `json:"name"`
	Path         string    `json:"path"`
	Branch       string    `json:"branch"`
	MainRepo     string    `json:"main_repo"` // Path to the main repo this worktree belongs to
	DiscoveredAt time.Time `json:"discovered_at"`
	LastActivity time.Time `json:"last_activity"`

	// Activity indicators
	HasServer bool `json:"has_server"` // We have a server registered for this
	HasClaude bool `json:"has_claude"` // Claude Code is active (detected via socket/process)
	HasVSCode bool `json:"has_vscode"` // VS Code is open (detected via process)
	GitDirty  bool `json:"git_dirty"`  // Has uncommitted changes

	// Detailed agent info (populated when HasClaude is true)
	Agent *AgentInfo `json:"agent,omitempty"`
}

// Discover finds all worktrees for a given repo
func Discover(repoPath string) ([]*Worktree, error) {
	absPath, err := filepath.Abs(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Use git worktree list to find all worktrees
	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	cmd.Dir = absPath
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list worktrees: %w", err)
	}

	worktrees, err := parseWorktreeList(string(output))
	if err != nil {
		return nil, err
	}

	// Detect activity for each worktree
	for _, wt := range worktrees {
		if err := DetectActivity(wt); err != nil {
			// Log error but continue
			continue
		}
	}

	return worktrees, nil
}

// parseWorktreeList parses the output of `git worktree list --porcelain`
func parseWorktreeList(output string) ([]*Worktree, error) {
	var worktrees []*Worktree
	var current *Worktree
	var mainRepoPath string

	lines := strings.Split(output, "\n")
	for i, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "worktree ") {
			if current != nil {
				worktrees = append(worktrees, current)
			}

			path := strings.TrimPrefix(line, "worktree ")
			current = &Worktree{
				Path:         path,
				DiscoveredAt: time.Now(),
				LastActivity: time.Now(),
			}

			// First worktree is the main repo
			if i == 0 || mainRepoPath == "" {
				mainRepoPath = path
			}
			current.MainRepo = mainRepoPath

		} else if strings.HasPrefix(line, "branch ") && current != nil {
			branch := strings.TrimPrefix(line, "branch ")
			branch = strings.TrimPrefix(branch, "refs/heads/")
			current.Branch = branch
			// For main repo (first worktree), use directory name instead of branch
			// This makes standalone repos show as "myapp" instead of "main"
			if current.Path == mainRepoPath {
				current.Name = sanitizeBranchName(filepath.Base(current.Path))
			} else {
				current.Name = sanitizeBranchName(branch)
			}

		} else if strings.HasPrefix(line, "HEAD ") && current != nil && current.Branch == "" {
			// Detached HEAD state
			current.Branch = "HEAD"
			current.Name = "detached-head"

		} else if strings.HasPrefix(line, "detached") && current != nil {
			// Mark as detached
			if current.Branch == "" {
				current.Branch = "HEAD"
				current.Name = "detached-head"
			}
		}
	}

	if current != nil {
		worktrees = append(worktrees, current)
	}

	return worktrees, nil
}

// DetectActivity checks for various activities in a worktree.
// All checks run in parallel for performance.
func DetectActivity(wt *Worktree) error {
	var wg sync.WaitGroup
	var agent *AgentInfo
	var hasVSCode, gitDirty bool

	// Run all detection checks in parallel
	wg.Add(3)

	go func() {
		defer wg.Done()
		agent = detectAgent(wt.Path)
	}()

	go func() {
		defer wg.Done()
		hasVSCode = detectVSCode(wt.Path)
	}()

	go func() {
		defer wg.Done()
		gitDirty = detectGitDirty(wt.Path)
	}()

	wg.Wait()

	wt.Agent = agent
	wt.HasClaude = agent != nil
	wt.HasVSCode = hasVSCode
	wt.GitDirty = gitDirty

	// Update last activity time if any activity detected
	if wt.HasClaude || wt.HasVSCode || wt.GitDirty {
		wt.LastActivity = time.Now()
	}

	return nil
}

// detectAgent checks for AI agent activity and returns detailed info
func detectAgent(path string) *AgentInfo {
	// Check for Claude Code first
	if agent := detectClaudeAgent(path); agent != nil {
		return agent
	}

	// Add other agent detection here in the future (Cursor, Copilot, etc.)

	return nil
}

// detectClaudeAgent checks for Claude Code activity
func detectClaudeAgent(path string) *AgentInfo {
	// Find Claude Code processes using ps aux
	// Claude processes have "claude" in the command (with optional flags)
	cmd := exec.Command("bash", "-c", "ps aux | grep -E '[c]laude\\s*(--|-|$)' | awk '{print $2}'")
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	pids := strings.Fields(strings.TrimSpace(string(output)))
	if len(pids) == 0 {
		return nil
	}

	// Check each claude process's working directory using lsof
	for _, pidStr := range pids {
		cwd := getProcessCwd(pidStr)
		if cwd != "" && cwd == path {
			pid := 0
			fmt.Sscanf(pidStr, "%d", &pid)

			// Get process start time and command
			startTime := getProcessStartTime(pidStr)
			command := getProcessCommand(pidStr)

			return &AgentInfo{
				Type:      "claude",
				PID:       pid,
				Path:      cwd,
				StartTime: startTime,
				Command:   command,
			}
		}
	}

	return nil
}

// getProcessStartTime returns the start time of a process
func getProcessStartTime(pid string) time.Time {
	// Use ps to get process start time
	cmd := exec.Command("ps", "-p", pid, "-o", "lstart=")
	output, err := cmd.Output()
	if err != nil {
		return time.Time{}
	}

	// Parse the date string (format: "Mon Jan  2 15:04:05 2006")
	timeStr := strings.TrimSpace(string(output))
	t, err := time.Parse("Mon Jan  2 15:04:05 2006", timeStr)
	if err != nil {
		// Try alternative format
		t, err = time.Parse("Mon Jan 2 15:04:05 2006", timeStr)
		if err != nil {
			return time.Time{}
		}
	}
	return t
}

// getProcessCommand returns the full command line of a process
func getProcessCommand(pid string) string {
	cmd := exec.Command("ps", "-p", pid, "-o", "command=")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

// getProcessCwd returns the current working directory of a process
func getProcessCwd(pid string) string {
	cmd := exec.Command("lsof", "-p", pid)
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "cwd") {
			// lsof output format: "node  PID user  cwd  DIR  ...  /path/to/dir"
			fields := strings.Fields(line)
			if len(fields) >= 9 {
				return fields[len(fields)-1]
			}
		}
	}
	return ""
}

// detectVSCode checks for VS Code activity
func detectVSCode(path string) bool {
	// Check for .vscode-server directory (remote development)
	vscodeServerPath := filepath.Join(path, ".vscode-server")
	if info, err := os.Stat(vscodeServerPath); err == nil && info.IsDir() {
		return true
	}

	// Check for code process with this path
	return checkProcessWithPath("code", path)
}

// detectGitDirty checks if the worktree has uncommitted changes
func detectGitDirty(path string) bool {
	cmd := exec.Command("git", "-C", path, "status", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	// If output is not empty, there are changes
	return len(strings.TrimSpace(string(output))) > 0
}

// checkProcessWithPath checks if a process with the given name has the path as an argument
func checkProcessWithPath(processName, path string) bool {
	// Use ps to find processes
	cmd := exec.Command("ps", "aux")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, processName) && strings.Contains(line, path) {
			return true
		}
	}

	return false
}

// sanitizeBranchName converts a branch name to a URL-safe name
func sanitizeBranchName(branch string) string {
	// Replace / with -
	name := strings.ReplaceAll(branch, "/", "-")

	// Replace other special characters
	name = strings.ReplaceAll(name, "_", "-")
	name = strings.ReplaceAll(name, " ", "-")

	// Convert to lowercase
	name = strings.ToLower(name)

	// Remove any remaining special characters
	var result strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		}
	}

	return result.String()
}

// FindAll discovers all git repositories in a directory tree
func FindAll(basePath string, maxDepth int) ([]*Worktree, error) {
	var allWorktrees []*Worktree
	seen := make(map[string]bool)

	var scan func(path string, depth int) error
	scan = func(path string, depth int) error {
		if maxDepth >= 0 && depth > maxDepth {
			return nil
		}

		// Check if this is a git repository
		gitPath := filepath.Join(path, ".git")
		if info, err := os.Stat(gitPath); err == nil && info.IsDir() {
			// Found a main git repo, discover its worktrees
			worktrees, err := Discover(path)
			if err == nil {
				for _, wt := range worktrees {
					if !seen[wt.Path] {
						seen[wt.Path] = true
						allWorktrees = append(allWorktrees, wt)
					}
				}
			}
			// Don't descend into git repos
			return nil
		}

		// Not a git repo, scan subdirectories
		entries, err := os.ReadDir(path)
		if err != nil {
			return nil //nolint:nilerr // Intentionally skip unreadable directories and continue walk
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

			if err := scan(filepath.Join(path, name), depth+1); err != nil {
				return err
			}
		}

		return nil
	}

	if err := scan(basePath, 0); err != nil {
		return nil, err
	}

	return allWorktrees, nil
}
