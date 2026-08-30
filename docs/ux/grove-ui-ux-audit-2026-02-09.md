# Grove UI/UX Audit (MenuBar First)

Date: 2026-02-09  
Scope: MenuBar app, TUI, CLI  
Mode: Audit + redesign planning only (no implementation in this document)

## Executive View

Grove has a strong core idea (worktree-centric local development with lightweight controls), but the interaction model is currently split across three different mental models:

- CLI presents mostly **worktree-aware server management**.
- TUI presents **server actions**, with partial or incorrect start/restart guidance.
- MenuBar presents a feature-rich **server dashboard**, but with high UI/state complexity and hidden behavior.

The top UX issue is model drift: users cannot rely on one consistent answer to "what should I do next" across surfaces.

## Cross-Surface Workflow Matrix

Legend:
- Good = clear, consistent, and complete
- Partial = works but confusing/inconsistent/discoverability gaps
- Poor = incorrect or misleading behavior

| Workflow | CLI | TUI | MenuBar | Notes |
|---|---|---|---|---|
| Discover/register worktrees | Partial (`discover` works, register/start coupling is confusing) | Poor (no discovery flow) | Partial (empty state prompts command) | `discover --register --start` depends on command availability and can silently skip starts |
| Start server | Good (`grove start` from worktree dir) | Poor (shows `grove start <name>` guidance) | Partial (works via run-in-directory, but no clear progress/state) | TUI guidance conflicts with actual CLI semantics |
| Stop server | Good (`grove stop [name]`, `--all`) | Good (can stop selected) | Good (single and bulk actions) | Different feedback quality across surfaces |
| Restart server | Good (`grove restart [name]`) | Poor (shows wrong restart instruction) | Partial (indirect; mostly stop+start patterns) | TUI/CLI mismatch increases operator error |
| Open server URL | Good (`grove open`) | Good (`b`) | Good (row action + Open All) | Shortcut expectations differ by surface |
| View logs | Good (`grove logs`) | Good (`l`, `L`) | Good (`⌘L`, dedicated window) | Strong capability, but UX patterns differ significantly |
| Error handling/recovery | Partial (errors often terse/no next step) | Partial (toast-style notices, but some misleading) | Partial (single `error` string + local queue) | Recovery instructions inconsistent and sometimes wrong |
| Scope/filtering | Good (`ls` supports tags/groups/full) | Partial (filter list only) | Partial (search and grouping, but scope preference not wired into data filtering) | Capability parity missing across surfaces |

## Information Architecture Findings

### Mental model mismatch
- Code and docs use `workspace`, `worktree`, and `server` interchangeably without one canonical definition.
- `ls` now returns discovered worktrees and server data, but MenuBar still behaves primarily as a server list.

### Action model mismatch
- Start semantics are context-dependent in CLI (`grove start` in worktree dir), but TUI copy suggests name-based start.
- Restart semantics exist in CLI (`grove restart`) but are not reflected in TUI guidance.

### Surface drift
- README shortcut table does not match TUI keymap.
- MenuBar shortcut behavior is partly visible via tooltips/footer, partly hidden in event monitor behavior.

## Severity-Ranked Findings

### Critical

1. Incorrect TUI start guidance causes invalid commands and user confusion  
   - Evidence: `internal/tui/app_enhanced.go` shows `Use 'grove start %s' in terminal` message.  
   - Impact: Directly teaches a command pattern that is incompatible with current CLI start semantics.

2. Incorrect TUI restart guidance ignores dedicated restart command  
   - Evidence: `internal/tui/app_enhanced.go` shows `Restart ... with 'grove start %s'`.  
   - Impact: Users take a slower/error-prone path; undermines trust in TUI guidance.

3. Docs vs shipped behavior mismatch on core keybindings  
   - Evidence: `README.md` lists `enter/space/o`; TUI keymap uses `s/x/b` in `internal/tui/app_enhanced.go`.  
   - Impact: Onboarding friction and immediate command failure for new users.

### High

4. MenuBar state complexity is concentrated in one large view  
   - Evidence: `menubar/.../Views/MenuView.swift` carries mixed concerns (search, nav, toast, queueing, keyboard monitor, list rendering, row actions).  
   - Impact: Feature velocity drops, regressions likely, interaction consistency hard to maintain.

5. MenuBar action feedback is uneven for async operations  
   - Evidence: many actions trigger `refresh()` after completion in `ServerManager` with limited in-row loading state.  
   - Impact: Users cannot reliably tell if action is pending, succeeded, or failed.

