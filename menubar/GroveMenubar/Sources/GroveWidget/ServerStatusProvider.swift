import WidgetKit
import SwiftUI

/// Timeline entry holding the server snapshot at a point in time.
struct ServerEntry: TimelineEntry {
    let date: Date
    let servers: [WidgetServer]

    var runningCount: Int {
        servers.filter(\.isRunning).count
    }

    var totalCount: Int {
        servers.count
    }

    var allHealthy: Bool {
        servers.filter(\.isRunning).allSatisfy { $0.health == .healthy || $0.health == .unknown }
    }

    var statusSummary: String {
        if totalCount == 0 { return "No servers" }
        return "\(runningCount)/\(totalCount) running"
    }

    var healthSummary: String {
        let unhealthy = servers.filter { $0.isRunning && $0.health == .unhealthy }
        if unhealthy.isEmpty { return "All healthy" }
        return "\(unhealthy.count) unhealthy"
    }

    static let placeholder = ServerEntry(
        date: .now,
        servers: [
            WidgetServer(name: "api-server", branch: "main", path: "", port: 3000,
                         status: .running, health: .healthy, startedAt: Date().addingTimeInterval(-3600), url: nil),
            WidgetServer(name: "web-frontend", branch: "feature-auth", path: "", port: 3001,
                         status: .running, health: .healthy, startedAt: Date().addingTimeInterval(-5400), url: nil),
            WidgetServer(name: "background", branch: "main", path: "", port: 3002,
                         status: .stopped, health: .unknown, startedAt: nil, url: nil),
        ]
    )
}

/// Provides timeline snapshots for the widget by reading registry.json.
struct ServerStatusProvider: TimelineProvider {
    typealias Entry = ServerEntry

    func placeholder(in context: Context) -> ServerEntry {
        .placeholder
    }

    func getSnapshot(in context: Context, completion: @escaping (ServerEntry) -> Void) {
        if context.isPreview {
            completion(.placeholder)
            return
        }
        completion(currentEntry())
    }

    func getTimeline(in context: Context, completion: @escaping (Timeline<ServerEntry>) -> Void) {
        let entry = currentEntry()
        let nextUpdate = Calendar.current.date(byAdding: .minute, value: 5, to: .now)!
        completion(Timeline(entries: [entry], policy: .after(nextUpdate)))
    }

    private func currentEntry() -> ServerEntry {
        ServerEntry(date: .now, servers: RegistryReader.readServers())
    }
}
