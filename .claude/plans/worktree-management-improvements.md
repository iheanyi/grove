# Grove Worktree Management Improvements

## Overview

This spec outlines improvements to Grove's worktree management system, focusing on better UX for switching, discovery, cleanup, and multi-agent workflows.

## Interview Summary

**Date**: 2026-01-03
**Interviewer**: Claude
**Stakeholder**: Iheanyi

---

## 1. Switch Command UX

### Decision: Replace Shell (cd-style) + Future Daemon Option

**Current behavior**: `grove switch` opens a new terminal tab and prints a reminder to run `grove start`.

**New behavior**:
- Primary: Shell function approach for in-place directory change
- Add `grove cd <name>` that outputs the path (for shell integration)
- Shell function: `grovecd() { cd "$(grove cd "$@")" && grove start 2>/dev/null }`
- Future: Optional daemon that watches for directory changes and auto-starts servers

**Implementation**:
```bash
# In ~/.bashrc or ~/.zshrc
grovecd() {
  local path
  path=$(grove cd "$@") || return 1
  cd "$path" || return 1
  # Auto-start if .grove.yaml exists and server not running
  if [ -f .grove.yaml ] && ! grove status --quiet 2>/dev/null; then
    grove start
  fi
}
```

**CLI changes**:
- Add `grove cd <name>` - outputs worktree path (for shell integration)
- Keep `grove switch <name>` - opens new terminal tab (current behavior)
- Add `grove switch --cd` - just outputs path without opening terminal

---

## 2. Naming Strategy

### Decision: User Specifies with Full Path Default

**Current**: `<repo>-<branch>` (e.g., `myapp-feature-auth-oauth-provider`)

**New behavior**:
- Default: Full sanitized path (current behavior)
- Add `--name` flag to `grove new`: `grove new feature/oauth --name oauth`
- If collision detected, prompt user for custom name

**Implementation**:
```go
// In new.go
newCmd.Flags().StringP("name", "n", "", "Custom name for the worktree (default: sanitized branch name)")

// If name collision:
if exists && customName == "" {
    fmt.Printf("Name '%s' already exists. Specify a custom name with --name\n", name)
    return
}
```

---

## 3. Data Model Unification

### Decision: Workspace with Optional Server

**Current**: Separate `Worktree` and `Server` registry entries that can get out of sync.

**New model**:
```go
type Workspace struct {
    // Identity
    Name      string    `json:"name"`
    Path      string    `json:"path"`
    Branch    string    `json:"branch"`
    MainRepo  string    `json:"main_repo,omitempty"`

    // Git state
    GitDirty  bool      `json:"git_dirty"`

    // Activity detection
    HasClaude bool      `json:"has_claude"`
    HasVSCode bool      `json:"has_vscode"`
    LastActivity time.Time `json:"last_activity"`

    // Server (optional - nil if no server running/registered)
    Server    *ServerState `json:"server,omitempty"`

    // Metadata
    CreatedAt    time.Time `json:"created_at"`
    DiscoveredAt time.Time `json:"discovered_at,omitempty"`
    Tags         []string  `json:"tags,omitempty"`
}

type ServerState struct {
    Port      int       `json:"port"`
    PID       int       `json:"pid,omitempty"`
    Status    string    `json:"status"`
    URL       string    `json:"url"`
    Command   []string  `json:"command,omitempty"`
    LogFile   string    `json:"log_file,omitempty"`
    StartedAt time.Time `json:"started_at,omitempty"`
    Health    string    `json:"health,omitempty"`
}
```

**Migration**:
- Merge existing Server and Worktree entries by name
- Prefer Server data where both exist
- One-time migration on registry load

---

## 4. Deletion and Cleanup

### Decision: Safe Delete with Prompt + Prune Offers Deletion

**New command: `grove delete <name>`**
```
grove delete feature-auth

Worktree: feature-auth
Path: ~/projects/myapp-feature-auth
Branch: feature/auth
Status: No uncommitted changes

This will:
  - Remove git worktree (git worktree remove)
  - Delete directory
  - Remove from registry
  - Delete log files

Continue? [y/N]:
```

**Safety checks**:
- Warn if uncommitted changes: "Warning: 3 uncommitted changes. Delete anyway? [y/N]"
- Warn if server running: "Warning: Server is running. Stop and delete? [y/N]"
- Add `--force` to skip prompts

