import Foundation

struct ServerGroup: Identifiable {
    let id: String
    let name: String
    let path: String
    var servers: [Server]

    var isRunning: Bool {
        servers.contains { $0.isRunning }
    }

    var runningCount: Int {
        servers.filter { $0.isRunning }.count
    }

    var totalCount: Int {
        servers.count
    }
}

class ServerGrouper {
    static func groupServers(_ servers: [Server]) -> [ServerGroup] {
        // Group servers by their parent directory
        var groups: [String: [Server]] = [:]

        for server in servers {
            let groupKey = extractGroupKey(from: server.path)
            groups[groupKey, default: []].append(server)
        }

        // Convert to ServerGroup objects
        return groups.map { key, servers in
            ServerGroup(
                id: key,
                name: extractGroupName(from: key),
                path: key,
                servers: servers.sorted { $0.name < $1.name }
            )
        }.sorted { $0.name < $1.name }
    }

    private static func extractGroupKey(from path: String) -> String {
        // Extract the parent directory as the group key
        let url = URL(fileURLWithPath: path)
        let parentPath = url.deletingLastPathComponent().path

        // If the parent is the home directory or root, use the immediate parent
        let homeDir = NSHomeDirectory()
        if parentPath == homeDir || parentPath == "/" {
            return path
        }

        return parentPath
    }

    private static func extractGroupName(from path: String) -> String {
        // Extract a friendly name from the path
        let url = URL(fileURLWithPath: path)
        let lastComponent = url.lastPathComponent

        // If it looks like a home directory path, show the last 2 components
        if path.contains(NSHomeDirectory()) {
            let components = url.pathComponents
            if components.count >= 2 {
                return components.suffix(2).joined(separator: "/")
            }
        }

        return lastComponent
    }

    // Check if servers should be grouped (only group if there are multiple groups)
    static func shouldGroup(_ servers: [Server]) -> Bool {
        let groups = groupServers(servers)
        return groups.count > 1
    }
}

// UserDefaults extension for collapsed groups
class CollapsedGroupsManager {
    static let shared = CollapsedGroupsManager()
    private let defaults = UserDefaults.standard
    private let key = "collapsedServerGroups"

    private init() {}

    func isCollapsed(_ groupId: String) -> Bool {
        let collapsed = defaults.stringArray(forKey: key) ?? []
        return collapsed.contains(groupId)
    }

    func setCollapsed(_ groupId: String, collapsed: Bool) {
        var collapsedGroups = defaults.stringArray(forKey: key) ?? []

        if collapsed {
            if !collapsedGroups.contains(groupId) {
                collapsedGroups.append(groupId)
            }
        } else {
            collapsedGroups.removeAll { $0 == groupId }
        }

        defaults.set(collapsedGroups, forKey: key)
    }

    func toggleCollapsed(_ groupId: String) {
        setCollapsed(groupId, collapsed: !isCollapsed(groupId))
    }
}
