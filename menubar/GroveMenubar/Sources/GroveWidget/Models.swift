import Foundation

/// Lightweight server model for the widget, parsed directly from registry.json.
/// This is intentionally independent from the main app's Server model to keep
/// the widget extension self-contained.
struct WidgetServer: Identifiable {
    let name: String
    let branch: String?
    let path: String
    let port: Int?
    let status: ServerStatus
    let health: HealthStatus
    let startedAt: Date?
    let url: String?

    var id: String { name }

    var isRunning: Bool {
        status == .running || status == .starting
    }

    var displayPort: String {
        guard let port = port, port > 0 else { return "—" }
        return ":\(port)"
    }

    var uptimeString: String {
        guard let startedAt = startedAt, isRunning else { return "—" }
        let elapsed = Date().timeIntervalSince(startedAt)
        let hours = Int(elapsed) / 3600
        let minutes = (Int(elapsed) % 3600) / 60
        if hours > 0 {
            return String(format: "%02dh%02dm", hours, minutes)
        }
        return String(format: "%dm", minutes)
    }

    var statusSFSymbol: String {
        switch status {
        case .running: return "circle.fill"
        case .starting: return "circle.dotted"
        case .stopped: return "circle"
        case .crashed: return "xmark.circle.fill"
        case .stopping: return "circle.dashed"
        }
    }

    enum ServerStatus: String {
        case running
        case stopped
        case starting
        case stopping
        case crashed
    }

    enum HealthStatus: String {
        case healthy
        case unhealthy
        case unknown
    }
}

// MARK: - Registry JSON Parsing

/// Top-level registry.json structure.
/// The registry contains both `workspaces` (new format) and `servers` (legacy),
/// with workspaces taking priority.
struct RegistryFile: Decodable {
    let workspaces: [String: WorkspaceJSON]?
    let servers: [String: ServerJSON]?
}

struct WorkspaceJSON: Decodable {
    let name: String
    let path: String
    let branch: String?
    let server: ServerStateJSON?

    enum CodingKeys: String, CodingKey {
        case name, path, branch, server
    }
}

struct ServerStateJSON: Decodable {
    let port: Int?
    let pid: Int?
    let status: String?
    let url: String?
    let health: String?
    let startedAt: String?
    let stoppedAt: String?

    enum CodingKeys: String, CodingKey {
        case port, pid, status, url, health
        case startedAt = "started_at"
        case stoppedAt = "stopped_at"
    }
}

/// Legacy server format in registry.json
struct ServerJSON: Decodable {
    let name: String
    let port: Int?
    let pid: Int?
    let path: String
    let url: String?
    let status: String?
    let health: String?
    let branch: String?
    let startedAt: String?

    enum CodingKeys: String, CodingKey {
        case name, port, pid, path, url, status, health, branch
        case startedAt = "started_at"
    }
}