**Prune enhancement**:
```
grove prune

Stale registry entries (paths no longer exist):
  - myapp-old-feature (was at ~/projects/myapp-old-feature)

Stale worktrees (inactive >30 days):
  - myapp-experiment (last active: 45 days ago)
    Path: ~/projects/myapp-experiment
    Status: clean (no uncommitted changes)

Remove stale registry entries? [Y/n]:
Delete stale worktree directories? [y/N]:
```

---

## 5. Discovery and Filtering

### Decision: Fuzzy Picker + Tags + Smart Grouping

**Fuzzy picker**:
- Add `grove select --worktrees` (or just enhance existing `grove select`)
- fzf-style interactive search across all worktrees
- Show: name, branch, status (server running?), last activity

**Tag system**:
```bash
grove tag myapp-auth urgent frontend     # Add tags
grove tag myapp-auth --remove urgent     # Remove tag
grove ls --tag urgent                     # Filter by tag
grove ls --tag frontend --tag backend    # Multiple tags (OR)
```

**Smart grouping** (enhance `grove ls`):
```bash
grove ls                      # Default: group by main repo
grove ls --group activity     # Group by: active, recent, stale
grove ls --group status       # Group by: running, stopped, dirty
grove ls --flat               # No grouping, just list
```

---

## 6. Menubar Scope

### Decision: Configurable

**New preference**: "Show in menubar"
- Servers only (current behavior)
- Active worktrees (has server OR recent activity)
- All worktrees

**Implementation**: Add to PreferencesManager.swift
```swift
enum MenubarScope: String {
    case serversOnly = "servers_only"
    case activeWorktrees = "active_worktrees"
    case allWorktrees = "all_worktrees"
}

@AppStorage("menubarScope") var menubarScope: MenubarScope = .serversOnly
```

---

## 7. Remote Branch Handling

### Decision: Interactive Prompt

**When `grove new feature-x` and `origin/feature-x` exists**:
```
Branch 'feature-x' exists on remote (origin/feature-x).
Track existing remote branch? [Y/n]:

(Y) → git worktree add -b feature-x --track origin/feature-x
(n) → git worktree add -b feature-x (create new local branch)
```

**Implementation**:
```go
// Check if remote branch exists
remoteBranch := fmt.Sprintf("origin/%s", branchName)
if branchExists(remoteBranch) {
    fmt.Printf("Branch '%s' exists on remote (%s).\n", branchName, remoteBranch)
    if promptYesNo("Track existing remote branch?", true) {
        // Add --track flag
    }
}
```

---

## 8. GitHub Integration

### Decision: Yes, Optional (via --full flag)

**Enhance `grove ls --full`** to show for all worktrees (not just servers):
- PR status (open, merged, draft)
- CI status (passing, failing, pending)
- Review status (approved, changes requested, pending)

**No new flags needed** - `--full` already exists, just extend to worktrees.

---

## 9. Name Collision Handling

### Decision: Prompt User

**When centralized worktrees_dir has collision**:
```
grove new feature-auth

Directory conflict: ~/worktrees/api/feature-auth already exists
(from a different 'api' project)

Options:
  1. Use different name: grove new feature-auth --name myorg-api-auth
  2. Use different directory: grove new feature-auth --dir ~/worktrees/myorg-api
  3. Cancel

Choice [1/2/3]:
```

---

## 10. Templates

### Decision: Simple Inheritance (Deferred)

**Current approach is sufficient**:
- Worktrees inherit `.grove.yaml` from the git repo
- Different branches can have different `.grove.yaml` configs
- No new template system needed initially

**Future consideration**: Pattern-based templates in `~/.config/grove/templates/`

---

## 11. Windows Support

### Decision: Not Needed

macOS/Linux only. Windows users can use WSL.

---

## 12. Multi-Agent Dashboard (Major Feature)

### Pain Point
Running multiple AI agents across worktrees with no centralized view of:
- What each agent is working on
- How to test/preview their changes
- Which are ready for review

### Decision: Comprehensive Dashboard (TUI + Menubar + Web)

#### 12.1 Agent Tracking Data

