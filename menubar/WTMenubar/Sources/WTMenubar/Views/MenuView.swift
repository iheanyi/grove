import SwiftUI

struct MenuView: View {
    @EnvironmentObject var serverManager: ServerManager

    var body: some View {
        if serverManager.isStreamingLogs {
            LogsView()
                .environmentObject(serverManager)
        } else {
            mainMenuView
        }
    }

    private var mainMenuView: some View {
        VStack(alignment: .leading, spacing: 0) {
            // Header
            HStack {
                Text("wt")
                    .font(.headline)
                    .foregroundColor(.wtPrimary)
                Spacer()
                if serverManager.isLoading {
                    ProgressView()
                        .scaleEffect(0.5)
                }
                Button {
                    serverManager.refresh()
                } label: {
                    Image(systemName: "arrow.clockwise")
                }
                .buttonStyle(.plain)
            }
            .padding(.horizontal)
            .padding(.vertical, 8)

            Divider()

            // Servers
            if serverManager.servers.isEmpty {
                VStack(spacing: 8) {
                    Image(systemName: "server.rack")
                        .font(.largeTitle)
                        .foregroundColor(.gray)
                    Text("No servers registered")
                        .foregroundColor(.secondary)
                    Text("Run 'wt start' in terminal")
                        .font(.caption)
                        .foregroundColor(.secondary)
                }
                .frame(maxWidth: .infinity)
                .padding(.vertical, 20)
            } else {
                ScrollView {
                    VStack(spacing: 0) {
                        // Running servers
                        let running = serverManager.servers.filter { $0.isRunning }
                        if !running.isEmpty {
                            SectionHeader(title: "Running", count: running.count)
                            ForEach(running) { server in
                                ServerRowView(server: server)
                            }
                        }

                        // Stopped servers
                        let stopped = serverManager.servers.filter { !$0.isRunning }
                        if !stopped.isEmpty {
                            SectionHeader(title: "Stopped", count: stopped.count)
                            ForEach(stopped) { server in
                                ServerRowView(server: server)
                            }
                        }
                    }
                }
                .frame(maxHeight: 300)
            }

            Divider()

            // Proxy status
            ProxyStatusView()
                .padding(.horizontal)
                .padding(.vertical, 8)

            Divider()

            // Actions
            VStack(spacing: 4) {
                ActionButton(
                    title: "Open TUI",
                    icon: "terminal",
                    action: serverManager.openTUI
                )

                ActionButton(
                    title: "Quit",
                    icon: "xmark.circle",
                    action: { NSApplication.shared.terminate(nil) }
                )
            }
            .padding(.horizontal)
            .padding(.vertical, 8)
        }
        .frame(width: 300)
    }
}

struct SectionHeader: View {
    let title: String
    let count: Int

    var body: some View {
        HStack {
            Text(title.uppercased())
                .font(.caption)
                .foregroundColor(.secondary)
            Spacer()
            Text("\(count)")
                .font(.caption)
                .foregroundColor(.secondary)
        }
        .padding(.horizontal)
        .padding(.vertical, 4)
        .background(Color(NSColor.windowBackgroundColor).opacity(0.5))
    }
}

struct ServerRowView: View {
    @EnvironmentObject var serverManager: ServerManager
    let server: Server
    @State private var isHovered = false

    var body: some View {
        HStack {
            // Status indicator
            Image(systemName: server.statusIcon)
                .foregroundColor(server.statusColor)
                .frame(width: 16)

            // Server info
            VStack(alignment: .leading, spacing: 2) {
                Text(server.name)
                    .font(.system(.body, design: .monospaced))
                Text(String(format: ":%d", server.port))
                    .font(.caption)
                    .foregroundColor(.secondary)
            }

            Spacer()

            // Actions
            if isHovered || server.isRunning {
                HStack(spacing: 8) {
                    // Logs button - always available if server has log file
                    if server.logFile != nil {
                        Button {
                            serverManager.startStreamingLogs(for: server)
                        } label: {
                            Image(systemName: "doc.text")
                        }
                        .buttonStyle(.plain)
                        .help("View logs")
                    }

                    if server.isRunning {
                        Button {
                            serverManager.openServer(server)
                        } label: {
                            Image(systemName: "arrow.up.right.square")
                        }
                        .buttonStyle(.plain)
                        .help("Open in browser")

                        Button {
                            serverManager.copyURL(server)
                        } label: {
                            Image(systemName: "doc.on.doc")
                        }
                        .buttonStyle(.plain)
                        .help("Copy URL")

                        Button {
                            serverManager.stopServer(server)
                        } label: {
                            Image(systemName: "stop.fill")
                                .foregroundColor(.red)
                        }
                        .buttonStyle(.plain)
                        .help("Stop server")
                    }
                }
            }
        }
        .padding(.horizontal)
        .padding(.vertical, 6)
        .background(isHovered ? Color.gray.opacity(0.1) : Color.clear)
        .onHover { hovering in
            isHovered = hovering
        }
    }
}

struct ProxyStatusView: View {
    @EnvironmentObject var serverManager: ServerManager

    var body: some View {
        HStack {
            if serverManager.isSubdomainMode {
                // Subdomain mode - show proxy controls
                if let proxy = serverManager.proxy {
                    Image(systemName: proxy.isRunning ? "checkmark.circle.fill" : "xmark.circle")
                        .foregroundColor(proxy.isRunning ? .green : .gray)

                    VStack(alignment: .leading, spacing: 0) {
                        Text("Proxy")
                            .font(.caption)
                        if proxy.isRunning {
                            Text(String(format: ":%d/:%d", proxy.httpPort, proxy.httpsPort))
                                .font(.caption2)
                                .foregroundColor(.secondary)
                        } else {
                            Text("Not running")
                                .font(.caption2)
                                .foregroundColor(.secondary)
                        }
                    }

                    Spacer()

                    Button {
                        if proxy.isRunning {
                            serverManager.stopProxy()
                        } else {
                            serverManager.startProxy()
                        }
                    } label: {
                        Text(proxy.isRunning ? "Stop" : "Start")
                            .font(.caption)
                    }
                    .buttonStyle(.bordered)
                    .controlSize(.small)
                }
            } else {
                // Port mode - show mode info
                Image(systemName: "network")
                    .foregroundColor(.blue)

                VStack(alignment: .leading, spacing: 0) {
                    Text("URL Mode: Port")
                        .font(.caption)
                    Text("Access servers via localhost:PORT")
                        .font(.caption2)
                        .foregroundColor(.secondary)
                }

                Spacer()
            }
        }
    }
}

struct ActionButton: View {
    let title: String
    let icon: String
    let action: () -> Void
    @State private var isHovered = false

    var body: some View {
        Button(action: action) {
            HStack {
                Image(systemName: icon)
                    .frame(width: 20)
                Text(title)
                Spacer()
            }
            .padding(.horizontal, 8)
            .padding(.vertical, 4)
            .background(isHovered ? Color.gray.opacity(0.1) : Color.clear)
            .cornerRadius(4)
        }
        .buttonStyle(.plain)
        .onHover { hovering in
            isHovered = hovering
        }
    }
}

// Preview disabled for SPM builds
// #Preview {
//     MenuView()
//         .environmentObject(ServerManager())
// }
