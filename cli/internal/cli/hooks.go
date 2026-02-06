package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var hooksCmd = &cobra.Command{
	Use:   "hooks",
	Short: "Manage Claude Code hooks for Grove integration",
	Long: `Manage Claude Code hooks that help AI agents use Grove effectively.

These hooks remind AI agents to:
- Use 'grove start' instead of running dev servers directly
- Use 'grove new' instead of 'git worktree add'
- Check grove status at session start
- Update documentation when features are added`,
}

var hooksInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install Claude Code hooks for Grove",
	Long: `Install Claude Code hooks into the current project.

This creates hooks in .claude/settings.json that:
- Show grove status at session start
- Intercept direct dev server commands (npm run dev, rails s, etc.)
- Intercept git worktree add commands
- Remind about documentation updates

The hooks are project-local and won't affect other projects.`,
	RunE: runHooksInstall,
}

var hooksUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove Grove hooks from Claude Code",
	Long:  `Remove Grove-related hooks from .claude/settings.json`,
	RunE:  runHooksUninstall,
}

func init() {
	hooksCmd.GroupID = "config"
	rootCmd.AddCommand(hooksCmd)
	hooksCmd.AddCommand(hooksInstallCmd)
	hooksCmd.AddCommand(hooksUninstallCmd)
}

// Hook script content
const groveSessionStartHook = `#!/bin/bash
# Grove SessionStart hook - shows grove status and active Tasuku task
set -e

input=$(cat)
cwd=$(echo "$input" | jq -r '.cwd // ""')

if [ -z "$cwd" ]; then
  exit 0
fi

cd "$cwd" 2>/dev/null || exit 0

# Check for active Tasuku task first
if command -v tk &> /dev/null; then
  active_task=$(tk task list --status in_progress --format json 2>/dev/null | jq -r '.[0] // empty')
  if [ -n "$active_task" ]; then
    task_id=$(echo "$active_task" | jq -r '.id // ""')
    task_desc=$(echo "$active_task" | jq -r '.description // ""')
    if [ -n "$task_id" ]; then
      echo "📋 Active task: $task_id"
      echo "   $task_desc"
      echo ""
    fi
  fi
fi

# Check if grove is available
if ! command -v grove &> /dev/null; then
  exit 0
fi

# Get grove status
servers=$(grove ls --json 2>/dev/null || echo '{"servers":[]}')
running=$(echo "$servers" | jq '[.servers[] | select(.status == "running")] | length')
total=$(echo "$servers" | jq '.servers | length')

if [ "$total" -gt 0 ]; then
  echo "Grove: $running/$total servers running"
  if [ "$running" -gt 0 ]; then
    echo "$servers" | jq -r '.servers[] | select(.status == "running") | "  - \(.name): \(.url)"'
  fi
  echo ""
  echo "Use 'grove start <cmd>' to start servers, 'grove new <branch>' to create worktrees."
fi

exit 0
`

const groveDevServerHook = `#!/bin/bash
# Grove PreToolUse hook - intercepts direct dev server commands
set -e

input=$(cat)
tool_name=$(echo "$input" | jq -r '.tool_name // ""')
command=$(echo "$input" | jq -r '.tool_input.command // ""')

# Only check Bash commands
if [ "$tool_name" != "Bash" ]; then
  exit 0
fi

# Check for common dev server commands
if echo "$command" | grep -qE '(npm run dev|yarn dev|pnpm dev|rails s|rails server|bin/dev|python.*manage\.py.*runserver|go run|cargo run.*server)'; then
  echo "💡 Consider using 'grove start $command' instead."
  echo "   Grove automatically:"
  echo "   - Sets PORT env var (your server should use process.env.PORT or ENV['PORT'])"
  echo "   - Allocates consistent ports per worktree (same branch = same port)"
  echo "   - Manages logs at ~/.config/grove/logs/<name>.log"
  echo "   - Tracks server status for the menubar app and MCP tools"
  echo ""
  echo "   Example: grove start npm run dev"
  echo ""
fi

exit 0
`

const groveWorktreeHook = `#!/bin/bash
# Grove PreToolUse hook - intercepts git worktree commands
set -e

input=$(cat)
tool_name=$(echo "$input" | jq -r '.tool_name // ""')
command=$(echo "$input" | jq -r '.tool_input.command // ""')

# Only check Bash commands
if [ "$tool_name" != "Bash" ]; then
  exit 0
fi

# Check for git worktree add
if echo "$command" | grep -qE 'git worktree add'; then
  echo "💡 Consider using 'grove new <branch-name>' instead of 'git worktree add'."
  echo "   Grove automatically:"
  echo "   - Creates worktrees in a consistent location (configurable via worktrees_dir)"
  echo "   - Registers worktrees for tracking and easy switching"
  echo "   - Supports --dir flag to override worktree location"
  echo "   - Enables 'grove switch <name>' to jump between worktrees"
  echo ""
  echo "   Example: grove new feature-auth"
  echo "   Example: grove new feature-auth --dir ~/worktrees"
  echo ""
fi

exit 0
`

