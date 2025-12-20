import SwiftUI

// MARK: - Toast System

enum ToastType {
    case success(String)
    case error(String)
    case info(String)

    var icon: String {
        switch self {
        case .success: return "checkmark.circle.fill"
        case .error: return "xmark.circle.fill"
        case .info: return "info.circle.fill"
        }
    }

    var color: Color {
        switch self {
        case .success: return .green
        case .error: return .red
        case .info: return .blue
        }
    }

    var message: String {
        switch self {
        case .success(let msg), .error(let msg), .info(let msg):
            return msg
        }
    }
}

// Environment key for toast notifications
private struct ShowCopiedToastKey: EnvironmentKey {
    static let defaultValue: Binding<Bool> = .constant(false)
}

extension EnvironmentValues {
    var showCopiedToast: Binding<Bool> {
        get { self[ShowCopiedToastKey.self] }
        set { self[ShowCopiedToastKey.self] = newValue }
    }
}

// Environment key for group index
private struct GroupIndexKey: EnvironmentKey {
    static let defaultValue: Int = 0
}

extension EnvironmentValues {
    var groupIndex: Int {
        get { self[GroupIndexKey.self] }
        set { self[GroupIndexKey.self] = newValue }
    }
}

struct MenuView: View {
    @EnvironmentObject var serverManager: ServerManager
    @State private var showingPreferences = false
    @State private var selectedServerIndex: Int?
    @FocusState private var isFocused: Bool
    @State private var searchText = ""
    @FocusState private var isSearchFocused: Bool
    @State private var showCopiedToast = false
    @State private var eventMonitor: Any?
    @State private var currentToast: ToastType?
    @State private var isRefreshing = false

    var body: some View {
        if serverManager.isStreamingLogs {
            LogsView()
                .environmentObject(serverManager)
        } else {
            mainMenuView
                .sheet(isPresented: $showingPreferences) {
                    PreferencesView()
                }
                .onAppear {
                    isFocused = true
                }
        }
    }

    // Filter servers based on search text
    private var filteredServers: [Server] {
        if searchText.isEmpty {
            return serverManager.servers
        }
        return serverManager.servers.filter { server in
            server.name.localizedCaseInsensitiveContains(searchText) ||
            server.path.localizedCaseInsensitiveContains(searchText) ||
            (server.githubInfo?.prNumber.map { "#\($0)".contains(searchText) } ?? false)
        }
    }

