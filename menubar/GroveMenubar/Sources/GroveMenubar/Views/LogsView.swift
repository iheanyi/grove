import SwiftUI

struct LogsView: View {
    @EnvironmentObject var serverManager: ServerManager
    @State private var autoScroll = true
    @State private var searchText = ""
    @State private var selectedLogLevel: LogLevel? = nil
    @State private var showLineNumbers = false
    @State private var showPopoutWindow = false

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
        VStack(spacing: 0) {
            // Header
            HStack {
                Button {
                    serverManager.stopStreamingLogs()
                } label: {
                    Image(systemName: "chevron.left")
                }
                .buttonStyle(.plain)

                if let server = serverManager.selectedServerForLogs {
                    Text(server.name)
                        .font(.headline)
                        .foregroundColor(.grovePrimary)
                        .lineLimit(1)
                        .truncationMode(.middle)
                        .help(server.name)
                }

                Spacer()
            }
            .padding(.horizontal)
            .padding(.vertical, 8)

            Divider()

            // Search and filter bar
            VStack(spacing: 8) {
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
                .padding(6)
                .background(Color(NSColor.controlBackgroundColor))
                .cornerRadius(6)

                // Log level filters
                HStack(spacing: 8) {
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
                                Text(level.rawValue)
                                    .font(.caption)
                            }
                            .padding(.horizontal, 8)
                            .padding(.vertical, 4)
                            .background(selectedLogLevel == level ? level.color.opacity(0.2) : Color.clear)
                            .foregroundColor(selectedLogLevel == level ? level.color : .secondary)
                            .cornerRadius(4)
                            .overlay(
                                RoundedRectangle(cornerRadius: 4)
                                    .stroke(selectedLogLevel == level ? level.color : Color.secondary.opacity(0.3), lineWidth: 1)
                            )
                        }
                        .buttonStyle(.plain)
                    }