**Extend Workspace model**:
```go
type Workspace struct {
    // ... existing fields ...

    // Agent tracking
    AgentInfo *AgentInfo `json:"agent_info,omitempty"`
}

type AgentInfo struct {
    // What they're working on
    ActiveIssue  string    `json:"active_issue,omitempty"`  // beads issue ID
    TaskSummary  string    `json:"task_summary,omitempty"`  // from beads or commit msg

    // Git activity
    FilesChanged int       `json:"files_changed"`
    LinesAdded   int       `json:"lines_added"`
    LinesRemoved int       `json:"lines_removed"`
    LastCommit   string    `json:"last_commit,omitempty"`

    // Status
    Status       string    `json:"status"`  // working, idle, ready_for_review
    LastActivity time.Time `json:"last_activity"`
}
```

**Data sources**:
- Git: `git diff --stat`, `git log -1`
- Beads: Parse `.beads/issues/` for in_progress issues
- Claude Code: Check for running claude process, parse recent activity

#### 12.2 TUI Dashboard (`grove agents`)

```
┌─ Agent Workspaces ──────────────────────────────────────────────┐
│                                                                  │
│ myapp-feature-auth          ● WORKING                           │
│   Task: Implement OAuth login (beads-abc)                       │
│   Changes: +142 -23 (5 files)                                   │
│   Server: http://localhost:3042 (running)                       │
│   Last activity: 2 min ago                                      │
│                                                                  │
│ myapp-feature-api           ✓ READY FOR REVIEW                  │
│   Task: Add rate limiting (beads-def)                           │
│   Changes: +89 -12 (3 files)                                    │
│   Server: http://localhost:3043 (running)                       │
│   Last activity: 15 min ago                                     │
│                                                                  │
│ myapp-bugfix-login          ○ IDLE                              │
│   No active task                                                │
│   Changes: none                                                 │
│   Server: stopped                                               │
│                                                                  │
├─────────────────────────────────────────────────────────────────┤
│ [o] Open in browser  [r] Review queue  [s] Start server  [q] Quit│
└─────────────────────────────────────────────────────────────────┘
```

**Keyboard shortcuts**:
- `o` - Open selected workspace's server in browser
- `r` - Show review queue (only READY FOR REVIEW workspaces)
- `s` - Start/stop server
- `d` - Show detailed diff
- `enter` - Switch to workspace (cd + focus)
- `tab` - Cycle through workspaces in browser (quick switch)

#### 12.3 Menubar Expansion

**New "Agents" section** in menubar (when enabled):
```
┌────────────────────────────────┐
│ Grove                      ●   │
├────────────────────────────────┤
│ ▼ Agents (3 active)            │
│   ● myapp-auth    Working      │
│   ✓ myapp-api     Ready        │
│   ○ myapp-bug     Idle         │
├────────────────────────────────┤
│ ▼ Servers                      │
│   ...                          │
└────────────────────────────────┘
```

**Right-click menu on agent**:
- Open in Browser
- View Diff
- Copy URL
- Mark as Reviewed
- Switch to Workspace

#### 12.4 Web Dashboard (`grove dashboard`)

**Start local web server**:
```bash
grove dashboard          # Opens http://localhost:3099
grove dashboard --port 8080
```

**Features**:
- Real-time status updates (WebSocket)
- Side-by-side browser frames for comparing branches
- Diff viewer with syntax highlighting
- Review queue with one-click open
- Activity timeline per workspace

#### 12.5 Review Queue

**Command**: `grove review`

```
Ready for Review:

1. myapp-feature-api
   Task: Add rate limiting
   Changes: +89 -12 (3 files)
   URL: http://localhost:3043

2. myapp-feature-ui
   Task: Update dashboard layout
   Changes: +234 -45 (8 files)
   URL: http://localhost:3044

Actions:
  [1-2] Open in browser
  [a]   Open all
  [d]   Show diff for selected
  [q]   Quit
```

#### 12.6 Quick Switch

**Browser cycling**: `grove cycle`
- Cycles through all running servers in default browser
- Each press opens next server URL
- Wraps around to first

**Implementation**: Store "current index" in temp file, increment on each call.

---

## Implementation Priority

### Phase 1: Foundation
1. Unify data model (Workspace with optional Server)
2. Add `grove cd` command for shell integration
3. Add `--name` flag to `grove new`
4. Add `grove delete` with safety checks

