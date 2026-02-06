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

    /// Duration in seconds before auto-dismiss
    var duration: TimeInterval {
        switch self {
        case .error: return 5.0
        case .success, .info: return 2.0
        }
    }

    var isError: Bool {
        if case .error = self { return true }
        return false
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
    @ObservedObject private var preferences = PreferencesManager.shared
    @Environment(\.openWindow) private var openWindow
    @Environment(\.openSettings) private var openSettings
    @State private var selectedServerIndex: Int?
    @FocusState private var isFocused: Bool
    @State private var searchText = ""
    @FocusState private var isSearchFocused: Bool
    @State private var showCopiedToast = false
    @State private var eventMonitor: Any?
    @State private var currentToast: ToastType?
    @State private var toastDismissTask: Task<Void, Never>?
    @State private var errorMessages: [String] = []
    @State private var isRefreshing = false
    @State private var isStoppedCollapsed = false
    // Keyboard navigation
    @State private var selectedNavIndex: Int? = nil
    // Sound tracking - previous server statuses for change detection
    @State private var previousServerStatuses: [String: String] = [:]

    var body: some View {
        mainMenuView
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

    // Pinned servers from filtered list
    private var pinnedFilteredServers: [Server] {
        filteredServers.filter { preferences.isServerPinned($0.name) }
    }

    // Non-pinned servers from filtered list
    private var unpinnedFilteredServers: [Server] {
        filteredServers.filter { !preferences.isServerPinned($0.name) }
    }

    // Flat list for keyboard navigation (pinned first, then rest)
    private var navigableServers: [Server] {
        pinnedFilteredServers + unpinnedFilteredServers
    }

    /// Check for server status changes and play sounds
    private func checkSoundEffects() {
        guard preferences.enableSounds else { return }
        for server in serverManager.servers {
            let prevStatus = previousServerStatuses[server.name]
            let currentStatus = server.status
            if let prev = prevStatus, prev != currentStatus {
                if currentStatus == "running" && prev == "starting" {
                    NSSound(named: "Glass")?.play()
                } else if currentStatus == "crashed" {
                    NSSound(named: "Basso")?.play()
                }
            }
            previousServerStatuses[server.name] = currentStatus
        }
    }

    /// Dismiss the current error and show the next one in the queue, if any
    private func dismissCurrentError() {
        if !errorMessages.isEmpty {
            errorMessages.removeFirst()
        }
        if let next = errorMessages.first {
            serverManager.error = next
        } else {
            serverManager.error = nil
        }
    }

    private var mainMenuView: some View {
        VStack(alignment: .leading, spacing: 0) {
            // Header with improved loading indicator
            HStack {
                Text("Grove")
                    .font(.headline)
                    .foregroundColor(.grovePrimary)

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

            // Error banner (shows most recent, with count if multiple)
            if let error = serverManager.error {
                HStack(spacing: 6) {
                    Image(systemName: "exclamationmark.triangle.fill")
                        .foregroundColor(.yellow)
                    Text(error)
                        .font(.caption)
                        .lineLimit(2)

                    if errorMessages.count > 1 {
                        Text("+\(errorMessages.count - 1)")
                            .font(.caption2)
                            .foregroundColor(.yellow)
                            .padding(.horizontal, 4)
                            .padding(.vertical, 1)
                            .background(Color.yellow.opacity(0.2))
                            .cornerRadius(3)
                    }

                    Spacer()
                    Button {
                        dismissCurrentError()
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
                    .font(.system(size: 11))

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
                            .font(.system(size: 11))
                    }
                    .buttonStyle(.plain)
                }
            }
            .padding(.horizontal, 10)
            .padding(.vertical, 5)

            Divider()

            // Compact toolbar
            HStack(spacing: 10) {
                if serverManager.hasRunningServers {
                    Button {
                        serverManager.stopAllServers()
                    } label: {
                        HStack(spacing: 3) {
                            Image(systemName: "stop.fill")
                                .font(.system(size: 10))
                            Text("Stop All")
                                .font(.system(size: 10, weight: .medium))
                        }
                        .foregroundColor(.red.opacity(0.8))
                    }
                    .buttonStyle(.plain)
                    .keyboardShortcut("s", modifiers: [.command, .shift])

                    Button {
                        serverManager.openAllRunningServers()
                    } label: {
                        HStack(spacing: 3) {
                            Image(systemName: "arrow.up.right.square")
                                .font(.system(size: 10))
                            Text("Open All")
                                .font(.system(size: 10, weight: .medium))
                        }
                        .foregroundColor(.secondary)
                    }
                    .buttonStyle(.plain)
                    .keyboardShortcut("o", modifiers: [.command, .shift])
                }

                Spacer()

                Button {
                    serverManager.refresh()
                } label: {
                    Image(systemName: "arrow.clockwise")
                        .font(.system(size: 11))
                        .foregroundColor(.secondary)
                }
                .buttonStyle(.plain)
                .keyboardShortcut("r", modifiers: .command)
                .help("Refresh (⌘R)")

                Button {
                    NSApp.activate(ignoringOtherApps: true)
                    openWindow(id: "log-viewer")
                } label: {
                    Image(systemName: "doc.text.magnifyingglass")
                        .font(.system(size: 11))
                        .foregroundColor(.secondary)
                }
                .buttonStyle(.plain)
                .keyboardShortcut("l", modifiers: .command)
                .help("View Logs (⌘L)")

                Button {
                    serverManager.openTUI()
                } label: {
                    Image(systemName: "terminal.fill")
                        .font(.system(size: 11))
                        .foregroundColor(.secondary)
                }
                .buttonStyle(.plain)
                .help("Open TUI")

                Button {
                    NSApp.activate(ignoringOtherApps: true)
                    openSettings()
                } label: {
                    Image(systemName: "gear")
                        .font(.system(size: 11))
                        .foregroundColor(.secondary)
                }
                .buttonStyle(.plain)
                .help("Settings (⌘,)")
            }
            .padding(.horizontal, 12)
            .padding(.vertical, 4)

            // Servers
            if serverManager.servers.isEmpty {
                // Enhanced empty state with onboarding
                VStack(spacing: 12) {
                    Image(systemName: "tree.fill")
                        .font(.system(size: 36))
                        .foregroundColor(.grovePrimary.opacity(0.6))

                    Text("No servers found")
                        .font(.headline)
                        .foregroundColor(.primary)

                    Text("Get started by running:")
                        .font(.subheadline)
                        .foregroundColor(.secondary)

                    HStack(spacing: 8) {
                        Text("grove discover --register")
                            .font(.system(.caption, design: .monospaced))
                            .padding(.horizontal, 10)
                            .padding(.vertical, 6)
                            .background(Color(NSColor.textBackgroundColor))
                            .cornerRadius(6)

                        Button {
                            NSPasteboard.general.clearContents()
                            NSPasteboard.general.setString("grove discover --register", forType: .string)
                            withAnimation {
                                currentToast = .success("Copied to clipboard")
                                showCopiedToast = true
                            }
                        } label: {
                            Image(systemName: "doc.on.doc")
                                .font(.caption)
                        }
                        .buttonStyle(.bordered)
                        .controlSize(.small)
                        .help("Copy command")
                    }

                    Text("This will find your git worktrees\nand register them with Grove.")
                        .font(.caption)
                        .foregroundColor(.secondary)
                        .multilineTextAlignment(.center)
                        .padding(.top, 4)

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
                        // Pinned servers section
                        let pinned = pinnedFilteredServers
                        if !pinned.isEmpty {
                            SectionHeader(title: "Pinned", count: pinned.count)
                            ForEach(Array(pinned.enumerated()), id: \.element.id) { index, server in
                                let navIdx = index
                                ServerRowView(
                                    server: server,
                                    searchText: searchText,
                                    isPinned: true,
                                    isNavSelected: selectedNavIndex == navIdx
                                )
                            }
                        }

                        // Non-pinned servers
                        let unpinned = unpinnedFilteredServers
                        let pinOffset = pinned.count

                        // Check if servers should be grouped
                        let shouldGroup = ServerGrouper.shouldGroup(unpinned)
                        if shouldGroup {
                            // Show grouped view
                            let groups = ServerGrouper.groupServers(unpinned)
                            ForEach(Array(groups.enumerated()), id: \.element.id) { index, group in
                                ServerGroupView(group: group, searchText: searchText)
                                    .environment(\.groupIndex, index)
                            }

                            // Show "Remove All Stopped" if there are stopped servers in grouped view
                            let stoppedInGroups = unpinned.filter { !$0.isRunning }
                            if !stoppedInGroups.isEmpty {
                                Divider()
                                    .padding(.vertical, 4)
                                HStack {
                                    Text("\(stoppedInGroups.count) stopped")
                                        .font(.caption)
                                        .foregroundColor(.secondary)
                                    Spacer()
                                    Button {
                                        serverManager.removeAllStoppedServers()
                                    } label: {
                                        Text("Remove All")
                                            .font(.caption2)
                                            .foregroundColor(.red)
                                    }
                                    .buttonStyle(.plain)
                                    .help("Remove all stopped servers from Grove")
                                }
                                .padding(.horizontal)
                                .padding(.bottom, 4)
                            }
                        } else {
                            // Show simple running/stopped sections
                            // Running servers
                            let running = unpinned.filter { $0.isRunning }
                            if !running.isEmpty {
                                SectionHeader(title: "Running", count: running.count)
                                ForEach(Array(running.enumerated()), id: \.element.id) { index, server in
                                    let navIdx = pinOffset + index
                                    ServerRowView(
                                        server: server,
                                        searchText: searchText,
                                        displayIndex: index + 1,
                                        isNavSelected: selectedNavIndex == navIdx
                                    )
                                }
                            }

                            // Stopped servers (collapsible)
                            let stopped = unpinned.filter { !$0.isRunning }
                            if !stopped.isEmpty {
                                SectionHeader(
                                    title: "Stopped",
                                    count: stopped.count,
                                    isCollapsible: true,
                                    isCollapsed: $isStoppedCollapsed,
                                    actionLabel: "Remove All",
                                    action: { serverManager.removeAllStoppedServers() }
                                )

                                if !isStoppedCollapsed {
                                    let runningCount = running.count
                                    ForEach(Array(stopped.enumerated()), id: \.element.id) { index, server in
                                        let navIdx = pinOffset + runningCount + index
                                        ServerRowView(
                                            server: server,
                                            searchText: searchText,
                                            displayIndex: runningCount + index + 1,
                                            isNavSelected: selectedNavIndex == navIdx
                                        )
                                    }
                                }
                            }
                        }
                    }
                    .environment(\.showCopiedToast, $showCopiedToast)
                }
                .frame(maxHeight: 450)
            }

            // Proxy status - only show in subdomain mode
            if serverManager.isSubdomainMode {
                Divider()

                ProxyStatusView()
                    .padding(.horizontal)
                    .padding(.vertical, 8)
            }

            // Footer
            Divider()

            HStack {
                // Keyboard hint
                if !serverManager.servers.isEmpty {
                    Text("j/k navigate · o open · ⌘F search")
                        .font(.system(size: 9))
                        .foregroundColor(.secondary.opacity(0.6))
                }

                Spacer()

                Button {
                    NSApplication.shared.terminate(nil)
                } label: {
                    Text("Quit")
                        .font(.system(size: 11))
                        .foregroundColor(.secondary)
                }
                .buttonStyle(.plain)
            }
            .padding(.horizontal, 12)
            .padding(.vertical, 6)
        }
        .frame(width: 340)
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
                .padding(.bottom, 40)
                .transition(.move(edge: .bottom).combined(with: .opacity))
                .animation(.spring(response: 0.3), value: showCopiedToast)
                .onTapGesture {
                    // Allow manual dismiss of any toast
                    withAnimation {
                        currentToast = nil
                        showCopiedToast = false
                    }
                }
            }
        }
        .onChange(of: currentToast?.message) {
            // Schedule auto-dismiss with duration based on toast type
            toastDismissTask?.cancel()
            if let toast = currentToast {
                toastDismissTask = Task {
                    try? await Task.sleep(nanoseconds: UInt64(toast.duration * 1_000_000_000))
                    if !Task.isCancelled {
                        await MainActor.run {
                            withAnimation {
                                currentToast = nil
                            }
                        }
                    }
                }
            }
        }
        .onChange(of: serverManager.error) {
            // Track errors in the queue
            if let error = serverManager.error, !error.isEmpty {
                errorMessages.append(error)
            }
        }
        .onChange(of: serverManager.runningCount) {
            checkSoundEffects()
        }
        .onChange(of: serverManager.hasCrashedServers) {
            checkSoundEffects()
        }
        .onAppear {
            // Initialize previous statuses for sound tracking
            for server in serverManager.servers {
                previousServerStatuses[server.name] = server.status
            }

            // Set up keyboard shortcuts handler (only once)
            guard eventMonitor == nil else { return }
            eventMonitor = NSEvent.addLocalMonitorForEvents(matching: .keyDown) { [self] event in
                // Skip shortcuts when a text field has focus
                let isTextFieldFocused = isSearchFocused || (NSApp.keyWindow?.firstResponder is NSTextView)

                if event.modifierFlags.contains(.command) {
                    if event.charactersIgnoringModifiers == "f" {
                        isSearchFocused = true
                        return nil
                    }
                }

                // j/k/Enter/o navigation (only when search isn't focused)
                if !isTextFieldFocused {
                    let chars = event.charactersIgnoringModifiers ?? ""
                    let serverCount = navigableServers.count

                    switch chars {
                    case "j":
                        // Move down
                        if serverCount > 0 {
                            if let idx = selectedNavIndex {
                                selectedNavIndex = min(idx + 1, serverCount - 1)
                            } else {
                                selectedNavIndex = 0
                            }
                        }
                        return nil
                    case "k":
                        // Move up
                        if serverCount > 0 {
                            if let idx = selectedNavIndex {
                                selectedNavIndex = max(idx - 1, 0)
                            } else {
                                selectedNavIndex = serverCount - 1
                            }
                        }
                        return nil
                    case "o":
                        // Open in browser
                        if let idx = selectedNavIndex, idx < navigableServers.count {
                            let server = navigableServers[idx]
                            if server.isRunning {
                                serverManager.openServer(server)
                            }
                        }
                        return nil
                    default:
                        break
                    }

                    // Enter key
                    if event.keyCode == 36 {
                        if let idx = selectedNavIndex, idx < navigableServers.count {
                            let server = navigableServers[idx]
                            if server.isRunning {
                                serverManager.openServer(server)
                            } else if server.displayStatus == "stopped" && server.hasServer == true {
                                serverManager.startServer(server)
                            }
                        }
                        return nil
                    }

                    // Escape clears selection
                    if event.keyCode == 53 {
                        selectedNavIndex = nil
                        return nil
                    }
                }

                // Number keys 1-9 for quick-start (only when no text field is focused)
                if !isTextFieldFocused,
                   let chars = event.charactersIgnoringModifiers,
                   let num = Int(chars),
                   num >= 1 && num <= 9 {
                    let servers = filteredServers.filter { $0.isRunning || ($0.displayStatus == "stopped" && $0.hasServer == true) }
                    if num <= servers.count {
                        let server = servers[num - 1]
                        if !server.isRunning && server.displayStatus == "stopped" && server.hasServer == true {
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
            selectedNavIndex = nil
        }
    }
}

struct SectionHeader: View {
    let title: String
    let count: Int
    var isCollapsible: Bool = false
    @Binding var isCollapsed: Bool
    var actionLabel: String? = nil
    var action: (() -> Void)? = nil

    init(title: String, count: Int, isCollapsible: Bool = false, isCollapsed: Binding<Bool> = .constant(false), actionLabel: String? = nil, action: (() -> Void)? = nil) {
        self.title = title
        self.count = count
        self.isCollapsible = isCollapsible
        self._isCollapsed = isCollapsed
        self.actionLabel = actionLabel
        self.action = action
    }

    var body: some View {
        HStack {
            HStack(spacing: 6) {
                if title == "Pinned" {
                    Image(systemName: "star.fill")
                        .foregroundColor(.yellow)
                        .font(.system(size: 8))
                } else {
                    Circle()
                        .fill(title == "Running" ? Color.green : Color.gray.opacity(0.5))
                        .frame(width: 6, height: 6)
                }
                Text(title.uppercased())
                    .font(.caption.weight(.medium))
                    .foregroundColor(.secondary)
            }
            Spacer()

            if let actionLabel = actionLabel, let action = action, !isCollapsed {
                Button {
                    action()
                } label: {
                    Text(actionLabel)
                        .font(.caption2)
                        .foregroundColor(.red)
                }
                .buttonStyle(.plain)
                .help("Remove all stopped servers from Grove")
            }

            Text("\(count)")
                .font(.caption.weight(.semibold))
                .foregroundColor(.secondary)
                .padding(.horizontal, 6)
                .padding(.vertical, 2)
                .background(Color.secondary.opacity(0.1))
                .cornerRadius(4)

            if isCollapsible {
                Image(systemName: isCollapsed ? "chevron.right" : "chevron.down")
                    .font(.caption)
                    .foregroundColor(.secondary)
            }
        }
        .padding(.horizontal)
        .padding(.vertical, 6)
        .background(Color(NSColor.windowBackgroundColor).opacity(0.6))
        .contentShape(Rectangle())
        .onTapGesture {
            if isCollapsible {
                withAnimation(.easeInOut(duration: 0.2)) {
                    isCollapsed.toggle()
                }
            }
        }
    }
}

struct ServerRowView: View {
    @EnvironmentObject var serverManager: ServerManager
    @ObservedObject private var preferences = PreferencesManager.shared
    @Environment(\.openWindow) private var openWindow
    let server: Server
    var searchText: String = ""
    var displayIndex: Int?
    var isPinned: Bool = false
    var isNavSelected: Bool = false
    @State private var isExpanded = false
    @State private var isHovered = false
    @Environment(\.showCopiedToast) private var showCopiedToast

    private func ciStatusHelp(_ status: GitHubInfo.CIStatus) -> String {
        switch status {
        case .success: return "CI: Passed"
        case .failure: return "CI: Failed"
        case .pending: return "CI: Running"
        case .unknown: return "CI: Unknown"
        }
    }

    /// Returns the color for the status indicator, considering both server status and health
    private var effectiveStatusColor: Color {
        // If server is running but unhealthy, show orange
        if server.isRunning && serverManager.healthStatus(for: server) == .unhealthy {
            return .orange
        }
        return server.statusColor
    }

    /// Returns an SF Symbol name that conveys status through shape (accessibility for color-blind users)
    private var statusSymbol: String {
        if server.isRunning {
            let health = serverManager.healthStatus(for: server)
            switch health {
            case .healthy:
                return "checkmark.circle.fill"
            case .unhealthy:
                return "xmark.circle.fill"
            case .unknown:
                return "circle.fill"
            }
        }
        switch server.displayStatus {
        case "crashed":
            return "exclamationmark.triangle.fill"
        case "starting":
            return "circle.dotted"
        default:
            return "circle"
        }
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            // Main row content
            HStack(spacing: 8) {
                // Pin indicator
                if isPinned {
                    Image(systemName: "star.fill")
                        .foregroundColor(.yellow)
                        .font(.system(size: 8))
                        .frame(width: 8)
                }

                // Status indicator with shape variation for accessibility
                Image(systemName: statusSymbol)
                    .foregroundColor(effectiveStatusColor)
                    .font(.system(size: 8))
                    .frame(width: 8, height: 8)

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
                        HStack(spacing: 2) {
                            Text(server.displayName)
                                .font(.system(.body, design: .monospaced))
                                .lineLimit(1)

                            if preferences.showPort, let port = server.port, port > 0 {
                                Text(":\(String(port))")
                                    .font(.system(.caption, design: .monospaced))
                                    .foregroundColor(.grovePrimary)
                            }
                        }

                        if preferences.showUptime, let uptime = server.formattedUptime, server.isRunning {
                            Text(uptime)
                                .font(.caption2)
                                .foregroundColor(.secondary)
                                .padding(.horizontal, 4)
                                .padding(.vertical, 1)
                                .background(Color.secondary.opacity(0.1))
                                .cornerRadius(3)
                        }

                        // GitHub badges
                        if preferences.showGitHubInfo, let github = server.githubInfo {
                            if let prNumber = github.prNumber {
                                Button {
                                    if let urlString = github.prURL, let url = URL(string: urlString) {
                                        NSWorkspace.shared.open(url)
                                    }
                                } label: {
                                    HStack(spacing: 3) {
                                        Text("#\(prNumber)")
                                            .font(.caption)
                                        if github.ciStatus != .unknown {
                                            Image(systemName: github.ciStatus.icon)
                                                .font(.system(size: 9))
                                        }
                                    }
                                    .foregroundColor(github.ciStatus == .failure ? .orange : (github.ciStatus == .success ? .green : .blue))
                                    .padding(.horizontal, 5)
                                    .padding(.vertical, 2)
                                    .background(
                                        Capsule()
                                            .fill(github.ciStatus == .failure ? Color.orange.opacity(0.15) : (github.ciStatus == .success ? Color.green.opacity(0.15) : Color.blue.opacity(0.15)))
                                    )
                                }
                                .buttonStyle(.plain)
                                .help("PR #\(prNumber) • \(ciStatusHelp(github.ciStatus))")
                            } else if github.ciStatus != .unknown {
                                HStack(spacing: 3) {
                                    Text("CI")
                                        .font(.system(size: 9, weight: .medium))
                                    Image(systemName: github.ciStatus.icon)
                                        .font(.system(size: 9))
                                }
                                .foregroundColor(github.ciStatus == .failure ? .orange : (github.ciStatus == .success ? .green : .secondary))
                                .padding(.horizontal, 5)
                                .padding(.vertical, 2)
                                .background(
                                    Capsule()
                                        .fill(github.ciStatus == .failure ? Color.orange.opacity(0.15) : (github.ciStatus == .success ? Color.green.opacity(0.15) : Color.secondary.opacity(0.1)))
                                )
                                .help(ciStatusHelp(github.ciStatus))
                            }
                        }

                        // Agent badge
                        if let agent = serverManager.agent(for: server) {
                            Image(systemName: agent.iconName)
                                .font(.system(size: 9))
                                .foregroundColor(.purple)
                                .help("\(agent.displayType) active")
                        }

                        // Git dirty indicator
                        if server.gitDirty == true {
                            Circle()
                                .fill(Color.orange)
                                .frame(width: 5, height: 5)
                                .help("Uncommitted changes")
                        }
                    }

                    // Resource usage hint for running servers
                    if server.isRunning, let resources = serverManager.serverResources[server.name] {
                        Text(resources.summary)
                            .font(.system(size: 9, design: .monospaced))
                            .foregroundColor(.secondary.opacity(0.8))
                    }
                }

                Spacer()

                // Chevron to indicate expandability + quick start/stop
                HStack(spacing: 8) {
                    if !isExpanded {
                        if server.isRunning {
                            Button {
                                serverManager.stopServer(server)
                            } label: {
                                Image(systemName: "stop.circle.fill")
                                    .font(.system(size: 14))
                                    .foregroundColor(.red)
                            }
                            .buttonStyle(.plain)
                            .help("Stop server")
                        } else if server.displayStatus == "stopped" && server.hasServer == true {
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

                    Image(systemName: "chevron.right")
                        .font(.system(size: 10, weight: .semibold))
                        .foregroundColor(.secondary)
                        .rotationEffect(.degrees(isExpanded ? 90 : 0))
                }
            }
            .padding(.horizontal, 12)
            .padding(.vertical, 10)
            .contentShape(Rectangle())
            .onTapGesture {
                withAnimation(.easeInOut(duration: 0.2)) {
                    isExpanded.toggle()
                }
            }

            // Expanded action bar
            if isExpanded {
                HStack(spacing: 12) {
                    // Primary action
                    if server.isRunning {
                        ActionChip(icon: "arrow.up.right.square", label: "Open") {
                            serverManager.openServer(server)
                        }

                        ActionChip(icon: "stop.fill", label: "Stop", destructive: true) {
                            serverManager.stopServer(server)
                        }
                    } else if server.displayStatus == "stopped" && server.hasServer == true {
                        ActionChip(icon: "play.fill", label: "Start", primary: true) {
                            serverManager.startServer(server)
                        }
                    }

                    if server.logFile != nil {
                        ActionChip(icon: "doc.text", label: "Logs") {
                            serverManager.startStreamingLogs(for: server)
                            NSApp.activate(ignoringOtherApps: true)
                            openWindow(id: "log-viewer")
                        }
                    }

                    Spacer()

                    // More menu for less common actions
                    Menu {
                        Button {
                            serverManager.openInTerminal(server)
                        } label: {
                            Label("Open Terminal", systemImage: "terminal")
                        }

                        Button {
                            serverManager.openInFinder(server)
                        } label: {
                            Label("Open in Finder", systemImage: "folder")
                        }

                        Button {
                            serverManager.openInVSCode(server)
                        } label: {
                            Label("Open in VS Code", systemImage: "chevron.left.forwardslash.chevron.right")
                        }

                        Divider()

                        if server.isRunning {
                            Button {
                                serverManager.copyURL(server)
                            } label: {
                                Label("Copy URL", systemImage: "link")
                            }
                        }

                        Button {
                            serverManager.copyPath(server)
                        } label: {
                            Label("Copy Path", systemImage: "doc.on.doc")
                        }

                        Divider()

                        Button(role: .destructive) {
                            serverManager.detachServer(server)
                        } label: {
                            Label("Remove from Grove", systemImage: "xmark.circle")
                        }
                    } label: {
                        Image(systemName: "ellipsis.circle")
                            .font(.system(size: 14))
                            .foregroundColor(.secondary)
                    }
                    .menuIndicator(.hidden)
                    .fixedSize()
                }
                .padding(.horizontal, 12)
                .padding(.bottom, 8)

                // Resource usage (running servers)
                if server.isRunning, let resources = serverManager.serverResources[server.name] {
                    HStack(spacing: 14) {
                        HStack(spacing: 4) {
                            Image(systemName: "cpu")
                                .font(.system(size: 9))
                                .foregroundColor(.secondary)
                            Text(resources.cpuDisplay)
                                .font(.system(size: 10, design: .monospaced))
                        }
                        HStack(spacing: 4) {
                            Image(systemName: "memorychip")
                                .font(.system(size: 9))
                                .foregroundColor(.secondary)
                            Text(resources.memoryDisplay)
                                .font(.system(size: 10, design: .monospaced))
                        }
                    }
                    .padding(.horizontal, 12)
                    .padding(.bottom, 6)
                }

                // Rich GitHub details
                if let github = server.githubInfo, github.prNumber != nil {
                    HStack(spacing: 8) {
                        if let review = github.reviewStatus {
                            HStack(spacing: 3) {
                                Image(systemName: review.icon)
                                    .font(.system(size: 9))
                                Text(review.label)
                                    .font(.system(size: 10))
                            }
                            .foregroundColor(review.color)
                        }

                        if github.commentCount > 0 {
                            HStack(spacing: 3) {
                                Image(systemName: "bubble.left")
                                    .font(.system(size: 9))
                                Text("\(github.commentCount)")
                                    .font(.system(size: 10))
                            }
                            .foregroundColor(.secondary)
                        }

                        if github.hasMergeConflicts {
                            HStack(spacing: 3) {
                                Image(systemName: "exclamationmark.triangle.fill")
                                    .font(.system(size: 9))
                                Text("Conflicts")
                                    .font(.system(size: 10))
                            }
                            .foregroundColor(.red)
                        }
                    }
                    .padding(.horizontal, 12)
                    .padding(.bottom, 6)
                }

                // Active agent in this worktree
                if let agent = serverManager.agent(for: server) {
                    HStack(spacing: 8) {
                        Image(systemName: agent.iconName)
                            .font(.system(size: 10))
                            .foregroundColor(.purple)

                        VStack(alignment: .leading, spacing: 1) {
                            Text(agent.displayType)
                                .font(.system(size: 10, weight: .medium))
                            if let task = agent.shortTaskDisplay {
                                Text(task)
                                    .font(.system(size: 9, design: .monospaced))
                                    .foregroundColor(.orange)
                                    .lineLimit(1)
                            }
                        }

                        if let duration = agent.duration {
                            Text(duration)
                                .font(.system(size: 9, design: .monospaced))
                                .foregroundColor(.secondary)
                        }

                        Spacer()

                        // Only show "Open" for agents with launchable editors (e.g. Cursor)
                        if agent.type == "cursor" {
                            Button {
                                serverManager.openAgent(agent)
                            } label: {
                                HStack(spacing: 3) {
                                    Image(systemName: "arrow.up.forward.app")
                                        .font(.system(size: 9))
                                    Text("Open")
                                        .font(.system(size: 10))
                                }
                                .foregroundColor(.purple)
                            }
                            .buttonStyle(.plain)
                        }
                    }
                    .padding(.horizontal, 12)
                    .padding(.vertical, 6)
                    .background(Color.purple.opacity(0.06))
                    .cornerRadius(4)
                    .padding(.horizontal, 8)
                    .padding(.bottom, 8)
                }
            }
        }
        .background(
            RoundedRectangle(cornerRadius: 6)
                .fill(
                    isNavSelected ? Color.grovePrimary.opacity(0.15) :
                    (isExpanded ? Color.grovePrimary.opacity(0.08) :
                    (isHovered ? Color.grovePrimary.opacity(0.04) : Color.clear))
                )
        )
        .overlay(
            RoundedRectangle(cornerRadius: 6)
                .strokeBorder(isNavSelected ? Color.grovePrimary.opacity(0.4) : Color.clear, lineWidth: 1)
        )
        .onHover { hovering in
            withAnimation(.easeInOut(duration: 0.15)) {
                isHovered = hovering
            }
        }
        .contextMenu {
            // Pin/Unpin toggle
            Button {
                preferences.togglePinned(server.name)
            } label: {
                Label(
                    preferences.isServerPinned(server.name) ? "Unpin" : "Pin",
                    systemImage: preferences.isServerPinned(server.name) ? "star.slash" : "star"
                )
            }

            Divider()

            if server.isRunning {
                Button {
                    serverManager.openServer(server)
                } label: {
                    Label("Open in Browser", systemImage: "arrow.up.right.square")
                }

                Button {
                    serverManager.copyURL(server)
                } label: {
                    Label("Copy URL", systemImage: "link")
                }
            }

            if server.logFile != nil {
                Button {
                    serverManager.startStreamingLogs(for: server)
                    NSApp.activate(ignoringOtherApps: true)
                    openWindow(id: "log-viewer")
                } label: {
                    Label("View Logs", systemImage: "doc.text")
                }
            }

            Divider()

            Button {
                serverManager.openInTerminal(server)
            } label: {
                Label("Open in Terminal", systemImage: "terminal")
            }

            Button {
                serverManager.openInVSCode(server)
            } label: {
                Label("Open in VS Code", systemImage: "chevron.left.forwardslash.chevron.right")
            }

            Button {
                serverManager.openInFinder(server)
            } label: {
                Label("Open in Finder", systemImage: "folder")
            }

            Button {
                serverManager.copyPath(server)
            } label: {
                Label("Copy Path", systemImage: "doc.on.doc")
            }

            if server.isRunning {
                Divider()

                Button(role: .destructive) {
                    serverManager.stopServer(server)
                } label: {
                    Label("Stop Server", systemImage: "stop.fill")
                }
            }
        }
    }
}

struct ProxyStatusView: View {
    @EnvironmentObject var serverManager: ServerManager

    var body: some View {
        HStack {
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
        }
    }
}

struct ActionButton: View {
    let title: String
    let icon: String
    var shortcut: String? = nil
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
                if let shortcut = shortcut {
                    Text(shortcut)
                        .font(.system(size: 11, design: .monospaced))
                        .foregroundColor(.secondary)
                }
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

// MARK: - Action Chip

struct ActionChip: View {
    let icon: String
    let label: String
    var primary: Bool = false
    var destructive: Bool = false
    let action: () -> Void

    var body: some View {
        Button(action: action) {
            HStack(spacing: 4) {
                Image(systemName: icon)
                    .font(.system(size: 10))
                Text(label)
                    .font(.caption2)
            }
            .padding(.horizontal, 8)
            .padding(.vertical, 4)
            .background(
                RoundedRectangle(cornerRadius: 4)
                    .fill(backgroundColor)
            )
            .foregroundColor(foregroundColor)
        }
        .buttonStyle(.plain)
    }

    private var backgroundColor: Color {
        if primary {
            return Color.green.opacity(0.15)
        } else if destructive {
            return Color.red.opacity(0.15)
        } else {
            return Color.secondary.opacity(0.1)
        }
    }

    private var foregroundColor: Color {
        if primary {
            return .green
        } else if destructive {
            return .red
        } else {
            return .primary
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

// MARK: - Server Detail Popover

struct ServerDetailPopover: View {
    let server: Server
    let onOpenTerminal: () -> Void
    let onOpenVSCode: () -> Void
    let onOpenBrowser: () -> Void
    @Environment(\.dismiss) private var dismiss

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            // Header with full name
            HStack {
                Image(systemName: server.isRunning ? "checkmark.circle.fill" : (server.displayStatus == "crashed" ? "exclamationmark.triangle.fill" : "circle"))
                    .foregroundColor(server.statusColor)
                    .font(.system(size: 10))

                Text(server.displayName)
                    .font(.system(.headline, design: .monospaced))
                    .textSelection(.enabled)

                Spacer()

                if server.isRunning {
                    Text(server.displayStatus.uppercased())
                        .font(.caption2)
                        .fontWeight(.semibold)
                        .foregroundColor(.green)
                        .padding(.horizontal, 6)
                        .padding(.vertical, 2)
                        .background(Color.green.opacity(0.15))
                        .cornerRadius(4)
                }
            }

            Divider()

            // Details
            VStack(alignment: .leading, spacing: 8) {
                if let branch = server.branch {
                    DetailRow(icon: "arrow.triangle.branch", label: "Branch", value: branch)
                }

                DetailRow(icon: "folder", label: "Path", value: server.path, selectable: true)

                if let port = server.port, port > 0 {
                    DetailRow(icon: "network", label: "Port", value: ":\(port)")
                }

                if let url = server.url {
                    DetailRow(icon: "link", label: "URL", value: url, selectable: true)
                }

                if let uptime = server.formattedUptime {
                    DetailRow(icon: "clock", label: "Uptime", value: uptime)
                }
            }

            Divider()

            // Quick Actions
            HStack(spacing: 8) {
                Button {
                    onOpenTerminal()
                    dismiss()
                } label: {
                    Label("Terminal", systemImage: "terminal")
                        .font(.caption)
                }
                .buttonStyle(.bordered)
                .controlSize(.small)

                Button {
                    onOpenVSCode()
                    dismiss()
                } label: {
                    Label("VS Code", systemImage: "chevron.left.forwardslash.chevron.right")
                        .font(.caption)
                }
                .buttonStyle(.bordered)
                .controlSize(.small)

                if server.isRunning {
                    Button {
                        onOpenBrowser()
                        dismiss()
                    } label: {
                        Label("Open", systemImage: "arrow.up.right.square")
                            .font(.caption)
                    }
                    .buttonStyle(.borderedProminent)
                    .controlSize(.small)
                }
            }
        }
        .padding()
        .frame(width: 320)
    }
}

struct DetailRow: View {
    let icon: String
    let label: String
    let value: String
    var selectable: Bool = false

    var body: some View {
        HStack(alignment: .top, spacing: 8) {
            Image(systemName: icon)
                .font(.caption)
                .foregroundColor(.secondary)
                .frame(width: 16)

            Text(label)
                .font(.caption)
                .foregroundColor(.secondary)
                .frame(width: 50, alignment: .leading)

            if selectable {
                Text(value)
                    .font(.system(.caption, design: .monospaced))
                    .textSelection(.enabled)
                    .lineLimit(2)
                    .truncationMode(.middle)
            } else {
                Text(value)
                    .font(.system(.caption, design: .monospaced))
                    .lineLimit(1)
            }
        }
    }
}

// Preview disabled for SPM builds
// #Preview {
//     MenuView()
//         .environmentObject(ServerManager())
// }
