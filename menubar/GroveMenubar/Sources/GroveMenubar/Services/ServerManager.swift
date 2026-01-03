import Foundation
import SwiftUI
import Combine

class ServerManager: ObservableObject {
    @Published var servers: [Server] = []
    @Published var proxy: ProxyInfo?
    @Published var urlMode: String = "port"
    @Published var isLoading = false
    @Published var error: String?
    @Published var selectedServerForLogs: Server?
    @Published var logLines: [String] = []
    @Published var isStreamingLogs = false
    @Published var serverHealth: [String: HealthStatus] = [:]  // Track health per server

    enum HealthStatus: String {
        case healthy
        case unhealthy
        case unknown
    }

    private var refreshTimer: Timer?
    private var logTimer: Timer?
    private var lastLogPosition: UInt64 = 0
    private let grovePath: String
    private var previousServerStates: [String: String] = [:]  // Track previous server statuses
    private let githubService = GitHubService.shared
    private let preferences = PreferencesManager.shared
    private var isGitHubFetchInProgress = false  // Prevent overlapping GitHub fetches
    private var isHealthCheckInProgress = false  // Prevent overlapping health checks

    // Wake-from-sleep handling - start with cooldown ON to handle fresh app starts after wake
    private var isWakeCooldown = true
    private var wakeCooldownWorkItem: DispatchWorkItem?
    private var sleepObserver: NSObjectProtocol?
    private var wakeObserver: NSObjectProtocol?

    var isPortMode: Bool { urlMode == "port" }
    var isSubdomainMode: Bool { urlMode == "subdomain" }

    init() {
        let initStart = CFAbsoluteTimeGetCurrent()
        print("[Grove] init() START - thread: \(Thread.isMainThread ? "MAIN" : "bg")")

        // Find grove binary synchronously from known paths (fast, no process spawn)
        // We avoid running `which` here to prevent blocking the main thread
        let findStart = CFAbsoluteTimeGetCurrent()
        self.grovePath = Self.findGroveBinaryFast() ?? "/usr/local/bin/grove"
        print("[Grove] findGroveBinaryFast took \(String(format: "%.3f", CFAbsoluteTimeGetCurrent() - findStart))s -> \(grovePath)")

        // Register for sleep/wake notifications to handle network availability
        setupSleepWakeObservers()
        print("[Grove] setupSleepWakeObservers done")

        // Start with cooldown enabled - handles fresh app start after wake
        // Clear cooldown after 3 seconds when network should be ready
        let workItem = DispatchWorkItem { [weak self] in
            print("[Grove] Cooldown ended, triggering refresh")
            self?.isWakeCooldown = false
            // Trigger a refresh to fetch GitHub info now that network should be ready
            self?.refresh()
        }
        wakeCooldownWorkItem = workItem
        DispatchQueue.main.asyncAfter(deadline: .now() + 3.0, execute: workItem)

        // Delay initial operations slightly to let system stabilize after launch/wake
        // This prevents blocking the main thread during the critical app startup period
        DispatchQueue.main.asyncAfter(deadline: .now() + 0.3) { [weak self] in
            print("[Grove] Initial refresh starting (0.3s after init)")
            // Run cleanup first to remove stale entries (non-existent paths)
            self?.runCleanup()
            self?.refresh()
            self?.startAutoRefresh()
        }

        print("[Grove] init() END - took \(String(format: "%.3f", CFAbsoluteTimeGetCurrent() - initStart))s")
    }

    deinit {
        refreshTimer?.invalidate()
        logTimer?.invalidate()
        wakeCooldownWorkItem?.cancel()

        // Remove sleep/wake observers
        if let observer = sleepObserver {
            NSWorkspace.shared.notificationCenter.removeObserver(observer)
        }
        if let observer = wakeObserver {
            NSWorkspace.shared.notificationCenter.removeObserver(observer)
        }
    }

    // MARK: - Sleep/Wake Handling

    private func setupSleepWakeObservers() {
        let workspace = NSWorkspace.shared

        // When system is about to sleep, cancel any pending operations
        sleepObserver = workspace.notificationCenter.addObserver(
            forName: NSWorkspace.willSleepNotification,
            object: nil,
            queue: .main
        ) { [weak self] _ in
            self?.handleSystemWillSleep()
        }

        // When system wakes, wait before doing network operations
        wakeObserver = workspace.notificationCenter.addObserver(
            forName: NSWorkspace.didWakeNotification,
            object: nil,
            queue: .main
        ) { [weak self] _ in
            self?.handleSystemDidWake()
        }
    }