    private var mainMenuView: some View {
        VStack(alignment: .leading, spacing: 0) {
            // Header with improved loading indicator
            HStack {
                HStack(spacing: 6) {
                    Image(systemName: "bolt.fill")
                        .foregroundColor(.grovePrimary)
                        .font(.system(size: 14))
                    Text("Grove")
                        .font(.headline)
                        .foregroundColor(.grovePrimary)
                }

                Spacer()

                // Running count badge
                if serverManager.runningCount > 0 {
                    Text("\(serverManager.runningCount) running")
                        .font(.caption2)
                        .foregroundColor(.white)
                        .padding(.horizontal, 6)
                        .padding(.vertical, 2)
                        .background(Color.green)
                        .cornerRadius(8)
                }

                // Loading indicator with animation
                if serverManager.isLoading || isRefreshing {
                    Image(systemName: "arrow.clockwise")
                        .font(.system(size: 12))
                        .foregroundColor(.secondary)
                        .rotationEffect(.degrees(isRefreshing ? 360 : 0))
                        .animation(
                            isRefreshing ? .linear(duration: 1).repeatForever(autoreverses: false) : .default,
                            value: isRefreshing
                        )
                }
            }
            .padding(.horizontal)
            .padding(.vertical, 8)

            // Error banner (if any)
            if let error = serverManager.error {
                HStack(spacing: 6) {
                    Image(systemName: "exclamationmark.triangle.fill")
                        .foregroundColor(.yellow)
                    Text(error)
                        .font(.caption)
                        .lineLimit(2)
                    Spacer()
                    Button {
                        serverManager.error = nil
                    } label: {
                        Image(systemName: "xmark")
                            .font(.caption)
                    }
                    .buttonStyle(.plain)
                }
                .padding(.horizontal, 10)
                .padding(.vertical, 6)
                .background(Color.yellow.opacity(0.15))
            }

            Divider()

            // Search field
            HStack(spacing: 8) {
                Image(systemName: "magnifyingglass")
                    .foregroundColor(.secondary)
                    .font(.system(size: 12))

                TextField("Search servers...", text: $searchText)
                    .textFieldStyle(.plain)
                    .font(.system(size: 12))
                    .focused($isSearchFocused)

                if !searchText.isEmpty {
                    Button {
                        searchText = ""
                    } label: {
                        Image(systemName: "xmark.circle.fill")
                            .foregroundColor(.secondary)
                            .font(.system(size: 12))
                    }
                    .buttonStyle(.plain)
                }
            }
            .padding(.horizontal, 10)
            .padding(.vertical, 6)
            .background(Color(NSColor.controlBackgroundColor))

            Divider()

            // Quick Actions Bar
            HStack(spacing: 12) {
                Button {
                    serverManager.stopAllServers()
                } label: {
                    HStack(spacing: 4) {
                        Image(systemName: "stop.fill")
                        Text("Stop All")
                            .font(.caption)
                    }
                }
                .buttonStyle(.bordered)
                .controlSize(.small)
                .disabled(!serverManager.hasRunningServers)
                .keyboardShortcut("s", modifiers: [.command, .shift])

                Button {
                    serverManager.openAllRunningServers()
                } label: {
                    HStack(spacing: 4) {
                        Image(systemName: "arrow.up.right.square")
                        Text("Open All")
                            .font(.caption)
                    }
                }
                .buttonStyle(.bordered)
                .controlSize(.small)
                .disabled(!serverManager.hasRunningServers)
                .keyboardShortcut("o", modifiers: [.command, .shift])

                Button {
                    serverManager.refresh()
                } label: {
                    HStack(spacing: 4) {
                        Image(systemName: "arrow.clockwise")
                        Text("Refresh")
                            .font(.caption)
                    }
                }
                .buttonStyle(.bordered)
                .controlSize(.small)
                .keyboardShortcut("r", modifiers: .command)

                Spacer()

                Button {
                    showingPreferences = true
                } label: {
                    Image(systemName: "gear")
                }
                .buttonStyle(.plain)
                .help("Settings")
            }
            .padding(.horizontal)
            .padding(.vertical, 6)
            .background(Color(NSColor.controlBackgroundColor))

            Divider()

            // Servers
            if serverManager.servers.isEmpty {
                // Enhanced empty state with onboarding
                VStack(spacing: 12) {
                    Image(systemName: "bolt.badge.clock")
                        .font(.system(size: 36))
                        .foregroundColor(.grovePrimary.opacity(0.6))

                    Text("No worktrees discovered")
                        .font(.headline)
                        .foregroundColor(.primary)

                    Text("Get started with Grove")
                        .font(.subheadline)
                        .foregroundColor(.secondary)

                    VStack(alignment: .leading, spacing: 8) {
                        OnboardingStep(number: 1, text: "Navigate to your project directory")
                        OnboardingStep(number: 2, text: "Run: grove start <command>")
                        OnboardingStep(number: 3, text: "Or: grove discover --register")
                    }
                    .padding(.vertical, 8)

                    Button {
                        serverManager.openTUI()
                    } label: {
                        HStack {
                            Image(systemName: "terminal")
                            Text("Open Terminal")
                        }
                        .font(.caption)
                    }
                    .buttonStyle(.borderedProminent)
                    .tint(.grovePrimary)
                    .controlSize(.small)
                }
                .frame(maxWidth: .infinity)
                .padding(.vertical, 24)
                .padding(.horizontal)
            } else if filteredServers.isEmpty {
                VStack(spacing: 12) {
                    Image(systemName: "magnifyingglass")
                        .font(.system(size: 28))
                        .foregroundColor(.secondary)
                    Text("No matches for '\(searchText)'")
                        .font(.subheadline)
                        .foregroundColor(.secondary)
                    Button {
                        searchText = ""
                    } label: {
                        Text("Clear search")
                            .font(.caption)
                    }
                    .buttonStyle(.bordered)
                    .controlSize(.small)
                }
                .frame(maxWidth: .infinity)
                .padding(.vertical, 20)
            } else {
                ScrollView {
                    VStack(spacing: 0) {
                        // Check if servers should be grouped
                        if ServerGrouper.shouldGroup(filteredServers) {
                            // Show grouped view
                            let groups = ServerGrouper.groupServers(filteredServers)
                            ForEach(Array(groups.enumerated()), id: \.element.id) { index, group in
                                ServerGroupView(group: group, searchText: searchText)
                                    .environment(\.groupIndex, index)
                            }
                        } else {
                            // Show simple running/stopped sections
                            // Running servers
                            let running = filteredServers.filter { $0.isRunning }
                            if !running.isEmpty {
                                SectionHeader(title: "Running", count: running.count)
                                ForEach(Array(running.enumerated()), id: \.element.id) { index, server in
                                    ServerRowView(server: server, searchText: searchText, displayIndex: index + 1)
                                }
                            }

                            // Stopped servers
                            let stopped = filteredServers.filter { !$0.isRunning }
                            if !stopped.isEmpty {
                                SectionHeader(title: "Stopped", count: stopped.count)
                                ForEach(Array(stopped.enumerated()), id: \.element.id) { index, server in
                                    ServerRowView(server: server, searchText: searchText, displayIndex: running.count + index + 1)
                                }
                            }
                        }
                    }
                    .environment(\.showCopiedToast, $showCopiedToast)
                }
                .frame(maxHeight: 300)
            }

            Divider()

            // Proxy status
            ProxyStatusView()
                .padding(.horizontal)
                .padding(.vertical, 8)

            Divider()

            // Actions with keyboard hints
            VStack(spacing: 2) {
                ActionButton(
                    title: "Open TUI",
                    icon: "terminal.fill",
                    action: serverManager.openTUI
                )

                Divider()
                    .padding(.vertical, 4)

                ActionButton(
                    title: "Quit Grove",
                    icon: "power",
                    destructive: true,
                    action: { NSApplication.shared.terminate(nil) }
                )
            }
            .padding(.horizontal, 8)
            .padding(.vertical, 8)
        }
        .frame(width: 320)
        .overlay(alignment: .bottom) {
            // Toast overlay
            if showCopiedToast || currentToast != nil {
                let toast = currentToast ?? .success("Copied to clipboard")
                HStack(spacing: 8) {
                    Image(systemName: toast.icon)
                        .foregroundColor(toast.color)
                    Text(toast.message)
                        .font(.caption)
                        .foregroundColor(.primary)
                }
                .padding(.horizontal, 14)
                .padding(.vertical, 8)
                .background(.ultraThinMaterial)
                .cornerRadius(10)
                .shadow(color: .black.opacity(0.15), radius: 8, y: 4)
                .padding(.bottom, 70)
                .transition(.move(edge: .bottom).combined(with: .opacity))
                .animation(.spring(response: 0.3), value: showCopiedToast)
            }
        }
        .onAppear {
            // Set up keyboard shortcuts handler (only once)
            guard eventMonitor == nil else { return }
            eventMonitor = NSEvent.addLocalMonitorForEvents(matching: .keyDown) { [self] event in
                if event.modifierFlags.contains(.command) {
                    if event.charactersIgnoringModifiers == "f" {
                        isSearchFocused = true
                        return nil
                    }
                }
                // Number keys 1-9 for quick-start
                if let chars = event.charactersIgnoringModifiers,
                   let num = Int(chars),
                   num >= 1 && num <= 9 {
                    let servers = filteredServers.filter { $0.isRunning || $0.displayStatus == "stopped" }
                    if num <= servers.count {
                        let server = servers[num - 1]
                        if !server.isRunning && server.displayStatus == "stopped" {
                            serverManager.startServer(server)
                        } else if server.isRunning {
                            serverManager.openServer(server)
                        }
                        return nil
                    }
                }
                return event
            }
        }
        .onDisappear {
            // Clean up the event monitor to prevent leaks
            if let monitor = eventMonitor {
                NSEvent.removeMonitor(monitor)
                eventMonitor = nil
            }
        }
    }
}

