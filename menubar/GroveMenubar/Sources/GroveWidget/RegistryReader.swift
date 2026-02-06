import Foundation

/// Reads server data directly from ~/.config/grove/registry.json.
/// The widget cannot execute CLI commands, so it reads the registry file that
/// the Grove CLI writes to. This is the same file used by the main app.
enum RegistryReader {
    static var registryPath: String {
        NSHomeDirectory() + "/.config/grove/registry.json"
    }

    /// Read and parse all servers from the registry file.
    static func readServers() -> [WidgetServer] {
        guard let data = FileManager.default.contents(atPath: registryPath) else {
            return []
        }

        let decoder = JSONDecoder()

        guard let registry = try? decoder.decode(RegistryFile.self, from: data) else {
            return []
        }

        // Prefer the unified `workspaces` format; fall back to legacy `servers`
        if let workspaces = registry.workspaces, !workspaces.isEmpty {
            return workspaces.values.map { ws in
                serverFromWorkspace(ws)
            }.sorted { $0.name.localizedCaseInsensitiveCompare($1.name) == .orderedAscending }
        }

        if let servers = registry.servers, !servers.isEmpty {
            return servers.values.map { s in
                serverFromLegacy(s)
            }.sorted { $0.name.localizedCaseInsensitiveCompare($1.name) == .orderedAscending }
        }

        return []
    }

    // MARK: - Private

    private static func serverFromWorkspace(_ ws: WorkspaceJSON) -> WidgetServer {
        WidgetServer(
            name: ws.name,
            branch: ws.branch,
            path: ws.path,
            port: ws.server?.port,
            status: parseStatus(ws.server?.status),
            health: parseHealth(ws.server?.health),
            startedAt: parseDate(ws.server?.startedAt),
            url: ws.server?.url
        )
    }

    private static func serverFromLegacy(_ s: ServerJSON) -> WidgetServer {
        WidgetServer(
            name: s.name,
            branch: s.branch,
            path: s.path,
            port: s.port,
            status: parseStatus(s.status),
            health: parseHealth(s.health),
            startedAt: parseDate(s.startedAt),
            url: s.url
        )
    }

    private static func parseStatus(_ raw: String?) -> WidgetServer.ServerStatus {
        guard let raw = raw else { return .stopped }
        return WidgetServer.ServerStatus(rawValue: raw) ?? .stopped
    }

    private static func parseHealth(_ raw: String?) -> WidgetServer.HealthStatus {
        guard let raw = raw else { return .unknown }
        return WidgetServer.HealthStatus(rawValue: raw) ?? .unknown
    }

    private static let iso8601Formatter: ISO8601DateFormatter = {
        let f = ISO8601DateFormatter()
        f.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        return f
    }()

    private static let iso8601FallbackFormatter: ISO8601DateFormatter = {
        let f = ISO8601DateFormatter()
        f.formatOptions = [.withInternetDateTime]
        return f
    }()

    private static func parseDate(_ raw: String?) -> Date? {
        guard let raw = raw, !raw.isEmpty else { return nil }
        // Go's time.Time zero value serializes as "0001-01-01T00:00:00Z"
        if raw.hasPrefix("0001-") { return nil }
        return iso8601Formatter.date(from: raw)
            ?? iso8601FallbackFormatter.date(from: raw)
    }
}
