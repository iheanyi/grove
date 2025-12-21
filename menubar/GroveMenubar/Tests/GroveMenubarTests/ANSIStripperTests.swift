import XCTest
@testable import GroveMenubar

final class ANSIStripperTests: XCTestCase {

    // MARK: - Basic Cases

    func testStripEmptyString() {
        XCTAssertEqual(ANSIStripper.strip(""), "")
    }

    func testStripPlainText() {
        let input = "Hello, World!"
        XCTAssertEqual(ANSIStripper.strip(input), input)
    }

    func testStripNoANSICodes() {
        let input = "Started GET \"/\" for 127.0.0.1 at 2025-01-15"
        XCTAssertEqual(ANSIStripper.strip(input), input)
    }

    // MARK: - Color Codes

    func testStripBoldCode() {
        let input = "\u{1B}[1mBold Text\u{1B}[0m"
        XCTAssertEqual(ANSIStripper.strip(input), "Bold Text")
    }

    func testStripColorCode() {
        let input = "\u{1B}[35mPurple Text\u{1B}[0m"
        XCTAssertEqual(ANSIStripper.strip(input), "Purple Text")
    }

    func testStripCombinedBoldAndColor() {
        let input = "\u{1B}[1m\u{1B}[35mBold Purple\u{1B}[0m"
        XCTAssertEqual(ANSIStripper.strip(input), "Bold Purple")
    }

    func testStripMultipleCodes() {
        let input = "\u{1B}[1m\u{1B}[36mActiveRecord\u{1B}[0m \u{1B}[1m\u{1B}[34mSELECT\u{1B}[0m"
        XCTAssertEqual(ANSIStripper.strip(input), "ActiveRecord SELECT")
    }

    // MARK: - Rails Log Format

    func testStripRailsSQLLog() {
        // Simulates: "  [1m[35m (1.4ms)[0m  [1m[34mSELECT * FROM users[0m"
        let input = "  \u{1B}[1m\u{1B}[35m (1.4ms)\u{1B}[0m  \u{1B}[1m\u{1B}[34mSELECT * FROM users\u{1B}[0m"
        let expected = "   (1.4ms)  SELECT * FROM users"
        XCTAssertEqual(ANSIStripper.strip(input), expected)
    }

    func testStripRailsActiveRecordLog() {
        let input = "\u{1B}[1m\u{1B}[36mActiveRecord::SchemaMigration Load (1.2ms)\u{1B}[0m  \u{1B}[1m\u{1B}[34mSELECT \"schema_migrations\".\"version\" FROM \"schema_migrations\"\u{1B}[0m"
        let expected = "ActiveRecord::SchemaMigration Load (1.2ms)  SELECT \"schema_migrations\".\"version\" FROM \"schema_migrations\""
        XCTAssertEqual(ANSIStripper.strip(input), expected)
    }

    // MARK: - Various ANSI Codes

    func testStripResetCode() {
        let input = "Text\u{1B}[0m More"
        XCTAssertEqual(ANSIStripper.strip(input), "Text More")
    }

    func testStripUnderlineCode() {
        let input = "\u{1B}[4mUnderlined\u{1B}[24m"
        XCTAssertEqual(ANSIStripper.strip(input), "Underlined")
    }

    func testStripBlinkCode() {
        let input = "\u{1B}[5mBlinking\u{1B}[25m"
        XCTAssertEqual(ANSIStripper.strip(input), "Blinking")
    }

    func testStripBackgroundColorCode() {
        let input = "\u{1B}[44mBlue Background\u{1B}[0m"
        XCTAssertEqual(ANSIStripper.strip(input), "Blue Background")
    }

    func testStrip256ColorCode() {
        let input = "\u{1B}[38;5;196mRed 256 color\u{1B}[0m"
        XCTAssertEqual(ANSIStripper.strip(input), "Red 256 color")
    }

    func testStripTrueColorCode() {
        let input = "\u{1B}[38;2;255;0;0mTrue red\u{1B}[0m"
        XCTAssertEqual(ANSIStripper.strip(input), "True red")
    }

    // MARK: - Edge Cases

    func testStripCodeAtStart() {
        let input = "\u{1B}[31mError: Something went wrong"
        XCTAssertEqual(ANSIStripper.strip(input), "Error: Something went wrong")
    }

    func testStripCodeAtEnd() {
        let input = "Text with trailing code\u{1B}[0m"
        XCTAssertEqual(ANSIStripper.strip(input), "Text with trailing code")
    }

    func testStripConsecutiveCodes() {
        let input = "\u{1B}[1m\u{1B}[4m\u{1B}[31mBold Underline Red\u{1B}[0m"
        XCTAssertEqual(ANSIStripper.strip(input), "Bold Underline Red")
    }

    func testPreserveSquareBracketsWithoutEscape() {
        // Regular square brackets should NOT be stripped
        let input = "[INFO] This is a log message [with brackets]"
        XCTAssertEqual(ANSIStripper.strip(input), input)
    }

    func testPreserveTimestamps() {
        let input = "[2025-01-15 10:30:00] Started GET \"/\""
        XCTAssertEqual(ANSIStripper.strip(input), input)
    }
}