    private func handleSystemWillSleep() {
        // Cancel any pending wake cooldown
        wakeCooldownWorkItem?.cancel()

        // Mark that we should skip network operations
        isWakeCooldown = true
        isGitHubFetchInProgress = false
    }

    private func handleSystemDidWake() {
        // Cancel any existing cooldown
        wakeCooldownWorkItem?.cancel()

        // Start cooldown period - skip GitHub fetches for 3 seconds
        isWakeCooldown = true

        // Schedule end of cooldown
        let workItem = DispatchWorkItem { [weak self] in
            self?.isWakeCooldown = false
            // Do a fresh refresh now that network should be ready
            self?.refresh()
        }
        wakeCooldownWorkItem = workItem
        DispatchQueue.main.asyncAfter(deadline: .now() + 3.0, execute: workItem)
    }

    // MARK: - Status

    var statusIcon: String {
        return "bolt.fill"
    }

    var statusColor: Color {
        if servers.contains(where: { $0.status == "crashed" }) {
            return .red
        }
        if servers.contains(where: { $0.status == "starting" }) {
            return .yellow
        }
        if servers.contains(where: { $0.isRunning }) {
            return .green
        }
        return .gray
    }

    var runningCount: Int {
        servers.filter { $0.isRunning }.count
    }

    var hasRunningServers: Bool {
        servers.contains(where: { $0.isRunning })
    }

    var hasCrashedServers: Bool {
        servers.contains(where: { $0.status == "crashed" })
    }

    var hasStartingServers: Bool {
        servers.contains(where: { $0.status == "starting" })
    }

    var hasUnhealthyServers: Bool {
        serverHealth.values.contains(.unhealthy)
    }

    func healthStatus(for server: Server) -> HealthStatus {
        serverHealth[server.name] ?? .unknown
    }

    // MARK: - Actions

    func refresh() {
        let refreshStart = CFAbsoluteTimeGetCurrent()
        print("[Grove] refresh() START - thread: \(Thread.isMainThread ? "MAIN" : "bg"), cooldown: \(isWakeCooldown)")

        isLoading = true
        error = nil

        print("[Grove] refresh() calling runGrove...")
        runGrove(["ls", "--json", "--fast"]) { [weak self] result in
            print("[Grove] refresh() runGrove completed in \(String(format: "%.3f", CFAbsoluteTimeGetCurrent() - refreshStart))s - thread: \(Thread.isMainThread ? "MAIN" : "bg")")

            switch result {
            case .success(let output):
                let parseStart = CFAbsoluteTimeGetCurrent()
                guard let data = output.data(using: .utf8),
                      let status = try? JSONDecoder().decode(WTStatus.self, from: data) else {
                    print("[Grove] refresh() JSON parse FAILED")
                    DispatchQueue.main.async {
                        self?.isLoading = false
                    }
                    return
                }
                print("[Grove] refresh() JSON parsed in \(String(format: "%.3f", CFAbsoluteTimeGetCurrent() - parseStart))s, \(status.servers.count) servers")

                let newServers = status.servers.sorted { $0.name < $1.name }

                print("[Grove] refresh() dispatching to main...")
                DispatchQueue.main.async {
                    guard let self = self else { return }
                    print("[Grove] refresh() on main thread, updating UI...")

                    self.checkForStatusChanges(newServers: newServers)
                    self.cleanupRemovedServers(currentServers: newServers)

                    self.servers = newServers
                    self.proxy = status.proxy
                    self.urlMode = status.urlMode
                    self.isLoading = false

                    print("[Grove] refresh() UI updated, cooldown=\(self.isWakeCooldown), calling fetchGitHubInfoForServers...")
                    self.fetchGitHubInfoForServers()
                    self.checkServerHealth()
                    print("[Grove] refresh() DONE - total \(String(format: "%.3f", CFAbsoluteTimeGetCurrent() - refreshStart))s")
                }

            case .failure(let err):
                print("[Grove] refresh() FAILED: \(err.localizedDescription)")
                DispatchQueue.main.async {
                    self?.isLoading = false
                    self?.error = err.localizedDescription
                }
            }
        }
    }

    func stopServer(_ server: Server) {
        runGrove(["stop", server.name]) { [weak self] _ in
            DispatchQueue.main.async {
                self?.refresh()
            }
        }
    }

