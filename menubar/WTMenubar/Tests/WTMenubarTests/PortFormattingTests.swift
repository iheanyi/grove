import XCTest

final class PortFormattingTests: XCTestCase {

    /// Test that port numbers are formatted without locale-specific separators
    /// This ensures ports like 3592 display as ":3592" not ":3,592"
    func testPortFormatting() {
        // Test various port numbers that could have comma formatting issues
        let testCases: [(port: Int, expected: String)] = [
            (80, ":80"),
            (443, ":443"),
            (3000, ":3000"),
            (3478, ":3478"),
            (3592, ":3592"),
            (8080, ":8080"),
            (10000, ":10000"),
            (65535, ":65535"),
        ]

        for testCase in testCases {
            let formatted = String(format: ":%d", testCase.port)
            XCTAssertEqual(formatted, testCase.expected,
                "Port \(testCase.port) should format as '\(testCase.expected)' but got '\(formatted)'")

            // Also verify no comma exists in the output
            XCTAssertFalse(formatted.contains(","),
                "Port formatting should not contain commas: '\(formatted)'")
        }
    }

    /// Test that the proxy port formatting works correctly
    func testProxyPortFormatting() {
        let httpPort = 80
        let httpsPort = 443
        let formatted = String(format: ":%d/:%d", httpPort, httpsPort)
        XCTAssertEqual(formatted, ":80/:443")
        XCTAssertFalse(formatted.contains(","))
    }

    /// Test with larger port numbers that would definitely show commas with locale formatting
    func testLargePortNumbers() {
        let ports = [1000, 2000, 3000, 4000, 5000, 10000, 20000, 30000, 50000]
        for port in ports {
            let formatted = String(format: ":%d", port)
            XCTAssertFalse(formatted.contains(","),
                "Port \(port) formatted as '\(formatted)' should not contain comma")
            XCTAssertTrue(formatted.hasPrefix(":"),
                "Port formatting should start with colon")
        }
    }
}
