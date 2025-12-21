import XCTest
import SwiftUI
@testable import GroveMenubar

final class ServerComputedPropertiesTests: XCTestCase {

    // MARK: - Uptime Formatting

    func testFormattedUptimeHoursMinutes() {
        let server = createServer(uptime: "2h34m12s")
        XCTAssertEqual(server.formattedUptime, "2h 34m")
    }

    func testFormattedUptimeMinutesOnly() {
        let server = createServer(uptime: "45m23s")
        XCTAssertEqual(server.formattedUptime, "45m")
    }

    func testFormattedUptimeSecondsOnly() {
        let server = createServer(uptime: "15s")
        XCTAssertEqual(server.formattedUptime, "15s")
    }

    func testFormattedUptimeNil() {
        let server = createServer(uptime: nil)
        XCTAssertNil(server.formattedUptime)
    }

    func testFormattedUptimeZero() {
        let server = createServer(uptime: "0s")
        XCTAssertEqual(server.formattedUptime, "0s")
    }

    func testFormattedUptimeLargeHours() {
        let server = createServer(uptime: "120h30m5s")
        XCTAssertEqual(server.formattedUptime, "120h 30m")
    }

    // MARK: - Status Icon

    func testStatusIconRunning() {
        let server = createServer(status: "running")
        XCTAssertEqual(server.statusIcon, "circle.fill")
    }

    func testStatusIconStopped() {
        let server = createServer(status: "stopped")
        XCTAssertEqual(server.statusIcon, "circle")
    }

    func testStatusIconCrashed() {
        let server = createServer(status: "crashed")
        XCTAssertEqual(server.statusIcon, "xmark.circle.fill")
    }

    func testStatusIconStarting() {
        let server = createServer(status: "starting")
        XCTAssertEqual(server.statusIcon, "circle.dotted")
    }

    func testStatusIconNil() {
        let server = createServer(status: nil)
        XCTAssertEqual(server.statusIcon, "circle")
    }

    // MARK: - Status Color

    func testStatusColorRunning() {
        let server = createServer(status: "running")
        XCTAssertEqual(server.statusColor, .green)
    }

    func testStatusColorStopped() {
        let server = createServer(status: "stopped")
        XCTAssertEqual(server.statusColor, .gray)
    }

    func testStatusColorCrashed() {
        let server = createServer(status: "crashed")
        XCTAssertEqual(server.statusColor, .red)
    }

    func testStatusColorStarting() {
        let server = createServer(status: "starting")
        XCTAssertEqual(server.statusColor, .yellow)
    }

    func testStatusColorNil() {
        let server = createServer(status: nil)
        XCTAssertEqual(server.statusColor, .gray)
    }

    // MARK: - Health Color

    func testHealthColorHealthy() {
        let server = createServer(health: "healthy")
        XCTAssertEqual(server.healthColor, .green)
    }

    func testHealthColorUnhealthy() {
        let server = createServer(health: "unhealthy")
        XCTAssertEqual(server.healthColor, .red)
    }

    func testHealthColorOther() {
        let server = createServer(health: "pending")
        XCTAssertEqual(server.healthColor, .yellow)
    }

    func testHealthColorNil() {
        let server = createServer(health: nil)
        XCTAssertEqual(server.healthColor, .gray)
    }

    // MARK: - isRunning

    func testIsRunningTrue() {
        XCTAssertTrue(createServer(status: "running").isRunning)
        XCTAssertTrue(createServer(status: "starting").isRunning)
    }

    func testIsRunningFalse() {
        XCTAssertFalse(createServer(status: "stopped").isRunning)
        XCTAssertFalse(createServer(status: "crashed").isRunning)
        XCTAssertFalse(createServer(status: nil).isRunning)
    }

    // MARK: - Display Properties

    func testDisplayURLWithURL() {
        let server = createServer(url: "https://myapp.localhost", port: 3000)
        XCTAssertEqual(server.displayURL, "https://myapp.localhost")
    }

    func testDisplayURLWithoutURL() {
        let server = createServer(url: nil, port: 3000)
        XCTAssertEqual(server.displayURL, "http://localhost:3000")
    }

    func testDisplayURLWithoutURLOrPort() {
        let server = createServer(url: nil, port: nil)
        XCTAssertEqual(server.displayURL, "http://localhost:0")
    }

    func testDisplayPort() {
        XCTAssertEqual(createServer(port: 3000).displayPort, 3000)
        XCTAssertEqual(createServer(port: nil).displayPort, 0)
    }

    func testDisplayStatus() {
        XCTAssertEqual(createServer(status: "running").displayStatus, "running")
        XCTAssertEqual(createServer(status: nil).displayStatus, "stopped")
    }

    // MARK: - GitHub CI Status

    func testCIStatusIcon() {
        XCTAssertEqual(GitHubInfo.CIStatus.success.icon, "checkmark.circle.fill")
        XCTAssertEqual(GitHubInfo.CIStatus.failure.icon, "xmark.circle.fill")
        XCTAssertEqual(GitHubInfo.CIStatus.pending.icon, "clock.fill")
        XCTAssertEqual(GitHubInfo.CIStatus.unknown.icon, "questionmark.circle")
    }

    func testCIStatusColor() {
        XCTAssertEqual(GitHubInfo.CIStatus.success.color, .green)
        XCTAssertEqual(GitHubInfo.CIStatus.failure.color, .red)
        XCTAssertEqual(GitHubInfo.CIStatus.pending.color, .yellow)
        XCTAssertEqual(GitHubInfo.CIStatus.unknown.color, .gray)
    }

    // MARK: - Helper

    private func createServer(
        name: String = "test",
        url: String? = nil,
        port: Int? = nil,
        status: String? = nil,
        health: String? = nil,
        uptime: String? = nil
    ) -> Server {
        return Server(
            name: name,
            url: url,
            subdomains: nil,
            port: port,
            status: status,
            health: health,
            path: "/test/path",
            branch: "main",
            uptime: uptime,
            pid: nil,
            logFile: nil,
            hasServer: true,
            hasClaude: false,
            hasVSCode: false,
            gitDirty: false,
            githubInfo: nil
        )
    }
}
