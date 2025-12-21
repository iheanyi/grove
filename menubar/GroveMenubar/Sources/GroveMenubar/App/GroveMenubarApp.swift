import SwiftUI

@main
struct GroveMenubarApp: App {
    @StateObject private var serverManager = ServerManager()

    init() {
        print("[Grove] GroveMenubarApp.init() - app starting")
    }

    var body: some Scene {
        let _ = print("[Grove] GroveMenubarApp.body evaluated")
        // Menubar
        MenuBarExtra {
            let _ = print("[Grove] MenuBarExtra content evaluated")
            MenuView()
                .environmentObject(serverManager)
        } label: {
            Label {
                Text("Grove")
            } icon: {
                ZStack(alignment: .bottomTrailing) {
                    Image(systemName: "bolt.fill")
                        .symbolRenderingMode(.monochrome)
                        .foregroundStyle(.primary)

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
