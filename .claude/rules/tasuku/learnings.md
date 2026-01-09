# Tasuku Learnings

_Auto-synced from .tasuku/context/learnings.md_

## Rules

- Never use `_ = err` to silence linter errors - always handle errors properly by reporting them to the user. Even for non-fatal errors, show warnings so users understand what happened. Silent failures make debugging impossible.
- Always define colors in `internal/styles/styles.go` instead of hardcoding hex values like `lipgloss.Color("#A78BFA")` inline. This centralizes theming, prevents drift between files, and makes it easy to update colors globally. Import the styles package and reference colors as `styles.Accent`, `styles.Error`, etc.
- Always use standard bubbles components (spinner, progress, textinput, viewport) instead of custom implementations. The list component has built-in fuzzy filtering via `SetFilteringEnabled(true)` - pressing "/" activates it automatically. Don't implement custom search bars that duplicate this functionality.
- Always use custom ItemDelegate for bubbles/list when items need colored status indicators. Never embed ANSI escape codes directly in Title()/Description() methods - the list's default delegate applies its own styling which corrupts embedded ANSI codes and breaks filtering. Instead: (1) Return plain text from Title()/Description()/FilterValue(), (2) Add helper methods like StatusIcon(), StatusStyle() to items, (3) Create custom delegate that renders status styling during Render().
- Never call `list.SetItems()` while `list.FilterState() != list.Unfiltered` in bubbletea. The 2-second tickMsg refresh was calling SetItems() which disrupts the filter state and causes items to disappear during filtering. Always check FilterState before calling SetItems, SetItem, or any method that modifies the list items.
- In bubbletea, goroutines started in tea.Cmd functions cannot communicate back to the TUI directly. Use a subscription pattern instead: (1) tea.Cmd returns a message when an event occurs, (2) Update() handles the message and starts a new tea.Cmd to watch for the next event. Never start a long-running goroutine that tries to modify model state - this causes race conditions and broken UIs.

## Insights

- When using bubbles/list with filtering: (1) Don't use SetShowHelp(false) as it hides the filter input bar, (2) Check `FilterState() != list.Unfiltered` not just `== list.Filtering` to handle both active filtering AND applied filter states, (3) Don't intercept `esc` key - the list uses it to cancel filtering.