    func stopAllServers() {
        let runningServers = servers.filter { $0.isRunning }
        guard !runningServers.isEmpty else { return }

        for server in runningServers {
            runGrove(["stop", server.name]) { _ in }
        }

        // Refresh after a short delay to allow all stops to complete
        DispatchQueue.main.asyncAfter(deadline: .now() + 1.0) { [weak self] in
            self?.refresh()
        }
    }

    func startServer(_ server: Server) {
        // grove start needs to run from within the worktree directory
        runGroveInDirectory(server.path, args: ["start"]) { [weak self] result in
            DispatchQueue.main.async {
                switch result {
                case .success:
                    self?.refresh()
                case .failure(let error):
                    self?.error = "Failed to start \(server.name): \(error.localizedDescription)"
                    self?.refresh()
                }
            }
        }
    }

    // MARK: - Group Actions

    func startAllInGroup(_ group: ServerGroup) {
        let stoppedServers = group.servers.filter { !$0.isRunning }
        guard !stoppedServers.isEmpty else { return }

        for server in stoppedServers {
            // grove start needs to run from within the worktree directory
            runGroveInDirectory(server.path, args: ["start"]) { _ in }
        }

        // Refresh after a short delay
        DispatchQueue.main.asyncAfter(deadline: .now() + 1.0) { [weak self] in
            self?.refresh()
        }
    }

    func stopAllInGroup(_ group: ServerGroup) {
        let runningServers = group.servers.filter { $0.isRunning }
        guard !runningServers.isEmpty else { return }

        for server in runningServers {
            runGrove(["stop", server.name]) { _ in }
        }

        // Refresh after a short delay
        DispatchQueue.main.asyncAfter(deadline: .now() + 1.0) { [weak self] in
            self?.refresh()
        }
    }

    func openServer(_ server: Server) {
        if let url = URL(string: server.displayURL) {
            preferences.openURL(url)
        }
    }

    func copyURL(_ server: Server) {
        NSPasteboard.general.clearContents()
        NSPasteboard.general.setString(server.displayURL, forType: .string)
    }

    func openAllRunningServers() {
        let runningServers = servers.filter { $0.isRunning }
        for server in runningServers {
            if let url = URL(string: server.displayURL) {
                preferences.openURL(url)
            }
        }
    }

    // MARK: - Quick Navigation

    func openInTerminal(_ server: Server) {
        PreferencesManager.shared.openInTerminal(path: server.path)
    }

    func openInVSCode(_ server: Server) {
        let task = Process()
        task.executableURL = URL(fileURLWithPath: "/usr/bin/env")
        task.arguments = ["code", server.path]

        let pipe = Pipe()
        task.standardOutput = pipe
        task.standardError = pipe

        do {
            try task.run()
        } catch {
            // If VS Code command fails, try opening with 'open' command
            let openTask = Process()
            openTask.executableURL = URL(fileURLWithPath: "/usr/bin/open")
            openTask.arguments = ["-a", "Visual Studio Code", server.path]
            try? openTask.run()
        }
    }

    func openInFinder(_ server: Server) {
        NSWorkspace.shared.selectFile(nil, inFileViewerRootedAtPath: server.path)
    }

    func copyPath(_ server: Server) {
        NSPasteboard.general.clearContents()
        NSPasteboard.general.setString(server.path, forType: .string)
    }

    func detachServer(_ server: Server) {
        runGrove(["detach", server.name]) { [weak self] _ in
            DispatchQueue.main.async {
                self?.refresh()
            }
        }
    }

    func removeAllStoppedServers() {
        let stoppedServers = servers.filter { !$0.isRunning }
        guard !stoppedServers.isEmpty else { return }

        // Pause auto-refresh during batch operation
        refreshTimer?.invalidate()
        refreshTimer = nil

        let group = DispatchGroup()

        for server in stoppedServers {
            group.enter()
            runGrove(["detach", server.name]) { _ in
                group.leave()
            }
        }

        // Refresh once all detaches complete, then restart auto-refresh
        group.notify(queue: .main) { [weak self] in
            self?.refresh()
            self?.startAutoRefresh()
        }
    }

    /// Remove all servers in a group from Grove (detach them)
    func removeAllServersInGroup(_ serverGroup: ServerGroup) {
        guard !serverGroup.servers.isEmpty else { return }

        // Pause auto-refresh during batch operation
        refreshTimer?.invalidate()
        refreshTimer = nil

        let group = DispatchGroup()

        for server in serverGroup.servers {
            group.enter()
            runGrove(["detach", server.name]) { _ in
                group.leave()
            }
        }

        // Refresh once all detaches complete, then restart auto-refresh
        group.notify(queue: .main) { [weak self] in
            self?.refresh()
            self?.startAutoRefresh()
        }
    }

