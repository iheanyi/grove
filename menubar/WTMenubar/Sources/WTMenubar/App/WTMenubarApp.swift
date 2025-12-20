import SwiftUI

@main
struct WTMenubarApp: App {
    @StateObject private var serverManager = ServerManager()

    var body: some Scene {
        MenuBarExtra {
            MenuView()
                .environmentObject(serverManager)
        } label: {
            Label {
                Text("wt")
            } icon: {
                Image(systemName: serverManager.statusIcon)
                    .symbolRenderingMode(.palette)
                    .foregroundStyle(serverManager.statusColor, .primary)
            }
        }
        .menuBarExtraStyle(.window)
    }
}
