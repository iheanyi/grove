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

    var isPortMode: Bool { urlMode == "port" }
    var isSubdomainMode: Bool { urlMode == "subdomain" }

    private static func debugLog(_ message: String) {
        let logFile = "/tmp/wtmenubar-debug.log"
        let timestamp = ISO8601DateFormatter().string(from: Date())
        let line = "[\(timestamp)] \(message)\n"
        if let data = line.data(using: .utf8) {
            if FileManager.default.fileExists(atPath: logFile) {
                if let handle = FileHandle(forWritingAtPath: logFile) {
                    handle.seekToEndOfFile()
                    handle.write(data)
                    handle.closeFile()
                }
            } else {
                FileManager.default.createFile(atPath: logFile, contents: data)
            }
        }
    }

    init() {
        // Find wt binary
        if let path = Self.findWTBinary() {
            self.wtPath = path
            Self.debugLog("Found wt at: \(path)")
        } else {
            self.wtPath = "/usr/local/bin/wt"
            Self.debugLog("WARNING: wt not found, defaulting to: \(self.wtPath)")
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
        if servers.contains(where: { $0.isRunning }) {
            return "bolt.fill"
        }
        return "bolt"
    }

    var statusColor: Color {
        if servers.contains(where: { $0.status == "crashed" }) {
            return .red
        }
        if servers.contains(where: { $0.isRunning }) {
            return .green
        }
        return .gray
    }

    var runningCount: Int {
        servers.filter { $0.isRunning }.count
    }

    // MARK: - Actions

    func refresh() {
        isLoading = true
        error = nil
        Self.debugLog("refresh() called, wtPath=\(wtPath)")

        runWT(["ls", "--json"]) { [weak self] result in
            DispatchQueue.main.async {
                self?.isLoading = false
                switch result {
                case .success(let output):
                    Self.debugLog("runWT success, output length: \(output.count)")
                    self?.parseStatus(output)
                case .failure(let err):
                    Self.debugLog("runWT FAILED: \(err)")
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

    func openServer(_ server: Server) {
        if let url = URL(string: server.url) {
            NSWorkspace.shared.open(url)
        }
    }

    func copyURL(_ server: Server) {
        NSPasteboard.general.clearContents()
        NSPasteboard.general.setString(server.url, forType: .string)
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
        refreshTimer = Timer.scheduledTimer(withTimeInterval: 5.0, repeats: true) { [weak self] _ in
            self?.refresh()
        }
    }

    private func parseStatus(_ output: String) {
        Self.debugLog("parseStatus called with output length: \(output.count)")
        guard let data = output.data(using: .utf8) else {
            Self.debugLog("ERROR: Could not convert output to data")
            return
        }

        do {
            let status = try JSONDecoder().decode(WTStatus.self, from: data)
            self.servers = status.servers.sorted { $0.name < $1.name }
            self.proxy = status.proxy
            self.urlMode = status.urlMode
            Self.debugLog("Parsed \(status.servers.count) servers")
        } catch {
            self.error = "Failed to parse status: \(error.localizedDescription)"
            Self.debugLog("ERROR parsing: \(error)")
            Self.debugLog("Raw output: \(output.prefix(500))")
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
