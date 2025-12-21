import Foundation
import SwiftUI
import ServiceManagement

class PreferencesManager: ObservableObject {
    static let shared = PreferencesManager()

    private let defaults = UserDefaults.standard

    // Keys
    private enum Keys {
        static let launchAtLogin = "launchAtLogin"
        static let notifyOnServerStart = "notifyOnServerStart"
        static let notifyOnServerStop = "notifyOnServerStop"
        static let notifyOnServerCrash = "notifyOnServerCrash"
        static let refreshInterval = "refreshInterval"
        static let defaultBrowser = "defaultBrowser"
        static let defaultTerminal = "defaultTerminal"
        static let theme = "theme"
        static let showDockIcon = "showDockIcon"
        static let showGitHubInfo = "showGitHubInfo"
        static let showUptime = "showUptime"
        static let showPort = "showPort"
    }

    // Launch at login
    @Published var launchAtLogin: Bool {
        didSet {
            defaults.set(launchAtLogin, forKey: Keys.launchAtLogin)
            updateLaunchAtLogin()
        }
    }

    // Notification preferences
    @Published var notifyOnServerStart: Bool {
        didSet { defaults.set(notifyOnServerStart, forKey: Keys.notifyOnServerStart) }
    }

    @Published var notifyOnServerStop: Bool {
        didSet { defaults.set(notifyOnServerStop, forKey: Keys.notifyOnServerStop) }
    }

    @Published var notifyOnServerCrash: Bool {
        didSet { defaults.set(notifyOnServerCrash, forKey: Keys.notifyOnServerCrash) }
    }

    // Refresh interval (in seconds)
    @Published var refreshInterval: Double {
        didSet { defaults.set(refreshInterval, forKey: Keys.refreshInterval) }
    }

    // Default browser (bundle identifier)
    @Published var defaultBrowser: String {
        didSet { defaults.set(defaultBrowser, forKey: Keys.defaultBrowser) }
    }

    // Default terminal (bundle identifier)
    @Published var defaultTerminal: String {
        didSet { defaults.set(defaultTerminal, forKey: Keys.defaultTerminal) }
    }

    // Theme selection
    @Published var theme: AppTheme {
        didSet {
            defaults.set(theme.rawValue, forKey: Keys.theme)
            applyTheme()
        }
    }

    // Show dock icon
    @Published var showDockIcon: Bool {
        didSet {
            defaults.set(showDockIcon, forKey: Keys.showDockIcon)
            updateDockIcon()
        }
    }

    // Show GitHub PR/CI info (can cause slowness on wake)
    @Published var showGitHubInfo: Bool {
        didSet {
            defaults.set(showGitHubInfo, forKey: Keys.showGitHubInfo)
        }
    }

    // Show uptime badge on server rows
    @Published var showUptime: Bool {
        didSet {
            defaults.set(showUptime, forKey: Keys.showUptime)
        }
    }

    // Show port number on server rows
    @Published var showPort: Bool {
        didSet {
            defaults.set(showPort, forKey: Keys.showPort)
        }
    }

    private init() {
        // Load from defaults
        self.launchAtLogin = defaults.bool(forKey: Keys.launchAtLogin)
        self.notifyOnServerStart = defaults.object(forKey: Keys.notifyOnServerStart) as? Bool ?? true
        self.notifyOnServerStop = defaults.object(forKey: Keys.notifyOnServerStop) as? Bool ?? false
        self.notifyOnServerCrash = defaults.object(forKey: Keys.notifyOnServerCrash) as? Bool ?? true
        self.refreshInterval = defaults.object(forKey: Keys.refreshInterval) as? Double ?? 5.0
        self.defaultBrowser = defaults.string(forKey: Keys.defaultBrowser) ?? "system"
        self.defaultTerminal = defaults.string(forKey: Keys.defaultTerminal) ?? "com.apple.Terminal"

        let themeString = defaults.string(forKey: Keys.theme) ?? AppTheme.system.rawValue
        self.theme = AppTheme(rawValue: themeString) ?? .system
        self.showDockIcon = defaults.bool(forKey: Keys.showDockIcon)
        // Default to OFF to avoid wake-from-sleep issues
        self.showGitHubInfo = defaults.object(forKey: Keys.showGitHubInfo) as? Bool ?? false
        // Default to ON for uptime and port
        self.showUptime = defaults.object(forKey: Keys.showUptime) as? Bool ?? true
        self.showPort = defaults.object(forKey: Keys.showPort) as? Bool ?? true

        applyTheme()
        updateDockIcon()
    }

