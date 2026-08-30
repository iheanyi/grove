package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// chdir changes to dir for the duration of the test, restoring the original on cleanup.
func chdir(t *testing.T, dir string) {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir %s: %v", dir, err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(orig); err != nil {
			t.Errorf("restore working directory: %v", err)
		}
	})
}

func newHooksTestDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	chdir(t, dir)
	return dir
}

func readSettings(t *testing.T, dir string) map[string]interface{} {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, ".claude", "settings.json"))
	if err != nil {
		t.Fatalf("read settings.json: %v", err)
	}
	var out map[string]interface{}
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("parse settings.json: %v", err)
	}
	return out
}

// commandsForEvent collects all "command" strings nested inside a hook event array.
func commandsForEvent(t *testing.T, settings map[string]interface{}, event string) []string {
	t.Helper()
	hooks, ok := settings["hooks"].(map[string]interface{})
	if !ok {
		return nil
	}
	eventHooks, ok := hooks[event].([]interface{})
	if !ok {
		return nil
	}
	var cmds []string
	for _, h := range eventHooks {
		hm, ok := h.(map[string]interface{})
		if !ok {
			continue
		}
		if cmd, ok := hm["command"].(string); ok {
			cmds = append(cmds, cmd)
		}
		if nested, ok := hm["hooks"].([]interface{}); ok {
			for _, nh := range nested {
				nm, ok := nh.(map[string]interface{})
				if !ok {
					continue
				}
				if cmd, ok := nm["command"].(string); ok {
					cmds = append(cmds, cmd)
				}
			}
		}
	}
	return cmds
}

func countCommandsForEvent(t *testing.T, settings map[string]interface{}, event string) int {
	return len(commandsForEvent(t, settings, event))
}

// --- install tests ---

func TestHooksInstall_CreatesAllScripts(t *testing.T) {
	dir := newHooksTestDir(t)

	if err := runHooksInstall(nil, nil); err != nil {
		t.Fatalf("runHooksInstall: %v", err)
	}

	expected := []string{
		"grove-session-start.sh",
		"grove-dev-server.sh",
		"grove-worktree.sh",
		"grove-doc-reminder.sh",
		"grove-worktree-create.sh",
		"grove-worktree-remove.sh",
	}
	for _, name := range expected {
		path := filepath.Join(dir, ".claude", "hooks", name)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("expected script %s to exist: %v", name, err)
		}
	}
}

func TestHooksInstall_WorktreeHooksInSettings(t *testing.T) {
	dir := newHooksTestDir(t)

	if err := runHooksInstall(nil, nil); err != nil {
		t.Fatalf("runHooksInstall: %v", err)
	}

	settings := readSettings(t, dir)

	// WorktreeCreate must reference the create script via $CLAUDE_PROJECT_DIR
	createCmds := commandsForEvent(t, settings, "WorktreeCreate")
	if len(createCmds) == 0 {
		t.Fatal("expected WorktreeCreate hook commands, got none")
	}
	found := false
	for _, cmd := range createCmds {
		if strings.Contains(cmd, "grove-worktree-create.sh") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("WorktreeCreate commands do not reference grove-worktree-create.sh: %v", createCmds)
	}
	for _, cmd := range createCmds {
		if strings.Contains(cmd, "grove-worktree-create.sh") && !strings.Contains(cmd, "$CLAUDE_PROJECT_DIR") {
			t.Errorf("WorktreeCreate command should use $CLAUDE_PROJECT_DIR, got: %s", cmd)
		}
	}

	// WorktreeRemove must reference the remove script via $CLAUDE_PROJECT_DIR
	removeCmds := commandsForEvent(t, settings, "WorktreeRemove")
	if len(removeCmds) == 0 {
		t.Fatal("expected WorktreeRemove hook commands, got none")
	}
	found = false
	for _, cmd := range removeCmds {
		if strings.Contains(cmd, "grove-worktree-remove.sh") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("WorktreeRemove commands do not reference grove-worktree-remove.sh: %v", removeCmds)
	}
	for _, cmd := range removeCmds {
		if strings.Contains(cmd, "grove-worktree-remove.sh") && !strings.Contains(cmd, "$CLAUDE_PROJECT_DIR") {
			t.Errorf("WorktreeRemove command should use $CLAUDE_PROJECT_DIR, got: %s", cmd)
		}
	}
}

func TestHooksInstall_AllExpectedEventsPresent(t *testing.T) {
	dir := newHooksTestDir(t)

	if err := runHooksInstall(nil, nil); err != nil {
		t.Fatalf("runHooksInstall: %v", err)
	}

	settings := readSettings(t, dir)
	for _, event := range []string{"SessionStart", "PreToolUse", "Stop", "WorktreeCreate", "WorktreeRemove"} {
		if n := countCommandsForEvent(t, settings, event); n == 0 {
			t.Errorf("expected at least one command for event %s", event)
		}
	}
}

