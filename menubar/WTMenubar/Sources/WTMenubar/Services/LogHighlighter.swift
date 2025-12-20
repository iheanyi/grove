import SwiftUI

/// Highlights log lines with syntax coloring for Rails and structured logs
struct LogHighlighter {

    // MARK: - Colors

    static let colors = LogColors()

    struct LogColors {
        let error = Color.red
        let warn = Color.orange
        let info = Color.blue
        let debug = Color.gray
        let success = Color.green

        let httpGet = Color.green
        let httpPost = Color.blue
        let httpPut = Color.orange
        let httpPatch = Color.yellow
        let httpDelete = Color.red

        let timestamp = Color.gray
        let duration = Color.purple
        let number = Color.cyan
        let string = Color.green
        let key = Color.blue
        let controller = Color.yellow
        let statusOk = Color.green
        let statusRedirect = Color.yellow
        let statusClientError = Color.orange
        let statusServerError = Color.red
    }

    // MARK: - Main Highlight Function

    static func highlight(_ line: String) -> AttributedString {
        var result = AttributedString(line)
        let nsRange = NSRange(location: 0, length: line.utf16.count)

        // Apply highlights in order (later ones override earlier)
        highlightTimestamps(in: &result, line: line, nsRange: nsRange)
        highlightLogLevels(in: &result, line: line, nsRange: nsRange)
        highlightHTTPMethods(in: &result, line: line, nsRange: nsRange)
        highlightStatusCodes(in: &result, line: line, nsRange: nsRange)
        highlightDurations(in: &result, line: line, nsRange: nsRange)
        highlightRailsPatterns(in: &result, line: line, nsRange: nsRange)
        highlightKeyValuePairs(in: &result, line: line, nsRange: nsRange)
        highlightJSON(in: &result, line: line, nsRange: nsRange)

        return result
    }

    // MARK: - Pattern Highlighters

    private static func highlightTimestamps(in result: inout AttributedString, line: String, nsRange: NSRange) {
        // ISO 8601: 2025-01-15T10:30:15Z
        // Rails: 2025-01-15 10:30:15 -0500
        // Common: [2025-01-15 10:30:15]
        let patterns = [
            #"\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:?\d{2})?"#,
            #"\[\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}\]"#,
            #"\d{2}:\d{2}:\d{2}\.\d+"#
        ]

        for pattern in patterns {
            applyColor(colors.timestamp, pattern: pattern, in: &result, line: line)
        }
    }

