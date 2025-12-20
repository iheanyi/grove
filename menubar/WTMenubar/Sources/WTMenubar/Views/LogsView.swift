import SwiftUI

struct LogsView: View {
    @EnvironmentObject var serverManager: ServerManager
    @State private var autoScroll = true

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
                        .foregroundColor(.wtPrimary)
                }

                Spacer()

                // Auto-scroll toggle
                Toggle(isOn: $autoScroll) {
                    Image(systemName: "arrow.down.to.line")
                }
                .toggleStyle(.button)
                .buttonStyle(.plain)
                .help("Auto-scroll to bottom")

                Button {
                    serverManager.clearLogs()
                } label: {
                    Image(systemName: "trash")
                }
                .buttonStyle(.plain)
                .help("Clear logs")

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
                        ForEach(Array(serverManager.logLines.enumerated()), id: \.offset) { index, line in
                            LogLineView(line: line)
                                .id(index)
                        }
                    }
                    .padding(.horizontal, 8)
                    .padding(.vertical, 4)
                }
                .onChange(of: serverManager.logLines.count) { _ in
                    if autoScroll, let lastIndex = serverManager.logLines.indices.last {
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

                Text("\(serverManager.logLines.count) lines")
                    .font(.caption2)
                    .foregroundColor(.secondary)
            }
            .padding(.horizontal)
            .padding(.vertical, 4)
        }
        .frame(width: 500, height: 400)
    }
}

struct LogLineView: View {
    let line: String

    var body: some View {
        Text(line)
            .font(.system(size: 11, design: .monospaced))
            .foregroundColor(lineColor)
            .textSelection(.enabled)
            .frame(maxWidth: .infinity, alignment: .leading)
    }

    private var lineColor: Color {
        let lowercased = line.lowercased()
        if lowercased.contains("error") || lowercased.contains("fatal") || lowercased.contains("fail") {
            return .red
        } else if lowercased.contains("warn") {
            return .orange
        } else if lowercased.contains("debug") {
            return .gray
        }
        return .primary
    }
}