    /// Stop all running servers in a group
    func stopAllServersInGroup(_ serverGroup: ServerGroup) {
        let runningServers = serverGroup.servers.filter { $0.isRunning }
        guard !runningServers.isEmpty else { return }

        // Pause auto-refresh during batch operation
        refreshTimer?.invalidate()
        refreshTimer = nil

        let group = DispatchGroup()

        for server in runningServers {
            group.enter()
            runGrove(["stop", server.name]) { _ in
                group.leave()
            }
        }

        // Refresh once all stops complete, then restart auto-refresh
        group.notify(queue: .main) { [weak self] in
            self?.refresh()
            self?.startAutoRefresh()
        }
    }

    func startProxy() {
        runGrove(["proxy", "start"]) { [weak self] _ in
            DispatchQueue.main.async {
                self?.refresh()
            }
        }
    }

    func stopProxy() {
        runGrove(["proxy", "stop"]) { [weak self] _ in
            DispatchQueue.main.async {
                self?.refresh()
            }
        }
    }

    /// Run cleanup to remove stale entries (non-existent paths, dead processes)
    /// This runs silently in the background
    private func runCleanup() {
        runGrove(["cleanup"]) { result in
            switch result {
            case .success(let output):
                if !output.contains("No stale entries") {
                    print("[Grove] Cleanup: \(output.trimmingCharacters(in: .whitespacesAndNewlines))")
                }
            case .failure(let error):
                print("[Grove] Cleanup failed: \(error.localizedDescription)")
            }
        }
    }

    func openTUI() {
        // Open configured terminal with grove TUI command
        let groveDir = (grovePath as NSString).deletingLastPathComponent
        let grovePath = self.grovePath
        let terminal = preferences.defaultTerminal

        // Run on background thread to prevent blocking main thread
        DispatchQueue.global(qos: .userInitiated).async {
            switch terminal {
            case "com.apple.Terminal":
                let script = """
                tell application "Terminal"
                    activate
                    do script "cd '\(groveDir)' && \(grovePath)"
                end tell
                """
                Self.runAppleScriptAsync(script)
            case "com.googlecode.iterm2":
                let script = """
                tell application "iTerm"
                    activate
                    try
                        set newWindow to (create window with default profile)
                        tell current session of newWindow
                            write text "cd '\(groveDir)' && \(grovePath)"
                        end tell
                    on error
                        tell current window
                            create tab with default profile
                            tell current session
                                write text "cd '\(groveDir)' && \(grovePath)"
                            end tell
                        end tell
                    end try
                end tell
                """
                Self.runAppleScriptAsync(script)
            case "com.mitchellh.ghostty":
                // Try using the Ghostty CLI directly
                let ghosttyPaths = [
                    "/Applications/Ghostty.app/Contents/MacOS/ghostty",
                    "/opt/homebrew/bin/ghostty",
                    "\(NSHomeDirectory())/.local/bin/ghostty"
                ]

                var launched = false
                for ghosttyPath in ghosttyPaths {
                    if FileManager.default.fileExists(atPath: ghosttyPath) {
                        let task = Process()
                        task.executableURL = URL(fileURLWithPath: ghosttyPath)
                        task.arguments = ["--working-directory=\(groveDir)", "-e", grovePath]
                        if (try? task.run()) != nil {
                            launched = true
                            break
                        }
                    }
                }

                // Fallback: open Ghostty and send command via AppleScript
                if !launched {
                    let script = """
                    tell application "Ghostty"
                        activate
                    end tell
                    delay 0.5
                    tell application "System Events"
                        keystroke "cd '\(groveDir)' && \(grovePath)"
                        keystroke return
                    end tell
                    """
                    Self.runAppleScriptAsync(script)
                }
            case "dev.warp.Warp-Stable":
                // Warp: open the directory, then run the command
                let script = """
                tell application "Warp"
                    activate
                end tell
                delay 0.3
                tell application "System Events"
                    keystroke "cd '\(groveDir)' && \(grovePath)"
                    keystroke return
                end tell
                """
                Self.runAppleScriptAsync(script)
            default:
                // Fallback to Terminal.app
                let script = """
                tell application "Terminal"
                    activate
                    do script "cd '\(groveDir)' && \(grovePath)"
                end tell
                """
                Self.runAppleScriptAsync(script)
            }
        }
    }

