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

	// Tasuku integration
	ActiveTask  string `json:"active_task,omitempty"`  // Current Tasuku task ID (if any)
	TaskSummary string `json:"task_summary,omitempty"` // Task description for display
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
	HasGemini bool `json:"has_gemini"` // Gemini CLI is active
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
	wt.HasClaude = agent != nil && agent.Type == "claude"
	wt.HasGemini = agent != nil && agent.Type == "gemini"
	wt.HasVSCode = hasVSCode
	wt.GitDirty = gitDirty

	// If agent detected, check for active Tasuku task
	if agent != nil {
		taskID, taskDesc := GetActiveTask(wt.Path)
		if taskID != "" {
			agent.ActiveTask = taskID
			agent.TaskSummary = taskDesc
		}
	}

	// Update last activity time if any activity detected
	if wt.Agent != nil || wt.HasVSCode || wt.GitDirty {
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

	// Check for Gemini CLI
	if agent := detectGeminiAgent(path); agent != nil {
		return agent
	}

	// Add other agent detection here in the future (Cursor, Copilot, etc.)

	return nil
}

// detectGeminiAgent checks for Gemini CLI activity
func detectGeminiAgent(path string) *AgentInfo {
	// Find Gemini CLI processes using ps aux
	// The [g] trick prevents grep from matching itself
	cmd := exec.Command("bash", "-c", "ps aux | grep -E '[g]emini(-cli)?' | awk '{print $2}'")
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	pids := strings.Fields(strings.TrimSpace(string(output)))
	if len(pids) == 0 {
		return nil
	}

	// Check each gemini process's working directory using lsof
	for _, pidStr := range pids {
		cwd := getProcessCwd(pidStr)
		if cwd != "" && cwd == path {
			pid := 0
			_, _ = fmt.Sscanf(pidStr, "%d", &pid)

			// Get process start time and command
			startTime := getProcessStartTime(pidStr)
			command := getProcessCommand(pidStr)

			return &AgentInfo{
				Type:      "gemini",
				PID:       pid,
				Path:      cwd,
				StartTime: startTime,
				Command:   command,
			}
		}
	}

	return nil
}

// detectClaudeAgent checks for Claude Code activity
func detectClaudeAgent(path string) *AgentInfo {
	// Find Claude Code processes using ps aux
	// The [c] trick prevents grep from matching itself
	cmd := exec.Command("bash", "-c", "ps aux | grep '[c]laude' | awk '{print $2}'")
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
			_, _ = fmt.Sscanf(pidStr, "%d", &pid)

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

// DetectAllAgents finds all active AI agents across all directories.
// This is more efficient than calling DetectActivity for each worktree
// because it finds all agent processes once and batches the lsof calls.
func DetectAllAgents() map[string]*AgentInfo {
	agents := make(map[string]*AgentInfo)

	// Find all Claude and Gemini processes at once
	claudeAgents := detectAllClaudeAgents()
	for path, agent := range claudeAgents {
		agents[path] = agent
	}

	geminiAgents := detectAllGeminiAgents()
	for path, agent := range geminiAgents {
		if _, exists := agents[path]; !exists {
			agents[path] = agent
		}
	}

	return agents
}

// detectAllClaudeAgents finds all Claude Code processes and returns a map of path -> AgentInfo
func detectAllClaudeAgents() map[string]*AgentInfo {
	agents := make(map[string]*AgentInfo)

	// Find Claude Code processes using ps aux
	cmd := exec.Command("bash", "-c", "ps aux | grep '[c]laude' | awk '{print $2}'")
	output, err := cmd.Output()
	if err != nil {
		return agents
	}

	pids := strings.Fields(strings.TrimSpace(string(output)))
	if len(pids) == 0 {
		return agents
	}

	// Get CWDs for all PIDs at once using a single lsof call
	// lsof -d cwd -a -p PID1,PID2,... is more efficient
	pidList := strings.Join(pids, ",")
	lsofCmd := exec.Command("lsof", "-d", "cwd", "-a", "-p", pidList)
	lsofOutput, err := lsofCmd.Output()
	if err != nil {
		// Fall back to individual lookups if batch fails
		return detectAgentsFallback(pids, "claude")
	}

	// Parse lsof output to extract PID -> CWD mapping
	pidToCwd := parseLsofOutput(string(lsofOutput))

	// Build AgentInfo for each unique path
	for pid, cwd := range pidToCwd {
		if _, exists := agents[cwd]; exists {
			continue // Already have an agent for this path
		}

		startTime := getProcessStartTime(pid)
		command := getProcessCommand(pid)
		pidInt := 0
		_, _ = fmt.Sscanf(pid, "%d", &pidInt)

		agents[cwd] = &AgentInfo{
			Type:      "claude",
			PID:       pidInt,
			Path:      cwd,
			StartTime: startTime,
			Command:   command,
		}
	}

	return agents
}

// detectAllGeminiAgents finds all Gemini CLI processes and returns a map of path -> AgentInfo
func detectAllGeminiAgents() map[string]*AgentInfo {
	agents := make(map[string]*AgentInfo)

	// Find Gemini CLI processes
	cmd := exec.Command("bash", "-c", "ps aux | grep -E '[g]emini(-cli)?' | awk '{print $2}'")
	output, err := cmd.Output()
	if err != nil {
		return agents
	}

	pids := strings.Fields(strings.TrimSpace(string(output)))
	if len(pids) == 0 {
		return agents
	}

	// Get CWDs for all PIDs at once
	pidList := strings.Join(pids, ",")
	lsofCmd := exec.Command("lsof", "-d", "cwd", "-a", "-p", pidList)
	lsofOutput, err := lsofCmd.Output()
	if err != nil {
		return detectAgentsFallback(pids, "gemini")
	}

	// Parse lsof output
	pidToCwd := parseLsofOutput(string(lsofOutput))

	for pid, cwd := range pidToCwd {
		if _, exists := agents[cwd]; exists {
			continue
		}

		startTime := getProcessStartTime(pid)
		command := getProcessCommand(pid)
		pidInt := 0
		_, _ = fmt.Sscanf(pid, "%d", &pidInt)

		agents[cwd] = &AgentInfo{
			Type:      "gemini",
			PID:       pidInt,
			Path:      cwd,
			StartTime: startTime,
			Command:   command,
		}
	}

	return agents
}

// parseLsofOutput parses lsof output to extract PID -> CWD mapping
func parseLsofOutput(output string) map[string]string {
	result := make(map[string]string)
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		// lsof -d cwd output format: COMMAND PID USER FD TYPE ... NAME
		// Skip header line
		if strings.HasPrefix(line, "COMMAND") || line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) >= 9 {
			pid := fields[1]
			cwd := fields[len(fields)-1]
			result[pid] = cwd
		}
	}

	return result
}

