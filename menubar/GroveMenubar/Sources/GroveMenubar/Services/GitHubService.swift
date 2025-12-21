import Foundation

class GitHubService {
    static let shared = GitHubService()

    private let cache = GitHubCache()

    private init() {}

    // MARK: - Public API

    func fetchGitHubInfo(for server: Server, completion: @escaping (GitHubInfo?) -> Void) {
        // Check cache first
        if let cached = cache.get(for: server.path) {
            completion(cached)
            return
        }

        // Fetch fresh data in background
        DispatchQueue.global(qos: .utility).async { [weak self] in
            guard let self = self else { return }

            let info = self.fetchGitHubInfoSync(for: server)

            // Cache the result
            if let info = info {
                self.cache.set(info, for: server.path)
            }

            DispatchQueue.main.async {
                completion(info)
            }
        }
    }

    // MARK: - Private Helpers

    private func fetchGitHubInfoSync(for server: Server) -> GitHubInfo? {
        // Get current branch
        guard let branch = getCurrentBranch(at: server.path) else {
            return nil
        }

        // Fetch PR info
        let prInfo = fetchPRInfo(at: server.path, branch: branch)

        // Fetch CI status
        let ciStatus = fetchCIStatus(at: server.path)

        return GitHubInfo(
            prNumber: prInfo?.number,
            prURL: prInfo?.url,
            prState: prInfo?.state,
            ciStatus: ciStatus,
            lastUpdated: Date()
        )
    }

    private func getCurrentBranch(at path: String) -> String? {
        let result = runCommand("/usr/bin/git", args: ["-C", path, "rev-parse", "--abbrev-ref", "HEAD"])
        return result?.trimmingCharacters(in: .whitespacesAndNewlines)
    }

    private func fetchPRInfo(at path: String, branch: String) -> (number: Int, url: String, state: String)? {
        // Use gh CLI to get PR info
        guard let output = runCommand("/opt/homebrew/bin/gh", args: [
            "pr", "list",
            "--head", branch,
            "--json", "number,url,state",
            "--repo", ":owner/:repo"
        ], workingDir: path) else {
            // Try without --repo flag (will auto-detect from git remote)
            guard let output = runCommand("/opt/homebrew/bin/gh", args: [
                "pr", "list",
                "--head", branch,
                "--json", "number,url,state"
            ], workingDir: path) else {
                return nil
            }
            return parsePRJSON(output)
        }

        return parsePRJSON(output)
    }

    private func parsePRJSON(_ json: String) -> (number: Int, url: String, state: String)? {
        guard let data = json.data(using: .utf8),
              let array = try? JSONSerialization.jsonObject(with: data) as? [[String: Any]],
              let first = array.first,
              let number = first["number"] as? Int,
              let url = first["url"] as? String,
              let state = first["state"] as? String else {
            return nil
        }

        return (number, url, state)
    }

    private func fetchCIStatus(at path: String) -> GitHubInfo.CIStatus {
        // Get current commit SHA
        guard let sha = runCommand("/usr/bin/git", args: ["-C", path, "rev-parse", "HEAD"])?
            .trimmingCharacters(in: .whitespacesAndNewlines) else {
            return .unknown
        }

        // Get repository info
        guard let repoInfo = getRepoInfo(at: path) else {
            return .unknown
        }

        // Fetch check runs from GitHub API
        guard let output = runCommand("/opt/homebrew/bin/gh", args: [
            "api",
            "repos/\(repoInfo.owner)/\(repoInfo.repo)/commits/\(sha)/check-runs",
            "--jq", ".check_runs[].conclusion"
        ], workingDir: path) else {
            return .unknown
        }

        // Parse conclusions
        let conclusions = output.components(separatedBy: .newlines)
            .map { $0.trimmingCharacters(in: .whitespacesAndNewlines) }
            .filter { !$0.isEmpty }

        if conclusions.isEmpty {
            return .unknown
        }

        if conclusions.contains("failure") || conclusions.contains("timed_out") || conclusions.contains("cancelled") {
            return .failure
        }

        if conclusions.contains("") || conclusions.contains("null") {
            return .pending
        }

        if conclusions.allSatisfy({ $0 == "success" || $0 == "skipped" || $0 == "neutral" }) {
            return .success
        }

        return .unknown
    }

    private func getRepoInfo(at path: String) -> (owner: String, repo: String)? {
        guard let remoteURL = runCommand("/usr/bin/git", args: ["-C", path, "remote", "get-url", "origin"])?
            .trimmingCharacters(in: .whitespacesAndNewlines) else {
            return nil
        }

        // Parse GitHub URL (handles both SSH and HTTPS)
        // SSH: git@github.com:owner/repo.git
        // HTTPS: https://github.com/owner/repo.git
        let patterns = [
            "github.com[:/]([^/]+)/([^/]+?)(?:\\.git)?$"
        ]

        for pattern in patterns {
            if let regex = try? NSRegularExpression(pattern: pattern),
               let match = regex.firstMatch(in: remoteURL, range: NSRange(remoteURL.startIndex..., in: remoteURL)) {
                if let ownerRange = Range(match.range(at: 1), in: remoteURL),
                   let repoRange = Range(match.range(at: 2), in: remoteURL) {
                    let owner = String(remoteURL[ownerRange])
                    let repo = String(remoteURL[repoRange])
                    return (owner, repo)
                }
            }
        }

        return nil
    }

    /// Default timeout for git/gh commands (5 seconds)
    private static let commandTimeout: TimeInterval = 5.0

    private func runCommand(_ command: String, args: [String], workingDir: String? = nil) -> String? {
        let task = Process()
        task.executableURL = URL(fileURLWithPath: command)
        task.arguments = args

        if let workingDir = workingDir {
            task.currentDirectoryURL = URL(fileURLWithPath: workingDir)
        }

        let pipe = Pipe()
        task.standardOutput = pipe
        task.standardError = Pipe()

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
            DispatchQueue.global().asyncAfter(deadline: .now() + Self.commandTimeout, execute: timeoutWorkItem)

            task.waitUntilExit()

            // Cancel timeout if process finished in time
            timeoutWorkItem.cancel()

            if timedOut {
                print("[GitHubService] Command timed out: \(command) \(args.joined(separator: " "))")
                return nil
            }

            guard task.terminationStatus == 0 else { return nil }

            let data = pipe.fileHandleForReading.readDataToEndOfFile()
            return String(data: data, encoding: .utf8)
        } catch {
            timeoutWorkItem.cancel()
            return nil
        }
    }
}

// MARK: - Cache

private class GitHubCache {
    private var cache: [String: CacheEntry] = [:]
    private let cacheTimeout: TimeInterval = 300 // 5 minutes
    private let lock = NSLock() // Thread-safe access

    struct CacheEntry {
        let info: GitHubInfo
        let timestamp: Date
    }

    func get(for path: String) -> GitHubInfo? {
        lock.lock()
        defer { lock.unlock() }

        guard let entry = cache[path] else { return nil }

        // Check if cache is still valid
        if Date().timeIntervalSince(entry.timestamp) > cacheTimeout {
            cache.removeValue(forKey: path)
            return nil
        }

        return entry.info
    }

    func set(_ info: GitHubInfo, for path: String) {
        lock.lock()
        defer { lock.unlock() }
        cache[path] = CacheEntry(info: info, timestamp: Date())
    }

    func clear() {
        lock.lock()
        defer { lock.unlock() }
        cache.removeAll()
    }

    func invalidate(for path: String) {
        lock.lock()
        defer { lock.unlock() }
        cache.removeValue(forKey: path)
    }
}
