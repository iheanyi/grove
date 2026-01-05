# Grove - Development Guide

Grove is a development server manager for git worktrees with two main components:
1. **CLI** (`cli/`) - Go CLI with TUI for managing dev servers
2. **Menubar** (`menubar/`) - Native macOS menubar app (Swift/SwiftUI)

## Project Structure

```
grove/
├── cli/                    # Go CLI application
│   ├── cmd/grove/         # Main entry point
│   ├── internal/          # Internal packages
│   │   ├── cli/          # CLI commands (cobra)
│   │   ├── tui/          # Terminal UI (bubbletea)
│   │   ├── worktree/     # Git worktree detection
│   │   ├── port/         # Port allocation
│   │   ├── registry/     # Server registry (JSON)
│   │   ├── process/      # Process management
│   │   └── config/       # Configuration
│   └── go.mod
├── menubar/               # macOS menubar app
│   └── GroveMenubar/
│       ├── Sources/GroveMenubar/
│       │   ├── App/      # App entry point
│       │   ├── Models/   # Data models
│       │   ├── Views/    # SwiftUI views
│       │   └── Services/ # Business logic
│       └── Package.swift
└── docs/
```

## CLI (Go)

### Build & Run

```bash
cd cli
go build -o grove ./cmd/grove
./grove          # Launch TUI
./grove ls       # List servers
./grove start    # Start server in current worktree
```

### Key Commands

- `grove` or `grove ui` - Launch interactive TUI
- `grove start [command]` - Start dev server (must run from worktree directory)
- `grove stop <name>` - Stop a server by name
- `grove ls --json` - List servers (JSON for menubar integration)
- `grove discover --register` - Discover and register worktrees

### Architecture

- **Cobra** for CLI commands
- **Bubbletea** for TUI
- **Lipgloss** for TUI styling
- Registry stored at `~/.config/grove/registry.json`
- Per-project config: `.grove.yaml`

### Testing

```bash
cd cli
go test ./...
```

### Adding Commands

1. Create file in `internal/cli/` (e.g., `newcmd.go`)
2. Define cobra command
3. Register in `internal/cli/root.go`

## Menubar App (Swift)

### Requirements

- macOS 14.0+ (Sonoma)
- Xcode 15+ or Swift 5.9+

### Build & Run

```bash
cd menubar/GroveMenubar
swift build
.build/arm64-apple-macosx/debug/GroveMenubar
```

### Architecture

- **SwiftUI** with MenuBarExtra (`.window` style)
- Communicates with CLI via `grove ls --json`
- Settings use native SwiftUI Settings scene

### Key Files

- `App/GroveMenubarApp.swift` - App entry, scenes (MenuBarExtra, Window, Settings)
- `Views/MenuView.swift` - Main menubar dropdown UI
- `Views/SettingsView.swift` - Settings tabs
- `Views/LogViewerWindow.swift` - Pop-out log viewer
- `Services/ServerManager.swift` - Server state, CLI communication
- `Services/PreferencesManager.swift` - User preferences, terminal config
- `Services/NotificationService.swift` - macOS notifications

### SwiftUI Patterns Used

```swift
// Opening settings (macOS 14+)
@Environment(\.openSettings) private var openSettings
Button { openSettings() } label: { ... }

// Opening windows
@Environment(\.openWindow) private var openWindow
Button { openWindow(id: "log-viewer") } label: { ... }
```

### CLI Communication

The menubar app finds and executes the `grove` binary:

```swift
// ServerManager.swift
private func runGrove(_ args: [String], completion: ...)
private func runGroveInDirectory(_ dir: String, args: [...])  // For `grove start`
```

Important: `grove start` must run from within the worktree directory, not with a name argument.

### Testing Changes

1. Build: `swift build`
2. Kill existing: `pkill -x GroveMenubar`
3. Run: `.build/arm64-apple-macosx/debug/GroveMenubar &`

## Common Tasks

### Adding a new setting

1. Add property to `PreferencesManager.swift` with UserDefaults key
2. Add UI control in `SettingsView.swift`
3. Use setting where needed via `PreferencesManager.shared`

### Adding a new server action

1. Add method to `ServerManager.swift`
2. Call from UI in `MenuView.swift` or `ServerRowView`
3. For actions needing worktree directory, use `runGroveInDirectory`

### Adding a new CLI command

1. Create command file in `cli/internal/cli/`
2. Register in root command
3. If menubar needs it, update `ServerManager.swift`

## Configuration Files

### Global: `~/.config/grove/config.yaml`
```yaml
port_min: 3000
port_max: 3999
tld: localhost
```

### Per-project: `.grove.yaml`
```yaml
name: myapp
command: bin/dev
port: 3000
env:
  RAILS_ENV: development
```

## Debugging

### CLI
```bash
GROVE_DEBUG=1 grove ls
```

### Menubar
- Check Console.app for logs
- Print statements appear in terminal when running debug build
- Use `NotificationService` logs: "NotificationService: ..."

## Code Style

### Go
- Standard Go formatting (`gofmt`)
- Error handling: return errors, don't panic
- Use `internal/` for non-exported packages

### Swift
- SwiftUI declarative patterns
- Use `@StateObject` for owned state, `@EnvironmentObject` for shared
- Prefer `async/await` over completion handlers for new code
- Use `DispatchQueue.main.async` for UI updates from background threads

## Commit Messages

Follow conventional commits:
```
feat: Add new feature
fix: Fix bug
refactor: Code refactoring
docs: Documentation updates
```