                    Spacer()
                }
            }
            .padding(.horizontal)
            .padding(.vertical, 8)
            .background(Color(NSColor.windowBackgroundColor).opacity(0.5))

            Divider()

            // Toolbar
            HStack {
                // Auto-scroll toggle
                Toggle(isOn: $autoScroll) {
                    HStack(spacing: 4) {
                        Image(systemName: "arrow.down.to.line")
                        Text("Auto-scroll")
                            .font(.caption)
                    }
                }
                .toggleStyle(.button)
                .buttonStyle(.bordered)
                .controlSize(.small)
                .help("Auto-scroll to bottom")

                // Line numbers toggle
                Toggle(isOn: $showLineNumbers) {
                    HStack(spacing: 4) {
                        Image(systemName: "list.number")
                        Text("Lines")
                            .font(.caption)
                    }
                }
                .toggleStyle(.button)
                .buttonStyle(.bordered)
                .controlSize(.small)
                .help("Show line numbers")

                Spacer()

                // Copy All button
                Button {
                    copyAllLogs()
                } label: {
                    HStack(spacing: 4) {
                        Image(systemName: "doc.on.doc")
                        Text("Copy All")
                            .font(.caption)
                    }
                }
                .buttonStyle(.bordered)
                .controlSize(.small)
                .help("Copy all logs to clipboard")

                // Popout button
                Button {
                    showPopoutWindow.toggle()
                } label: {
                    HStack(spacing: 4) {
                        Image(systemName: "arrow.up.left.and.arrow.down.right")
                        Text("Popout")
                            .font(.caption)
                    }
                }
                .buttonStyle(.bordered)
                .controlSize(.small)
                .help("Open logs in separate window")

                // Clear button
                Button {
                    serverManager.clearLogs()
                } label: {
                    Image(systemName: "trash")
                }
                .buttonStyle(.plain)
                .help("Clear logs")

                // Show in Finder button
                if let server = serverManager.selectedServerForLogs {
                    Button {
                        serverManager.openLogsInFinder(server)
                    } label: {
                        Image(systemName: "folder")
                    }
                    .buttonStyle(.plain)
                    .help("Show in Finder")
                }
            }
            .padding(.horizontal)
            .padding(.vertical, 8)

            Divider()

            // Log content
            ScrollViewReader { proxy in
                ScrollView {
                    LazyVStack(alignment: .leading, spacing: 1) {
                        ForEach(Array(filteredLogs.enumerated()), id: \.offset) { index, line in
                            LogLineView(
                                line: line.text,
                                lineNumber: showLineNumbers ? line.originalIndex + 1 : nil
                            )
                            .id(index)
                        }
                    }
                    .padding(.horizontal, 8)
                    .padding(.vertical, 4)
                }
                .onChange(of: serverManager.logLines.count) { oldCount, newCount in
                    if autoScroll && newCount > oldCount, let lastIndex = filteredLogs.indices.last {
                        withAnimation(.easeOut(duration: 0.1)) {
                            proxy.scrollTo(lastIndex, anchor: .bottom)
                        }
                    }
                }
            }
            .background(Color(NSColor.textBackgroundColor))

            Divider()

            // Status bar
            HStack {
                if serverManager.isStreamingLogs {
                    Circle()
                        .fill(.green)
                        .frame(width: 6, height: 6)
                    Text("Streaming")
                        .font(.caption2)
                        .foregroundColor(.secondary)
                }

                Spacer()

                Text("\(filteredLogs.count) / \(serverManager.logLines.count) lines")
                    .font(.caption2)
                    .foregroundColor(.secondary)
            }
            .padding(.horizontal)
            .padding(.vertical, 4)
        }
        .frame(width: 700, height: 500)
        .sheet(isPresented: $showPopoutWindow) {
            LogsPopoutWindow()
                .environmentObject(serverManager)
        }
    }

    // MARK: - Computed Properties

    private var filteredLogs: [(text: String, originalIndex: Int)] {
        var logs = serverManager.logLines.enumerated().map { (text: $0.element, originalIndex: $0.offset) }

        // Apply search filter
        if !searchText.isEmpty {
            logs = logs.filter { $0.text.localizedCaseInsensitiveContains(searchText) }
        }

        // Apply log level filter
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

struct LogLineView: View {
    let line: String
    let lineNumber: Int?

    var body: some View {
        HStack(alignment: .top, spacing: 8) {
            if let lineNumber = lineNumber {
                Text("\(lineNumber)")
                    .font(.system(size: 11, design: .monospaced))
                    .foregroundColor(.secondary)
                    .frame(width: 40, alignment: .trailing)
            }

            Text(highlightedLine)
                .font(.system(size: 11, design: .monospaced))
                .textSelection(.enabled)
                .frame(maxWidth: .infinity, alignment: .leading)
        }
    }

    private var highlightedLine: AttributedString {
        LogHighlighter.highlight(line)
    }
}

// MARK: - Popout Window

struct LogsPopoutWindow: View {
    @EnvironmentObject var serverManager: ServerManager
    @State private var autoScroll = true
    @State private var searchText = ""
    @State private var selectedLogLevel: LogsView.LogLevel? = nil
    @State private var showLineNumbers = false
    @Environment(\.dismiss) var dismiss

    var body: some View {
        VStack(spacing: 0) {
            // Header
            HStack {
                if let server = serverManager.selectedServerForLogs {
                    Text(server.name + " - Logs")
                        .font(.headline)
                }

                Spacer()

                Button("Close") {
                    dismiss()
                }
            }
            .padding()

            Divider()

            // Search and filter bar
            VStack(spacing: 8) {
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
                .padding(6)
                .background(Color(NSColor.controlBackgroundColor))
                .cornerRadius(6)

                // Log level filters
                HStack(spacing: 8) {
                    ForEach(LogsView.LogLevel.allCases, id: \.self) { level in
                        Button {
                            if selectedLogLevel == level {
                                selectedLogLevel = nil
                            } else {
                                selectedLogLevel = level
                            }
                        } label: {
                            HStack(spacing: 4) {
                                Image(systemName: level.icon)
                                Text(level.rawValue)
                                    .font(.caption)
                            }
                            .padding(.horizontal, 8)
                            .padding(.vertical, 4)
                            .background(selectedLogLevel == level ? level.color.opacity(0.2) : Color.clear)
                            .foregroundColor(selectedLogLevel == level ? level.color : .secondary)
                            .cornerRadius(4)
                            .overlay(
                                RoundedRectangle(cornerRadius: 4)
                                    .stroke(selectedLogLevel == level ? level.color : Color.secondary.opacity(0.3), lineWidth: 1)
                            )
                        }
                        .buttonStyle(.plain)
                    }

                    Spacer()
                }
            }
            .padding(.horizontal)
            .padding(.vertical, 8)

            Divider()

            // Toolbar
            HStack {
                Toggle(isOn: $autoScroll) {
                    HStack(spacing: 4) {
                        Image(systemName: "arrow.down.to.line")
                        Text("Auto-scroll")
                            .font(.caption)
                    }
                }
                .toggleStyle(.button)
                .buttonStyle(.bordered)
                .controlSize(.small)

                Toggle(isOn: $showLineNumbers) {
                    HStack(spacing: 4) {
                        Image(systemName: "list.number")
                        Text("Lines")
                            .font(.caption)
                    }
                }
                .toggleStyle(.button)
                .buttonStyle(.bordered)
                .controlSize(.small)

                Spacer()

                Button {
                    copyAllLogs()
                } label: {
                    HStack(spacing: 4) {
                        Image(systemName: "doc.on.doc")
                        Text("Copy All")
                            .font(.caption)
                    }
                }
                .buttonStyle(.bordered)
                .controlSize(.small)

                Button {
                    serverManager.clearLogs()
                } label: {
                    Image(systemName: "trash")
                }
                .buttonStyle(.plain)
            }
            .padding(.horizontal)
            .padding(.vertical, 8)

            Divider()

            // Log content
            ScrollViewReader { proxy in
                ScrollView {
                    LazyVStack(alignment: .leading, spacing: 1) {
                        ForEach(Array(filteredLogs.enumerated()), id: \.offset) { index, line in
                            LogLineView(
                                line: line.text,
                                lineNumber: showLineNumbers ? line.originalIndex + 1 : nil
                            )
                            .id(index)
                        }
                    }
                    .padding(.horizontal, 8)
                    .padding(.vertical, 4)
                }
                .onChange(of: serverManager.logLines.count) { oldCount, newCount in
                    if autoScroll && newCount > oldCount, let lastIndex = filteredLogs.indices.last {
                        withAnimation(.easeOut(duration: 0.1)) {
                            proxy.scrollTo(lastIndex, anchor: .bottom)
                        }
                    }
                }
            }
            .background(Color(NSColor.textBackgroundColor))

            Divider()

            // Status bar
            HStack {
                if serverManager.isStreamingLogs {
                    Circle()
                        .fill(.green)
                        .frame(width: 6, height: 6)
                    Text("Streaming")
                        .font(.caption2)
                        .foregroundColor(.secondary)
                }

                Spacer()

                Text("\(filteredLogs.count) / \(serverManager.logLines.count) lines")
                    .font(.caption2)
                    .foregroundColor(.secondary)
            }
            .padding(.horizontal)
            .padding(.vertical, 4)
        }
        .frame(minWidth: 900, minHeight: 700)
    }

    // MARK: - Computed Properties

    private var filteredLogs: [(text: String, originalIndex: Int)] {
        var logs = serverManager.logLines.enumerated().map { (text: $0.element, originalIndex: $0.offset) }

        // Apply search filter
        if !searchText.isEmpty {
            logs = logs.filter { $0.text.localizedCaseInsensitiveContains(searchText) }
        }

        // Apply log level filter
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
