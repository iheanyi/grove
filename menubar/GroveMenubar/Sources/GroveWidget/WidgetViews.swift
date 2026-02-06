import SwiftUI
import WidgetKit

// MARK: - Small Widget

/// Compact grid showing server status dots with names.
/// Fits 4-6 servers in a grid layout.
struct SmallWidgetView: View {
    let entry: ServerEntry

    private let columns = [
        GridItem(.flexible(), spacing: 6),
        GridItem(.flexible(), spacing: 6),
    ]

    var body: some View {
        VStack(alignment: .leading, spacing: 6) {
            header

            if entry.servers.isEmpty {
                emptyState
            } else {
                serverGrid
            }

            Spacer(minLength: 0)

            Text(entry.statusSummary)
                .font(.caption2)
                .foregroundStyle(.secondary)
        }
        .containerBackground(.fill.tertiary, for: .widget)
    }

    private var header: some View {
        HStack(spacing: 4) {
            Image(systemName: "tree.fill")
                .font(.caption)
                .foregroundStyle(.green)
            Text("Grove")
                .font(.caption.bold())
            Spacer()
        }
    }

    private var emptyState: some View {
        VStack(spacing: 4) {
            Spacer()
            Image(systemName: "server.rack")
                .font(.title3)
                .foregroundStyle(.secondary)
            Text("No servers")
                .font(.caption2)
                .foregroundStyle(.secondary)
            Spacer()
        }
        .frame(maxWidth: .infinity)
    }

    private var serverGrid: some View {
        LazyVGrid(columns: columns, alignment: .leading, spacing: 4) {
            ForEach(Array(entry.servers.prefix(6))) { server in
                HStack(spacing: 3) {
                    Image(systemName: server.statusSFSymbol)
                        .font(.system(size: 7))
                        .foregroundStyle(statusColor(for: server))
                    Text(server.name)
                        .font(.system(size: 10))
                        .lineLimit(1)
                        .truncationMode(.tail)
                }
            }
        }
    }
}

// MARK: - Medium Widget

/// Server list with name, status icon, port, and uptime.
struct MediumWidgetView: View {
    let entry: ServerEntry

    var body: some View {
        VStack(alignment: .leading, spacing: 4) {
            headerRow

            if entry.servers.isEmpty {
                mediumEmptyState
            } else {
                serverList
            }

            Spacer(minLength: 0)
        }
        .containerBackground(.fill.tertiary, for: .widget)
    }

    private var headerRow: some View {
        HStack {
            HStack(spacing: 4) {
                Image(systemName: "tree.fill")
                    .font(.caption)
                    .foregroundStyle(.green)
                Text("Grove")
                    .font(.caption.bold())
            }
            Spacer()
            Text(entry.statusSummary)
                .font(.caption2)
                .foregroundStyle(.secondary)
        }
    }

    private var mediumEmptyState: some View {
        VStack(spacing: 6) {
            Spacer()
            Image(systemName: "server.rack")
                .font(.title2)
                .foregroundStyle(.secondary)
            Text("No servers registered")
                .font(.caption)
                .foregroundStyle(.secondary)
            Text("Run `grove start` to begin")
                .font(.caption2)
                .foregroundStyle(.tertiary)
            Spacer()
        }
        .frame(maxWidth: .infinity)
    }

    private var serverList: some View {
        VStack(alignment: .leading, spacing: 2) {
            Divider()
            ForEach(Array(entry.servers.prefix(5))) { server in
                MediumServerRow(server: server)
            }
            if entry.servers.count > 5 {
                Text("+\(entry.servers.count - 5) more")
                    .font(.caption2)
                    .foregroundStyle(.tertiary)
            }
        }
    }
}

struct MediumServerRow: View {
    let server: WidgetServer

    var body: some View {
        HStack(spacing: 6) {
            Image(systemName: server.statusSFSymbol)
                .font(.system(size: 8))
                .foregroundStyle(statusColor(for: server))
                .frame(width: 10)

            Text(server.name)
                .font(.system(size: 11, weight: .medium))
                .lineLimit(1)

            Spacer()

            Text(server.displayPort)
                .font(.system(size: 10, design: .monospaced))
                .foregroundStyle(.secondary)

            Text(server.uptimeString)
                .font(.system(size: 10, design: .monospaced))
                .foregroundStyle(.secondary)
                .frame(width: 46, alignment: .trailing)
        }
        .padding(.vertical, 1)
    }
}

