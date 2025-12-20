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
    private let wtPath: String
    private var previousServerStates: [String: String] = [:]  // Track previous server statuses
    private let githubService = GitHubService.shared
    private let preferences = PreferencesManager.shared

    var isPortMode: Bool { urlMode == "port" }
    var isSubdomainMode: Bool { urlMode == "subdomain" }

    init() {
        // Find wt binary
        if let path = Self.findWTBinary() {
            self.wtPath = path
        } else {
            self.wtPath = "/usr/local/bin/wt"
        }

        refresh()
        startAutoRefresh()
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
        isLoading = true
        error = nil

        runWT(["ls", "--json"]) { [weak self] result in
            DispatchQueue.main.async {
                self?.isLoading = false
                switch result {
                case .success(let output):
                    self?.parseStatus(output)
                case .failure(let err):
                    self?.error = err.localizedDescription
                }
            }
        }
    }

    func stopServer(_ server: Server) {
        runWT(["stop", server.name]) { [weak self] _ in
            DispatchQueue.main.async {
                self?.refresh()
            }
        }
    }

    func stopAllServers() {
        let runningServers = servers.filter { $0.isRunning }
        guard !runningServers.isEmpty else { return }

        for server in runningServers {
            runWT(["stop", server.name]) { _ in }
        }

        // Refresh after a short delay to allow all stops to complete
        DispatchQueue.main.asyncAfter(deadline: .now() + 1.0) { [weak self] in
            self?.refresh()
        }
    }

    func startServer(_ server: Server) {
        runWT(["start", server.name]) { [weak self] _ in
            DispatchQueue.main.async {
                self?.refresh()
            }
        }
    }

    // MARK: - Group Actions

    func startAllInGroup(_ group: ServerGroup) {
        let stoppedServers = group.servers.filter { !$0.isRunning }
        guard !stoppedServers.isEmpty else { return }

        for server in stoppedServers {
            runWT(["start", server.name]) { _ in }
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
            runWT(["stop", server.name]) { _ in }
        }

        // Refresh after a short delay
        DispatchQueue.main.asyncAfter(deadline: .now() + 1.0) { [weak self] in
            self?.refresh()
        }
    }

    func openServer(_ server: Server) {
        if let url = URL(string: server.url) {
            preferences.openURL(url)
        }
    }

    func copyURL(_ server: Server) {
        NSPasteboard.general.clearContents()
        NSPasteboard.general.setString(server.url, forType: .string)
    }

    func openAllRunningServers() {
        let runningServers = servers.filter { $0.isRunning }
        for server in runningServers {
            if let url = URL(string: server.url) {
                preferences.openURL(url)
            }
        }
    }

    // MARK: - Quick Navigation

    func openInTerminal(_ server: Server) {
        let script = """
        tell application "Terminal"
            activate
            do script "cd '\(server.path)'"
        end tell
        """
        if let appleScript = NSAppleScript(source: script) {
            var error: NSDictionary?
            appleScript.executeAndReturnError(&error)
        }
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
        runWT(["proxy", "start"]) { [weak self] _ in
            DispatchQueue.main.async {
                self?.refresh()
            }
        }
    }

    func stopProxy() {
        runWT(["proxy", "stop"]) { [weak self] _ in
            DispatchQueue.main.async {
                self?.refresh()
            }
        }
    }

    func openTUI() {
        // Open Terminal with wt command
        let script = """
        tell application "Terminal"
            activate
            do script "\(wtPath)"
        end tell
        """
        if let appleScript = NSAppleScript(source: script) {
            var error: NSDictionary?
            appleScript.executeAndReturnError(&error)
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

        DispatchQueue.global(qos: .userInitiated).async { [weak self] in
            guard let self = self else { return }

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

    private func parseStatus(_ output: String) {
        guard let data = output.data(using: .utf8) else { return }

        do {
            let status = try JSONDecoder().decode(WTStatus.self, from: data)
            let newServers = status.servers.sorted { $0.name < $1.name }

            // Check for status changes and send notifications
            checkForStatusChanges(newServers: newServers)

            self.servers = newServers
            self.proxy = status.proxy
            self.urlMode = status.urlMode

            // Fetch GitHub info for all servers in background
            fetchGitHubInfoForServers()
        } catch {
            self.error = "Failed to parse status: \(error.localizedDescription)"
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
            let currentStatus = server.status

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

    private func runWT(_ args: [String], completion: @escaping (Result<String, Error>) -> Void) {
        DispatchQueue.global(qos: .userInitiated).async {
            let task = Process()
            task.executableURL = URL(fileURLWithPath: self.wtPath)
            task.arguments = args

            let pipe = Pipe()
            task.standardOutput = pipe
            task.standardError = pipe

            do {
                try task.run()
                task.waitUntilExit()

                let data = pipe.fileHandleForReading.readDataToEndOfFile()
                let output = String(data: data, encoding: .utf8) ?? ""

                if task.terminationStatus == 0 {
                    completion(.success(output))
                } else {
                    completion(.failure(NSError(domain: "WTMenubar", code: Int(task.terminationStatus),
                        userInfo: [NSLocalizedDescriptionKey: output])))
                }
            } catch {
                completion(.failure(error))
            }
        }
    }

    private static func findWTBinary() -> String? {
        // Order matters - check development path first
        let paths = [
            "\(NSHomeDirectory())/development/go/bin/wt",
            "/usr/local/bin/wt",
            "/opt/homebrew/bin/wt",
            "\(NSHomeDirectory())/go/bin/wt",
            "\(NSHomeDirectory())/.local/bin/wt"
        ]

        for path in paths {
            if FileManager.default.fileExists(atPath: path) {
                return path
            }
        }

        // Try which command
        let task = Process()
        task.executableURL = URL(fileURLWithPath: "/usr/bin/which")
        task.arguments = ["wt"]

        let pipe = Pipe()
        task.standardOutput = pipe

        do {
            try task.run()
            task.waitUntilExit()

            let data = pipe.fileHandleForReading.readDataToEndOfFile()
            let output = String(data: data, encoding: .utf8)?.trimmingCharacters(in: .whitespacesAndNewlines)

            if task.terminationStatus == 0, let path = output, !path.isEmpty {
                return path
            }
        } catch {}

        return nil
    }
}