struct SectionHeader: View {
    let title: String
    let count: Int

    var body: some View {
        HStack {
            HStack(spacing: 6) {
                Circle()
                    .fill(title == "Running" ? Color.green : Color.gray.opacity(0.5))
                    .frame(width: 6, height: 6)
                Text(title.uppercased())
                    .font(.caption.weight(.medium))
                    .foregroundColor(.secondary)
            }
            Spacer()
            Text("\(count)")
                .font(.caption.weight(.semibold))
                .foregroundColor(.secondary)
                .padding(.horizontal, 6)
                .padding(.vertical, 2)
                .background(Color.secondary.opacity(0.1))
                .cornerRadius(4)
        }
        .padding(.horizontal)
        .padding(.vertical, 6)
        .background(Color(NSColor.windowBackgroundColor).opacity(0.6))
    }
}

struct ServerRowView: View {
    @EnvironmentObject var serverManager: ServerManager
    let server: Server
    var searchText: String = ""
    var displayIndex: Int?
    @State private var isHovered = false
    @State private var showingQuickActions = false
    @Environment(\.showCopiedToast) private var showCopiedToast

    private func ciStatusHelp(_ status: GitHubInfo.CIStatus) -> String {
        switch status {
        case .success: return "CI: Passed"
        case .failure: return "CI: Failed"
        case .pending: return "CI: Running"
        case .unknown: return "CI: Unknown"
        }
    }

