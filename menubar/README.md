# WTMenubar

A native macOS menubar companion app for the `wt` CLI tool.

## Features

- **Status at a glance**: Menubar icon changes color based on server status
  - Green: Servers running
  - Gray: No servers running
  - Red: Server crashed
- **Quick actions**: Start/stop servers without opening terminal
- **Open in browser**: One-click to open server URL
- **Copy URL**: Copy server URL to clipboard
- **View logs**: Real-time log streaming for any server
- **Proxy management**: Start/stop the reverse proxy (subdomain mode)
- **Open TUI**: Launch the terminal TUI from menubar

## Requirements

- macOS 13.0 (Ventura) or later
- `wt` CLI installed

## Building

```bash
cd WTMenubar

# Build the app bundle
make build

# Build and run
make run

# Clean build artifacts
make clean
```

The app bundle will be created at `.build/WTMenubar.app`.

## Installation

After building:

```bash
# Run directly
open .build/WTMenubar.app

# Or copy to Applications
cp -r .build/WTMenubar.app /Applications/
```

## Auto-launch on Login

1. Open System Settings > General > Login Items
2. Click "+" under "Open at Login"
3. Navigate to and select WTMenubar.app

## Configuration

The menubar app looks for the `wt` binary in these locations (in order):

1. `~/development/go/bin/wt`
2. `/usr/local/bin/wt`
3. `/opt/homebrew/bin/wt`
4. `~/go/bin/wt`
5. `~/.local/bin/wt`

If not found in any of these locations, it falls back to `/usr/local/bin/wt`.

## How It Works

The app communicates with the `wt` CLI by running `wt ls --json` to get server status. It automatically refreshes every 5 seconds.

## Project Structure

```
WTMenubar/
├── Package.swift              # Swift Package Manager manifest
├── Makefile                   # Build commands
├── Sources/WTMenubar/
│   ├── App/
│   │   └── WTMenubarApp.swift # App entry point (MenuBarExtra)
│   ├── Models/
│   │   └── Server.swift       # Data models for JSON parsing
│   ├── Services/
│   │   └── ServerManager.swift # wt CLI communication
│   └── Views/
│       ├── MenuView.swift     # Main menu dropdown
│       └── LogsView.swift     # Log viewer
└── Tests/
    └── WTMenubarTests/
```

## URL Modes

The app supports both URL modes:

- **Port mode** (default): Shows `http://localhost:PORT` URLs
- **Subdomain mode**: Shows `https://name.localhost` URLs with proxy controls

The mode is determined by your `wt` configuration (`~/.config/wt/config.yaml`).
