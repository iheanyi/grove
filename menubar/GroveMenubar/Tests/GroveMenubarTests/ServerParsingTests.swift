import XCTest
@testable import GroveMenubar

final class ServerParsingTests: XCTestCase {

    // MARK: - Full Server with All Fields

    func testParseFullServerWithAllFields() throws {
        let json = """
        {
            "worktrees": [
                {
                    "name": "feature-auth",
                    "path": "/Users/dev/myapp-feature-auth",
                    "branch": "feature/auth",
                    "url": "http://localhost:3042",
                    "port": 3042,
                    "status": "running",
                    "has_server": true,
                    "has_claude": true,
                    "has_vscode": false,
                    "git_dirty": true,
                    "pid": 12345,
                    "uptime": "2h34m"
                }
            ],
            "url_mode": "port"
        }
        """

        let data = json.data(using: .utf8)!
        let status = try JSONDecoder().decode(WTStatus.self, from: data)

        XCTAssertEqual(status.worktrees.count, 1)
        XCTAssertEqual(status.urlMode, "port")

        let server = status.servers.first!
        XCTAssertEqual(server.name, "feature-auth")
        XCTAssertEqual(server.path, "/Users/dev/myapp-feature-auth")
        XCTAssertEqual(server.branch, "feature/auth")
        XCTAssertEqual(server.url, "http://localhost:3042")
        XCTAssertEqual(server.port, 3042)
        XCTAssertEqual(server.status, "running")
        XCTAssertEqual(server.hasServer, true)
        XCTAssertEqual(server.hasClaude, true)
        XCTAssertEqual(server.hasVSCode, false)
        XCTAssertEqual(server.gitDirty, true)
        XCTAssertEqual(server.pid, 12345)
        XCTAssertEqual(server.uptime, "2h34m")
        XCTAssertTrue(server.isRunning)
        XCTAssertEqual(server.displayPort, 3042)
        XCTAssertEqual(server.displayURL, "http://localhost:3042")
        XCTAssertEqual(server.displayStatus, "running")
    }

    // MARK: - Worktree Without Server (Critical Test Case)

    func testParseWorktreeWithoutServer() throws {
        // This is the critical test case that was causing freezes:
        // Worktrees without servers don't have url, port, or status fields
        let json = """
        {
            "worktrees": [
                {
                    "name": "some-worktree",
                    "path": "/Users/dev/myapp-worktree",
                    "branch": "main",
                    "has_server": false,
                    "has_claude": false,
                    "has_vscode": true,
                    "git_dirty": false
                }
            ],
            "url_mode": "port"
        }
        """

        let data = json.data(using: .utf8)!
        let status = try JSONDecoder().decode(WTStatus.self, from: data)

        XCTAssertEqual(status.worktrees.count, 1)

        let server = status.servers.first!
        XCTAssertEqual(server.name, "some-worktree")
        XCTAssertEqual(server.path, "/Users/dev/myapp-worktree")
        XCTAssertEqual(server.branch, "main")
        XCTAssertNil(server.url, "url should be nil for worktrees without servers")
        XCTAssertNil(server.port, "port should be nil for worktrees without servers")
        XCTAssertNil(server.status, "status should be nil for worktrees without servers")
        XCTAssertEqual(server.hasServer, false)
        XCTAssertEqual(server.hasClaude, false)
        XCTAssertEqual(server.hasVSCode, true)
        XCTAssertEqual(server.gitDirty, false)
        XCTAssertFalse(server.isRunning)
        XCTAssertEqual(server.displayPort, 0, "displayPort should default to 0")
        XCTAssertEqual(server.displayStatus, "stopped", "displayStatus should default to 'stopped'")
    }

    // MARK: - Mixed Worktrees (With and Without Servers)

    func testParseMixedWorktrees() throws {
        let json = """
        {
            "worktrees": [
                {
                    "name": "main",
                    "path": "/Users/dev/myapp",
                    "branch": "main",
                    "url": "http://localhost:3000",
                    "port": 3000,
                    "status": "running",
                    "has_server": true,
                    "has_claude": false,
                    "has_vscode": false,
                    "git_dirty": false,
                    "pid": 1234
                },
                {
                    "name": "feature-branch",
                    "path": "/Users/dev/myapp-feature",
                    "branch": "feature/test",
                    "has_server": false,
                    "has_claude": true,
                    "has_vscode": false,
                    "git_dirty": true
                },
                {
                    "name": "stopped-server",
                    "path": "/Users/dev/myapp-stopped",
                    "branch": "develop",
                    "url": "http://localhost:3001",
                    "port": 3001,
                    "status": "stopped",
                    "has_server": true,
                    "has_claude": false,
                    "has_vscode": true,
                    "git_dirty": false
                }
            ],
            "url_mode": "port"
        }
        """

        let data = json.data(using: .utf8)!
        let status = try JSONDecoder().decode(WTStatus.self, from: data)

        XCTAssertEqual(status.worktrees.count, 3)

        // Running server
        let running = status.servers.first { $0.name == "main" }!
        XCTAssertTrue(running.isRunning)
        XCTAssertEqual(running.port, 3000)

        // Worktree without server
        let noServer = status.servers.first { $0.name == "feature-branch" }!
        XCTAssertNil(noServer.port)
        XCTAssertNil(noServer.status)
        XCTAssertFalse(noServer.isRunning)

        // Stopped server
        let stopped = status.servers.first { $0.name == "stopped-server" }!
        XCTAssertEqual(stopped.status, "stopped")
        XCTAssertFalse(stopped.isRunning)
    }

    // MARK: - Subdomain Mode with Proxy

