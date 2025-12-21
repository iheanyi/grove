import XCTest
import SwiftUI
@testable import GroveMenubar

final class LogHighlightingTests: XCTestCase {

    // MARK: - Basic Highlighting

    func testHighlightEmptyString() {
        let result = LogHighlighter.highlight("")
        XCTAssertEqual(String(result.characters), "")
    }

    func testHighlightSimpleText() {
        let input = "Hello, World!"
        let result = LogHighlighter.highlight(input)
        XCTAssertEqual(String(result.characters), input)
    }

    // MARK: - Timestamp Highlighting

    func testHighlightISO8601Timestamp() {
        let input = "2025-01-15T10:30:00Z Started server"
        let result = LogHighlighter.highlight(input)
        // Should not crash and preserve content
        XCTAssertEqual(String(result.characters), input)
    }

    func testHighlightBracketedTimestamp() {
        let input = "[2025-01-15 10:30:00] Request processed"
        let result = LogHighlighter.highlight(input)
        XCTAssertEqual(String(result.characters), input)
    }

    // MARK: - HTTP Method Highlighting

    func testHighlightHTTPMethods() {
        let methods = ["GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"]
        for method in methods {
            let input = "Started \(method) /api/users"
            let result = LogHighlighter.highlight(input)
            XCTAssertEqual(String(result.characters), input)
        }
    }

    // MARK: - Log Level Highlighting

    func testHighlightLogLevels() {
        let levels = ["ERROR", "WARN", "WARNING", "INFO", "DEBUG", "TRACE", "FATAL", "CRITICAL"]
        for level in levels {
            let input = "[\(level)] Something happened"
            let result = LogHighlighter.highlight(input)
            XCTAssertEqual(String(result.characters), input)
        }
    }

    // MARK: - Duration Highlighting

    func testHighlightDurations() {
        let inputs = [
            "Request completed in 123ms",
            "Duration: 45.6ms",
            "Processing took 2.5s",
            "Operation: 100μs"
        ]
        for input in inputs {
            let result = LogHighlighter.highlight(input)
            XCTAssertEqual(String(result.characters), input)
        }
    }

    // MARK: - Rails Patterns

    func testHighlightRailsPatterns() {
        let inputs = [
            "Started GET \"/users\" for 127.0.0.1",
            "Processing by UsersController#index",
            "Completed 200 OK in 45ms",
            "Rendered users/index.html.erb",
            "ActiveRecord: 12.3ms",
            "Views: 23.4ms",
            "Allocations: 1234"
        ]
        for input in inputs {
            let result = LogHighlighter.highlight(input)
            XCTAssertEqual(String(result.characters), input)
        }
    }

    // MARK: - JSON Highlighting

    func testHighlightJSON() {
        let input = """
        {"user": "john", "id": 123, "active": true}
        """
        let result = LogHighlighter.highlight(input)
        XCTAssertEqual(String(result.characters), input)
    }

    // MARK: - Special Characters (Edge Cases)

    func testHighlightWithUnicodeCharacters() {
        let input = "User 日本語 logged in with emoji 🎉"
        let result = LogHighlighter.highlight(input)
        XCTAssertEqual(String(result.characters), input)
    }

    func testHighlightWithSpecialCharacters() {
        let input = "Path: /users/123?foo=bar&baz=qux#section"
        let result = LogHighlighter.highlight(input)
        XCTAssertEqual(String(result.characters), input)
    }

    func testHighlightVeryLongLine() {
        // Very long lines (>2000 chars) are returned as-is for performance
        let input = String(repeating: "x", count: 10000)
        let result = LogHighlighter.highlight(input)
        XCTAssertEqual(String(result.characters), input)
    }

    func testHighlightLineAtMaxLength() {
        // Lines at exactly max length should still be highlighted
        let input = "[INFO] " + String(repeating: "x", count: 1990)
        let result = LogHighlighter.highlight(input)
        XCTAssertEqual(String(result.characters), input)
    }

    func testHighlightPerformanceWithManyMatches() {
        // Test that lines with many potential matches don't hang
        let input = "key1=val key2=val key3=val " + String(repeating: "key=val ", count: 100)
        let result = LogHighlighter.highlight(input)
        XCTAssertEqual(String(result.characters), input)
    }

    // MARK: - Status Codes

    func testHighlightStatusCodes() {
        let inputs = [
            "Completed 200 OK",
            "HTTP 301 Redirect",
            "status 404 Not Found",
            "Completed 500 Internal Server Error"
        ]
        for input in inputs {
            let result = LogHighlighter.highlight(input)
            XCTAssertEqual(String(result.characters), input)
        }
    }
}