func TestHooksInstall_IsIdempotent(t *testing.T) {
	dir := newHooksTestDir(t)

	if err := runHooksInstall(nil, nil); err != nil {
		t.Fatalf("first runHooksInstall: %v", err)
	}
	if err := runHooksInstall(nil, nil); err != nil {
		t.Fatalf("second runHooksInstall: %v", err)
	}

	settings := readSettings(t, dir)

	for _, event := range []string{"WorktreeCreate", "WorktreeRemove"} {
		if n := countCommandsForEvent(t, settings, event); n != 1 {
			t.Errorf("expected exactly 1 command for %s after double install, got %d", event, n)
		}
	}

	// SessionStart and Stop should each have exactly 1 entry too
	for _, event := range []string{"SessionStart", "Stop"} {
		if n := countCommandsForEvent(t, settings, event); n != 1 {
			t.Errorf("expected exactly 1 command for %s after double install, got %d", event, n)
		}
	}
}

func TestHooksInstall_ScriptsAreExecutable(t *testing.T) {
	dir := newHooksTestDir(t)

	if err := runHooksInstall(nil, nil); err != nil {
		t.Fatalf("runHooksInstall: %v", err)
	}

	for _, name := range []string{"grove-worktree-create.sh", "grove-worktree-remove.sh"} {
		path := filepath.Join(dir, ".claude", "hooks", name)
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("stat %s: %v", name, err)
		}
		if info.Mode()&0111 == 0 {
			t.Errorf("expected %s to be executable, got mode %v", name, info.Mode())
		}
	}
}

func TestHooksInstall_WorktreeCreateScriptContent(t *testing.T) {
	dir := newHooksTestDir(t)

	if err := runHooksInstall(nil, nil); err != nil {
		t.Fatalf("runHooksInstall: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, ".claude", "hooks", "grove-worktree-create.sh"))
	if err != nil {
		t.Fatalf("read grove-worktree-create.sh: %v", err)
	}
	s := string(content)

	checks := []string{"#!/bin/bash", "grove discover --register", "/.claude/worktrees/"}
	for _, want := range checks {
		if !strings.Contains(s, want) {
			t.Errorf("grove-worktree-create.sh missing %q", want)
		}
	}
}

func TestHooksInstall_WorktreeRemoveScriptContent(t *testing.T) {
	dir := newHooksTestDir(t)

	if err := runHooksInstall(nil, nil); err != nil {
		t.Fatalf("runHooksInstall: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, ".claude", "hooks", "grove-worktree-remove.sh"))
	if err != nil {
		t.Fatalf("read grove-worktree-remove.sh: %v", err)
	}
	s := string(content)

	checks := []string{"#!/bin/bash", "grove stop", "grove detach", "/.claude/worktrees/"}
	for _, want := range checks {
		if !strings.Contains(s, want) {
			t.Errorf("grove-worktree-remove.sh missing %q", want)
		}
	}
}

// --- uninstall tests ---

func TestHooksUninstall_RemovesWorktreeScripts(t *testing.T) {
	dir := newHooksTestDir(t)

	if err := runHooksInstall(nil, nil); err != nil {
		t.Fatalf("runHooksInstall: %v", err)
	}
	if err := runHooksUninstall(nil, nil); err != nil {
		t.Fatalf("runHooksUninstall: %v", err)
	}

	for _, name := range []string{"grove-worktree-create.sh", "grove-worktree-remove.sh"} {
		path := filepath.Join(dir, ".claude", "hooks", name)
		if _, err := os.Stat(path); err == nil {
			t.Errorf("expected %s to be removed after uninstall", name)
		}
	}
}

func TestHooksUninstall_RemovesWorktreeEventsFromSettings(t *testing.T) {
	dir := newHooksTestDir(t)

	if err := runHooksInstall(nil, nil); err != nil {
		t.Fatalf("runHooksInstall: %v", err)
	}
	if err := runHooksUninstall(nil, nil); err != nil {
		t.Fatalf("runHooksUninstall: %v", err)
	}

	settings := readSettings(t, dir)
	for _, event := range []string{"WorktreeCreate", "WorktreeRemove"} {
		if n := countCommandsForEvent(t, settings, event); n != 0 {
			t.Errorf("expected WorktreeCreate/WorktreeRemove removed after uninstall, but %s still has %d commands", event, n)
		}
	}
}

func TestHooksUninstall_ClearsAllGroveHooks(t *testing.T) {
	dir := newHooksTestDir(t)

	if err := runHooksInstall(nil, nil); err != nil {
		t.Fatalf("runHooksInstall: %v", err)
	}
	if err := runHooksUninstall(nil, nil); err != nil {
		t.Fatalf("runHooksUninstall: %v", err)
	}

	settings := readSettings(t, dir)
	// After uninstall all grove hooks are gone, so the hooks map should be absent or empty.
	if hooks, ok := settings["hooks"].(map[string]interface{}); ok && len(hooks) > 0 {
		t.Errorf("expected empty hooks after uninstall, got: %v", hooks)
	}
}

// --- helper tests ---

func TestContainsString(t *testing.T) {
	cases := []struct {
		s, sub string
		want   bool
	}{
		{"grove-worktree-create.sh", "grove-worktree-create.sh", true},
		{"$CLAUDE_PROJECT_DIR/.claude/hooks/grove-worktree-create.sh", "grove-worktree-create.sh", true},
		{"grove-", "grove-worktree-create.sh", false},
		{"", "grove-", false},
		{"grove-dev-server.sh", "grove-", true},
		{"abc", "", true},
	}
	for _, tc := range cases {
		if got := containsString(tc.s, tc.sub); got != tc.want {
			t.Errorf("containsString(%q, %q) = %v, want %v", tc.s, tc.sub, got, tc.want)
		}
	}
}
