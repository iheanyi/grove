import SwiftUI

/// Native macOS Settings window with tabs
struct SettingsView: View {
    var body: some View {
        TabView {
            GeneralSettingsTab()
                .tabItem {
                    Label("General", systemImage: "gear")
                }

            NotificationsSettingsTab()
                .tabItem {
                    Label("Notifications", systemImage: "bell")
                }

            AppearanceSettingsTab()
                .tabItem {
                    Label("Appearance", systemImage: "paintbrush")
                }

            ShortcutsSettingsTab()
                .tabItem {
                    Label("Shortcuts", systemImage: "keyboard")
                }

            AboutSettingsTab()
                .tabItem {
                    Label("About", systemImage: "info.circle")
                }
        }
        .frame(width: 450, height: 300)
    }
}

// MARK: - General Settings

struct GeneralSettingsTab: View {
    @ObservedObject var preferences = PreferencesManager.shared

    var body: some View {
        Form {
            Section {
                Toggle("Launch Grove at login", isOn: $preferences.launchAtLogin)

                Toggle("Show in Dock", isOn: $preferences.showDockIcon)

                LabeledContent("Refresh interval") {
                    Picker("", selection: Binding(
                        get: { Int(preferences.refreshInterval) },
                        set: { preferences.refreshInterval = Double($0) }
                    )) {
                        Text("1 second").tag(1)
                        Text("2 seconds").tag(2)
                        Text("5 seconds").tag(5)
                        Text("10 seconds").tag(10)
                        Text("30 seconds").tag(30)
                    }
                    .frame(width: 150)
                }
            }

            Section {
                LabeledContent("Default browser") {
                    Picker("", selection: $preferences.defaultBrowser) {
                        ForEach(preferences.getInstalledBrowsers()) { browser in
                            Text(browser.name).tag(browser.bundleId)
                        }
                    }
                    .frame(width: 180)
                }
            }
        }
        .formStyle(.grouped)
        .padding()
    }
}

// MARK: - Notifications Settings

struct NotificationsSettingsTab: View {
    @ObservedObject var preferences = PreferencesManager.shared

    var body: some View {
        Form {
            Section {
                Toggle("Server started", isOn: $preferences.notifyOnServerStart)
                Toggle("Server stopped", isOn: $preferences.notifyOnServerStop)
                Toggle("Server crashed", isOn: $preferences.notifyOnServerCrash)
            } header: {
                Text("Show notifications when:")
            }

            Section {
                Text("Notifications appear in macOS Notification Center. You can customize notification settings in System Settings > Notifications > Grove.")
                    .font(.caption)
                    .foregroundColor(.secondary)
            }
        }
        .formStyle(.grouped)
        .padding()
    }
}

// MARK: - Appearance Settings

struct AppearanceSettingsTab: View {
    @ObservedObject var preferences = PreferencesManager.shared

    var body: some View {
        Form {
            Section {
                Picker("Theme", selection: $preferences.theme) {
                    ForEach(AppTheme.allCases, id: \.self) { theme in
                        Text(theme.displayName).tag(theme)
                    }
                }
                .pickerStyle(.segmented)
            } header: {
                Text("Appearance")
            }

            Section {
                Text("Choose 'System' to automatically match your macOS appearance settings.")
                    .font(.caption)
                    .foregroundColor(.secondary)
            }
        }
        .formStyle(.grouped)
        .padding()
    }
}

// MARK: - Shortcuts Settings

struct ShortcutsSettingsTab: View {
    var body: some View {
        Form {
            Section {
                ShortcutInfoRow(shortcut: "1-9", description: "Quick select server by position")
                ShortcutInfoRow(shortcut: "⌘F", description: "Focus search field")
                ShortcutInfoRow(shortcut: "⌘R", description: "Refresh server list")
                ShortcutInfoRow(shortcut: "⌘⇧S", description: "Stop all servers")
                ShortcutInfoRow(shortcut: "⌘⇧O", description: "Open all running servers")
                ShortcutInfoRow(shortcut: "⌘L", description: "Open log viewer")
                ShortcutInfoRow(shortcut: "⌘,", description: "Open settings")
            } header: {
                Text("Keyboard Shortcuts")
            }
        }
        .formStyle(.grouped)
        .padding()
    }
}

struct ShortcutInfoRow: View {
    let shortcut: String
    let description: String

    var body: some View {
        HStack {
            Text(description)
            Spacer()
            Text(shortcut)
                .font(.system(.body, design: .monospaced))
                .foregroundColor(.secondary)
                .padding(.horizontal, 8)
                .padding(.vertical, 4)
                .background(Color.secondary.opacity(0.1))
                .cornerRadius(4)
        }
    }
}

// MARK: - About Settings

struct AboutSettingsTab: View {
    var body: some View {
        VStack(spacing: 20) {
            // App Icon
            Image(systemName: "bolt.fill")
                .font(.system(size: 64))
                .foregroundColor(.grovePrimary)

            // App Name and Version
            VStack(spacing: 4) {
                Text("Grove")
                    .font(.title)
                    .fontWeight(.bold)

                Text("Version \(appVersion)")
                    .font(.subheadline)
                    .foregroundColor(.secondary)
            }

            // Description
            Text("Manage your git worktrees and dev servers from the menubar")
                .font(.subheadline)
                .foregroundColor(.secondary)
                .multilineTextAlignment(.center)
                .padding(.horizontal)

            Spacer()

            // Links
            HStack(spacing: 20) {
                Link("GitHub", destination: URL(string: "https://github.com/iheanyi/grove")!)
                    .buttonStyle(.link)

                Link("Report Issue", destination: URL(string: "https://github.com/iheanyi/grove/issues")!)
                    .buttonStyle(.link)
            }
            .font(.caption)

            // Copyright
            Text("Made with Swift and Go")
                .font(.caption2)
                .foregroundColor(.secondary.opacity(0.7))
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .padding()
    }

    private var appVersion: String {
        Bundle.main.infoDictionary?["CFBundleShortVersionString"] as? String ?? "1.0.0"
    }
}