    private func updateLaunchAtLogin() {
        if #available(macOS 13.0, *) {
            do {
                if launchAtLogin {
                    try SMAppService.mainApp.register()
                } else {
                    try SMAppService.mainApp.unregister()
                }
            } catch {
                print("Failed to update launch at login: \(error)")
            }
        }
    }

    private func applyTheme() {
        switch theme {
        case .system:
            NSApp.appearance = nil
        case .light:
            NSApp.appearance = NSAppearance(named: .aqua)
        case .dark:
            NSApp.appearance = NSAppearance(named: .darkAqua)
        }
    }

    private func updateDockIcon() {
        if showDockIcon {
            NSApp.setActivationPolicy(.regular)
        } else {
            NSApp.setActivationPolicy(.accessory)
        }
    }

    // Get list of installed browsers
    func getInstalledBrowsers() -> [Browser] {
        var browsers: [Browser] = [
            Browser(name: "System Default", bundleId: "system")
        ]

        let commonBrowsers = [
            Browser(name: "Safari", bundleId: "com.apple.Safari"),
            Browser(name: "Google Chrome", bundleId: "com.google.Chrome"),
            Browser(name: "Firefox", bundleId: "org.mozilla.firefox"),
            Browser(name: "Microsoft Edge", bundleId: "com.microsoft.edgemac"),
            Browser(name: "Brave", bundleId: "com.brave.Browser"),
            Browser(name: "Arc", bundleId: "company.thebrowser.Browser"),
            Browser(name: "Dia", bundleId: "build.aspect.Dia"),
            Browser(name: "Opera", bundleId: "com.operasoftware.Opera"),
            Browser(name: "Vivaldi", bundleId: "com.vivaldi.Vivaldi")
        ]

        for browser in commonBrowsers {
            if NSWorkspace.shared.urlForApplication(withBundleIdentifier: browser.bundleId) != nil {
                browsers.append(browser)
            }
        }

        return browsers
    }

    // Get list of installed terminals
    func getInstalledTerminals() -> [TerminalApp] {
        var terminals: [TerminalApp] = []

        let commonTerminals = [
            TerminalApp(name: "Terminal", bundleId: "com.apple.Terminal"),
            TerminalApp(name: "Ghostty", bundleId: "com.mitchellh.ghostty"),
            TerminalApp(name: "iTerm", bundleId: "com.googlecode.iterm2"),
            TerminalApp(name: "Warp", bundleId: "dev.warp.Warp-Stable"),
            TerminalApp(name: "Alacritty", bundleId: "org.alacritty"),
            TerminalApp(name: "Kitty", bundleId: "net.kovidgoyal.kitty"),
            TerminalApp(name: "Hyper", bundleId: "co.zeit.hyper")
        ]

        for terminal in commonTerminals {
            if NSWorkspace.shared.urlForApplication(withBundleIdentifier: terminal.bundleId) != nil {
                terminals.append(terminal)
            }
        }

        return terminals
    }

    // Open a path in the configured terminal - runs on background thread to avoid blocking
    func openInTerminal(path: String) {
        let terminal = defaultTerminal

        // Run all terminal operations on background thread to prevent main thread blocking
        DispatchQueue.global(qos: .userInitiated).async {
            switch terminal {
            case "com.apple.Terminal":
                Self.openInAppleTerminalAsync(path: path)
            case "com.googlecode.iterm2":
                Self.openInITermAsync(path: path)
            case "com.mitchellh.ghostty":
                Self.openInGhostty(path: path)
            case "dev.warp.Warp-Stable":
                Self.openInWarp(path: path)
            default:
                // For other terminals, try generic approach
                DispatchQueue.main.async {
                    Self.openInGenericTerminal(path: path, bundleId: terminal)
                }
            }
        }
    }

    private static func openInAppleTerminalAsync(path: String) {
        let script = """
        tell application "Terminal"
            activate
            do script "cd '\(path)'"
        end tell
        """
        runAppleScriptAsync(script)
    }

    private static func openInITermAsync(path: String) {
        let script = """
        tell application "iTerm"
            activate
            try
                set newWindow to (create window with default profile)
                tell current session of newWindow
                    write text "cd '\(path)'"
                end tell
            on error
                tell current window
                    create tab with default profile
                    tell current session
                        write text "cd '\(path)'"
                    end tell
                end tell
            end try
        end tell
        """
        runAppleScriptAsync(script)
    }

    private static func openInGhostty(path: String) {
        // Try using the Ghostty CLI with --working-directory if available
        let ghosttyPaths = [
            "/Applications/Ghostty.app/Contents/MacOS/ghostty",
            "/opt/homebrew/bin/ghostty",
            "\(NSHomeDirectory())/.local/bin/ghostty"
        ]

        for ghosttyPath in ghosttyPaths {
            if FileManager.default.fileExists(atPath: ghosttyPath) {
                let task = Process()
                task.executableURL = URL(fileURLWithPath: ghosttyPath)
                task.arguments = ["--working-directory=\(path)"]

                do {
                    try task.run()
                    return
                } catch {
                    // Try next path
                    continue
                }
            }
        }

        // Fallback: Use AppleScript to open Ghostty and send a cd command
        // Note: This is less ideal but works as a fallback
        let script = """
        tell application "Ghostty"
            activate
        end tell
        delay 0.5
        tell application "System Events"
            keystroke "cd '\(path)'"
            keystroke return
        end tell
        """
        runAppleScriptAsync(script)
    }

    private static func openInWarp(path: String) {
        // Warp can be opened with a directory
        let task = Process()
        task.executableURL = URL(fileURLWithPath: "/usr/bin/open")
        task.arguments = ["-a", "Warp", path]

        do {
            try task.run()
        } catch {
            // Fallback
            DispatchQueue.main.async {
                if let appURL = NSWorkspace.shared.urlForApplication(withBundleIdentifier: "dev.warp.Warp-Stable") {
                    NSWorkspace.shared.open(appURL)
                }
            }
        }
    }

    private static func openInGenericTerminal(path: String, bundleId: String) {
        // Try to open the terminal app at the given path
        if let appURL = NSWorkspace.shared.urlForApplication(withBundleIdentifier: bundleId) {
            let config = NSWorkspace.OpenConfiguration()
            NSWorkspace.shared.open([URL(fileURLWithPath: path)], withApplicationAt: appURL, configuration: config)
        }
    }

    /// Run AppleScript on background thread - never blocks main thread
    private static func runAppleScriptAsync(_ script: String) {
        if let appleScript = NSAppleScript(source: script) {
            var error: NSDictionary?
            appleScript.executeAndReturnError(&error)
            if let error = error {
                print("AppleScript error: \(error)")
            }
        }
    }

    func openURL(_ url: URL) {
        if defaultBrowser == "system" {
            NSWorkspace.shared.open(url)
        } else if let browserURL = NSWorkspace.shared.urlForApplication(withBundleIdentifier: defaultBrowser) {
            NSWorkspace.shared.open([url],
                                   withApplicationAt: browserURL,
                                   configuration: NSWorkspace.OpenConfiguration())
        } else {
            // Fallback to system default if browser not found
            NSWorkspace.shared.open(url)
        }
    }
}

enum AppTheme: String, CaseIterable {
    case system = "System"
    case light = "Light"
    case dark = "Dark"

    var displayName: String {
        rawValue
    }
}

struct Browser: Identifiable {
    let name: String
    let bundleId: String

    var id: String { bundleId }
}

struct TerminalApp: Identifiable {
    let name: String
    let bundleId: String

    var id: String { bundleId }
}
