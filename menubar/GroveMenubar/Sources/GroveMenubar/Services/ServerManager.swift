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

    private var refreshTimer: Timer?
    private var logTimer: Timer?
    private var lastLogPosition: UInt64 = 0
    private let grovePath: String
    private var previousServerStates: [String: String] = [:]  // Track previous server statuses
    private let githubService = GitHubService.shared
    private let preferences = PreferencesManager.shared

    var isPortMode: Bool { urlMode == "port" }
    var isSubdomainMode: Bool { urlMode == "subdomain" }

    init() {
        let initStart = CFAbsoluteTimeGetCurrent()
        print("[DEBUG] ServerManager.init() started")

        // Find grove binary synchronously from known paths (fast, no process spawn)
        // We avoid running `which` here to prevent blocking the main thread
        self.grovePath = Self.findGroveBinaryFast() ?? "/usr/local/bin/grove"
        print("[DEBUG] Found grove at: \(grovePath) (took \(CFAbsoluteTimeGetCurrent() - initStart)s)")

        refresh()
        startAutoRefresh()
        print("[DEBUG] ServerManager.init() completed (took \(CFAbsoluteTimeGetCurrent() - initStart)s)")
    }

    deinit {
        refreshTimer?.invalidate()
        logTimer?.invalidate()
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

    // MARK: - Actions

    func refresh() {
        let refreshStart = CFAbsoluteTimeGetCurrent()
        print("[DEBUG] refresh() started on thread: \(Thread.isMainThread ? "MAIN" : "background")")

        isLoading = true
        error = nil

        runGrove(["ls", "--json"]) { [weak self] result in
            print("[DEBUG] runGrove completed (took \(CFAbsoluteTimeGetCurrent() - refreshStart)s)")
            switch result {
            case .success(let output):
                // Parse and filter on background thread to avoid blocking main thread
                self?.parseStatusAsync(output)
            case .failure(let err):
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

    func openTUI() {
        // Open configured terminal with grove TUI command
        // Get the directory of the grove binary and run it
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
                let task = Process()
                task.executableURL = URL(fileURLWithPath: "/usr/bin/open")
                task.arguments = ["-a", "Ghostty", "--args", "-e", grovePath]
                try? task.run()
            case "dev.warp.Warp-Stable":
                let task = Process()
                task.executableURL = URL(fileURLWithPath: "/usr/bin/open")
                task.arguments = ["-a", "Warp", groveDir]
                try? task.run()
            default:
                // Fallback to Terminal.app
                let script = """
                tell application "Terminal"
                    activate
                    do script "\(grovePath)"
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
                    let newLines = newContent.components(separatedBy: .newlines).filter { !$0.isEmpty }

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

    /// Parse status on background thread, then update UI on main thread
    private func parseStatusAsync(_ output: String) {
        let parseStart = CFAbsoluteTimeGetCurrent()
        print("[DEBUG] parseStatusAsync started on thread: \(Thread.isMainThread ? "MAIN" : "background")")

        // Already on background thread from runGrove
        guard let data = output.data(using: .utf8) else {
            DispatchQueue.main.async { [weak self] in
                self?.isLoading = false
            }
            return
        }

        do {
            let status = try JSONDecoder().decode(WTStatus.self, from: data)
            print("[DEBUG] JSON decode done (took \(CFAbsoluteTimeGetCurrent() - parseStart)s)")

            // Filter out worktrees whose paths no longer exist (done on background thread)
            let validServers = status.servers.filter { server in
                let exists = FileManager.default.fileExists(atPath: server.path)
                if !exists {
                    print("Worktree path no longer exists, filtering out: \(server.name) at \(server.path)")
                }
                return exists
            }
            print("[DEBUG] FileManager filter done (took \(CFAbsoluteTimeGetCurrent() - parseStart)s)")

            let newServers = validServers.sorted { $0.name < $1.name }
            let proxy = status.proxy
            let urlMode = status.urlMode

            // Update UI on main thread
            DispatchQueue.main.async { [weak self] in
                let mainStart = CFAbsoluteTimeGetCurrent()
                print("[DEBUG] Main thread update started")
                guard let self = self else { return }

                self.isLoading = false

                // Check for status changes and send notifications
                self.checkForStatusChanges(newServers: newServers)

                // Clean up previousServerStates for removed servers
                self.cleanupRemovedServers(currentServers: newServers)

                self.servers = newServers
                self.proxy = proxy
                self.urlMode = urlMode
                print("[DEBUG] Main thread update done (took \(CFAbsoluteTimeGetCurrent() - mainStart)s)")

                // Fetch GitHub info for all servers in background
                self.fetchGitHubInfoForServers()
            }
        } catch {
            DispatchQueue.main.async { [weak self] in
                self?.isLoading = false
                self?.error = "Failed to parse status: \(error.localizedDescription)"
            }
        }
    }

    private func cleanupRemovedServers(currentServers: [Server]) {
        let currentNames = Set(currentServers.map { $0.name })
        let staleNames = previousServerStates.keys.filter { !currentNames.contains($0) }

        for name in staleNames {
            previousServerStates.removeValue(forKey: name)
        }
    }

    private func fetchGitHubInfoForServers() {
        // Fetch GitHub info for each server
        for (index, server) in servers.enumerated() {
            githubService.fetchGitHubInfo(for: server) { [weak self] info in
                guard let self = self else { return }
                DispatchQueue.main.async {
                    // Update the server with GitHub info
                    if index < self.servers.count && self.servers[index].id == server.id {
                        var updatedServer = self.servers[index]
                        updatedServer.githubInfo = info
                        self.servers[index] = updatedServer
                    }
                }
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

    /// Default timeout for grove commands (10 seconds should be plenty for ls --json)
    private static let commandTimeout: TimeInterval = 10.0

    private func runGrove(_ args: [String], timeout: TimeInterval = commandTimeout, completion: @escaping (Result<String, Error>) -> Void) {
        let grovePath = self.grovePath

        DispatchQueue.global(qos: .userInitiated).async {
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
            if task.isRunning {
                task.terminate()
            }
        }

        do {
            try task.run()

            // Schedule timeout
            DispatchQueue.global().asyncAfter(deadline: .now() + timeout, execute: timeoutWorkItem)

            task.waitUntilExit()

            // Cancel timeout if process finished in time
            timeoutWorkItem.cancel()

            if timedOut {
                completion(.failure(NSError(domain: "GroveMenubar", code: -1,
                    userInfo: [NSLocalizedDescriptionKey: "Command timed out after \(Int(timeout)) seconds"])))
                return
            }

            let data = pipe.fileHandleForReading.readDataToEndOfFile()
            let output = String(data: data, encoding: .utf8) ?? ""

            if task.terminationStatus == 0 {
                completion(.success(output))
            } else {
                completion(.failure(NSError(domain: "GroveMenubar", code: Int(task.terminationStatus),
                    userInfo: [NSLocalizedDescriptionKey: output.isEmpty ? "Command failed with exit code \(task.terminationStatus)" : output])))
            }
        } catch {
            timeoutWorkItem.cancel()
            completion(.failure(error))
        }
    }

    /// Fast path lookup that doesn't spawn any processes - safe for main thread
    private static func findGroveBinaryFast() -> String? {
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
                return path
            }
        }

        return nil
    }
}
