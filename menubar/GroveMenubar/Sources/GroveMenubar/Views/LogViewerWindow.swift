import SwiftUI

/// Standalone log viewer window that can be opened from the menubar
struct LogViewerWindow: View {
    @EnvironmentObject var serverManager: ServerManager
    @State private var selectedServerName: String?
    @State private var isSplitView = false
    @State private var splitRightServerName: String?

    var body: some View {
        HSplitView {
            // Sidebar - Server List
            serverSidebar
                .frame(minWidth: 180, idealWidth: 200, maxWidth: 250)

            // Main Content - Logs
            if isSplitView {
                splitLogContent
            } else {
                LogPaneView(
                    servers: serverManager.servers,
                    selectedServerName: $selectedServerName
                )
            }
        }
        .frame(minWidth: 800, minHeight: 500)
        .onAppear {
            // Select first server with logs, or first running server
            if selectedServerName == nil {
                if let current = serverManager.selectedServerForLogs {
                    selectedServerName = current.name
                } else if let running = serverManager.servers.first(where: { $0.isRunning }) {
                    selectedServerName = running.name
                    serverManager.startStreamingLogs(for: running)
                }
            }
        }
    }

    // MARK: - Split Log Content

    private var splitLogContent: some View {
        HSplitView {
            LogPaneView(
                servers: serverManager.servers,
                selectedServerName: $selectedServerName,
                isCompact: true
            )
            .frame(minWidth: 300)

            Divider()

            LogPaneView(
                servers: serverManager.servers,
                selectedServerName: $splitRightServerName,
                isCompact: true
            )
            .frame(minWidth: 300)
        }
        .onAppear {
            // Default the right pane to the second running server
            if splitRightServerName == nil {
                let running = serverManager.servers.filter { $0.isRunning }
                if running.count > 1 {
                    splitRightServerName = running[1].name
                }
            }
        }
    }

    // MARK: - Server Sidebar

    private var serverSidebar: some View {
        VStack(spacing: 0) {
            // Header
            HStack {
                Text("Servers")
                    .font(.headline)
                    .foregroundColor(.secondary)
                Spacer()

                // Split view toggle
                Toggle(isOn: $isSplitView) {
                    Image(systemName: "rectangle.split.2x1")
                }
                .toggleStyle(.button)
                .buttonStyle(.bordered)
                .controlSize(.small)
                .help("Toggle split view")
            }
            .padding(.horizontal, 12)
            .padding(.vertical, 10)

            Divider()

            // Server list
            ScrollView {
                LazyVStack(spacing: 2) {
                    ForEach(serverManager.servers) { server in
                        ServerSidebarRow(
                            server: server,
                            isSelected: selectedServerName == server.name,
                            onSelect: {
                                selectedServerName = server.name
                                serverManager.startStreamingLogs(for: server)
                            }
                        )
                    }
                }
                .padding(.vertical, 8)
            }

            Divider()

            // Sidebar footer
            HStack {
                Text("\(serverManager.runningCount) running")
                    .font(.caption)
                    .foregroundColor(.secondary)
                Spacer()
            }
            .padding(.horizontal, 12)
            .padding(.vertical, 8)
            .background(Color(NSColor.windowBackgroundColor))
        }
        .background(Color(NSColor.controlBackgroundColor))
    }
}

// MARK: - Server Sidebar Row

struct ServerSidebarRow: View {
    let server: Server
    let isSelected: Bool
    let onSelect: () -> Void

    var body: some View {
        Button(action: onSelect) {
            HStack(spacing: 10) {
                Circle()
                    .fill(server.statusColor)
                    .frame(width: 8, height: 8)

                VStack(alignment: .leading, spacing: 2) {
                    Text(server.name)
                        .font(.system(size: 12, weight: isSelected ? .semibold : .regular))
                        .foregroundColor(isSelected ? .primary : .secondary)
                        .lineLimit(1)
                        .truncationMode(.middle)
                        .help(server.name)

                    if server.isRunning, let port = server.port {
                        Text(":\(String(port))")
                            .font(.system(size: 10, design: .monospaced))
                            .foregroundColor(.secondary)
                    }
                }

                Spacer()

                if server.logFile != nil {
                    Image(systemName: "doc.text")
                        .font(.system(size: 10))
                        .foregroundColor(.secondary.opacity(0.6))
                }
            }
            .padding(.horizontal, 12)
            .padding(.vertical, 8)
            .background(
                RoundedRectangle(cornerRadius: 6)
                    .fill(isSelected ? Color.accentColor.opacity(0.15) : Color.clear)
            )
            .contentShape(Rectangle())
        }
        .buttonStyle(.plain)
        .padding(.horizontal, 8)
    }
}

// MARK: - Log Line Row

struct LogLineRow: View {
    let line: String
    let lineNumber: Int?
    let fontSize: Double
    let searchText: String

    var body: some View {
        HStack(alignment: .top, spacing: 0) {
            if let lineNumber = lineNumber {
                Text("\(lineNumber)")
                    .font(.system(size: fontSize, design: .monospaced))
                    .foregroundColor(.secondary.opacity(0.5))
                    .frame(width: 50, alignment: .trailing)
                    .padding(.trailing, 12)

                Divider()
                    .frame(height: fontSize + 8)
                    .padding(.trailing, 12)
            }

            Text(highlightedLine)
                .font(.system(size: fontSize, design: .monospaced))
                .textSelection(.enabled)
                .frame(maxWidth: .infinity, alignment: .leading)
        }
        .padding(.vertical, 2)
    }

    private var highlightedLine: AttributedString {
        var result = LogHighlighter.highlight(line)

        // Additionally highlight search matches
        if !searchText.isEmpty {
            if let range = line.range(of: searchText, options: .caseInsensitive) {
                let startOffset = line.distance(from: line.startIndex, to: range.lowerBound)
                let length = line.distance(from: range.lowerBound, to: range.upperBound)

                var currentIndex = result.startIndex
                for _ in 0..<startOffset {
                    guard currentIndex < result.endIndex else { break }
                    currentIndex = result.index(afterCharacter: currentIndex)
                }

                var endIndex = currentIndex
                for _ in 0..<length {
                    guard endIndex < result.endIndex else { break }
                    endIndex = result.index(afterCharacter: endIndex)
                }

                if currentIndex < result.endIndex && endIndex <= result.endIndex {
                    result[currentIndex..<endIndex].backgroundColor = .yellow.opacity(0.3)
                    result[currentIndex..<endIndex].foregroundColor = .black
                }
            }
        }

        return result
    }
}
