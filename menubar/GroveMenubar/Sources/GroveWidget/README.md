# Grove Widget Extension

macOS WidgetKit extension that displays server status on your desktop or in Notification Center.

## Widget Sizes

- **Small** — Compact grid of server status dots (up to 6 servers)
- **Medium** — Server list with name, port, and uptime (up to 5 servers)
- **Large** — Full dashboard with health summary, branch info, and details

## How It Works

The widget reads `~/.config/grove/registry.json` directly — the same file the Grove CLI writes to. No CLI execution is needed. The timeline refreshes every 5 minutes.

## Xcode Setup

WidgetKit extensions require a proper Xcode project with code signing and bundle embedding. SPM alone cannot produce a widget extension bundle.

### Steps

1. Open Xcode → File → New → Project → macOS App
2. Or add a Widget Extension target to an existing Xcode project:
   - File → New → Target → Widget Extension
   - Product Name: `GroveWidget`
   - Bundle Identifier: `com.grove.menubar.widget`
3. Delete the generated template files
4. Add the source files from this directory to the new target
5. In the widget target's Build Settings:
   - Set `INFOPLIST_KEY_NSWidgetExtensionPointIdentifier` to `com.apple.widgetkit-extension`
6. Embed the widget extension in the main app target:
   - Main app target → General → Frameworks, Libraries, and Embedded Content → Add GroveWidget.appex

### File Access

The widget reads `~/.config/grove/registry.json` from the user's home directory. Since this is a standard file path (not sandboxed), the widget needs:

- **App Sandbox** disabled, OR
- **File access** entitlement for the config directory

For development, disabling App Sandbox is simplest. For distribution, use an App Group to share data between the main app and the widget.

## Development

The widget source compiles as part of the SPM package for validation:

```bash
swift build --target GroveWidget
```

This verifies the code is syntactically correct but does not produce a usable widget bundle.