const groveDocReminderHook = `#!/bin/bash
# Grove Stop hook - reminds about documentation and task status updates
set -e

input=$(cat)
cwd=$(echo "$input" | jq -r '.cwd // ""')

if [ -z "$cwd" ]; then
  exit 0
fi

cd "$cwd" 2>/dev/null || exit 0

# Check for active Tasuku task and remind about status
if command -v tk &> /dev/null; then
  active_task=$(tk task list --status in_progress --format json 2>/dev/null | jq -r '.[0].id // empty')
  if [ -n "$active_task" ]; then
    echo ""
    echo "📋 Task '$active_task' is still in progress."
    echo "   If work is complete, run: tk task done $active_task"
    echo "   If pausing, task will resume next session."
  fi
fi

# Check if we're in a git repo
if ! git rev-parse --git-dir &> /dev/null; then
  exit 0
fi

# Check for uncommitted code changes (excluding tests and docs)
code_changes=$(git diff --name-only --diff-filter=ACM 2>/dev/null | grep -E '\.(go|ts|tsx|js|jsx|py|rb|swift)$' | grep -vE '(_test|\.test|\.spec|_spec)' || true)

if [ -n "$code_changes" ]; then
  # Check if README or docs were also modified
  doc_changes=$(git diff --name-only 2>/dev/null | grep -iE '(readme|doc|mcp)' || true)

  if [ -z "$doc_changes" ]; then
    echo ""
    echo "📝 Reminder: Code files were modified. Consider updating:"
    echo "   - README.md (if user-facing features changed)"
    echo "   - MCP documentation (if CLI commands changed)"
  fi
fi

exit 0
`

// ClaudeSettings represents the structure of .claude/settings.json
type ClaudeSettings struct {
	Hooks       map[string][]HookMatcher   `json:"hooks,omitempty"`
	Permissions map[string][]string        `json:"permissions,omitempty"`
	Other       map[string]json.RawMessage `json:"-"` // Preserve unknown fields
}

type HookMatcher struct {
	Matcher string `json:"matcher,omitempty"`
	Hooks   []Hook `json:"hooks,omitempty"`
	// For simple hooks without matcher
	Type    string `json:"type,omitempty"`
	Command string `json:"command,omitempty"`
	Timeout int    `json:"timeout,omitempty"`
}

type Hook struct {
	Type    string `json:"type"`
	Command string `json:"command"`
	Timeout int    `json:"timeout,omitempty"`
}

func runHooksInstall(cmd *cobra.Command, args []string) error {
	// Ensure .claude directory exists
	claudeDir := ".claude"
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		return fmt.Errorf("failed to create .claude directory: %w", err)
	}

	// Ensure hooks directory exists
	hooksDir := filepath.Join(claudeDir, "hooks")
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		return fmt.Errorf("failed to create hooks directory: %w", err)
	}

	// Write hook scripts
	hookScripts := map[string]string{
		"grove-session-start.sh": groveSessionStartHook,
		"grove-dev-server.sh":    groveDevServerHook,
		"grove-worktree.sh":      groveWorktreeHook,
		"grove-doc-reminder.sh":  groveDocReminderHook,
	}

	for name, content := range hookScripts {
		path := filepath.Join(hooksDir, name)
		if err := os.WriteFile(path, []byte(content), 0755); err != nil {
			return fmt.Errorf("failed to write %s: %w", name, err)
		}
	}

	// Load or create settings.json
	settingsPath := filepath.Join(claudeDir, "settings.json")
	settings := make(map[string]interface{})

	if data, err := os.ReadFile(settingsPath); err == nil {
		if err := json.Unmarshal(data, &settings); err != nil {
			return fmt.Errorf("failed to parse existing settings: %w", err)
		}
	}

	// Get or create hooks section
	hooks, ok := settings["hooks"].(map[string]interface{})
	if !ok {
		hooks = make(map[string]interface{})
	}

	// Add SessionStart hook
	sessionStartHooks := getOrCreateHookArray(hooks, "SessionStart")
	if !hasGroveHook(sessionStartHooks, "grove-session-start.sh") {
		sessionStartHooks = append(sessionStartHooks, map[string]interface{}{
			"type":    "command",
			"command": ".claude/hooks/grove-session-start.sh",
		})
		hooks["SessionStart"] = sessionStartHooks
	}

	// Add PreToolUse hooks for dev server and worktree interception
	preToolUseHooks := getOrCreateHookArray(hooks, "PreToolUse")

	// Dev server hook
	if !hasGroveHook(preToolUseHooks, "grove-dev-server.sh") {
		preToolUseHooks = append(preToolUseHooks, map[string]interface{}{
			"matcher": "Bash",
			"hooks": []map[string]interface{}{
				{
					"type":    "command",
					"command": ".claude/hooks/grove-dev-server.sh",
				},
			},
		})
	}

	// Worktree hook
	if !hasGroveHook(preToolUseHooks, "grove-worktree.sh") {
		preToolUseHooks = append(preToolUseHooks, map[string]interface{}{
			"matcher": "Bash",
			"hooks": []map[string]interface{}{
				{
					"type":    "command",
					"command": ".claude/hooks/grove-worktree.sh",
				},
			},
		})
	}
	hooks["PreToolUse"] = preToolUseHooks

	// Add Stop hook for doc reminder
	stopHooks := getOrCreateHookArray(hooks, "Stop")
	if !hasGroveHook(stopHooks, "grove-doc-reminder.sh") {
		stopHooks = append(stopHooks, map[string]interface{}{
			"type":    "command",
			"command": ".claude/hooks/grove-doc-reminder.sh",
		})
		hooks["Stop"] = stopHooks
	}

	settings["hooks"] = hooks

	// Write updated settings
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	if err := os.WriteFile(settingsPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write settings: %w", err)
	}

	fmt.Println("✓ Installed Grove hooks for Claude Code")
	fmt.Println()
	fmt.Println("Hooks installed:")
	fmt.Println("  - SessionStart: Shows grove server status")
	fmt.Println("  - PreToolUse:   Suggests 'grove start' for dev server commands")
	fmt.Println("  - PreToolUse:   Suggests 'grove new' for git worktree commands")
	fmt.Println("  - Stop:         Reminds about documentation updates")
	fmt.Println()
	fmt.Println("Files created:")
	fmt.Println("  - .claude/settings.json")
	fmt.Println("  - .claude/hooks/grove-session-start.sh")
	fmt.Println("  - .claude/hooks/grove-dev-server.sh")
	fmt.Println("  - .claude/hooks/grove-worktree.sh")
	fmt.Println("  - .claude/hooks/grove-doc-reminder.sh")
	fmt.Println()
	fmt.Println("Note: Add .claude/settings.json to git to share hooks with your team.")
	fmt.Println("      Add .claude/hooks/ to git as well.")

	return nil
}