    func testParseSubdomainModeWithProxy() throws {
        let json = """
        {
            "worktrees": [
                {
                    "name": "feature-auth",
                    "path": "/Users/dev/myapp",
                    "branch": "feature/auth",
                    "url": "https://feature-auth.localhost",
                    "subdomains": "https://*.feature-auth.localhost",
                    "port": 3042,
                    "status": "running",
                    "has_server": true,
                    "has_claude": false,
                    "has_vscode": false,
                    "git_dirty": false
                }
            ],
            "proxy": {
                "status": "running",
                "http_port": 80,
                "https_port": 443,
                "pid": 5678
            },
            "url_mode": "subdomain"
        }
        """

        let data = json.data(using: .utf8)!
        let status = try JSONDecoder().decode(WTStatus.self, from: data)

        XCTAssertEqual(status.urlMode, "subdomain")
        XCTAssertTrue(status.isSubdomainMode)
        XCTAssertFalse(status.isPortMode)

        XCTAssertNotNil(status.proxy)
        XCTAssertEqual(status.proxy?.status, "running")
        XCTAssertEqual(status.proxy?.httpPort, 80)
        XCTAssertEqual(status.proxy?.httpsPort, 443)
        XCTAssertEqual(status.proxy?.pid, 5678)
        XCTAssertTrue(status.proxy?.isRunning ?? false)

        let server = status.servers.first!
        XCTAssertEqual(server.subdomains, "https://*.feature-auth.localhost")
    }

    // MARK: - Empty Worktrees

    func testParseEmptyWorktrees() throws {
        let json = """
        {
            "worktrees": [],
            "url_mode": "port"
        }
        """

        let data = json.data(using: .utf8)!
        let status = try JSONDecoder().decode(WTStatus.self, from: data)

        XCTAssertEqual(status.worktrees.count, 0)
        XCTAssertEqual(status.servers.count, 0)
    }

    // MARK: - Status Values

    func testServerStatusValues() throws {
        let statuses = ["running", "stopped", "starting", "crashed"]

        for testStatus in statuses {
            let json = """
            {
                "worktrees": [
                    {
                        "name": "test",
                        "path": "/test",
                        "status": "\(testStatus)",
                        "has_server": true,
                        "has_claude": false,
                        "has_vscode": false,
                        "git_dirty": false
                    }
                ],
                "url_mode": "port"
            }
            """

            let data = json.data(using: .utf8)!
            let status = try JSONDecoder().decode(WTStatus.self, from: data)
            let server = status.servers.first!

            XCTAssertEqual(server.status, testStatus)
            XCTAssertEqual(server.displayStatus, testStatus)

            // Check isRunning for each status
            if testStatus == "running" || testStatus == "starting" {
                XCTAssertTrue(server.isRunning, "\(testStatus) should be considered running")
            } else {
                XCTAssertFalse(server.isRunning, "\(testStatus) should not be considered running")
            }
        }
    }

    // MARK: - Computed Properties

    func testComputedPropertiesWithNilValues() {
        // Test that computed properties handle nil values correctly
        let json = """
        {
            "worktrees": [
                {
                    "name": "test",
                    "path": "/test",
                    "has_server": false,
                    "has_claude": false,
                    "has_vscode": false,
                    "git_dirty": false
                }
            ],
            "url_mode": "port"
        }
        """

        let data = json.data(using: .utf8)!
        guard let status = try? JSONDecoder().decode(WTStatus.self, from: data),
              let server = status.servers.first else {
            XCTFail("Failed to decode JSON")
            return
        }

        // Test computed properties return safe defaults
        XCTAssertEqual(server.displayPort, 0)
        XCTAssertEqual(server.displayStatus, "stopped")
        XCTAssertEqual(server.displayURL, "http://localhost:0")
        XCTAssertFalse(server.isRunning)
    }

    // MARK: - GitHub Info

    func testParseServerWithGitHubInfo() throws {
        // Note: GitHub info is set separately, not from CLI JSON
        // Just verify the field exists and can be set

        let json = """
        {
            "worktrees": [
                {
                    "name": "feature-pr",
                    "path": "/test",
                    "branch": "feature/123",
                    "has_server": true,
                    "has_claude": false,
                    "has_vscode": false,
                    "git_dirty": false
                }
            ],
            "url_mode": "port"
        }
        """

        let data = json.data(using: .utf8)!
        var status = try JSONDecoder().decode(WTStatus.self, from: data)

        XCTAssertNil(status.servers.first?.githubInfo)

        // Simulate setting GitHub info (as ServerManager does)
        var server = status.worktrees[0]
        server.githubInfo = GitHubInfo(
            prNumber: 123,
            prURL: "https://github.com/owner/repo/pull/123",
            prState: "open",
            ciStatus: .success,
            lastUpdated: Date()
        )

        XCTAssertEqual(server.githubInfo?.prNumber, 123)
        XCTAssertEqual(server.githubInfo?.ciStatus, .success)
    }

    // MARK: - Backward Compatibility

    func testWTStatusServersAlias() throws {
        // Verify the `servers` alias points to `worktrees`
        let json = """
        {
            "worktrees": [
                {
                    "name": "test1",
                    "path": "/test1",
                    "has_server": false,
                    "has_claude": false,
                    "has_vscode": false,
                    "git_dirty": false
                },
                {
                    "name": "test2",
                    "path": "/test2",
                    "has_server": false,
                    "has_claude": false,
                    "has_vscode": false,
                    "git_dirty": false
                }
            ],
            "url_mode": "port"
        }
        """

        let data = json.data(using: .utf8)!
        let status = try JSONDecoder().decode(WTStatus.self, from: data)

        XCTAssertEqual(status.worktrees.count, 2)
        XCTAssertEqual(status.servers.count, 2)
        XCTAssertEqual(status.worktrees.first?.name, status.servers.first?.name)
    }
}
