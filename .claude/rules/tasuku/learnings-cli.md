---
paths: cli/**/*.go
---

# Tasuku Learnings

_Auto-synced from .tasuku/context/learnings.md_

## Rules

- Always use `ansi.Truncate(s, width, "...")` from `github.com/charmbracelet/x/ansi` instead of custom byte-based truncation. Custom truncation like `s[:n-3] + "..."` breaks on: (1) multi-byte UTF-8 characters (cuts emoji/CJK in half), (2) wide characters (CJK = 2 cells), (3) ANSI escape codes (corrupts styling). The ansi package is already an indirect dependency via lipgloss.