    private static func highlightLogLevels(in result: inout AttributedString, line: String, nsRange: NSRange) {
        let levels: [(pattern: String, color: Color)] = [
            (#"\b(ERROR|FATAL|CRITICAL)\b"#, colors.error),
            (#"\b(WARN|WARNING)\b"#, colors.warn),
            (#"\bINFO\b"#, colors.info),
            (#"\b(DEBUG|TRACE)\b"#, colors.debug),
            (#"\[error\]"#, colors.error),
            (#"\[warn(ing)?\]"#, colors.warn),
            (#"\[info\]"#, colors.info),
            (#"\[debug\]"#, colors.debug),
        ]

        for (pattern, color) in levels {
            applyColor(color, pattern: pattern, in: &result, line: line, options: .caseInsensitive)
        }
    }

    private static func highlightHTTPMethods(in result: inout AttributedString, line: String, nsRange: NSRange) {
        let methods: [(pattern: String, color: Color)] = [
            (#"\bGET\b"#, colors.httpGet),
            (#"\bPOST\b"#, colors.httpPost),
            (#"\bPUT\b"#, colors.httpPut),
            (#"\bPATCH\b"#, colors.httpPatch),
            (#"\bDELETE\b"#, colors.httpDelete),
            (#"\bHEAD\b"#, colors.httpGet),
            (#"\bOPTIONS\b"#, colors.httpGet),
        ]

        for (pattern, color) in methods {
            applyColor(color, pattern: pattern, in: &result, line: line, bold: true)
        }
    }

    private static func highlightStatusCodes(in result: inout AttributedString, line: String, nsRange: NSRange) {
        // HTTP status codes
        let statuses: [(pattern: String, color: Color)] = [
            (#"\b2\d{2}\b"#, colors.statusOk),           // 2xx Success
            (#"\b3\d{2}\b"#, colors.statusRedirect),     // 3xx Redirect
            (#"\b4\d{2}\b"#, colors.statusClientError),  // 4xx Client Error
            (#"\b5\d{2}\b"#, colors.statusServerError),  // 5xx Server Error
        ]

        // Only highlight if it looks like a status code context
        if line.contains("Completed") || line.contains("HTTP") || line.contains("status") {
            for (pattern, color) in statuses {
                applyColor(color, pattern: pattern, in: &result, line: line, bold: true)
            }
        }
    }

    private static func highlightDurations(in result: inout AttributedString, line: String, nsRange: NSRange) {
        // Match durations: 12.3ms, 1.5s, 100μs
        let patterns = [
            #"\d+\.?\d*\s*ms\b"#,
            #"\d+\.?\d*\s*s\b"#,
            #"\d+\.?\d*\s*μs\b"#,
            #"Duration:\s*\d+\.?\d*ms"#,
            #"in\s+\d+\.?\d*ms"#,
        ]

        for pattern in patterns {
            applyColor(colors.duration, pattern: pattern, in: &result, line: line)
        }
    }

    private static func highlightRailsPatterns(in result: inout AttributedString, line: String, nsRange: NSRange) {
        // "Started GET" or "Started POST" etc
        applyColor(colors.info, pattern: #"^Started\b"#, in: &result, line: line, bold: true)

        // "Processing by Controller#action"
        if let regex = try? NSRegularExpression(pattern: #"Processing by (\w+#\w+)"#) {
            let matches = regex.matches(in: line, range: nsRange)
            for match in matches {
                if match.numberOfRanges > 1, let range = Range(match.range(at: 1), in: line) {
                    if let attrRange = Range(range, in: result) {
                        result[attrRange].foregroundColor = colors.controller
                        result[attrRange].font = .system(.body, design: .monospaced).bold()
                    }
                }
            }
        }

        // "Completed 200 OK"
        applyColor(colors.success, pattern: #"^Completed\b"#, in: &result, line: line, bold: true)

        // "Rendered view.html.erb"
        applyColor(colors.info, pattern: #"Rendered\s+[\w/]+\.[\w.]+"#, in: &result, line: line)

        // ActiveRecord timing
        applyColor(colors.duration, pattern: #"ActiveRecord:\s*\d+\.?\d*ms"#, in: &result, line: line)

        // Views timing
        applyColor(colors.duration, pattern: #"Views:\s*\d+\.?\d*ms"#, in: &result, line: line)

        // Allocations
        applyColor(colors.number, pattern: #"Allocations:\s*\d+"#, in: &result, line: line)
    }

    private static func highlightKeyValuePairs(in result: inout AttributedString, line: String, nsRange: NSRange) {
        // key=value or key: value patterns
        if let regex = try? NSRegularExpression(pattern: #"(\w+)[=:]\s*"#) {
            let matches = regex.matches(in: line, range: nsRange)
            for match in matches {
                if match.numberOfRanges > 1, let range = Range(match.range(at: 1), in: line) {
                    if let attrRange = Range(range, in: result) {
                        result[attrRange].foregroundColor = colors.key
                    }
                }
            }
        }
    }

    private static func highlightJSON(in result: inout AttributedString, line: String, nsRange: NSRange) {
        // Highlight JSON keys (simple detection)
        if line.contains("{") || line.contains("[") {
            // JSON string values
            applyColor(colors.string, pattern: #""[^"]*""#, in: &result, line: line)

            // JSON keys (before colon)
            if let regex = try? NSRegularExpression(pattern: #""(\w+)":\s*"#) {
                let matches = regex.matches(in: line, range: nsRange)
                for match in matches {
                    if match.numberOfRanges > 1, let range = Range(match.range(at: 1), in: line) {
                        if let attrRange = Range(range, in: result) {
                            result[attrRange].foregroundColor = colors.key
                        }
                    }
                }
            }

            // Numbers in JSON
            applyColor(colors.number, pattern: #":\s*-?\d+\.?\d*[,\}\]]"#, in: &result, line: line)

            // Booleans
            applyColor(colors.number, pattern: #"\b(true|false|null)\b"#, in: &result, line: line)
        }
    }

    // MARK: - Helper

    private static func applyColor(
        _ color: Color,
        pattern: String,
        in result: inout AttributedString,
        line: String,
        options: NSRegularExpression.Options = [],
        bold: Bool = false
    ) {
        guard let regex = try? NSRegularExpression(pattern: pattern, options: options) else { return }
        let nsRange = NSRange(location: 0, length: line.utf16.count)
        let matches = regex.matches(in: line, range: nsRange)

        for match in matches {
            if let range = Range(match.range, in: line),
               let attrRange = Range(range, in: result) {
                result[attrRange].foregroundColor = color
                if bold {
                    result[attrRange].font = .system(.body, design: .monospaced).bold()
                }
            }
        }
    }
}
