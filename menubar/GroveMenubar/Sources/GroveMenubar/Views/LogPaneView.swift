import SwiftUI
import AppKit

/// A self-contained log pane with search, filtering, error navigation, and export.
/// Used as the building block for both single and split log views.
struct LogPaneView: View {
    @EnvironmentObject var serverManager: ServerManager
    let servers: [Server]
    @Binding var selectedServerName: String?

    @State private var autoScroll = true
    @State private var searchText = ""
    @State private var selectedLogLevel: LogLevel? = nil
    @State private var showLineNumbers = true
    @State private var fontSize: Double = 11
    @State private var timeFilter: TimeFilter = .all
    @State private var currentErrorIndex: Int? = nil
    @State private var highlightedLineId: Int? = nil
    @State private var jumpToErrorOnly = false
    @State private var noMoreErrorsMessage: String? = nil

    /// Whether this pane is used inside a split view (compact toolbar)
    var isCompact: Bool = false

    var body: some View {
        VStack(spacing: 0) {
            // Server selector (for split view mode)
            if isCompact {
                compactServerSelector
                Divider()
            }

            // Toolbar
            paneToolbar

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

    // MARK: - Compact Server Selector

    private var compactServerSelector: some View {
        HStack(spacing: 8) {
            Picker("Server", selection: $selectedServerName) {
                Text("Select server...").tag(nil as String?)
                ForEach(servers) { server in
                    HStack(spacing: 6) {
                        Circle()
                            .fill(server.statusColor)
                            .frame(width: 6, height: 6)
                        Text(server.name)
                    }
                    .tag(server.name as String?)
                }
            }
            .labelsHidden()
            .frame(maxWidth: .infinity)
        }
        .padding(.horizontal, 12)
        .padding(.vertical, 6)
        .background(Color(NSColor.controlBackgroundColor))
        .onChange(of: selectedServerName) { _, newValue in
            if let name = newValue,
               let server = servers.first(where: { $0.name == name }) {
                serverManager.startStreamingLogs(for: server)
            }
        }
    }

    // MARK: - Toolbar

    private var paneToolbar: some View {
        HStack(spacing: 8) {
            if !isCompact {
                if let serverName = selectedServerName,
                   let server = servers.first(where: { $0.name == serverName }) {
                    HStack(spacing: 6) {
                        Circle()
                            .fill(server.statusColor)
                            .frame(width: 8, height: 8)
                        Text(server.name)
                            .font(.headline)
                            .lineLimit(1)
                            .truncationMode(.middle)
                        if server.isRunning, let port = server.port {
                            Text(":\(String(port))")
                                .font(.system(.subheadline, design: .monospaced))
                                .foregroundColor(.grovePrimary)
                        }
                    }
                }
                Spacer()
            }

            // Error navigation
            HStack(spacing: 4) {
                Button {
                    jumpToPreviousError()
                } label: {
                    Image(systemName: "chevron.up")
                }
                .buttonStyle(.bordered)
                .controlSize(.small)
                .help("Previous error (Cmd+Shift+E)")
                .keyboardShortcut("e", modifiers: [.command, .shift])

                Button {
                    jumpToNextError()
                } label: {
                    Image(systemName: "chevron.down")
                }
                .buttonStyle(.bordered)
                .controlSize(.small)
                .help("Next error (Cmd+E)")
                .keyboardShortcut("e", modifiers: .command)

                // Toggle: errors only vs errors+warnings
                Button {
                    jumpToErrorOnly.toggle()
                } label: {
                    Image(systemName: jumpToErrorOnly ? "exclamationmark.triangle.fill" : "exclamationmark.triangle")
                        .foregroundColor(jumpToErrorOnly ? .red : .secondary)
                }
                .buttonStyle(.bordered)
                .controlSize(.small)
                .help(jumpToErrorOnly ? "Jumping to errors only" : "Jumping to errors and warnings")
            }

            if !isCompact {
                Divider().frame(height: 20)
            }

            // Toggle controls
            HStack(spacing: 6) {
                Toggle(isOn: $autoScroll) {
                    Label("Auto-scroll", systemImage: "arrow.down.to.line")
                }
                .toggleStyle(.button)
                .buttonStyle(.bordered)
                .controlSize(.small)

                if !isCompact {
                    Toggle(isOn: $showLineNumbers) {
                        Label("Lines", systemImage: "list.number")
                    }
                    .toggleStyle(.button)
                    .buttonStyle(.bordered)
                    .controlSize(.small)

                    Menu {
                        ForEach([9, 10, 11, 12, 13, 14], id: \.self) { size in
                            Button("\(size) pt") { fontSize = Double(size) }
                        }
                    } label: {
                        Label("\(Int(fontSize))", systemImage: "textformat.size")
                    }
                    .menuIndicator(.hidden)
                    .fixedSize()
                }
            }

            if !isCompact {
                Divider().frame(height: 20)

                // Time filter
                Menu {
                    ForEach(TimeFilter.allCases) { filter in
                        Button {
                            timeFilter = filter
                        } label: {
                            HStack {
                                Text(filter.rawValue)
                                if timeFilter == filter {
                                    Image(systemName: "checkmark")
                                }
                            }
                        }
                    }
                } label: {
                    HStack(spacing: 4) {
                        Image(systemName: "clock")
                        Text(timeFilter.rawValue)
                            .font(.caption)
                    }
                    .padding(.horizontal, 6)
                    .padding(.vertical, 3)
                    .background(timeFilter != .all ? Color.accentColor.opacity(0.15) : Color.clear)
                    .cornerRadius(4)
                }
                .menuIndicator(.hidden)

                Divider().frame(height: 20)

                // Actions
                HStack(spacing: 6) {
                    Button {
                        copyAllLogs()
                    } label: {
                        Label("Copy", systemImage: "doc.on.doc")
                    }
                    .buttonStyle(.bordered)
                    .controlSize(.small)

                    Button {
                        exportLogs()
                    } label: {
                        Label("Export", systemImage: "square.and.arrow.up")
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
                       let server = servers.first(where: { $0.name == serverName }) {
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

            if isCompact {
                Spacer()
            }

            // "No more errors" indicator
            if let message = noMoreErrorsMessage {
                Text(message)
                    .font(.caption)
                    .foregroundColor(.secondary)
                    .padding(.horizontal, 8)
                    .padding(.vertical, 3)
                    .background(Color.secondary.opacity(0.1))
                    .cornerRadius(4)
                    .transition(.opacity)
            }
        }
        .padding(.horizontal, isCompact ? 8 : 16)
        .padding(.vertical, isCompact ? 6 : 10)
        .background(Color(NSColor.windowBackgroundColor))
    }

    // MARK: - Search and Filter Bar

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
        .padding(.horizontal, isCompact ? 8 : 16)
        .padding(.vertical, 8)
        .background(Color(NSColor.windowBackgroundColor).opacity(0.5))
    }

    // MARK: - Log Scroll View

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
                        .background(
                            highlightedLineId == index
                                ? Color.yellow.opacity(0.25)
                                : (line.isError ? Color.red.opacity(0.05) : Color.clear)
                        )
                    }
                }
                .padding(.horizontal, 12)
                .padding(.vertical, 8)
            }
            .onChange(of: serverManager.logLines.count) { oldCount, newCount in
                if autoScroll && newCount > oldCount, let lastIndex = filteredLogs.indices.last {
                    withAnimation(.easeOut(duration: 0.1)) {
                        proxy.scrollTo(lastIndex, anchor: .bottom)
                    }
                }
            }
            .onChange(of: highlightedLineId) { _, newValue in
                if let lineId = newValue {
                    withAnimation(.easeOut(duration: 0.2)) {
                        proxy.scrollTo(lineId, anchor: .center)
                    }
                    // Clear highlight after a brief flash
                    DispatchQueue.main.asyncAfter(deadline: .now() + 1.5) {
                        withAnimation(.easeOut(duration: 0.3)) {
                            if highlightedLineId == lineId {
                                highlightedLineId = nil
                            }
                        }
                    }
                }
            }
        }
        .background(Color(NSColor.textBackgroundColor))
    }

