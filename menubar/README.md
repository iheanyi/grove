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
- **Proxy management**: Start/stop the reverse proxy
- **Open TUI**: Launch the terminal TUI from menubar

## Requirements

- macOS 13.0 (Ventura) or later
- `wt` CLI installed and in PATH

## Building

```bash
cd menubar/WTMenubar
swift build
```

The binary will be at `.build/debug/WTMenubar`.

## Building for Release

```bash
cd menubar/WTMenubar
swift build -c release
```

## Creating an App Bundle

To create a proper `.app` bundle:

```bash
# Build release
swift build -c release

# Create app structure
mkdir -p WTMenubar.app/Contents/MacOS
mkdir -p WTMenubar.app/Contents/Resources

# Copy binary
cp .build/release/WTMenubar WTMenubar.app/Contents/MacOS/

# Create Info.plist
cat > WTMenubar.app/Contents/Info.plist << 'EOF'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleExecutable</key>
    <string>WTMenubar</string>
    <key>CFBundleIdentifier</key>
    <string>com.iheanyi.wtmenubar</string>
    <key>CFBundleName</key>
    <string>WTMenubar</string>
    <key>CFBundlePackageType</key>
    <string>APPL</string>
    <key>CFBundleShortVersionString</key>
    <string>1.0</string>
    <key>CFBundleVersion</key>
    <string>1</string>
    <key>LSMinimumSystemVersion</key>
    <string>13.0</string>
    <key>LSUIElement</key>
    <true/>
</dict>
</plist>
EOF
```

## Running

```bash
# Run from command line
.build/debug/WTMenubar

# Or if you created an app bundle
open WTMenubar.app
```

## Auto-launch on Login

1. Open System Settings > General > Login Items
2. Click "+" under "Open at Login"
3. Navigate to and select WTMenubar.app

## Configuration

The menubar app looks for the `wt` binary in these locations:
1. `/usr/local/bin/wt`
2. `/opt/homebrew/bin/wt`
3. `~/go/bin/wt`
4. `~/.local/bin/wt`
5. Falls back to `which wt`

## Screenshots

The menubar shows:
- Server list grouped by running/stopped status
- Port numbers for each server
- Quick action buttons on hover
- Proxy status with start/stop button
- Open TUI button