6. Error model is split across global and local channels  
   - Evidence: `ServerManager.error` (single string) and `MenuView.errorMessages` queue with manual dequeue logic.  
   - Impact: Duplicate/hidden errors and unclear recovery path.

7. Menubar scope preference appears underutilized in runtime filtering  
   - Evidence: `PreferencesManager.menubarScope` + Settings picker exist, but no corresponding filtering hook in `MenuView`/`ServerManager` paths inspected.  
   - Impact: user preference may not materially affect list contents.

### Medium

8. Discovery start flow can silently skip starts  
   - Evidence: `discover.go` starts only when `cmdToUse != ""`; when config exists but command not resolved at register time, start is skipped.  
   - Impact: `--register --start` feels unreliable.

9. CLI error copy often lacks deterministic next-step guidance  
   - Evidence: several command errors are accurate but not always actionable (`not running`, `no server registered`, etc.).  
   - Impact: slower recovery and more guesswork.

10. Keyboard affordances are discoverable only after inspection  
   - Evidence: TUI action panel/help is available, but primary paths are not aligned with README; MenuBar has custom key monitor behavior not obvious to users.  
   - Impact: high cognitive load for new users.

11. MenuBar contains potentially stale/unused UI components  
   - Evidence: references to additional views/popovers in codebase and limited wiring in main flow.  
   - Impact: maintenance cost and conceptual noise.

12. Command group IA in CLI mixes operational levels  
   - Evidence: root command grouping blends runtime controls with maintenance surfaces (`ui`, `menubar`, etc.).  
   - Impact: less predictable command discovery for occasional users.

## MenuBar-First Deep Audit Notes

### What is working
- Strong at-a-glance utility: running indicators, quick actions, and integrated log window.
- Empty state includes concrete bootstrap command.
- Keyboard support is richer than most menu utilities.

### What feels unsane today
- Too many interaction modes in one dropdown (search, grouped list, expand rows, context menus, quick keys, toasts, error queue).
- Action placement is fragmented (toolbar icons, row chips, context menu, number keys, enter behavior).
- State lifecycle is hard to reason about (`onAppear` monitor setup, queued error rendering, separate feedback channels).

### MenuBar redesign direction (audit conclusion)
- Keep MenuBar focused on **triage + quick actions**, not full management surface.
- Move advanced operations and dense controls to dedicated windows (Logs, future Workspaces, Settings).
- Adopt explicit action states (idle/running/pending/error) at row level and global level.

## CLI/TUI Audit Notes

### What is working
- CLI command breadth is strong, especially `ls`, `logs`, `restart`, and grouping/filtering capabilities.
- TUI provides fast inspection and log access.

### Main issues
- TUI helper copy drifted from command semantics.
- Canonical keybindings are not synchronized with README.
- Recovery messaging should always include a specific next command with correct context requirements.

## Recommended UX Standards (Cross-Surface)

1. One canonical vocabulary:
   - `Worktree` = git worktree path/branch identity
   - `Server` = runtime process attached to a worktree
   - `Workspace` (optional future) = combined view object for UI

2. Action contract consistency:
   - `start` is context-bound unless explicitly name-addressable command exists.
   - `restart` should be the canonical named action.

3. Feedback contract:
   - every async action emits `pending -> success|error` visibly in the initiating surface.

4. Error copy contract:
   - state what failed, why (if known), and one exact next command/path.

## Audit Acceptance Check

- Major workflows covered across CLI/TUI/MenuBar: yes.
- 10+ high-impact findings with severity and rationale: yes.
- MenuBar-first direction and cross-surface standards: yes.

## Evidence Sources

- `menubar/GroveMenubar/Sources/GroveMenubar/Views/MenuView.swift`
- `menubar/GroveMenubar/Sources/GroveMenubar/Services/ServerManager.swift`
- `menubar/GroveMenubar/Sources/GroveMenubar/Views/SettingsView.swift`
- `menubar/GroveMenubar/Sources/GroveMenubar/Services/PreferencesManager.swift`
- `internal/tui/app_enhanced.go`
- `internal/tui/components.go`
- `internal/cli/start.go`
- `internal/cli/restart.go`
- `internal/cli/stop.go`
- `internal/cli/discover.go`
- `internal/cli/ls.go`
- `internal/cli/root.go`
- `README.md`
