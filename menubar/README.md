# GroveMenubar

A native macOS menubar companion app for the `grove` CLI tool.

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
- `grove` CLI installed

## Building

```bash
cd GroveMenubar

# Build the app bundle
make build

# Build and run
make run

# Clean build artifacts
make clean
```

The app bundle will be created at `.build/GroveMenubar.app`.

## Installation

After building:

```bash
# Run directly
open .build/GroveMenubar.app

# Or copy to Applications
cp -r .build/GroveMenubar.app /Applications/
```

## Auto-launch on Login

1. Open System Settings > General > Login Items
2. Click "+" under "Open at Login"
3. Navigate to and select GroveMenubar.app

## Configuration

The menubar app looks for the `grove` binary in these locations (in order):

1. `~/development/go/bin/grove`
2. `/usr/local/bin/grove`
3. `/opt/homebrew/bin/grove`
4. `~/go/bin/grove`
5. `~/.local/bin/grove`

If not found in any of these locations, it falls back to `/usr/local/bin/grove`.

## How It Works

The app communicates with the `grove` CLI by running `grove ls --json` to get server status. It automatically refreshes every 5 seconds.

## Project Structure

```
GroveMenubar/
├── Package.swift              # Swift Package Manager manifest
├── Makefile                   # Build commands
├── Sources/GroveMenubar/
│   ├── App/
│   │   └── GroveMenubarApp.swift # App entry point (MenuBarExtra)
│   ├── Models/
│   │   └── Server.swift       # Data models for JSON parsing
│   ├── Services/
│   │   └── ServerManager.swift # grove CLI communication
│   └── Views/
│       ├── MenuView.swift     # Main menu dropdown
│       └── LogsView.swift     # Log viewer
└── Tests/
    └── GroveMenubarTests/
```

## URL Modes

The app supports both URL modes:

- **Port mode** (default): Shows `http://localhost:PORT` URLs
- **Subdomain mode**: Shows `https://name.localhost` URLs with proxy controls

The mode is determined by your `grove` configuration (`~/.config/grove/config.yaml`).
