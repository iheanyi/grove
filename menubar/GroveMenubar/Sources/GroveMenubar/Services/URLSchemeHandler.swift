import Foundation
import AppKit
import SwiftUI

/// Handles `grove://` URL scheme for deep linking into the app.
///
/// Supported URLs:
/// - `grove://open/<server-name>` - Open server in browser
/// - `grove://start/<server-name>` - Start a server
/// - `grove://stop/<server-name>` - Stop a server
/// - `grove://logs/<server-name>` - Open log viewer for server
/// - `grove://refresh` - Trigger a server list refresh
///
/// Note: The URL scheme must be registered in Info.plist with CFBundleURLTypes
/// for the system to route URLs to this app. See the README or dist/Info.plist.
class URLSchemeHandler {
    static let shared = URLSchemeHandler()

    private init() {}

    /// Handle an incoming grove:// URL.
    func handle(_ url: URL) {
        guard url.scheme == "grove" else { return }

        // The host is the command (open, start, stop, logs, refresh)
        guard let command = url.host else { return }

        // The path contains the server name (after the leading /)
        let serverName = url.path.trimmingCharacters(in: CharacterSet(charactersIn: "/"))

        print("[Grove] URLScheme: command=\(command), serverName=\(serverName)")

        switch command {
        case "open":
            guard !serverName.isEmpty else { return }
            openServer(named: serverName)

        case "start":
            guard !serverName.isEmpty else { return }
            startServer(named: serverName)

        case "stop":
            guard !serverName.isEmpty else { return }
            stopServer(named: serverName)

        case "logs":
            guard !serverName.isEmpty else { return }
            openLogs(for: serverName)

        case "refresh":
            refreshServers()

        default:
            print("[Grove] URLScheme: Unknown command '\(command)'")
        }
    }

    // MARK: - Actions

    private func openServer(named name: String) {
        guard let manager = ServerManagerAccessor.shared,
              let server = findServer(named: name, in: manager) else {
            print("[Grove] URLScheme: Server '\(name)' not found")
            return
        }
        manager.openServer(server)
    }

    private func startServer(named name: String) {
        guard let manager = ServerManagerAccessor.shared,
              let server = findServer(named: name, in: manager) else {
            print("[Grove] URLScheme: Server '\(name)' not found")
            return
        }
        manager.startServer(server)
    }

    private func stopServer(named name: String) {
        guard let manager = ServerManagerAccessor.shared,
              let server = findServer(named: name, in: manager) else {
            print("[Grove] URLScheme: Server '\(name)' not found")
            return
        }
        manager.stopServer(server)
    }

    private func openLogs(for name: String) {
        NSApp.activate(ignoringOtherApps: true)

        if let manager = ServerManagerAccessor.shared,
           let server = findServer(named: name, in: manager) {
            manager.startStreamingLogs(for: server)
        }

        DispatchQueue.main.async {
            NotificationCenter.default.post(
                name: NSNotification.Name("OpenLogViewer"),
                object: nil,
                userInfo: ["serverName": name]
            )
        }
    }

    private func refreshServers() {
        ServerManagerAccessor.shared?.refresh()
    }

    // MARK: - Helpers

    private func findServer(named name: String, in manager: ServerManager) -> Server? {
        manager.servers.first { $0.name.lowercased() == name.lowercased() }
    }
}
