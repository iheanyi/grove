import SwiftUI

/// Shared log level enum used by LogsView and LogViewerWindow
enum LogLevel: String, CaseIterable {
    case error = "ERROR"
    case warn = "WARN"
    case info = "INFO"
    case debug = "DEBUG"

    var color: Color {
        switch self {
        case .error: return .red
        case .warn: return .orange
        case .info: return .blue
        case .debug: return .gray
        }
    }

    var icon: String {
        switch self {
        case .error: return "exclamationmark.triangle.fill"
        case .warn: return "exclamationmark.circle.fill"
        case .info: return "info.circle.fill"
        case .debug: return "ant.circle.fill"
        }
    }

    /// Matches log lines against log level filters, including Rails-specific patterns
    func matches(_ text: String) -> Bool {
        let upperText = text.uppercased()

        switch self {
        case .error:
            if upperText.contains("ERROR") || upperText.contains("FATAL") ||
               upperText.contains("CRITICAL") || upperText.contains("EXCEPTION") {
                return true
            }
            // Rails error patterns
            if text.contains("Completed 5") ||
               text.contains("ActionController::RoutingError") ||
               text.contains("ActiveRecord::") && upperText.contains("ERROR") {
                return true
            }
            return false

        case .warn:
            if upperText.contains("WARN") || upperText.contains("WARNING") ||
               upperText.contains("DEPRECAT") {
                return true
            }
            // Rails 4xx responses
            if text.contains("Completed 4") {
                return true
            }
            return false

        case .info:
            if upperText.contains("INFO") {
                return true
            }
            // Rails request lifecycle
            if text.contains("Started ") || text.contains("Processing by") ||
               text.contains("Completed 2") || text.contains("Completed 3") ||
               text.contains("Rendered ") {
                return true
            }
            return false

        case .debug:
            if upperText.contains("DEBUG") || upperText.contains("TRACE") {
                return true
            }
            // SQL queries, cache hits, etc.
            if text.contains("SELECT ") || text.contains("INSERT ") ||
               text.contains("UPDATE ") || text.contains("DELETE ") ||
               text.contains("Cache") {
                return true
            }
            return false
        }
    }
}
