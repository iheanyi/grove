import AppIntents
import Foundation

// MARK: - Start Server Intent

struct StartServerIntent: AppIntent {
    static var title: LocalizedStringResource = "Start Grove Server"
    static var description = IntentDescription("Start a development server managed by Grove")

    @Parameter(title: "Server Name")
    var serverName: String

    func perform() async throws -> some IntentResult & ProvidesDialog {
        guard let manager = await MainActor.run(body: { ServerManagerAccessor.shared }) else {
            return .result(dialog: "Grove is not running")
        }

        let server = await MainActor.run { manager.servers.first { $0.name.lowercased() == serverName.lowercased() } }
        guard let server = server else {
            return .result(dialog: "Server '\(serverName)' not found")
        }

        if server.isRunning {
            return .result(dialog: "Server '\(serverName)' is already running")
        }

        await MainActor.run {
            manager.startServer(server)
        }
        return .result(dialog: "Starting server '\(serverName)'...")
    }
}

// MARK: - Stop Server Intent

struct StopServerIntent: AppIntent {
    static var title: LocalizedStringResource = "Stop Grove Server"
    static var description = IntentDescription("Stop a running development server")

    @Parameter(title: "Server Name")
    var serverName: String

    func perform() async throws -> some IntentResult & ProvidesDialog {
        guard let manager = await MainActor.run(body: { ServerManagerAccessor.shared }) else {
            return .result(dialog: "Grove is not running")
        }

        let server = await MainActor.run { manager.servers.first { $0.name.lowercased() == serverName.lowercased() } }
        guard let server = server else {
            return .result(dialog: "Server '\(serverName)' not found")
        }

        if !server.isRunning {
            return .result(dialog: "Server '\(serverName)' is not running")
        }

        await MainActor.run {
            manager.stopServer(server)
        }
        return .result(dialog: "Stopping server '\(serverName)'...")
    }
}

// MARK: - List Servers Intent

struct ListServersIntent: AppIntent {
    static var title: LocalizedStringResource = "List Grove Servers"
    static var description = IntentDescription("List all development servers managed by Grove")

    func perform() async throws -> some IntentResult & ProvidesDialog {
        guard let manager = await MainActor.run(body: { ServerManagerAccessor.shared }) else {
            return .result(dialog: "Grove is not running")
        }

        let servers = await MainActor.run { manager.servers }

        if servers.isEmpty {
            return .result(dialog: "No servers registered with Grove")
        }

        let lines = servers.map { server -> String in
            let status = server.isRunning ? "running" : server.displayStatus
            return "\(server.name): \(status)"
        }

        let summary = lines.joined(separator: "\n")
        return .result(dialog: "\(summary)")
    }
}

// MARK: - Is Server Running Intent

struct IsServerRunningIntent: AppIntent {
    static var title: LocalizedStringResource = "Is Grove Server Running"
    static var description = IntentDescription("Check if a specific development server is currently running")

    @Parameter(title: "Server Name")
    var serverName: String

    func perform() async throws -> some IntentResult & ProvidesDialog {
        guard let manager = await MainActor.run(body: { ServerManagerAccessor.shared }) else {
            return .result(dialog: "Grove is not running")
        }

        let server = await MainActor.run { manager.servers.first { $0.name.lowercased() == serverName.lowercased() } }
        guard let server = server else {
            return .result(dialog: "Server '\(serverName)' not found")
        }

        let status = server.isRunning ? "running" : server.displayStatus
        return .result(dialog: "Server '\(serverName)' is \(status)")
    }
}

// MARK: - Refresh Intent

struct RefreshServersIntent: AppIntent {
    static var title: LocalizedStringResource = "Refresh Grove Servers"
    static var description = IntentDescription("Refresh the server list in Grove")

    func perform() async throws -> some IntentResult & ProvidesDialog {
        await MainActor.run {
            ServerManagerAccessor.shared?.refresh()
        }
        return .result(dialog: "Refreshing server list...")
    }
}

// MARK: - App Shortcuts Provider

struct GroveShortcuts: AppShortcutsProvider {
    static var appShortcuts: [AppShortcut] {
        AppShortcut(
            intent: ListServersIntent(),
            phrases: [
                "List servers in \(.applicationName)",
                "Show \(.applicationName) servers"
            ],
            shortTitle: "List Servers",
            systemImageName: "list.bullet"
        )
        AppShortcut(
            intent: RefreshServersIntent(),
            phrases: [
                "Refresh \(.applicationName)",
                "Refresh \(.applicationName) servers"
            ],
            shortTitle: "Refresh",
            systemImageName: "arrow.clockwise"
        )
    }
}
