import Foundation

class GitHubService {
    static let shared = GitHubService()

    private let cache = GitHubCache()
    private let ghPath: String

    private init() {
        self.ghPath = Self.findGhBinary() ?? "/opt/homebrew/bin/gh"
    }

    /// Find the gh CLI binary by searching common paths and falling back to `which`
    private static func findGhBinary() -> String? {
        let paths = [
            "/opt/homebrew/bin/gh",
            "/usr/local/bin/gh",
            "\(NSHomeDirectory())/bin/gh"
        ]

        for path in paths {
            if FileManager.default.fileExists(atPath: path) {
                return path
            }
        }

        // Fallback: try `which gh`
        let task = Process()
        task.executableURL = URL(fileURLWithPath: "/usr/bin/which")
        task.arguments = ["gh"]

        let pipe = Pipe()
        task.standardOutput = pipe
        task.standardError = Pipe()

        do {
            try task.run()
            task.waitUntilExit()
            guard task.terminationStatus == 0 else { return nil }
            let data = pipe.fileHandleForReading.readDataToEndOfFile()
            let result = String(data: data, encoding: .utf8)?.trimmingCharacters(in: .whitespacesAndNewlines)
            return result?.isEmpty == false ? result : nil
        } catch {
            return nil
        }
    }

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

        // Fetch rich PR details if we have a PR number
        var reviewStatus: GitHubInfo.ReviewStatus? = nil
        var commentCount = 0
        var hasMergeConflicts = false

        if let prNumber = prInfo?.number {
            let richInfo = fetchRichPRInfo(at: server.path, prNumber: prNumber)
            reviewStatus = richInfo.reviewStatus
            commentCount = richInfo.commentCount
            hasMergeConflicts = richInfo.hasMergeConflicts
        }

        return GitHubInfo(
            prNumber: prInfo?.number,
            prURL: prInfo?.url,
            prState: prInfo?.state,
            ciStatus: ciStatus,
            reviewStatus: reviewStatus,
            commentCount: commentCount,
            hasMergeConflicts: hasMergeConflicts,
            lastUpdated: Date()
        )
    }

    private func fetchRichPRInfo(at path: String, prNumber: Int) -> (reviewStatus: GitHubInfo.ReviewStatus?, commentCount: Int, hasMergeConflicts: Bool) {
        guard let output = runCommand(ghPath, args: [
            "pr", "view", String(prNumber),
            "--json", "reviews,comments,mergeable"
        ], workingDir: path) else {
            return (nil, 0, false)
        }

        guard let data = output.data(using: .utf8),
              let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any] else {
            return (nil, 0, false)
        }

        // Parse review status (latest review per reviewer)
        var reviewStatus: GitHubInfo.ReviewStatus? = nil
        if let reviews = json["reviews"] as? [[String: Any]], !reviews.isEmpty {
            // Get the most recent review state
            if let lastReview = reviews.last,
               let state = lastReview["state"] as? String {
                switch state.uppercased() {
                case "APPROVED":
                    reviewStatus = .approved
                case "CHANGES_REQUESTED":
                    reviewStatus = .changesRequested
                case "COMMENTED", "PENDING":
                    reviewStatus = .pending
                default:
                    break
                }
            }
        }

        // Parse comment count
        var commentCount = 0
        if let comments = json["comments"] as? [[String: Any]] {
            commentCount = comments.count
        }

        // Parse mergeable status
        var hasMergeConflicts = false
        if let mergeable = json["mergeable"] as? String {
            hasMergeConflicts = mergeable == "CONFLICTING"
        }

        return (reviewStatus, commentCount, hasMergeConflicts)
    }

    private func getCurrentBranch(at path: String) -> String? {
        let result = runCommand("/usr/bin/git", args: ["-C", path, "rev-parse", "--abbrev-ref", "HEAD"])
        return result?.trimmingCharacters(in: .whitespacesAndNewlines)
    }

    private func fetchPRInfo(at path: String, branch: String) -> (number: Int, url: String, state: String)? {
        // Use gh CLI to get PR info
        guard let output = runCommand(ghPath, args: [
            "pr", "list",
            "--head", branch,
            "--json", "number,url,state",
            "--repo", ":owner/:repo"
        ], workingDir: path) else {
            // Try without --repo flag (will auto-detect from git remote)
            guard let output = runCommand(ghPath, args: [
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
        guard let output = runCommand(ghPath, args: [
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

    /// Default timeout for git/gh commands (3 seconds - fail fast when network is down)
    private static let commandTimeout: TimeInterval = 3.0

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

        // Thread-safe timeout flag
        let timedOutLock = NSLock()
        var timedOut = false
        let timeoutWorkItem = DispatchWorkItem {
            timedOutLock.lock()
            timedOut = true
            timedOutLock.unlock()
            if task.isRunning {
                task.terminate()
            }
        }

        do {
            try task.run()

            // Schedule timeout
            DispatchQueue.global().asyncAfter(deadline: .now() + Self.commandTimeout, execute: timeoutWorkItem)

            // Read pipe data FIRST to prevent deadlock on large output
            let data = pipe.fileHandleForReading.readDataToEndOfFile()

            task.waitUntilExit()

            // Cancel timeout if process finished in time
            timeoutWorkItem.cancel()

            timedOutLock.lock()
            let didTimeout = timedOut
            timedOutLock.unlock()

            if didTimeout {
                print("[GitHubService] Command timed out: \(command) \(args.joined(separator: " "))")
                return nil
            }

            guard task.terminationStatus == 0 else { return nil }

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