    // MARK: - Empty State

    private var emptyState: some View {
        VStack(spacing: 16) {
            Image(systemName: "doc.text.magnifyingglass")
                .font(.system(size: 48))
                .foregroundColor(.secondary.opacity(0.5))
            Text("No Server Selected")
                .font(.title2)
                .foregroundColor(.secondary)
            Text("Select a server to view its logs")
                .font(.subheadline)
                .foregroundColor(.secondary.opacity(0.8))
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .background(Color(NSColor.textBackgroundColor))
    }

    // MARK: - Status Bar

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

            // Error/warning counts
            let counts = logCounts
            if counts.errors > 0 || counts.warnings > 0 {
                HStack(spacing: 8) {
                    if counts.errors > 0 {
                        HStack(spacing: 3) {
                            Image(systemName: "exclamationmark.triangle.fill")
                                .font(.system(size: 9))
                            Text("\(counts.errors) error\(counts.errors == 1 ? "" : "s")")
                                .font(.caption)
                        }
                        .foregroundColor(.red)
                    }
                    if counts.warnings > 0 {
                        HStack(spacing: 3) {
                            Image(systemName: "exclamationmark.circle.fill")
                                .font(.system(size: 9))
                            Text("\(counts.warnings) warning\(counts.warnings == 1 ? "" : "s")")
                                .font(.caption)
                        }
                        .foregroundColor(.orange)
                    }
                }
            } else {
                Text("No issues")
                    .font(.caption)
                    .foregroundColor(.secondary)
            }

            Text("·")
                .font(.caption)
                .foregroundColor(.secondary)

            Text("\(filteredLogs.count) of \(serverManager.logLines.count) lines")
                .font(.caption)
                .foregroundColor(.secondary)
        }
        .padding(.horizontal, isCompact ? 8 : 16)
        .padding(.vertical, 6)
        .background(Color(NSColor.windowBackgroundColor))
    }

    // MARK: - Computed Properties

    private var filteredLogs: [(text: String, originalIndex: Int, isError: Bool)] {
        let selectedServer = selectedServerName.flatMap { name in
            servers.first(where: { $0.name == name })
        }
        let cutoff = timeFilter.cutoffDate(serverUptime: selectedServer?.uptime)

        var logs: [(text: String, originalIndex: Int, isError: Bool)] = serverManager.logLines.enumerated().map { index, line in
            let isErr = LogLevel.error.matches(line) || LogLevel.warn.matches(line)
            return (text: line, originalIndex: index, isError: isErr)
        }

        // Apply time filter
        if let cutoff = cutoff {
            logs = logs.filter { entry in
                guard let timestamp = TimeFilter.parseTimestamp(from: entry.text) else {
                    return true // Keep lines without parseable timestamps
                }
                return timestamp >= cutoff
            }
        }

        // Apply search filter
        if !searchText.isEmpty {
            logs = logs.filter { $0.text.localizedCaseInsensitiveContains(searchText) }
        }

        // Apply log level filter
        if let level = selectedLogLevel {
            logs = logs.filter { level.matches($0.text) }
        }

        return logs
    }

    /// Indices of error/warning lines in filtered logs
    private var errorLineIndices: [Int] {
        filteredLogs.enumerated().compactMap { index, entry in
            if jumpToErrorOnly {
                return LogLevel.error.matches(entry.text) ? index : nil
            } else {
                return (LogLevel.error.matches(entry.text) || LogLevel.warn.matches(entry.text)) ? index : nil
            }
        }
    }

    /// Counts of errors and warnings in the current filtered log set
    private var logCounts: (errors: Int, warnings: Int) {
        var errors = 0
        var warnings = 0
        for entry in filteredLogs {
            if LogLevel.error.matches(entry.text) {
                errors += 1
            } else if LogLevel.warn.matches(entry.text) {
                warnings += 1
            }
        }
        return (errors, warnings)
    }

    // MARK: - Error Navigation

    private func jumpToNextError() {
        let indices = errorLineIndices
        guard !indices.isEmpty else {
            showNoMoreErrors("No errors found")
            return
        }

        let current = currentErrorIndex ?? -1
        if let nextIndex = indices.first(where: { $0 > current }) {
            currentErrorIndex = nextIndex
            highlightedLineId = nextIndex
            autoScroll = false
            noMoreErrorsMessage = nil
        } else {
            showNoMoreErrors("No more errors below")
        }
    }

    private func jumpToPreviousError() {
        let indices = errorLineIndices
        guard !indices.isEmpty else {
            showNoMoreErrors("No errors found")
            return
        }

        let current = currentErrorIndex ?? filteredLogs.count
        if let prevIndex = indices.last(where: { $0 < current }) {
            currentErrorIndex = prevIndex
            highlightedLineId = prevIndex
            autoScroll = false
            noMoreErrorsMessage = nil
        } else {
            showNoMoreErrors("No more errors above")
        }
    }

    private func showNoMoreErrors(_ message: String) {
        withAnimation(.easeInOut(duration: 0.2)) {
            noMoreErrorsMessage = message
        }
        DispatchQueue.main.asyncAfter(deadline: .now() + 2.0) {
            withAnimation(.easeOut(duration: 0.3)) {
                noMoreErrorsMessage = nil
            }
        }
    }

    // MARK: - Actions

    private func copyAllLogs() {
        let text = filteredLogs.map { $0.text }.joined(separator: "\n")
        NSPasteboard.general.clearContents()
        NSPasteboard.general.setString(text, forType: .string)
    }

    private func exportLogs() {
        let panel = NSSavePanel()
        let serverName = selectedServerName ?? "grove"
        let dateStr = Self.exportDateFormatter.string(from: Date())
        panel.nameFieldStringValue = "\(serverName)-\(dateStr).log"
        panel.allowedContentTypes = [.plainText]
        panel.canCreateDirectories = true

        panel.begin { response in
            guard response == .OK, let url = panel.url else { return }

            let header = "# Exported from Grove - Server: \(serverName) - Filter: \(timeFilter.rawValue)\(selectedLogLevel != nil ? " - Level: \(selectedLogLevel!.rawValue)" : "")\(searchText.isEmpty ? "" : " - Search: \(searchText)")\n# Exported at: \(Self.exportTimestampFormatter.string(from: Date()))\n\n"
            let content = header + filteredLogs.map { $0.text }.joined(separator: "\n")

            do {
                try content.write(to: url, atomically: true, encoding: .utf8)
            } catch {
                print("[Grove] Failed to export logs: \(error.localizedDescription)")
            }
        }
    }

    private static let exportDateFormatter: DateFormatter = {
        let f = DateFormatter()
        f.dateFormat = "yyyy-MM-dd"
        f.locale = Locale(identifier: "en_US_POSIX")
        return f
    }()

    private static let exportTimestampFormatter: DateFormatter = {
        let f = DateFormatter()
        f.dateFormat = "yyyy-MM-dd HH:mm:ss"
        f.locale = Locale(identifier: "en_US_POSIX")
        return f
    }()
}