    /// Run AppleScript on background thread - never blocks main thread
    private static func runAppleScriptAsync(_ script: String) {
        if let appleScript = NSAppleScript(source: script) {
            var error: NSDictionary?
            appleScript.executeAndReturnError(&error)
            if let error = error {
                print("AppleScript error: \(error)")
            }
        }
    }

    // MARK: - Logs

    func startStreamingLogs(for server: Server) {
        stopStreamingLogs()

        selectedServerForLogs = server
        logLines = []
        lastLogPosition = 0
        isStreamingLogs = true

        // Initial load of last 100 lines
        loadInitialLogs(for: server)

        // Start streaming new lines
        logTimer = Timer.scheduledTimer(withTimeInterval: 0.5, repeats: true) { [weak self] _ in
            self?.streamNewLogs()
        }
    }

    func stopStreamingLogs() {
        logTimer?.invalidate()
        logTimer = nil
        isStreamingLogs = false
        selectedServerForLogs = nil
        logLines = []
        lastLogPosition = 0
    }

    private func loadInitialLogs(for server: Server) {
        guard let logFile = server.logFile else {
            logLines = ["No log file configured for this server"]
            return
        }

        DispatchQueue.global(qos: .userInitiated).async { [weak self] in
            guard let self = self else { return }

            do {
                let url = URL(fileURLWithPath: logFile)
                let data = try Data(contentsOf: url)
                let content = String(data: data, encoding: .utf8) ?? ""
                let lines = content.components(separatedBy: .newlines)
                    .map { ANSIStripper.strip($0) }  // Strip ANSI escape codes

                // Keep last 100 lines
                let recentLines = Array(lines.suffix(100))

                // Get file size for streaming position
                let attributes = try FileManager.default.attributesOfItem(atPath: logFile)
                let fileSize = attributes[.size] as? UInt64 ?? 0

                DispatchQueue.main.async {
                    self.logLines = recentLines.filter { !$0.isEmpty }
                    self.lastLogPosition = fileSize
                }
            } catch {
                DispatchQueue.main.async {
                    self.logLines = ["Error reading log file: \(error.localizedDescription)"]
                }
            }
        }
    }

    private func streamNewLogs() {
        guard let server = selectedServerForLogs,
              let logFile = server.logFile else { return }

        let serverPath = server.path

        // Do ALL file operations on background thread
        DispatchQueue.global(qos: .userInitiated).async { [weak self] in
            guard let self = self else { return }

            // Check if the server's path still exists (worktree might have been deleted)
            guard FileManager.default.fileExists(atPath: serverPath) else {
                DispatchQueue.main.async {
                    self.logLines.append("[Server path no longer exists - worktree may have been deleted]")
                    self.stopStreamingLogs()
                }
                return
            }

            // Check if log file still exists
            guard FileManager.default.fileExists(atPath: logFile) else {
                return // Log file doesn't exist yet, keep waiting
            }

            do {
                let attributes = try FileManager.default.attributesOfItem(atPath: logFile)
                let fileSize = attributes[.size] as? UInt64 ?? 0

                // Check if file has new content
                guard fileSize > self.lastLogPosition else { return }

                // Read new content
                let handle = try FileHandle(forReadingFrom: URL(fileURLWithPath: logFile))
                try handle.seek(toOffset: self.lastLogPosition)
                let newData = handle.readDataToEndOfFile()
                try handle.close()

                if let newContent = String(data: newData, encoding: .utf8) {
                    let newLines = newContent.components(separatedBy: .newlines)
                        .filter { !$0.isEmpty }
                        .map { ANSIStripper.strip($0) }  // Strip ANSI escape codes

                    DispatchQueue.main.async {
                        self.logLines.append(contentsOf: newLines)
                        // Keep only last 500 lines to prevent memory issues
                        if self.logLines.count > 500 {
                            self.logLines = Array(self.logLines.suffix(500))
                        }
                        self.lastLogPosition = fileSize
                    }
                }
            } catch {
                // Silently ignore read errors during streaming
            }
        }
    }

    func clearLogs() {
        logLines = []
    }

    func openLogsInFinder(_ server: Server) {
        if let logFile = server.logFile {
            NSWorkspace.shared.selectFile(logFile, inFileViewerRootedAtPath: "")
        }
    }

