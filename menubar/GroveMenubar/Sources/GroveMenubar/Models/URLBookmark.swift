import Foundation

struct URLBookmark: Codable, Identifiable, Equatable {
    let id: String
    let path: String
    let label: String

    init(path: String, label: String) {
        self.id = UUID().uuidString
        self.path = path
        self.label = label
    }
}

/// Manages URL bookmarks per server, persisted in UserDefaults
class BookmarkManager: ObservableObject {
    static let shared = BookmarkManager()

    private let defaults = UserDefaults.standard
    private let storageKey = "serverURLBookmarks"

    @Published private var bookmarks: [String: [URLBookmark]] = [:]

    static let suggestions: [(path: String, label: String)] = [
        ("/admin", "Admin"),
        ("/api/docs", "API Docs"),
        ("/sidekiq", "Sidekiq"),
        ("/graphiql", "GraphiQL"),
        ("/letter_opener", "Emails"),
        ("/rails/info", "Rails Info"),
    ]

    private init() {
        loadBookmarks()
    }

    func bookmarks(for serverName: String) -> [URLBookmark] {
        bookmarks[serverName] ?? []
    }

    func addBookmark(for serverName: String, path: String, label: String) {
        let bookmark = URLBookmark(path: path, label: label)
        var serverBookmarks = bookmarks[serverName] ?? []
        // Avoid duplicates by path
        guard !serverBookmarks.contains(where: { $0.path == path }) else { return }
        serverBookmarks.append(bookmark)
        bookmarks[serverName] = serverBookmarks
        saveBookmarks()
    }

    func removeBookmark(for serverName: String, bookmarkId: String) {
        var serverBookmarks = bookmarks[serverName] ?? []
        serverBookmarks.removeAll { $0.id == bookmarkId }
        bookmarks[serverName] = serverBookmarks
        saveBookmarks()
    }

    private func loadBookmarks() {
        guard let data = defaults.data(forKey: storageKey),
              let decoded = try? JSONDecoder().decode([String: [URLBookmark]].self, from: data) else {
            return
        }
        bookmarks = decoded
    }

    private func saveBookmarks() {
        if let data = try? JSONEncoder().encode(bookmarks) {
            defaults.set(data, forKey: storageKey)
        }
    }
}
