import SwiftUI

/// Highlights log lines with syntax coloring for Rails and structured logs
struct LogHighlighter {

    // MARK: - Configuration

    /// Maximum line length to highlight (longer lines are returned as-is for performance)
    private static let maxLineLength = 2000

    /// Maximum number of matches to highlight per pattern (prevents O(n²) on repetitive content)
    private static let maxMatchesPerPattern = 50

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

    // MARK: - Combined Regex Patterns (fewer regex runs = faster)

    /// Combined timestamp pattern (one regex instead of three)
    private static let timestampRegex = try? NSRegularExpression(
        pattern: #"\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:?\d{2})?|\[\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}\]|\d{2}:\d{2}:\d{2}\.\d+"#
    )

    /// Combined log level pattern (one regex instead of eight)
    private static let logLevelRegex = try? NSRegularExpression(
        pattern: #"\b(ERROR|FATAL|CRITICAL)\b|\b(WARN|WARNING)\b|\bINFO\b|\b(DEBUG|TRACE)\b|\[(error|warn(?:ing)?|info|debug)\]"#,
        options: .caseInsensitive
    )

    /// Combined HTTP method pattern
    private static let httpMethodRegex = try? NSRegularExpression(
        pattern: #"\b(GET|POST|PUT|PATCH|DELETE|HEAD|OPTIONS)\b"#
    )

    /// Combined duration pattern
    private static let durationRegex = try? NSRegularExpression(
        pattern: #"\d+\.?\d*\s*(ms|s|μs)\b|Duration:\s*\d+\.?\d*ms|in\s+\d+\.?\d*ms"#
    )

    /// Status code pattern (only used when context suggests status codes)
    private static let statusCodeRegex = try? NSRegularExpression(
        pattern: #"\b([2345]\d{2})\b"#
    )