    // MARK: - Health Checking

    /// Performs HTTP health checks on all running servers
    private func checkServerHealth() {
        // Skip during wake cooldown
        guard !isWakeCooldown else { return }

        // Prevent overlapping checks
        guard !isHealthCheckInProgress else { return }

        let runningServers = servers.filter { $0.isRunning }
        guard !runningServers.isEmpty else { return }

        isHealthCheckInProgress = true

        // Safety timeout
        DispatchQueue.main.asyncAfter(deadline: .now() + 15) { [weak self] in
            self?.isHealthCheckInProgress = false
        }

        let healthLock = NSLock()
        var healthUpdates: [String: HealthStatus] = [:]
        var completedCount = 0
        let totalCount = runningServers.count

        for server in runningServers {
            let serverName = server.name
            let serverURL = server.displayURL

            DispatchQueue.global(qos: .utility).async { [weak self] in
                let health = self?.pingServer(url: serverURL) ?? .unknown

                healthLock.lock()
                healthUpdates[serverName] = health
                completedCount += 1
                let isComplete = completedCount >= totalCount
                let currentUpdates = healthUpdates
                healthLock.unlock()

                if isComplete {
                    DispatchQueue.main.async {
                        self?.isHealthCheckInProgress = false
                        self?.applyHealthUpdates(currentUpdates)
                    }
                }
            }
        }
    }

    /// Ping a server URL to check if it's responding
    private func pingServer(url: String) -> HealthStatus {
        guard let serverURL = URL(string: url) else { return .unknown }

        var request = URLRequest(url: serverURL)
        request.httpMethod = "GET"
        request.timeoutInterval = 3.0  // 3 second timeout

        let semaphore = DispatchSemaphore(value: 0)
        var result: HealthStatus = .unknown

        let task = URLSession.shared.dataTask(with: request) { _, response, error in
            if let error = error {
                // Check for connection refused specifically
                let nsError = error as NSError
                if nsError.domain == NSURLErrorDomain {
                    switch nsError.code {
                    case NSURLErrorCannotConnectToHost, NSURLErrorNetworkConnectionLost, NSURLErrorTimedOut:
                        result = .unhealthy
                    default:
                        result = .unhealthy
                    }
                } else {
                    result = .unhealthy
                }
            } else if let httpResponse = response as? HTTPURLResponse {
                // Consider 2xx-4xx as healthy (server is responding)
                // 5xx might indicate server issues but it's still "up"
                result = httpResponse.statusCode < 500 ? .healthy : .unhealthy
            } else {
                result = .unknown
            }
            semaphore.signal()
        }

        task.resume()
        _ = semaphore.wait(timeout: .now() + 4.0)  // Wait up to 4 seconds

        return result
    }

    /// Apply health updates in a single batch
    private func applyHealthUpdates(_ updates: [String: HealthStatus]) {
        guard !updates.isEmpty else { return }

        for (name, health) in updates {
            let previousHealth = serverHealth[name]
            if previousHealth != health {
                serverHealth[name] = health

                // Notify if server became unhealthy
                if health == .unhealthy && previousHealth != .unhealthy {
                    print("[Grove] Server '\(name)' is unhealthy (connection refused)")
                }
            }
        }

        // Clean up health status for servers that are no longer running
        let runningNames = Set(servers.filter { $0.isRunning }.map { $0.name })
        for name in serverHealth.keys {
            if !runningNames.contains(name) {
                serverHealth.removeValue(forKey: name)
            }
        }
    }

    // MARK: - Private

    private func startAutoRefresh() {
        refreshTimer?.invalidate()
        refreshTimer = Timer.scheduledTimer(withTimeInterval: preferences.refreshInterval, repeats: true) { [weak self] _ in
            self?.refresh()
        }
    }

    func updateRefreshInterval() {
        startAutoRefresh()
    }

    private func cleanupRemovedServers(currentServers: [Server]) {
        let currentNames = Set(currentServers.map { $0.name })
        let staleNames = previousServerStates.keys.filter { !currentNames.contains($0) }

        for name in staleNames {
            previousServerStates.removeValue(forKey: name)
        }
    }