    private func highlightedText(_ text: String) -> Text {
        if searchText.isEmpty {
            return Text(text)
        }

        let parts = text.components(separatedBy: searchText)
        if parts.count <= 1 {
            return Text(text)
        }

        var result = Text("")
        for (index, part) in parts.enumerated() {
            result = result + Text(part)
            if index < parts.count - 1 {
                result = result + Text(searchText).foregroundColor(.grovePrimary).bold()
            }
        }
        return result
    }

    var body: some View {
        HStack(spacing: 8) {
            // Status and health indicators
            HStack(spacing: 4) {
                Circle()
                    .fill(server.statusColor)
                    .frame(width: 8, height: 8)

                if server.isRunning, server.health != nil {
                    Circle()
                        .fill(server.healthColor)
                        .frame(width: 6, height: 6)
                }
            }
            .frame(width: 20)

            // Display index for keyboard shortcuts
            if let index = displayIndex, index <= 9 {
                Text("\(index)")
                    .font(.caption2)
                    .foregroundColor(.secondary)
                    .frame(width: 12)
            }

            // Server info
            VStack(alignment: .leading, spacing: 2) {
                HStack(spacing: 6) {
                    highlightedText(server.name)
                        .font(.system(.body, design: .monospaced))

                    if let uptime = server.formattedUptime, server.isRunning {
                        Text(uptime)
                            .font(.caption2)
                            .foregroundColor(.secondary)
                            .padding(.horizontal, 4)
                            .padding(.vertical, 1)
                            .background(Color.secondary.opacity(0.1))
                            .cornerRadius(3)
                    }

                    // GitHub badges
                    if let github = server.githubInfo {
                        // CI Status badge
                        if github.ciStatus != .unknown {
                            Image(systemName: github.ciStatus.icon)
                                .foregroundColor(github.ciStatus.color)
                                .font(.caption)
                                .help(ciStatusHelp(github.ciStatus))
                        }

                        // PR badge
                        if let prNumber = github.prNumber {
                            Button {
                                if let urlString = github.prURL, let url = URL(string: urlString) {
                                    NSWorkspace.shared.open(url)
                                }
                            } label: {
                                Text("#\(prNumber)")
                                    .font(.caption)
                                    .foregroundColor(.blue)
                            }
                            .buttonStyle(.plain)
                            .help("Open PR #\(prNumber)")
                        }
                    }
                }

                if let port = server.port, port > 0 {
                    HStack(spacing: 4) {
                        Text(":")
                            .foregroundColor(.secondary)
                            .font(.system(size: 10))
                        Text("\(port)")
                            .font(.system(.callout, design: .monospaced))
                            .foregroundColor(.grovePrimary)
                            .fontWeight(.medium)
                    }
                }
            }

            Spacer()

            // Actions
            if isHovered || server.isRunning {
                HStack(spacing: 6) {
                    // Quick actions menu
                    Menu {
                        Button("Open in Terminal") {
                            serverManager.openInTerminal(server)
                        }

                        Button("Open in VS Code") {
                            serverManager.openInVSCode(server)
                        }

                        Button("Open in Finder") {
                            serverManager.openInFinder(server)
                        }

                        Divider()

                        Button("Copy Path") {
                            serverManager.copyPath(server)
                        }
                    } label: {
                        Image(systemName: "ellipsis.circle")
                            .font(.system(size: 12))
                    }
                    .menuStyle(.borderlessButton)
                    .fixedSize()
                    .help("Quick Actions")

                    // Logs button - always available if server has log file
                    if server.logFile != nil {
                        Button {
                            serverManager.startStreamingLogs(for: server)
                        } label: {
                            Image(systemName: "doc.text")
                                .font(.system(size: 12))
                        }
                        .buttonStyle(.plain)
                        .help("View logs")
                    }

                    if server.isRunning {
                        Button {
                            serverManager.openServer(server)
                        } label: {
                            Image(systemName: "arrow.up.right.square")
                                .font(.system(size: 12))
                        }
                        .buttonStyle(.plain)
                        .help("Open in browser")

                        Button {
                            serverManager.copyURL(server)
                            showCopiedToast.wrappedValue = true
                            DispatchQueue.main.asyncAfter(deadline: .now() + 2) {
                                showCopiedToast.wrappedValue = false
                            }
                        } label: {
                            Image(systemName: "doc.on.doc")
                                .font(.system(size: 12))
                        }
                        .buttonStyle(.plain)
                        .help("Copy URL")

                        Button {
                            serverManager.stopServer(server)
                        } label: {
                            Image(systemName: "stop.circle.fill")
                                .font(.system(size: 14))
                                .foregroundColor(.red)
                        }
                        .buttonStyle(.plain)
                        .help("Stop server")
                    } else if server.displayStatus == "stopped" {
                        Button {
                            serverManager.startServer(server)
                        } label: {
                            Image(systemName: "play.circle.fill")
                                .font(.system(size: 14))
                                .foregroundColor(.green)
                        }
                        .buttonStyle(.plain)
                        .help("Start server")
                    }
                }
            }
        }
        .padding(.horizontal, 12)
        .padding(.vertical, 10)
        .background(
            RoundedRectangle(cornerRadius: 6)
                .fill(isHovered ? Color.grovePrimary.opacity(0.08) : Color.clear)
        )
        .contentShape(Rectangle())
        .onHover { hovering in
            withAnimation(.easeInOut(duration: 0.15)) {
                isHovered = hovering
            }
        }
        .contextMenu {
            if server.isRunning {
                Button("Open in Browser") {
                    serverManager.openServer(server)
                }

                Button("Copy URL") {
                    serverManager.copyURL(server)
                    showCopiedToast.wrappedValue = true
                    DispatchQueue.main.asyncAfter(deadline: .now() + 2) {
                        showCopiedToast.wrappedValue = false
                    }
                }
            }

            if server.logFile != nil {
                Button("View Logs") {
                    serverManager.startStreamingLogs(for: server)
                }
            }

            Divider()

            Button("Open in Terminal") {
                serverManager.openInTerminal(server)
            }

            Button("Open in VS Code") {
                serverManager.openInVSCode(server)
            }

            Button("Open in Finder") {
                serverManager.openInFinder(server)
            }

            Button("Copy Path") {
                serverManager.copyPath(server)
            }

            if server.isRunning {
                Divider()

                Button("Stop Server") {
                    serverManager.stopServer(server)
                }
            }
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
    var destructive: Bool = false
    let action: () -> Void
    @State private var isHovered = false

    var body: some View {
        Button(action: action) {
            HStack(spacing: 10) {
                Image(systemName: icon)
                    .font(.system(size: 13))
                    .foregroundColor(destructive ? .red : .primary)
                    .frame(width: 20)
                Text(title)
                    .font(.system(size: 13))
                    .foregroundColor(destructive ? .red : .primary)
                Spacer()
            }
            .padding(.horizontal, 10)
            .padding(.vertical, 6)
            .background(
                RoundedRectangle(cornerRadius: 5)
                    .fill(isHovered ? (destructive ? Color.red.opacity(0.1) : Color.grovePrimary.opacity(0.1)) : Color.clear)
            )
        }
        .buttonStyle(.plain)
        .onHover { hovering in
            withAnimation(.easeInOut(duration: 0.12)) {
                isHovered = hovering
            }
        }
    }
}

// MARK: - Onboarding Step

struct OnboardingStep: View {
    let number: Int
    let text: String

    var body: some View {
        HStack(spacing: 10) {
            Text("\(number)")
                .font(.caption2.bold())
                .foregroundColor(.white)
                .frame(width: 18, height: 18)
                .background(Color.grovePrimary.opacity(0.8))
                .clipShape(Circle())

            Text(text)
                .font(.caption)
                .foregroundColor(.secondary)
        }
    }
}

// MARK: - Keyboard Shortcut Hint

struct KeyboardHint: View {
    let keys: String
    let action: String

    var body: some View {
        HStack(spacing: 4) {
            Text(keys)
                .font(.system(size: 10, design: .monospaced))
                .padding(.horizontal, 4)
                .padding(.vertical, 2)
                .background(Color.secondary.opacity(0.2))
                .cornerRadius(3)

            Text(action)
                .font(.caption2)
                .foregroundColor(.secondary)
        }
    }
}

// Preview disabled for SPM builds
// #Preview {
//     MenuView()
//         .environmentObject(ServerManager())
// }