func runHooksUninstall(cmd *cobra.Command, args []string) error {
	claudeDir := ".claude"
	settingsPath := filepath.Join(claudeDir, "settings.json")

	// Load settings
	settings := make(map[string]interface{})
	if data, err := os.ReadFile(settingsPath); err == nil {
		if err := json.Unmarshal(data, &settings); err != nil {
			return fmt.Errorf("failed to parse settings: %w", err)
		}
	} else {
		fmt.Println("No .claude/settings.json found - nothing to uninstall")
		return nil
	}

	// Remove grove hooks from settings
	hooks, ok := settings["hooks"].(map[string]interface{})
	if ok {
		for event, eventHooks := range hooks {
			if hookArray, ok := eventHooks.([]interface{}); ok {
				filtered := filterOutGroveHooks(hookArray)
				if len(filtered) == 0 {
					delete(hooks, event)
				} else {
					hooks[event] = filtered
				}
			}
		}
		if len(hooks) == 0 {
			delete(settings, "hooks")
		} else {
			settings["hooks"] = hooks
		}
	}

	// Write updated settings
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	if err := os.WriteFile(settingsPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write settings: %w", err)
	}

	// Remove hook scripts
	hooksDir := filepath.Join(claudeDir, "hooks")
	groveHooks := []string{
		"grove-session-start.sh",
		"grove-dev-server.sh",
		"grove-worktree.sh",
		"grove-doc-reminder.sh",
	}

	for _, name := range groveHooks {
		path := filepath.Join(hooksDir, name)
		os.Remove(path) // Ignore errors - file might not exist
	}

	fmt.Println("✓ Removed Grove hooks from Claude Code")

	return nil
}

func getOrCreateHookArray(hooks map[string]interface{}, event string) []interface{} {
	if existing, ok := hooks[event].([]interface{}); ok {
		return existing
	}
	return []interface{}{}
}

func hasGroveHook(hooks []interface{}, scriptName string) bool {
	for _, h := range hooks {
		hookMap, ok := h.(map[string]interface{})
		if !ok {
			continue
		}

		// Check direct command
		if cmd, ok := hookMap["command"].(string); ok {
			if containsString(cmd, scriptName) {
				return true
			}
		}

		// Check nested hooks
		if nestedHooks, ok := hookMap["hooks"].([]interface{}); ok {
			for _, nh := range nestedHooks {
				if nhMap, ok := nh.(map[string]interface{}); ok {
					if cmd, ok := nhMap["command"].(string); ok {
						if containsString(cmd, scriptName) {
							return true
						}
					}
				}
			}
		}
	}
	return false
}

func filterOutGroveHooks(hooks []interface{}) []interface{} {
	result := []interface{}{}
	for _, h := range hooks {
		hookMap, ok := h.(map[string]interface{})
		if !ok {
			result = append(result, h)
			continue
		}

		// Check if it's a grove hook
		isGrove := false

		if cmd, ok := hookMap["command"].(string); ok {
			if containsString(cmd, "grove-") {
				isGrove = true
			}
		}

		if nestedHooks, ok := hookMap["hooks"].([]interface{}); ok {
			for _, nh := range nestedHooks {
				if nhMap, ok := nh.(map[string]interface{}); ok {
					if cmd, ok := nhMap["command"].(string); ok {
						if containsString(cmd, "grove-") {
							isGrove = true
							break
						}
					}
				}
			}
		}

		if !isGrove {
			result = append(result, h)
		}
	}
	return result
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
