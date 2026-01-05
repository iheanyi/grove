import Foundation

/// Represents an active AI agent session (e.g., Claude Code)
struct Agent: Codable, Identifiable, Equatable {
    let worktree: String
    let path: String
    let branch: String
    let type: String
    let pid: Int
    let startTime: String?
    let duration: String?
    let activeTask: String?
    let taskSummary: String?

    var id: String { "\(type)-\(pid)" }

    enum CodingKeys: String, CodingKey {
        case worktree, path, branch, type, pid
        case startTime = "start_time"
        case duration
        case activeTask = "active_task"
        case taskSummary = "task_summary"
    }

    /// Display name for the agent type
    var displayType: String {
        switch type {
        case "claude":
            return "Claude Code"
        case "cursor":
            return "Cursor"
        case "copilot":
            return "GitHub Copilot"
        default:
            return type.capitalized
        }
    }

    /// Icon for the agent type
    var iconName: String {
        switch type {
        case "claude":
            return "brain.head.profile"
        case "cursor":
            return "cursorarrow.click.2"
        case "copilot":
            return "airplane"
        default:
            return "cpu"
        }
    }

    /// Shortened path for display (with ~ for home directory)
    var shortPath: String {
        let home = NSHomeDirectory()
        if path.hasPrefix(home) {
            return "~" + path.dropFirst(home.count)
        }
        return path
    }

    /// Short task display (task ID or nil)
    var shortTaskDisplay: String? {
        guard let task = activeTask, !task.isEmpty else { return nil }
        // Truncate if too long
        if task.count > 20 {
            return String(task.prefix(17)) + "..."
        }
        return task
    }

    /// Whether this agent has an active task
    var hasActiveTask: Bool {
        activeTask != nil && !activeTask!.isEmpty
    }
}