// detectAgentsFallback is a slower fallback that checks each PID individually
func detectAgentsFallback(pids []string, agentType string) map[string]*AgentInfo {
	agents := make(map[string]*AgentInfo)

	for _, pid := range pids {
		cwd := getProcessCwd(pid)
		if cwd == "" {
			continue
		}

		if _, exists := agents[cwd]; exists {
			continue
		}

		startTime := getProcessStartTime(pid)
		command := getProcessCommand(pid)
		pidInt := 0
		_, _ = fmt.Sscanf(pid, "%d", &pidInt)

		agents[cwd] = &AgentInfo{
			Type:      agentType,
			PID:       pidInt,
			Path:      cwd,
			StartTime: startTime,
			Command:   command,
		}
	}

	return agents
}

// DetectAllVSCode finds all VS Code processes and returns a set of paths where VS Code is active.
// This is more efficient than calling detectVSCode per-worktree since it runs ps aux once.
func DetectAllVSCode() map[string]bool {
	vscodePaths := make(map[string]bool)

	// Run ps aux once and look for VS Code processes with path arguments
	cmd := exec.Command("ps", "aux")
	output, err := cmd.Output()
	if err != nil {
		return vscodePaths
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if !strings.Contains(line, "code") && !strings.Contains(line, "Code") {
			continue
		}

		// Extract paths from the command line (look for common path patterns)
		fields := strings.Fields(line)
		for _, field := range fields {
			// Skip if it's not a path
			if !strings.HasPrefix(field, "/") {
				continue
			}
			// Check if it looks like a project directory (exists and is a directory)
			if info, err := os.Stat(field); err == nil && info.IsDir() {
				vscodePaths[field] = true
			}
		}
	}

	return vscodePaths
}

// DetectActivitiesBatch efficiently detects activities for multiple worktrees.
// It batches the expensive operations (lsof for agents, ps for VS Code) and
// parallelizes git status checks.
func DetectActivitiesBatch(worktrees []*Worktree) {
	if len(worktrees) == 0 {
		return
	}

	// Batch 1: Get all agents at once (single lsof call)
	agents := DetectAllAgents()

	// Batch 2: Get all VS Code paths at once (single ps call)
	vscodePaths := DetectAllVSCode()

	// Parallel: Run git status for each worktree
	var wg sync.WaitGroup
	results := make(chan struct {
		idx      int
		gitDirty bool
	}, len(worktrees))

	for i, wt := range worktrees {
		wg.Add(1)
		go func(idx int, path string) {
			defer wg.Done()
			results <- struct {
				idx      int
				gitDirty bool
			}{idx, detectGitDirty(path)}
		}(i, wt.Path)
	}

	// Close results channel when all goroutines complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect git dirty results
	gitDirty := make([]bool, len(worktrees))
	for result := range results {
		gitDirty[result.idx] = result.gitDirty
	}

	// Apply all results to worktrees
	for i, wt := range worktrees {
		// Agent detection
		if agent, exists := agents[wt.Path]; exists {
			wt.Agent = agent
			wt.HasClaude = agent.Type == "claude"
			wt.HasGemini = agent.Type == "gemini"

			// Check for active Tasuku task
			taskID, taskDesc := GetActiveTask(wt.Path)
			if taskID != "" {
				agent.ActiveTask = taskID
				agent.TaskSummary = taskDesc
			}
		} else {
			wt.Agent = nil
			wt.HasClaude = false
			wt.HasGemini = false
		}

		// VS Code detection (check for exact match or parent directory)
		wt.HasVSCode = vscodePaths[wt.Path]
		if !wt.HasVSCode {
			// Check if VS Code is open on a parent directory
			for vsPath := range vscodePaths {
				if strings.HasPrefix(wt.Path, vsPath+"/") {
					wt.HasVSCode = true
					break
				}
			}
		}

		// Git dirty
		wt.GitDirty = gitDirty[i]

		// Update last activity
		if wt.Agent != nil || wt.HasVSCode || wt.GitDirty {
			wt.LastActivity = time.Now()
		}
	}
}
