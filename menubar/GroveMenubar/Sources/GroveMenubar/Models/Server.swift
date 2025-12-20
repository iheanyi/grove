import Foundation

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

    enum CodingKeys: String, CodingKey {
        case name, url, subdomains, port, status, health, path, uptime, pid
        case logFile = "log_file"
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
        default:
            return .gray
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

struct GroveStatus: Codable {
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
    static let wtPrimary = Color(red: 124/255, green: 58/255, blue: 237/255)
    static let wtGreen = Color(red: 16/255, green: 185/255, blue: 129/255)
    static let wtRed = Color(red: 239/255, green: 68/255, blue: 68/255)
}
