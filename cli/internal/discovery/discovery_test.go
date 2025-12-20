package discovery

import (
	"testing"
)

func TestSanitizeBranchName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"feature/auth", "feature-auth"},
		{"feature/user-management", "feature-user-management"},
		{"FEATURE/AUTH", "feature-auth"},
		{"fix_bug_123", "fix-bug-123"},
		{"release/v1.0.0", "release-v100"},
		{"main", "main"},
		{"feature/test space", "feature-test-space"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := sanitizeBranchName(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeBranchName(%q) = %q; want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseWorktreeList(t *testing.T) {
	// Test parsing git worktree list --porcelain output
	output := `worktree /Users/test/myproject
HEAD abc123def456
branch refs/heads/main

worktree /Users/test/myproject-feature
HEAD def456abc789
branch refs/heads/feature/auth

worktree /Users/test/myproject-bugfix
HEAD 789abc123def
branch refs/heads/bugfix/123
`

	worktrees, err := parseWorktreeList(output)
	if err != nil {
		t.Fatalf("parseWorktreeList() error = %v", err)
	}

	if len(worktrees) != 3 {
		t.Errorf("parseWorktreeList() returned %d worktrees; want 3", len(worktrees))
	}

	// Check first worktree (main repo)
	if worktrees[0].Path != "/Users/test/myproject" {
		t.Errorf("worktrees[0].Path = %q; want %q", worktrees[0].Path, "/Users/test/myproject")
	}
	if worktrees[0].Branch != "main" {
		t.Errorf("worktrees[0].Branch = %q; want %q", worktrees[0].Branch, "main")
	}
	if worktrees[0].Name != "main" {
		t.Errorf("worktrees[0].Name = %q; want %q", worktrees[0].Name, "main")
	}

	// Check second worktree
	if worktrees[1].Branch != "feature/auth" {
		t.Errorf("worktrees[1].Branch = %q; want %q", worktrees[1].Branch, "feature/auth")
	}
	if worktrees[1].Name != "feature-auth" {
		t.Errorf("worktrees[1].Name = %q; want %q", worktrees[1].Name, "feature-auth")
	}

	// Check third worktree
	if worktrees[2].Branch != "bugfix/123" {
		t.Errorf("worktrees[2].Branch = %q; want %q", worktrees[2].Branch, "bugfix/123")
	}
	if worktrees[2].Name != "bugfix-123" {
		t.Errorf("worktrees[2].Name = %q; want %q", worktrees[2].Name, "bugfix-123")
	}

	// All worktrees should have the same MainRepo
	for i, wt := range worktrees {
		if wt.MainRepo != "/Users/test/myproject" {
			t.Errorf("worktrees[%d].MainRepo = %q; want %q", i, wt.MainRepo, "/Users/test/myproject")
		}
	}
}

func TestDetachedHead(t *testing.T) {
	output := `worktree /Users/test/myproject
HEAD abc123def456
detached
`

	worktrees, err := parseWorktreeList(output)
	if err != nil {
		t.Fatalf("parseWorktreeList() error = %v", err)
	}

	if len(worktrees) != 1 {
		t.Errorf("parseWorktreeList() returned %d worktrees; want 1", len(worktrees))
	}

	if worktrees[0].Branch != "HEAD" {
		t.Errorf("worktrees[0].Branch = %q; want %q", worktrees[0].Branch, "HEAD")
	}
	if worktrees[0].Name != "detached-head" {
		t.Errorf("worktrees[0].Name = %q; want %q", worktrees[0].Name, "detached-head")
	}
}
