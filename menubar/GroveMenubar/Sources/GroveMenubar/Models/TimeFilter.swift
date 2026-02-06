import Foundation

/// Time-based filter options for log viewing
enum TimeFilter: String, CaseIterable, Identifiable {
    case all = "All Time"
    case lastMinute = "Last 1 min"
    case last5Minutes = "Last 5 min"
    case last15Minutes = "Last 15 min"
    case last30Minutes = "Last 30 min"
    case lastHour = "Last 1 hour"
    case sinceRestart = "Since restart"

    var id: String { rawValue }

    /// Returns the cutoff date for this filter, or nil for "all time"
    func cutoffDate(serverUptime: String?) -> Date? {
        let now = Date()
        switch self {
        case .all:
            return nil
        case .lastMinute:
            return now.addingTimeInterval(-60)
        case .last5Minutes:
            return now.addingTimeInterval(-5 * 60)
        case .last15Minutes:
            return now.addingTimeInterval(-15 * 60)
        case .last30Minutes:
            return now.addingTimeInterval(-30 * 60)
        case .lastHour:
            return now.addingTimeInterval(-60 * 60)
        case .sinceRestart:
            guard let uptime = serverUptime else { return nil }
            let seconds = TimeFilter.parseUptimeToSeconds(uptime)
            guard seconds > 0 else { return nil }
            return now.addingTimeInterval(-Double(seconds))
        }
    }

    /// Parse an uptime string like "2h34m12s" into total seconds
    static func parseUptimeToSeconds(_ uptime: String) -> Int {
        var hours = 0
        var minutes = 0
        var seconds = 0
        var currentNumber = ""

        for char in uptime {
            if char.isNumber {
                currentNumber.append(char)
            } else if let value = Int(currentNumber) {
                switch char {
                case "h": hours = value
                case "m": minutes = value
                case "s": seconds = value
                default: break
                }
                currentNumber = ""
            }
        }

        return hours * 3600 + minutes * 60 + seconds
    }

    /// Attempt to parse a timestamp from a log line
    static func parseTimestamp(from line: String) -> Date? {
        // Try ISO 8601 first: 2024-01-15T14:30:22.123Z or 2024-01-15T14:30:22+00:00
        if let date = iso8601Formatter.date(from: String(line.prefix(30).trimmingCharacters(in: .whitespaces))) {
            return date
        }

        // Try to extract timestamp patterns from the line
        let patterns: [(NSRegularExpression, (String) -> Date?)] = [
            (iso8601Regex, { parseISO8601($0) }),
            (railsTimestampRegex, { parseRailsTimestamp($0) }),
            (timeOnlyRegex, { parseTimeOnly($0) }),
        ]

        for (regex, parser) in patterns {
            let nsRange = NSRange(location: 0, length: min(line.utf16.count, 60))
            if let match = regex.firstMatch(in: line, options: [], range: nsRange),
               let range = Range(match.range, in: line) {
                let matched = String(line[range])
                if let date = parser(matched) {
                    return date
                }
            }
        }

        return nil
    }

    // MARK: - Regex Patterns

    private static let iso8601Regex: NSRegularExpression = {
        try! NSRegularExpression(pattern: #"\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:?\d{2})?"#)
    }()

    private static let railsTimestampRegex: NSRegularExpression = {
        try! NSRegularExpression(pattern: #"\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}"#)
    }()

    private static let timeOnlyRegex: NSRegularExpression = {
        try! NSRegularExpression(pattern: #"\d{2}:\d{2}:\d{2}(?:\.\d+)?"#)
    }()

    // MARK: - Parsers

    private static let iso8601Formatter: ISO8601DateFormatter = {
        let f = ISO8601DateFormatter()
        f.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        return f
    }()

    private static let railsFormatter: DateFormatter = {
        let f = DateFormatter()
        f.dateFormat = "yyyy-MM-dd HH:mm:ss"
        f.locale = Locale(identifier: "en_US_POSIX")
        return f
    }()

    private static let timeOnlyFormatter: DateFormatter = {
        let f = DateFormatter()
        f.dateFormat = "HH:mm:ss"
        f.locale = Locale(identifier: "en_US_POSIX")
        return f
    }()

    private static func parseISO8601(_ string: String) -> Date? {
        // Try with fractional seconds
        if let date = iso8601Formatter.date(from: string) {
            return date
        }
        // Try without fractional seconds
        let f = ISO8601DateFormatter()
        f.formatOptions = [.withInternetDateTime]
        return f.date(from: string)
    }

    private static func parseRailsTimestamp(_ string: String) -> Date? {
        railsFormatter.date(from: string)
    }

    private static func parseTimeOnly(_ string: String) -> Date? {
        // For time-only, assume today's date
        let timeStr = string.contains(".") ? String(string.prefix(8)) : string
        guard let time = timeOnlyFormatter.date(from: timeStr) else { return nil }

        let calendar = Calendar.current
        let now = Date()
        let timeComponents = calendar.dateComponents([.hour, .minute, .second], from: time)
        var todayComponents = calendar.dateComponents([.year, .month, .day], from: now)
        todayComponents.hour = timeComponents.hour
        todayComponents.minute = timeComponents.minute
        todayComponents.second = timeComponents.second

        return calendar.date(from: todayComponents)
    }
}
