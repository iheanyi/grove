import XCTest
@testable import GroveMenubar

final class LogStreamStateTests: XCTestCase {
    func testClearingInvalidatesInflightReadUntilTruncationCompletes() {
        var state = LogStreamState()
        state.position = 128
        let readGeneration = state.generation

        state.beginClear()

        XCTAssertTrue(state.isClearing)
        XCTAssertFalse(state.canApply(readGeneration))
        XCTAssertEqual(state.position, 128)

        state.finishClear()

        XCTAssertFalse(state.isClearing)
        XCTAssertEqual(state.position, 0)
        XCTAssertTrue(state.canApply(state.generation))
    }

    func testReadOffsetRestartsAfterLogCompaction() {
        var state = LogStreamState()
        state.position = 128

        XCTAssertEqual(state.readOffset(forFileSize: 64), 0)
        XCTAssertNil(state.readOffset(forFileSize: 128))
        XCTAssertEqual(state.readOffset(forFileSize: 160), 128)
    }
}