    private func fetchGitHubInfoForServers() {
        // Skip if GitHub info is disabled in preferences
        guard preferences.showGitHubInfo else {
            print("[Grove] fetchGitHubInfoForServers() SKIPPED - disabled in preferences")
            return
        }

        // Skip GitHub fetching during wake cooldown (network may not be ready)
        guard !isWakeCooldown else {
            print("[Grove] fetchGitHubInfoForServers() SKIPPED - wake cooldown active")
            return
        }

        // Prevent overlapping fetches
        guard !isGitHubFetchInProgress else {
            print("[Grove] fetchGitHubInfoForServers() SKIPPED - already in progress")
            return
        }

        print("[Grove] fetchGitHubInfoForServers() START")

        // Collect all GitHub info updates and batch them
        let serverCount = servers.count
        guard serverCount > 0 else { return }

        isGitHubFetchInProgress = true

        // Safety timeout - reset flag if fetches take too long (30 seconds max)
        DispatchQueue.main.asyncAfter(deadline: .now() + 30) { [weak self] in
            if self?.isGitHubFetchInProgress == true {
                self?.isGitHubFetchInProgress = false
            }
        }

        // Use a dictionary to collect updates (thread-safe collection)
        let updatesLock = NSLock()
        var updates: [String: GitHubInfo] = [:]
        var completedCount = 0

        for server in servers {
            let serverId = server.id
            githubService.fetchGitHubInfo(for: server) { [weak self] info in
                updatesLock.lock()

                // Add to updates if we have valid info
                if let info = info {
                    updates[serverId] = info
                }

                completedCount += 1
                let isComplete = completedCount >= serverCount
                let currentUpdates = updates
                updatesLock.unlock()

                // When all fetches complete, apply updates in a single batch
                if isComplete {
                    DispatchQueue.main.async {
                        self?.isGitHubFetchInProgress = false
                    }
                    self?.applyGitHubUpdates(currentUpdates)
                }
            }
        }
    }

    /// Apply GitHub info updates in a single batch to minimize re-renders
    private func applyGitHubUpdates(_ updates: [String: GitHubInfo]) {
        guard !updates.isEmpty else { return }

        DispatchQueue.main.async { [weak self] in
            guard let self = self else { return }

            // Apply all updates in one mutation
            var updatedServers = self.servers
            var hasChanges = false

            for (serverId, info) in updates {
                if let index = updatedServers.firstIndex(where: { $0.id == serverId }) {
                    // Only update if info actually changed
                    if updatedServers[index].githubInfo != info {
                        updatedServers[index].githubInfo = info
                        hasChanges = true
                    }
                }
            }

            // Only trigger @Published if there were actual changes
            if hasChanges {
                self.servers = updatedServers
            }
        }
    }

    private func checkForStatusChanges(newServers: [Server]) {
        for server in newServers {
            let previousStatus = previousServerStates[server.name]
            let currentStatus = server.displayStatus

            // Store current status for next comparison
            previousServerStates[server.name] = currentStatus

            // Skip if this is the first time we're seeing this server
            guard let previous = previousStatus else { continue }

            // Check for status changes
            if previous != currentStatus {
                handleStatusChange(server: server, from: previous, to: currentStatus)
            }
        }
    }

    private func handleStatusChange(server: Server, from previousStatus: String, to currentStatus: String) {
        // Server crashed
        if currentStatus == "crashed" && previousStatus != "crashed" {
            NotificationService.shared.notifyServerCrashed(serverName: server.name)
        }

        // Server became healthy (starting -> running)
        if currentStatus == "running" && previousStatus == "starting" {
            NotificationService.shared.notifyServerHealthy(serverName: server.name)
        }

        // Server stopped (could be idle timeout)
        // Note: We can't distinguish between manual stop and idle timeout from status alone
        // This would need additional info from the wt CLI
        if currentStatus == "stopped" && (previousStatus == "running" || previousStatus == "starting") {
            // For now, we'll just send a generic stopped notification
            // In the future, if wt provides idle timeout info, we can check it here
            NotificationService.shared.notifyServerIdleTimeout(serverName: server.name)
        }
    }

    /// Default timeout for grove commands (5 seconds - fail fast)
    private static let commandTimeout: TimeInterval = 5.0

    private func runGrove(_ args: [String], timeout: TimeInterval = commandTimeout, completion: @escaping (Result<String, Error>) -> Void) {
        let grovePath = self.grovePath
        let argsStr = args.joined(separator: " ")
        print("[Grove] runGrove(\(argsStr)) dispatching to background...")

        DispatchQueue.global(qos: .userInitiated).async {
            print("[Grove] runGrove(\(argsStr)) on background thread, starting process...")
            Self.runProcessWithTimeout(
                executablePath: grovePath,
                args: args,
                workingDirectory: nil,
                timeout: timeout,
                completion: completion
            )
        }
    }

