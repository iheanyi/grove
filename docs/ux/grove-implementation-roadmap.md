# Grove UX Implementation Roadmap

Status: Proposed  
Sequencing principle: quick wins first, then structural changes, then polish.

## Phase 1: Quick Wins (Low Risk, High Trust)

Objective: remove incorrect guidance and obvious UX drift immediately.

### Work items

1. Fix TUI messaging for start/restart semantics
- Update misleading messages in `internal/tui/app_enhanced.go`.
- Use actionable guidance that matches CLI behavior.

2. Sync README shortcut documentation with actual keymap
- Align `README.md` TUI keyboard table with `EnhancedKeyMap`.

3. Normalize CLI/TUI recovery copy
- Add deterministic next-step guidance for common errors (`not running`, `not registered`, bad context).

4. Ensure `menubarScope` has visible effect
- Wire preference to filtering path in MenuBar list rendering.

### Verification checklist
- [ ] No user-facing guidance suggests unsupported command signatures.
- [ ] README keybindings match TUI behavior exactly.
- [ ] MenuBar scope option changes visible list behavior.
- [ ] At least 5 common error messages include a concrete next action.

## Phase 2: Structural Improvements (Medium Risk)

Objective: make MenuBar maintainable and interaction-consistent.

### Work items

1. Decompose MenuView
- Split into shell/header/list/row/message components.
- Move keyboard handling to dedicated coordinator.

2. Unify action feedback state
- Replace split global string + local queue with one typed feedback channel.
- Introduce row-level pending state for long-running actions.

3. Clarify action hierarchy
- Single primary row action.
- Overflow menu for secondary operations.
- Reduce duplicated commands across toolbar/context/row where possible.

4. Align grouping and terminology
- Standardize labels toward worktree/workspace semantics while preserving backward compatibility.

### Verification checklist
- [ ] `MenuView.swift` reduced significantly and responsibilities separated.
- [ ] No duplicate error queues in UI state.
- [ ] Action placement follows one predictable pattern.
- [ ] Keyboard navigation behavior documented and deterministic.

## Phase 3: Polishing and System Quality (Higher Effort)

Objective: improve confidence, discoverability, and consistency at scale.

### Work items

1. Canonical interaction docs
- Add developer-facing UX contract doc for action/copy/feedback standards.

2. Shared string/help strategy
- Centralize user-visible action/help text for parity across surfaces.

3. Test coverage for UX contracts
- CLI unit tests for command guidance text patterns.
- TUI tests for keymap/help parity where feasible.
- MenuBar interaction tests for scope filtering and action states.

4. Optional enhancements
- Add first-run hints/tutorial callouts in MenuBar.
- Add explicit “what happened” activity panel for recent actions.

### Verification checklist
- [ ] Cross-surface terminology is consistent.
- [ ] Key user journeys pass acceptance tests.
- [ ] New contributors can identify canonical UX rules in one place.

## Risk Register

- **Behavior drift risk**: docs and keymaps diverge again.
  - Mitigation: enforce parity checks in CI or release checklist.

- **Refactor risk in MenuBar**: regressions from view decomposition.
  - Mitigation: split in small PRs with visual/manual validation after each.

- **Performance risk**: additional state layers in MenuBar.
  - Mitigation: keep expensive parsing/process calls off main thread, follow existing `PERFORMANCE.md` guidance.

## Suggested Ticket Backlog

- P1-01: Fix TUI start/restart instructional copy
- P1-02: Update README keyboard shortcuts to match shipped TUI
- P1-03: Make MenuBar scope preference functional in list filtering
- P1-04: Improve high-frequency CLI error copy with next-step actions
- P2-01: Extract Menu header/list/row/message components
- P2-02: Introduce unified action feedback store for MenuBar
- P2-03: Normalize row actions and shortcut discoverability
- P2-04: Standardize worktree/server terminology in MenuBar text
- P3-01: Create UX contract docs and contributor checklist
- P3-02: Add regression tests for key user guidance strings
