import SwiftUI

/// Standalone log viewer window that can be opened from the menubar
struct LogViewerWindow: View {
    @EnvironmentObject var serverManager: ServerManager
    @State private var selectedServerName: String?
    @State private var autoScroll = true
    @State private var searchText = ""
    @State private var selectedLogLevel: LogLevel? = nil
    @State private var showLineNumbers = true
    @State private var fontSize: Double = 11

    enum LogLevel: String, CaseIterable {
        case error = "ERROR"
        case warn = "WARN"
        case info = "INFO"
        case debug = "DEBUG"

        var color: Color {
            switch self {
            case .error: return .red
            case .warn: return .orange
            case .info: return .blue
            case .debug: return .gray
            }
        }

        var icon: String {
            switch self {
            case .error: return "exclamationmark.triangle.fill"
            case .warn: return "exclamationmark.circle.fill"
            case .info: return "info.circle.fill"
            case .debug: return "ant.circle.fill"
            }
        }
    }

    var body: some View {
        HSplitView {
            // Sidebar - Server List
            serverSidebar
                .frame(minWidth: 180, idealWidth: 200, maxWidth: 250)

            // Main Content - Logs
            logContent
        }
        .frame(minWidth: 800, minHeight: 500)
        .onAppear {
            // Select first server with logs, or first running server
            if selectedServerName == nil {
                if let current = serverManager.selectedServerForLogs {
                    selectedServerName = current.name
                } else if let running = serverManager.servers.first(where: { $0.isRunning }) {
                    selectedServerName = running.name
                    if let server = serverManager.servers.first(where: { $0.name == running.name }) {
                        serverManager.startStreamingLogs(for: server)
                    }
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

    // MARK: - Log Content

    private var logContent: some View {
        VStack(spacing: 0) {
            // Toolbar
            logToolbar

            Divider()

            // Search and filter bar
            searchAndFilterBar

            Divider()

            // Log content
            if selectedServerName != nil {
                logScrollView
            } else {
                emptyState
            }

            Divider()

            // Status bar
            statusBar
        }
    }

    private var logToolbar: some View {
        HStack(spacing: 12) {
            if let serverName = selectedServerName,
               let server = serverManager.servers.first(where: { $0.name == serverName }) {
                // Server info
                HStack(spacing: 8) {
                    Circle()
                        .fill(server.statusColor)
                        .frame(width: 8, height: 8)

                    Text(server.name)
                        .font(.headline)

                    if server.isRunning, let port = server.port {
                        Text(":\(String(port))")
                            .font(.system(.subheadline, design: .monospaced))
                            .foregroundColor(.grovePrimary)
                    }
                }
            }

            Spacer()

            // Toggle controls
            HStack(spacing: 8) {
                Toggle(isOn: $autoScroll) {
                    Label("Auto-scroll", systemImage: "arrow.down.to.line")
                }
                .toggleStyle(.button)
                .buttonStyle(.bordered)
                .controlSize(.small)

                Toggle(isOn: $showLineNumbers) {
                    Label("Lines", systemImage: "list.number")
                }
                .toggleStyle(.button)
                .buttonStyle(.bordered)
                .controlSize(.small)

                // Font size
                Menu {
                    ForEach([9, 10, 11, 12, 13, 14], id: \.self) { size in
                        Button("\(size) pt") {
                            fontSize = Double(size)
                        }
                    }
                } label: {
                    Label("\(Int(fontSize))", systemImage: "textformat.size")
                }
                .menuStyle(.borderlessButton)
                .frame(width: 60)
            }

            Divider()
                .frame(height: 20)

            // Actions
            HStack(spacing: 8) {
                Button {
                    copyAllLogs()
                } label: {
                    Label("Copy", systemImage: "doc.on.doc")
                }
                .buttonStyle(.bordered)
                .controlSize(.small)

                Button {
                    serverManager.clearLogs()
                } label: {
                    Label("Clear", systemImage: "trash")
                }
                .buttonStyle(.bordered)
                .controlSize(.small)

                if let serverName = selectedServerName,
                   let server = serverManager.servers.first(where: { $0.name == serverName }) {
                    Button {
                        serverManager.openLogsInFinder(server)
                    } label: {
                        Label("Reveal", systemImage: "folder")
                    }
                    .buttonStyle(.bordered)
                    .controlSize(.small)
                }
            }
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 10)
        .background(Color(NSColor.windowBackgroundColor))
    }

    private var searchAndFilterBar: some View {
        HStack(spacing: 12) {
            // Search field
            HStack {
                Image(systemName: "magnifyingglass")
                    .foregroundColor(.secondary)
                TextField("Search logs...", text: $searchText)
                    .textFieldStyle(.plain)
                if !searchText.isEmpty {
                    Button {
                        searchText = ""
                    } label: {
                        Image(systemName: "xmark.circle.fill")
                            .foregroundColor(.secondary)
                    }
                    .buttonStyle(.plain)
                }
            }
            .padding(8)
            .background(Color(NSColor.controlBackgroundColor))
            .cornerRadius(8)
            .frame(maxWidth: 300)

            // Log level filters
            HStack(spacing: 6) {
                ForEach(LogLevel.allCases, id: \.self) { level in
                    Button {
                        if selectedLogLevel == level {
                            selectedLogLevel = nil
                        } else {
                            selectedLogLevel = level
                        }
                    } label: {
                        HStack(spacing: 4) {
                            Image(systemName: level.icon)
                                .font(.system(size: 10))
                            Text(level.rawValue)
                                .font(.caption)
                        }
                        .padding(.horizontal, 8)
                        .padding(.vertical, 5)
                        .background(selectedLogLevel == level ? level.color.opacity(0.2) : Color.clear)
                        .foregroundColor(selectedLogLevel == level ? level.color : .secondary)
                        .cornerRadius(6)
                        .overlay(
                            RoundedRectangle(cornerRadius: 6)
                                .stroke(selectedLogLevel == level ? level.color : Color.secondary.opacity(0.3), lineWidth: 1)
                        )
                    }
                    .buttonStyle(.plain)
                }
            }

            Spacer()

            // Match count
            if !searchText.isEmpty {
                Text("\(filteredLogs.count) matches")
                    .font(.caption)
                    .foregroundColor(.secondary)
                    .padding(.horizontal, 8)
                    .padding(.vertical, 4)
                    .background(Color.secondary.opacity(0.1))
                    .cornerRadius(4)
            }
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 8)
        .background(Color(NSColor.windowBackgroundColor).opacity(0.5))
    }

    private var logScrollView: some View {
        ScrollViewReader { proxy in
            ScrollView {
                LazyVStack(alignment: .leading, spacing: 0) {
                    ForEach(Array(filteredLogs.enumerated()), id: \.offset) { index, line in
                        LogLineRow(
                            line: line.text,
                            lineNumber: showLineNumbers ? line.originalIndex + 1 : nil,
                            fontSize: fontSize,
                            searchText: searchText
                        )
                        .id(index)
                    }
                }
                .padding(.horizontal, 12)
                .padding(.vertical, 8)
            }
            .onChange(of: serverManager.logLines.count) { _ in
                if autoScroll, let lastIndex = filteredLogs.indices.last {
                    withAnimation(.easeOut(duration: 0.1)) {
                        proxy.scrollTo(lastIndex, anchor: .bottom)
                    }
                }
            }
        }
        .background(Color(NSColor.textBackgroundColor))
    }

    private var emptyState: some View {
        VStack(spacing: 16) {
            Image(systemName: "doc.text.magnifyingglass")
                .font(.system(size: 48))
                .foregroundColor(.secondary.opacity(0.5))

            Text("No Server Selected")
                .font(.title2)
                .foregroundColor(.secondary)

            Text("Select a server from the sidebar to view its logs")
                .font(.subheadline)
                .foregroundColor(.secondary.opacity(0.8))
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .background(Color(NSColor.textBackgroundColor))
    }

    private var statusBar: some View {
        HStack {
            if serverManager.isStreamingLogs {
                HStack(spacing: 6) {
                    Circle()
                        .fill(.green)
                        .frame(width: 6, height: 6)
                    Text("Live")
                        .font(.caption)
                        .foregroundColor(.green)
                }
                .padding(.horizontal, 8)
                .padding(.vertical, 3)
                .background(Color.green.opacity(0.1))
                .cornerRadius(4)
            } else {
                HStack(spacing: 6) {
                    Circle()
                        .fill(.gray)
                        .frame(width: 6, height: 6)
                    Text("Paused")
                        .font(.caption)
                        .foregroundColor(.secondary)
                }
            }

            Spacer()

            Text("\(filteredLogs.count) of \(serverManager.logLines.count) lines")
                .font(.caption)
                .foregroundColor(.secondary)
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 6)
        .background(Color(NSColor.windowBackgroundColor))
    }

    // MARK: - Computed Properties

    private var filteredLogs: [(text: String, originalIndex: Int)] {
        var logs = serverManager.logLines.enumerated().map { (text: $0.element, originalIndex: $0.offset) }

        if !searchText.isEmpty {
            logs = logs.filter { $0.text.localizedCaseInsensitiveContains(searchText) }
        }

        if let level = selectedLogLevel {
            logs = logs.filter { line in
                line.text.contains(level.rawValue) ||
                line.text.contains(level.rawValue.lowercased()) ||
                line.text.contains("[\(level.rawValue.lowercased())]")
            }
        }

        return logs
    }

    // MARK: - Actions

    private func copyAllLogs() {
        let allLogs = serverManager.logLines.joined(separator: "\n")
        NSPasteboard.general.clearContents()
        NSPasteboard.general.setString(allLogs, forType: .string)
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
            let lowerLine = line.lowercased()
            let lowerSearch = searchText.lowercased()

            if let range = lowerLine.range(of: lowerSearch) {
                // Convert to AttributedString range
                if let attrRange = Range(range, in: result) {
                    result[attrRange].backgroundColor = .yellow.opacity(0.3)
                    result[attrRange].foregroundColor = .black
                }
            }
        }

        return result
    }
}
