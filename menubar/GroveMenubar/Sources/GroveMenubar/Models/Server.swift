import Foundation
import SwiftUI

// MARK: - GitHub Models

struct GitHubInfo: Codable, Equatable {
    let prNumber: Int?
    let prURL: String?
    let prState: String?
    let ciStatus: CIStatus
    let lastUpdated: Date

    enum CIStatus: String, Codable {
        case success
        case failure
        case pending
        case unknown

        var icon: String {
            switch self {
            case .success: return "checkmark.circle.fill"
            case .failure: return "xmark.circle.fill"
            case .pending: return "clock.fill"
            case .unknown: return "questionmark.circle"
            }
        }

        var color: Color {
            switch self {
            case .success: return .green
            case .failure: return .red
            case .pending: return .yellow
            case .unknown: return .gray
            }
        }
    }

    static let empty = GitHubInfo(
        prNumber: nil,
        prURL: nil,
        prState: nil,
        ciStatus: .unknown,
        lastUpdated: Date()
    )
}

struct Server: Identifiable, Codable {
    let name: String
    let url: String
    let subdomains: String?  // Only present in subdomain mode
    let port: Int
    let status: String
    let health: String?
    let path: String
    let uptime: String?
    let pid: Int?
    let logFile: String?
    var githubInfo: GitHubInfo?

    enum CodingKeys: String, CodingKey {
        case name, url, subdomains, port, status, health, path, uptime, pid
        case logFile = "log_file"
        case githubInfo
    }

    var id: String { name }

    var isRunning: Bool {
        status == "running" || status == "starting"
    }

    var statusIcon: String {
        switch status {
        case "running":
            return "circle.fill"
        case "stopped":
            return "circle"
        case "crashed":
            return "xmark.circle.fill"
        case "starting":
            return "circle.dotted"
        default:
            return "circle"
        }
    }

    var statusColor: Color {
        switch status {
        case "running":
            return .green
        case "stopped":
            return .gray
        case "crashed":
            return .red
        case "starting":
            return .yellow
        default:
            return .gray
        }
    }

    var healthColor: Color {
        guard let health = health else { return .gray }
        switch health {
        case "healthy":
            return .green
        case "unhealthy":
            return .red
        default:
            return .yellow
        }
    }

    var formattedUptime: String? {
        guard let uptime = uptime else { return nil }

        // Parse uptime string like "2h34m12s" or "45m23s" or "15s"
        var hours = 0
        var minutes = 0
        var seconds = 0

        let scanner = Scanner(string: uptime)
        scanner.charactersToBeSkipped = CharacterSet.letters

        while !scanner.isAtEnd {
            if let value = scanner.scanInt() {
                let currentIndex = scanner.currentIndex
                if currentIndex < uptime.endIndex {
                    let nextChar = uptime[currentIndex]
                    switch nextChar {
                    case "h":
                        hours = value
                    case "m":
                        minutes = value
                    case "s":
                        seconds = value
                    default:
                        break
                    }
                }
            }
            scanner.scanCharacters(from: CharacterSet.letters)
        }

        // Format based on duration
        if hours > 0 {
            return String(format: "%dh %dm", hours, minutes)
        } else if minutes > 0 {
            return String(format: "%dm", minutes)
        } else {
            return String(format: "%ds", seconds)
        }
    }
}

struct ProxyInfo: Codable {
    let status: String
    let httpPort: Int
    let httpsPort: Int
    let pid: Int?

    enum CodingKeys: String, CodingKey {
        case status
        case httpPort = "http_port"
        case httpsPort = "https_port"
        case pid
    }

    var isRunning: Bool {
        status == "running"
    }
}

struct WTStatus: Codable {
    let servers: [Server]
    let proxy: ProxyInfo?  // Only present in subdomain mode
    let urlMode: String    // "port" or "subdomain"

    enum CodingKeys: String, CodingKey {
        case servers, proxy
        case urlMode = "url_mode"
    }

    var isPortMode: Bool {
        urlMode == "port"
    }

    var isSubdomainMode: Bool {
        urlMode == "subdomain"
    }
}

import SwiftUI

extension Color {
    static let grovePrimary = Color(red: 124/255, green: 58/255, blue: 237/255)
    static let groveGreen = Color(red: 16/255, green: 185/255, blue: 129/255)
    static let groveRed = Color(red: 239/255, green: 68/255, blue: 68/255)
}
