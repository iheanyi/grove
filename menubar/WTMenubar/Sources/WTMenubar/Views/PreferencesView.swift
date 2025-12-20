import SwiftUI

struct PreferencesView: View {
    @ObservedObject var preferences = PreferencesManager.shared
    @Environment(\.dismiss) private var dismiss

    var body: some View {
        VStack(spacing: 0) {
            // Header
            HStack {
                Text("Preferences")
                    .font(.headline)
                    .foregroundColor(.wtPrimary)

                Spacer()

                Button {
                    dismiss()
                } label: {
                    Image(systemName: "xmark.circle.fill")
                        .foregroundColor(.secondary)
                }
                .buttonStyle(.plain)
            }
            .padding()

            Divider()

            ScrollView {
                VStack(alignment: .leading, spacing: 20) {
                    // General Settings
                    PreferenceSection(title: "General") {
                        Toggle("Launch at login", isOn: $preferences.launchAtLogin)
                            .toggleStyle(.switch)
                    }

                    Divider()

                    // Notifications
                    PreferenceSection(title: "Notifications") {
                        VStack(alignment: .leading, spacing: 8) {
                            Toggle("Notify when server starts", isOn: $preferences.notifyOnServerStart)
                                .toggleStyle(.switch)

                            Toggle("Notify when server stops", isOn: $preferences.notifyOnServerStop)
                                .toggleStyle(.switch)

                            Toggle("Notify when server crashes", isOn: $preferences.notifyOnServerCrash)
                                .toggleStyle(.switch)
                        }
                    }

                    Divider()

                    // Refresh Settings
                    PreferenceSection(title: "Refresh") {
                        VStack(alignment: .leading, spacing: 8) {
                            Text("Refresh interval: \(Int(preferences.refreshInterval)) seconds")
                                .font(.caption)
                                .foregroundColor(.secondary)

                            Slider(value: $preferences.refreshInterval, in: 1...30, step: 1) {
                                Text("Refresh interval")
                            }
                        }
                    }

                    Divider()

                    // Browser Settings
                    PreferenceSection(title: "Browser") {
                        VStack(alignment: .leading, spacing: 8) {
                            Text("Default browser")
                                .font(.caption)
                                .foregroundColor(.secondary)

                            Picker("Browser", selection: $preferences.defaultBrowser) {
                                ForEach(preferences.getInstalledBrowsers()) { browser in
                                    Text(browser.name).tag(browser.bundleId)
                                }
                            }
                            .pickerStyle(.menu)
                            .labelsHidden()
                        }
                    }

                    Divider()

                    // Theme Settings
                    PreferenceSection(title: "Appearance") {
                        VStack(alignment: .leading, spacing: 8) {
                            Text("Theme")
                                .font(.caption)
                                .foregroundColor(.secondary)

                            Picker("Theme", selection: $preferences.theme) {
                                ForEach(AppTheme.allCases, id: \.self) { theme in
                                    Text(theme.displayName).tag(theme)
                                }
                            }
                            .pickerStyle(.segmented)
                            .labelsHidden()
                        }
                    }

                    Divider()

                    // Keyboard Shortcuts Info
                    PreferenceSection(title: "Keyboard Shortcuts") {
                        VStack(alignment: .leading, spacing: 4) {
                            ShortcutRow(key: "⌘1-9", description: "Quick select server by position")
                            ShortcutRow(key: "⌘O", description: "Open selected server in browser")
                            ShortcutRow(key: "⌘L", description: "View logs for selected server")
                            ShortcutRow(key: "⌘C", description: "Copy URL of selected server")
                            ShortcutRow(key: "⌘S", description: "Stop all servers")
                            ShortcutRow(key: "⌘R", description: "Refresh server list")
                        }
                        .font(.caption)
                    }
                }
                .padding()
            }
        }
        .frame(width: 400, height: 500)
    }
}

struct PreferenceSection<Content: View>: View {
    let title: String
    @ViewBuilder let content: Content

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(title)
                .font(.subheadline)
                .fontWeight(.semibold)

            content
        }
    }
}

struct ShortcutRow: View {
    let key: String
    let description: String

    var body: some View {
        HStack {
            Text(key)
                .font(.system(.caption, design: .monospaced))
                .foregroundColor(.secondary)
                .frame(width: 60, alignment: .leading)

            Text(description)
                .foregroundColor(.primary)

            Spacer()
        }
    }
}