    // Rails-specific patterns
    private static let controllerRegex = try? NSRegularExpression(pattern: #"Processing by (\w+#\w+)"#)
    private static let startedRegex = try? NSRegularExpression(pattern: #"^Started\b"#)
    private static let completedRegex = try? NSRegularExpression(pattern: #"^Completed\b"#)
    private static let renderedRegex = try? NSRegularExpression(pattern: #"Rendered\s+[\w/]+\.[\w.]+"#)
    private static let railsTimingRegex = try? NSRegularExpression(pattern: #"(ActiveRecord|Views|Allocations):\s*\d+\.?\d*(ms)?"#)

    // JSON/key-value patterns
    private static let keyValueRegex = try? NSRegularExpression(pattern: #"(\w+)[=:]\s*"#)
    private static let jsonKeyRegex = try? NSRegularExpression(pattern: #""(\w+)":\s*"#)
    private static let jsonStringRegex = try? NSRegularExpression(pattern: #""[^"]{0,200}""#) // Limit string length
    private static let jsonPrimitivesRegex = try? NSRegularExpression(pattern: #"\b(true|false|null|-?\d+\.?\d*)\b"#)

    // MARK: - Main Highlight Function

    static func highlight(_ line: String) -> AttributedString {
        // Fast path: skip very long lines (likely data dumps, not readable logs)
        guard line.count <= maxLineLength else {
            return AttributedString(line)
        }

        // Fast path: empty or very short lines
        guard line.count > 2 else {
            return AttributedString(line)
        }

        var result = AttributedString(line)
        let nsRange = NSRange(location: 0, length: line.utf16.count)

        // Apply highlights in order (later ones override earlier)
        // Use early-exit checks to skip unnecessary regex runs

        // Timestamps (only if line likely contains one)
        if lineContainsDigits(line) {
            highlightTimestamps(in: &result, line: line, nsRange: nsRange)
            highlightDurations(in: &result, line: line, nsRange: nsRange)
        }

        // Log levels (only if line contains bracket or uppercase)
        if line.contains("[") || containsUppercase(line) {
            highlightLogLevels(in: &result, line: line, nsRange: nsRange)
        }

        // HTTP methods (only if line might contain them)
        if containsHTTPMethodCandidate(line) {
            highlightHTTPMethods(in: &result, line: line, nsRange: nsRange)
        }

        // Status codes (only in specific contexts)
        if line.contains("Completed") || line.contains("HTTP") || line.contains("status") {
            highlightStatusCodes(in: &result, line: line, nsRange: nsRange)
        }

        // Rails patterns (only if line starts with Rails keywords)
        if line.hasPrefix("Started") || line.hasPrefix("Completed") ||
           line.contains("Processing by") || line.contains("Rendered") ||
           line.contains("ActiveRecord") || line.contains("Views:") || line.contains("Allocations:") {
            highlightRailsPatterns(in: &result, line: line, nsRange: nsRange)
        }

        // Key-value pairs (only if line contains = or :)
        if line.contains("=") || line.contains(":") {
            highlightKeyValuePairs(in: &result, line: line, nsRange: nsRange)
        }

        // JSON (only if line contains braces)
        if line.contains("{") || line.contains("[") {
            highlightJSON(in: &result, line: line, nsRange: nsRange)
        }

        return result
    }

    // MARK: - Fast Character Checks

    @inline(__always)
    private static func lineContainsDigits(_ line: String) -> Bool {
        line.contains { $0.isNumber }
    }

    @inline(__always)
    private static func containsUppercase(_ line: String) -> Bool {
        // Check first 100 chars only for performance
        let prefix = line.prefix(100)
        return prefix.contains { $0.isUppercase }
    }

    @inline(__always)
    private static func containsHTTPMethodCandidate(_ line: String) -> Bool {
        // Quick check for common HTTP method starting letters
        let upper = line.uppercased()
        return upper.contains("GET") || upper.contains("POST") || upper.contains("PUT") ||
               upper.contains("DELETE") || upper.contains("PATCH")
    }

    // MARK: - Pattern Highlighters

    private static func highlightTimestamps(in result: inout AttributedString, line: String, nsRange: NSRange) {
        guard let regex = timestampRegex else { return }
        applyRegex(regex, color: colors.timestamp, in: &result, line: line, nsRange: nsRange)
    }

    private static func highlightLogLevels(in result: inout AttributedString, line: String, nsRange: NSRange) {
        guard let regex = logLevelRegex else { return }

        let matches = regex.matches(in: line, options: [], range: nsRange)
        for match in matches.prefix(maxMatchesPerPattern) {
            guard let range = Range(match.range, in: line),
                  let attrRange = Range(range, in: result) else { continue }

            let matchedText = String(line[range]).uppercased()

            // Determine color based on matched text
            let color: Color
            if matchedText.contains("ERROR") || matchedText.contains("FATAL") || matchedText.contains("CRITICAL") {
                color = colors.error
            } else if matchedText.contains("WARN") {
                color = colors.warn
            } else if matchedText.contains("INFO") {
                color = colors.info
            } else {
                color = colors.debug
            }

            result[attrRange].foregroundColor = color
        }
    }

    private static func highlightHTTPMethods(in result: inout AttributedString, line: String, nsRange: NSRange) {
        guard let regex = httpMethodRegex else { return }

        let matches = regex.matches(in: line, options: [], range: nsRange)
        for match in matches.prefix(maxMatchesPerPattern) {
            guard let range = Range(match.range, in: line),
                  let attrRange = Range(range, in: result) else { continue }

            let method = String(line[range])
            let color: Color
            switch method {
            case "GET", "HEAD", "OPTIONS": color = colors.httpGet
            case "POST": color = colors.httpPost
            case "PUT": color = colors.httpPut
            case "PATCH": color = colors.httpPatch
            case "DELETE": color = colors.httpDelete
            default: color = colors.httpGet
            }

            result[attrRange].foregroundColor = color
            result[attrRange].font = .system(.body, design: .monospaced).bold()
        }
    }

    private static func highlightStatusCodes(in result: inout AttributedString, line: String, nsRange: NSRange) {
        guard let regex = statusCodeRegex else { return }

        let matches = regex.matches(in: line, options: [], range: nsRange)
        for match in matches.prefix(maxMatchesPerPattern) {
            guard let range = Range(match.range, in: line),
                  let attrRange = Range(range, in: result),
                  let code = Int(line[range]) else { continue }

            let color: Color
            switch code {
            case 200..<300: color = colors.statusOk
            case 300..<400: color = colors.statusRedirect
            case 400..<500: color = colors.statusClientError
            case 500..<600: color = colors.statusServerError
            default: continue
            }

            result[attrRange].foregroundColor = color
            result[attrRange].font = .system(.body, design: .monospaced).bold()
        }
    }

    private static func highlightDurations(in result: inout AttributedString, line: String, nsRange: NSRange) {
        guard let regex = durationRegex else { return }
        applyRegex(regex, color: colors.duration, in: &result, line: line, nsRange: nsRange)
    }

    private static func highlightRailsPatterns(in result: inout AttributedString, line: String, nsRange: NSRange) {
        // "Started GET" - use firstMatch since it only appears once
        if let regex = startedRegex, let match = regex.firstMatch(in: line, options: [], range: nsRange) {
            if let range = Range(match.range, in: line), let attrRange = Range(range, in: result) {
                result[attrRange].foregroundColor = colors.info
                result[attrRange].font = .system(.body, design: .monospaced).bold()
            }
        }

        // "Processing by Controller#action"
        if let regex = controllerRegex, let match = regex.firstMatch(in: line, options: [], range: nsRange) {
            if match.numberOfRanges > 1, let range = Range(match.range(at: 1), in: line) {
                if let attrRange = Range(range, in: result) {
                    result[attrRange].foregroundColor = colors.controller
                    result[attrRange].font = .system(.body, design: .monospaced).bold()
                }
            }
        }

        // "Completed 200 OK"
        if let regex = completedRegex, let match = regex.firstMatch(in: line, options: [], range: nsRange) {
            if let range = Range(match.range, in: line), let attrRange = Range(range, in: result) {
                result[attrRange].foregroundColor = colors.success
                result[attrRange].font = .system(.body, design: .monospaced).bold()
            }
        }

        // "Rendered view.html.erb"
        if let regex = renderedRegex, let match = regex.firstMatch(in: line, options: [], range: nsRange) {
            if let range = Range(match.range, in: line), let attrRange = Range(range, in: result) {
                result[attrRange].foregroundColor = colors.info
            }
        }

        // Rails timing (ActiveRecord, Views, Allocations)
        if let regex = railsTimingRegex {
            applyRegex(regex, color: colors.duration, in: &result, line: line, nsRange: nsRange)
        }
    }

    private static func highlightKeyValuePairs(in result: inout AttributedString, line: String, nsRange: NSRange) {
        guard let regex = keyValueRegex else { return }

        let matches = regex.matches(in: line, options: [], range: nsRange)
        for match in matches.prefix(maxMatchesPerPattern) {
            if match.numberOfRanges > 1, let range = Range(match.range(at: 1), in: line) {
                if let attrRange = Range(range, in: result) {
                    result[attrRange].foregroundColor = colors.key
                }
            }
        }
    }

    private static func highlightJSON(in result: inout AttributedString, line: String, nsRange: NSRange) {
        // JSON string values (limited length to prevent slow regex on large strings)
        if let regex = jsonStringRegex {
            applyRegex(regex, color: colors.string, in: &result, line: line, nsRange: nsRange)
        }

        // JSON keys
        if let regex = jsonKeyRegex {
            let matches = regex.matches(in: line, options: [], range: nsRange)
            for match in matches.prefix(maxMatchesPerPattern) {
                if match.numberOfRanges > 1, let range = Range(match.range(at: 1), in: line) {
                    if let attrRange = Range(range, in: result) {
                        result[attrRange].foregroundColor = colors.key
                    }
                }
            }
        }

        // JSON primitives (numbers, booleans, null)
        if let regex = jsonPrimitivesRegex {
            applyRegex(regex, color: colors.number, in: &result, line: line, nsRange: nsRange)
        }
    }

    // MARK: - Helper

    private static func applyRegex(
        _ regex: NSRegularExpression,
        color: Color,
        in result: inout AttributedString,
        line: String,
        nsRange: NSRange,
        bold: Bool = false
    ) {
        let matches = regex.matches(in: line, options: [], range: nsRange)
        for match in matches.prefix(maxMatchesPerPattern) {
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
