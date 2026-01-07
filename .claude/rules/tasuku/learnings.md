# Tasuku Learnings

_Auto-synced from .tasuku/context/learnings.md_

## Rules

- Never use `_ = err` to silence linter errors - always handle errors properly by reporting them to the user. Even for non-fatal errors, show warnings so users understand what happened. Silent failures make debugging impossible.

