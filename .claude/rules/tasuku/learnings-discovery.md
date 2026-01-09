---
paths: cli/internal/discovery/**/*.go
---

# Tasuku Learnings

_Auto-synced from .tasuku/context/learnings.md_

## Insights

- When using grep in shell commands, avoid complex patterns with `$` (end anchor) when piping to awk - it doesn't work as expected. Keep grep patterns simple like `grep '[c]laude'` (the bracket trick prevents self-matching). The working directory check via lsof is the real filter anyway.