### Phase 2: Discovery & Cleanup
5. Enhance prune to offer directory deletion
6. Add tag system
7. Add smart grouping to `grove ls`
8. Add fuzzy picker for worktrees

### Phase 3: Remote & GitHub
9. Interactive prompt for remote branches
10. Extend `--full` to show GitHub info for worktrees

### Phase 4: Multi-Agent Dashboard
11. Add AgentInfo tracking
12. Implement TUI dashboard (`grove agents`)
13. Expand menubar with Agents section
14. Add review queue (`grove review`)
15. Implement web dashboard (`grove dashboard`)
16. Add quick switch cycling

### Phase 5: Polish
17. Configurable menubar scope
18. Enhanced name collision handling
19. Documentation updates

---

## Web Dashboard Tech Stack

### Decision: Go + Embedded SvelteKit

**Rationale:**
- Svelte bundle size is ~1.6KB gzipped vs React's ~42KB - critical for single binary
- Svelte excels at real-time dashboards with reactive data
- SvelteKit's `adapter-static` produces files perfect for `go:embed`
- No virtual DOM overhead - direct DOM updates for better performance
- Used by Apple Music, Spotify, NYT for performance-critical UIs

**Architecture:**
```
cli/
├── cmd/grove/
├── internal/
│   ├── dashboard/              # Go WebSocket + HTTP server
│   │   ├── server.go           # Serves embedded SvelteKit build
│   │   ├── websocket.go        # Real-time agent updates
│   │   ├── api.go              # REST endpoints for dashboard
│   │   └── embed.go            # //go:embed web/build/*
│   └── ...
└── web/                        # SvelteKit project
    ├── src/
    │   ├── routes/
    │   │   ├── +page.svelte         # Main dashboard
    │   │   ├── +layout.svelte       # Shared layout
    │   │   ├── agents/+page.svelte  # Agent monitoring
    │   │   └── review/+page.svelte  # Review queue
    │   ├── lib/
    │   │   ├── stores/              # Svelte stores for state
    │   │   ├── components/          # Reusable components
    │   │   └── websocket.ts         # WS client for real-time
    │   └── app.css                  # Tailwind CSS
    ├── svelte.config.js             # Uses adapter-static
    ├── tailwind.config.js
    └── package.json
```

**Key Components:**

1. **Go Backend** (`internal/dashboard/`)
   - `server.go`: HTTP server serving embedded static files
   - `websocket.go`: Broadcasts workspace/agent updates in real-time
   - `api.go`: REST endpoints (`/api/workspaces`, `/api/agents`, etc.)

2. **SvelteKit Frontend** (`web/`)
   - Uses `adapter-static` for static file generation
   - Tailwind CSS for styling (via CDN or embedded)
   - WebSocket store for real-time reactivity
   - Components: WorkspaceCard, AgentStatus, DiffViewer, ReviewQueue

3. **Build Process:**
   ```makefile
   .PHONY: build-web build

   build-web:
   	cd web && npm install && npm run build

   build: build-web
   	go build -o grove ./cmd/grove

   dev-web:
   	cd web && npm run dev
   ```

4. **Development Workflow:**
   - Run `npm run dev` in `web/` for frontend hot-reload
   - Run `go run ./cmd/grove dashboard --dev` to proxy to Vite dev server
   - Production: `go build` embeds the built static files

**WebSocket Protocol:**
```typescript
// Client subscribes to updates
ws.send(JSON.stringify({ type: 'subscribe', topics: ['workspaces', 'agents'] }))

// Server broadcasts changes
{
  type: 'workspace_updated',
  payload: {
    name: 'myapp-feature-auth',
    server: { status: 'running', url: '...' },
    agent: { status: 'working', task: '...' }
  }
}
```

---

## Open Questions

1. **Beads integration depth**: Should grove automatically update beads issues when changes are committed? Or keep it read-only?

2. **Multi-project support**: When running agents across different projects (not just worktrees of one repo), how should grouping work?

3. **Notification preferences**: Should there be notifications when an agent's worktree becomes "ready for review"?

4. **Persistence of review status**: Where to store "reviewed" / "needs changes" state? In registry? As git notes?

---

## Related Issues

- `claude-helper-p1z`: Update MCP and README with recent features
- `claude-helper-1mt`: Add grove hooks install command for Claude Code integration
