import Foundation

/// Manages recent command history per server, persisted in UserDefaults
class QuickCommandHistory: ObservableObject {
    static let shared = QuickCommandHistory()

    private let defaults = UserDefaults.standard
    private let storageKey = "quickCommandHistory"
    private let maxPerServer = 5

    @Published private var history: [String: [String]] = [:]

    static let suggestions = [
        "rails console",
        "rails db:migrate",
        "bundle exec rspec",
        "npm run test",
        "npm run build",
        "yarn test",
        "make test",
        "git status",
        "git log --oneline -10",
    ]

    private init() {
        loadHistory()
    }

    func recentCommands(for serverName: String) -> [String] {
        history[serverName] ?? []
    }

    func addCommand(_ command: String, for serverName: String) {
        var commands = history[serverName] ?? []
        // Remove if already exists (move to front)
        commands.removeAll { $0 == command }
        commands.insert(command, at: 0)
        // Keep only the most recent
        if commands.count > maxPerServer {
            commands = Array(commands.prefix(maxPerServer))
        }
        history[serverName] = commands
        saveHistory()
    }

    private func loadHistory() {
        guard let data = defaults.data(forKey: storageKey),
              let decoded = try? JSONDecoder().decode([String: [String]].self, from: data) else {
            return
        }
        history = decoded
    }

    private func saveHistory() {
        if let data = try? JSONEncoder().encode(history) {
            defaults.set(data, forKey: storageKey)
        }
    }
}