    /// Run grove command from a specific working directory (needed for `grove start` which requires being in the worktree)
    /// Uses a longer timeout since start can take time
    private func runGroveInDirectory(_ directory: String, args: [String], completion: @escaping (Result<String, Error>) -> Void) {
        let grovePath = self.grovePath

        DispatchQueue.global(qos: .userInitiated).async {
            Self.runProcessWithTimeout(
                executablePath: grovePath,
                args: args,
                workingDirectory: directory,
                timeout: 30.0, // Longer timeout for start commands
                completion: completion
            )
        }
    }

    /// Run a process with timeout to prevent indefinite hangs
    private static func runProcessWithTimeout(
        executablePath: String,
        args: [String],
        workingDirectory: String?,
        timeout: TimeInterval,
        completion: @escaping (Result<String, Error>) -> Void
    ) {
        let processStart = CFAbsoluteTimeGetCurrent()
        let argsStr = args.joined(separator: " ")
        print("[Grove] runProcessWithTimeout START: \(executablePath) \(argsStr)")

        let task = Process()
        task.executableURL = URL(fileURLWithPath: executablePath)
        task.arguments = args

        if let workingDirectory = workingDirectory {
            task.currentDirectoryURL = URL(fileURLWithPath: workingDirectory)
        }

        let pipe = Pipe()
        task.standardOutput = pipe
        task.standardError = pipe

        // Set up timeout
        var timedOut = false
        let timeoutWorkItem = DispatchWorkItem {
            timedOut = true
            print("[Grove] runProcessWithTimeout TIMEOUT after \(timeout)s: \(argsStr)")
            if task.isRunning {
                task.terminate()
            }
        }

        do {
            print("[Grove] runProcessWithTimeout launching process...")
            try task.run()
            print("[Grove] runProcessWithTimeout process launched, waiting...")

            // Schedule timeout
            DispatchQueue.global().asyncAfter(deadline: .now() + timeout, execute: timeoutWorkItem)

            task.waitUntilExit()
            print("[Grove] runProcessWithTimeout process exited in \(String(format: "%.3f", CFAbsoluteTimeGetCurrent() - processStart))s")

            // Cancel timeout if process finished in time
            timeoutWorkItem.cancel()

            if timedOut {
                completion(.failure(NSError(domain: "GroveMenubar", code: -1,
                    userInfo: [NSLocalizedDescriptionKey: "Command timed out after \(Int(timeout)) seconds"])))
                return
            }

            print("[Grove] runProcessWithTimeout reading output...")
            let data = pipe.fileHandleForReading.readDataToEndOfFile()
            let output = String(data: data, encoding: .utf8) ?? ""
            print("[Grove] runProcessWithTimeout got \(data.count) bytes, status=\(task.terminationStatus)")

            if task.terminationStatus == 0 {
                completion(.success(output))
            } else {
                completion(.failure(NSError(domain: "GroveMenubar", code: Int(task.terminationStatus),
                    userInfo: [NSLocalizedDescriptionKey: output.isEmpty ? "Command failed with exit code \(task.terminationStatus)" : output])))
            }
        } catch {
            print("[Grove] runProcessWithTimeout EXCEPTION: \(error)")
            timeoutWorkItem.cancel()
            completion(.failure(error))
        }
    }

    /// Fast path lookup that doesn't spawn any processes - safe for main thread
    /// Uses UserDefaults cache to avoid repeated filesystem checks after wake
    private static func findGroveBinaryFast() -> String? {
        // Check cache first - much faster than filesystem after wake
        let cacheKey = "cachedGrovePath"
        if let cached = UserDefaults.standard.string(forKey: cacheKey),
           FileManager.default.fileExists(atPath: cached) {
            return cached
        }

        let paths = [
            "\(NSHomeDirectory())/development/claude-helper/cli/grove",
            "\(NSHomeDirectory())/development/go/bin/grove",
            "/usr/local/bin/grove",
            "/opt/homebrew/bin/grove",
            "\(NSHomeDirectory())/go/bin/grove",
            "\(NSHomeDirectory())/.local/bin/grove"
        ]

        for path in paths {
            if FileManager.default.fileExists(atPath: path) {
                // Cache for next launch
                UserDefaults.standard.set(path, forKey: cacheKey)
                return path
            }
        }

        return nil
    }
}
