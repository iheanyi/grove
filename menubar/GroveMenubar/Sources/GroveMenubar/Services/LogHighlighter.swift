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

    // MARK: - Cached Regex Patterns (Performance Optimization)

    private static let timestampPatterns: [NSRegularExpression] = {
        [
            #"\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:?\d{2})?"#,
            #"\[\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}\]"#,
            #"\d{2}:\d{2}:\d{2}\.\d+"#
        ].compactMap { try? NSRegularExpression(pattern: $0) }
    }()

    private static let logLevelPatterns: [(NSRegularExpression, Color)] = {
        [
            (#"\b(ERROR|FATAL|CRITICAL)\b"#, colors.error),
            (#"\b(WARN|WARNING)\b"#, colors.warn),
            (#"\bINFO\b"#, colors.info),
            (#"\b(DEBUG|TRACE)\b"#, colors.debug),
            (#"\[error\]"#, colors.error),
            (#"\[warn(ing)?\]"#, colors.warn),
            (#"\[info\]"#, colors.info),
            (#"\[debug\]"#, colors.debug),
        ].compactMap { pattern, color -> (NSRegularExpression, Color)? in
            guard let regex = try? NSRegularExpression(pattern: pattern, options: .caseInsensitive) else { return nil }
            return (regex, color)
        }
    }()

    private static let httpMethodPatterns: [(NSRegularExpression, Color)] = {
        [
            (#"\bGET\b"#, colors.httpGet),
            (#"\bPOST\b"#, colors.httpPost),
            (#"\bPUT\b"#, colors.httpPut),
            (#"\bPATCH\b"#, colors.httpPatch),
            (#"\bDELETE\b"#, colors.httpDelete),
            (#"\bHEAD\b"#, colors.httpGet),
            (#"\bOPTIONS\b"#, colors.httpGet),
        ].compactMap { pattern, color -> (NSRegularExpression, Color)? in
            guard let regex = try? NSRegularExpression(pattern: pattern) else { return nil }
            return (regex, color)
        }
    }()

    private static let statusCodePatterns: [(NSRegularExpression, Color)] = {
        [
            (#"\b2\d{2}\b"#, colors.statusOk),
            (#"\b3\d{2}\b"#, colors.statusRedirect),
            (#"\b4\d{2}\b"#, colors.statusClientError),
            (#"\b5\d{2}\b"#, colors.statusServerError),
        ].compactMap { pattern, color -> (NSRegularExpression, Color)? in
            guard let regex = try? NSRegularExpression(pattern: pattern) else { return nil }
            return (regex, color)
        }
    }()

    private static let durationPatterns: [NSRegularExpression] = {
        [
            #"\d+\.?\d*\s*ms\b"#,
            #"\d+\.?\d*\s*s\b"#,
            #"\d+\.?\d*\s*μs\b"#,
            #"Duration:\s*\d+\.?\d*ms"#,
            #"in\s+\d+\.?\d*ms"#,
        ].compactMap { try? NSRegularExpression(pattern: $0) }
    }()

    private static let controllerRegex = try? NSRegularExpression(pattern: #"Processing by (\w+#\w+)"#)
    private static let startedRegex = try? NSRegularExpression(pattern: #"^Started\b"#)
    private static let completedRegex = try? NSRegularExpression(pattern: #"^Completed\b"#)
    private static let renderedRegex = try? NSRegularExpression(pattern: #"Rendered\s+[\w/]+\.[\w.]+"#)
    private static let activeRecordRegex = try? NSRegularExpression(pattern: #"ActiveRecord:\s*\d+\.?\d*ms"#)
    private static let viewsRegex = try? NSRegularExpression(pattern: #"Views:\s*\d+\.?\d*ms"#)
    private static let allocationsRegex = try? NSRegularExpression(pattern: #"Allocations:\s*\d+"#)
    private static let keyValueRegex = try? NSRegularExpression(pattern: #"(\w+)[=:]\s*"#)
    private static let jsonKeyRegex = try? NSRegularExpression(pattern: #""(\w+)":\s*"#)
    private static let jsonStringRegex = try? NSRegularExpression(pattern: #""[^"]*""#)
    private static let jsonNumberRegex = try? NSRegularExpression(pattern: #":\s*-?\d+\.?\d*[,\}\]]"#)
    private static let jsonBoolRegex = try? NSRegularExpression(pattern: #"\b(true|false|null)\b"#)

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

    // MARK: - Pattern Highlighters (Using Cached Regex)

    private static func highlightTimestamps(in result: inout AttributedString, line: String, nsRange: NSRange) {
        for regex in timestampPatterns {
            applyRegex(regex, color: colors.timestamp, in: &result, line: line, nsRange: nsRange)
        }
    }

    private static func highlightLogLevels(in result: inout AttributedString, line: String, nsRange: NSRange) {
        for (regex, color) in logLevelPatterns {
            applyRegex(regex, color: color, in: &result, line: line, nsRange: nsRange)
        }
    }

    private static func highlightHTTPMethods(in result: inout AttributedString, line: String, nsRange: NSRange) {
        for (regex, color) in httpMethodPatterns {
            applyRegex(regex, color: color, in: &result, line: line, nsRange: nsRange, bold: true)
        }
    }

    private static func highlightStatusCodes(in result: inout AttributedString, line: String, nsRange: NSRange) {
        // Only highlight if it looks like a status code context
        guard line.contains("Completed") || line.contains("HTTP") || line.contains("status") else { return }

        for (regex, color) in statusCodePatterns {
            applyRegex(regex, color: color, in: &result, line: line, nsRange: nsRange, bold: true)
        }
    }

    private static func highlightDurations(in result: inout AttributedString, line: String, nsRange: NSRange) {
        for regex in durationPatterns {
            applyRegex(regex, color: colors.duration, in: &result, line: line, nsRange: nsRange)
        }
    }

    private static func highlightRailsPatterns(in result: inout AttributedString, line: String, nsRange: NSRange) {
        // "Started GET" or "Started POST" etc
        if let regex = startedRegex {
            applyRegex(regex, color: colors.info, in: &result, line: line, nsRange: nsRange, bold: true)
        }

        // "Processing by Controller#action"
        if let regex = controllerRegex {
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
        if let regex = completedRegex {
            applyRegex(regex, color: colors.success, in: &result, line: line, nsRange: nsRange, bold: true)
        }

        // "Rendered view.html.erb"
        if let regex = renderedRegex {
            applyRegex(regex, color: colors.info, in: &result, line: line, nsRange: nsRange)
        }

        // ActiveRecord timing
        if let regex = activeRecordRegex {
            applyRegex(regex, color: colors.duration, in: &result, line: line, nsRange: nsRange)
        }

        // Views timing
        if let regex = viewsRegex {
            applyRegex(regex, color: colors.duration, in: &result, line: line, nsRange: nsRange)
        }

        // Allocations
        if let regex = allocationsRegex {
            applyRegex(regex, color: colors.number, in: &result, line: line, nsRange: nsRange)
        }
    }

    private static func highlightKeyValuePairs(in result: inout AttributedString, line: String, nsRange: NSRange) {
        guard let regex = keyValueRegex else { return }
        let matches = regex.matches(in: line, range: nsRange)
        for match in matches {
            if match.numberOfRanges > 1, let range = Range(match.range(at: 1), in: line) {
                if let attrRange = Range(range, in: result) {
                    result[attrRange].foregroundColor = colors.key
                }
            }
        }
    }

    private static func highlightJSON(in result: inout AttributedString, line: String, nsRange: NSRange) {
        // Only process if line contains JSON-like content
        guard line.contains("{") || line.contains("[") else { return }

        // JSON string values
        if let regex = jsonStringRegex {
            applyRegex(regex, color: colors.string, in: &result, line: line, nsRange: nsRange)
        }

        // JSON keys (before colon)
        if let regex = jsonKeyRegex {
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
        if let regex = jsonNumberRegex {
            applyRegex(regex, color: colors.number, in: &result, line: line, nsRange: nsRange)
        }

        // Booleans
        if let regex = jsonBoolRegex {
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
