import SwiftUI
import AppKit

@main
struct GroveMenubarApp: App {
    @StateObject private var serverManager = ServerManager()

    init() {
        print("[Grove] GroveMenubarApp.init() - app starting")
    }

    private var menuBarIcon: NSImage? {
        guard let url = Bundle.module.url(forResource: "MenuBarIcon", withExtension: "png"),
              let image = NSImage(contentsOf: url) else {
            print("[Grove] Failed to load MenuBarIcon from bundle")
            return nil
        }
        image.isTemplate = true
        image.size = NSSize(width: 18, height: 18)
        return image
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
                    if let nsImage = menuBarIcon {
                        Image(nsImage: nsImage)
                    } else {
                        Image(systemName: "tree.fill")
                    }

                    if serverManager.hasCrashedServers {
                        Circle()
                            .fill(Color.red)
                            .frame(width: 6, height: 6)
                            .offset(x: 2, y: 2)
                    } else if serverManager.hasUnhealthyServers {
                        Circle()
                            .fill(Color.orange)
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
                .onAppear {
                    NSApp.activate(ignoringOtherApps: true)
                }
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