// MARK: - Large Widget

/// Full server dashboard with health summary and PR info.
struct LargeWidgetView: View {
    let entry: ServerEntry

    var body: some View {
        VStack(alignment: .leading, spacing: 6) {
            largeHeader
            summaryBar

            if entry.servers.isEmpty {
                largeEmptyState
            } else {
                Divider()
                serverDetails
            }

            Spacer(minLength: 0)
        }
        .containerBackground(.fill.tertiary, for: .widget)
    }

    private var largeHeader: some View {
        HStack(spacing: 4) {
            Image(systemName: "tree.fill")
                .font(.subheadline)
                .foregroundStyle(.green)
            Text("Grove Server Dashboard")
                .font(.subheadline.bold())
            Spacer()
        }
    }

    private var summaryBar: some View {
        HStack(spacing: 8) {
            Label(entry.statusSummary, systemImage: "server.rack")
            Text("·")
            Label {
                Text(entry.healthSummary)
            } icon: {
                Image(systemName: entry.allHealthy ? "checkmark.circle" : "exclamationmark.triangle")
                    .foregroundStyle(entry.allHealthy ? .green : .orange)
            }
            Spacer()
        }
        .font(.caption2)
        .foregroundStyle(.secondary)
    }

    private var largeEmptyState: some View {
        VStack(spacing: 8) {
            Spacer()
            Image(systemName: "server.rack")
                .font(.largeTitle)
                .foregroundStyle(.secondary)
            Text("No servers registered")
                .font(.callout)
                .foregroundStyle(.secondary)
            Text("Run `grove start` in a worktree to begin")
                .font(.caption)
                .foregroundStyle(.tertiary)
            Spacer()
        }
        .frame(maxWidth: .infinity)
    }

    private var serverDetails: some View {
        VStack(alignment: .leading, spacing: 2) {
            // Running servers first
            let running = entry.servers.filter(\.isRunning)
            let stopped = entry.servers.filter { !$0.isRunning }

            ForEach(Array(running.prefix(6))) { server in
                LargeServerRow(server: server)
            }

            if !running.isEmpty && !stopped.isEmpty {
                Divider()
                    .padding(.vertical, 2)
            }

            ForEach(Array(stopped.prefix(max(0, 8 - running.count)))) { server in
                LargeServerRow(server: server)
            }

            let totalShown = min(running.count, 6) + min(stopped.count, max(0, 8 - running.count))
            if entry.servers.count > totalShown {
                Text("+\(entry.servers.count - totalShown) more")
                    .font(.caption2)
                    .foregroundStyle(.tertiary)
                    .padding(.top, 2)
            }
        }
    }
}

struct LargeServerRow: View {
    let server: WidgetServer

    var body: some View {
        VStack(alignment: .leading, spacing: 1) {
            HStack(spacing: 6) {
                Image(systemName: server.statusSFSymbol)
                    .font(.system(size: 9))
                    .foregroundStyle(statusColor(for: server))
                    .frame(width: 12)

                Text(server.name)
                    .font(.system(size: 12, weight: .medium))
                    .lineLimit(1)

                Spacer()

                if server.health == .unhealthy {
                    Image(systemName: "exclamationmark.triangle.fill")
                        .font(.system(size: 9))
                        .foregroundStyle(.orange)
                }
            }

            if server.isRunning {
                HStack(spacing: 6) {
                    Color.clear.frame(width: 12) // align with name above

                    Text(server.displayPort)
                        .font(.system(size: 10, design: .monospaced))

                    Text("·")

                    Text(server.uptimeString)
                        .font(.system(size: 10, design: .monospaced))

                    if let branch = server.branch {
                        Text("·")
                        Image(systemName: "arrow.triangle.branch")
                            .font(.system(size: 8))
                        Text(branch)
                            .font(.system(size: 10))
                            .lineLimit(1)
                    }

                    Spacer()
                }
                .foregroundStyle(.secondary)
            }
        }
        .padding(.vertical, 2)
    }
}

// MARK: - Shared Helpers

func statusColor(for server: WidgetServer) -> Color {
    switch server.status {
    case .running: return .green
    case .starting: return .yellow
    case .stopped: return .gray
    case .crashed: return .red
    case .stopping: return .orange
    }
}
