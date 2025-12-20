import SwiftUI

@main
struct GroveMenubarApp: App {
    @StateObject private var serverManager = ServerManager()

    var body: some Scene {
        // Menubar
        MenuBarExtra {
            MenuView()
                .environmentObject(serverManager)
        } label: {
            Label {
                Text("Grove")
            } icon: {
                // Use ZStack to overlay status indicator on icon
                ZStack(alignment: .bottomTrailing) {
                    Image(systemName: "bolt.fill")
                        .symbolRenderingMode(.monochrome)
                        .foregroundStyle(.primary)

                    // Status indicator dot
                    if serverManager.hasCrashedServers {
                        Circle()
                            .fill(Color.red)
                            .frame(width: 6, height: 6)
                            .offset(x: 2, y: 2)
                    } else if serverManager.hasStartingServers {
                        Circle()
                            .fill(Color.yellow)
                            .frame(width: 6, height: 6)
                            .offset(x: 2, y: 2)
                    } else if serverManager.hasRunningServers {
                        Circle()
                            .fill(Color.green)
                            .frame(width: 6, height: 6)
                            .offset(x: 2, y: 2)
                    }
                }
            }
        }
        .menuBarExtraStyle(.window)

        // Log Viewer Window (opened on demand)
        Window("Grove Logs", id: "log-viewer") {
            LogViewerWindow()
                .environmentObject(serverManager)
        }
        .windowStyle(.automatic)
        .windowResizability(.contentMinSize)
        .defaultSize(width: 900, height: 600)
        .keyboardShortcut("l", modifiers: [.command])

        // Settings Window (native macOS settings)
        Settings {
            SettingsView()
        }
    }
}
