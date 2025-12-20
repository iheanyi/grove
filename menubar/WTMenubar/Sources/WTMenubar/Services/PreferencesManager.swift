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
        static let theme = "theme"
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

    // Theme selection
    @Published var theme: AppTheme {
        didSet {
            defaults.set(theme.rawValue, forKey: Keys.theme)
            applyTheme()
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

        let themeString = defaults.string(forKey: Keys.theme) ?? AppTheme.system.rawValue
        self.theme = AppTheme(rawValue: themeString) ?? .system

        applyTheme()
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
            Browser(name: "Opera", bundleId: "com.operasoftware.Opera"),
            Browser(name: "Vivaldi", bundleId: "com.vivaldi.Vivaldi")
        ]

        for browser in commonBrowsers {
            if let path = NSWorkspace.shared.urlForApplication(withBundleIdentifier: browser.bundleId) {
                browsers.append(browser)
            }
        }

        return browsers
    }

    func openURL(_ url: URL) {
        if defaultBrowser == "system" {
            NSWorkspace.shared.open(url)
        } else {
            NSWorkspace.shared.open([url],
                                   withApplicationAt: NSWorkspace.shared.urlForApplication(withBundleIdentifier: defaultBrowser)!,
                                   configuration: NSWorkspace.OpenConfiguration())
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
