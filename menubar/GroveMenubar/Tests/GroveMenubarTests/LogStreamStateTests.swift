import XCTest
@testable import GroveMenubar

final class LogStreamStateTests: XCTestCase {
    func testClearingInvalidatesInflightReadUntilTruncationCompletes() {
        var state = LogStreamState()
        state.position = 128
        let readGeneration = state.generation

        let clearGeneration = state.beginClear()

        XCTAssertTrue(state.isClearing)
        XCTAssertFalse(state.canApply(readGeneration))
        XCTAssertEqual(state.position, 128)

        XCTAssertTrue(state.finishClear(generation: clearGeneration))

        XCTAssertFalse(state.isClearing)
        XCTAssertEqual(state.position, 0)
        XCTAssertTrue(state.canApply(state.generation))
    }

    func testReadPlanReplacesDisplayedLinesAfterLogCompaction() {
        var state = LogStreamState()
        state.position = 128

        XCTAssertEqual(state.readPlan(forFileSize: 64), .replace)
        XCTAssertNil(state.readPlan(forFileSize: 128))
        XCTAssertEqual(state.readPlan(forFileSize: 160), .append(from: 128))
    }

    func testLateClearCompletionCannotMutateNewStream() {
        var state = LogStreamState()
        state.position = 128
        let clearGeneration = state.beginClear()

        state.reset()
        state.position = 64

        XCTAssertFalse(state.finishClear(generation: clearGeneration))
        XCTAssertEqual(state.position, 64)
        XCTAssertFalse(state.isClearing)
    }
}
