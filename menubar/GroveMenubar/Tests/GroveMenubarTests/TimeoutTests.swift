import XCTest
@testable import GroveMenubar

final class TimeoutTests: XCTestCase {

    // MARK: - Process Timeout Tests

    func testProcessCompletesBeforeTimeout() {
        let expectation = XCTestExpectation(description: "Process completes")

        let task = Process()
        task.executableURL = URL(fileURLWithPath: "/bin/echo")
        task.arguments = ["hello"]

        let pipe = Pipe()
        task.standardOutput = pipe

        var timedOut = false
        let timeoutWorkItem = DispatchWorkItem {
            timedOut = true
            if task.isRunning {
                task.terminate()
            }
        }

        do {
            try task.run()

            // Schedule timeout (1 second - should be plenty for echo)
            DispatchQueue.global().asyncAfter(deadline: .now() + 1.0, execute: timeoutWorkItem)

            task.waitUntilExit()
            timeoutWorkItem.cancel()

            XCTAssertFalse(timedOut, "Process should complete before timeout")
            XCTAssertEqual(task.terminationStatus, 0)

            let data = pipe.fileHandleForReading.readDataToEndOfFile()
            let output = String(data: data, encoding: .utf8)?.trimmingCharacters(in: .whitespacesAndNewlines)
            XCTAssertEqual(output, "hello")

            expectation.fulfill()
        } catch {
            XCTFail("Process failed to run: \(error)")
        }

        wait(for: [expectation], timeout: 2.0)
    }

    func testProcessTimesOut() {
        let expectation = XCTestExpectation(description: "Process times out")

        let task = Process()
        task.executableURL = URL(fileURLWithPath: "/bin/sleep")
        task.arguments = ["10"] // Sleep for 10 seconds

        var timedOut = false
        let timeoutWorkItem = DispatchWorkItem {
            timedOut = true
            if task.isRunning {
                task.terminate()
            }
        }

        do {
            try task.run()

            // Schedule short timeout (0.1 seconds)
            DispatchQueue.global().asyncAfter(deadline: .now() + 0.1, execute: timeoutWorkItem)

            task.waitUntilExit()
            timeoutWorkItem.cancel()

            XCTAssertTrue(timedOut, "Process should have timed out")
            expectation.fulfill()
        } catch {
            XCTFail("Process failed to run: \(error)")
        }

        wait(for: [expectation], timeout: 2.0)
    }

    // MARK: - Timeout Cancellation

    func testTimeoutCancelledWhenProcessCompletes() {
        let expectation = XCTestExpectation(description: "Timeout cancelled")

        var timeoutCalled = false

        let timeoutWorkItem = DispatchWorkItem {
            timeoutCalled = true
        }

        // Cancel immediately
        timeoutWorkItem.cancel()

        // Wait a bit and verify it wasn't called
        DispatchQueue.main.asyncAfter(deadline: .now() + 0.2) {
            XCTAssertFalse(timeoutCalled, "Cancelled work item should not execute")
            expectation.fulfill()
        }

        wait(for: [expectation], timeout: 1.0)
    }
}
