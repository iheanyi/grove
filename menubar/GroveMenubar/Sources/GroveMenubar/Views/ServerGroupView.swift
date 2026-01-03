import SwiftUI

struct ServerGroupView: View {
    @EnvironmentObject var serverManager: ServerManager
    let group: ServerGroup
    var searchText: String = ""
    @State private var isCollapsed: Bool = false

    var body: some View {
        VStack(spacing: 0) {
            // Group header
            Button {
                isCollapsed.toggle()
                CollapsedGroupsManager.shared.setCollapsed(group.id, collapsed: isCollapsed)
            } label: {
                HStack {
                    Image(systemName: isCollapsed ? "chevron.right" : "chevron.down")
                        .font(.caption)
                        .foregroundColor(.secondary)
                        .frame(width: 12)

                    Text(group.name)
                        .font(.caption)
                        .foregroundColor(.secondary)

                    Spacer()

                    if group.isRunning {
                        Circle()
                            .fill(Color.green)
                            .frame(width: 6, height: 6)
                    }

                    Text("\(group.runningCount)/\(group.totalCount)")
                        .font(.caption2)
                        .foregroundColor(.secondary)
                }
                .padding(.horizontal)
                .padding(.vertical, 4)
                .background(Color(NSColor.windowBackgroundColor).opacity(0.5))
            }
            .buttonStyle(.plain)
            .contextMenu {
                if group.runningCount > 0 {
                    Button {
                        serverManager.stopAllServersInGroup(group)
                    } label: {
                        Label("Stop All in \(group.name)", systemImage: "stop.fill")
                    }

                    Divider()
                }

                Button(role: .destructive) {
                    serverManager.removeAllServersInGroup(group)
                } label: {
                    Label("Remove All from Grove", systemImage: "xmark.circle")
                }
            }

            // Group servers
            if !isCollapsed {
                ForEach(Array(group.servers.enumerated()), id: \.element.id) { index, server in
                    ServerRowView(server: server, searchText: searchText, displayIndex: index + 1)
                }
            }
        }
        .onAppear {
            isCollapsed = CollapsedGroupsManager.shared.isCollapsed(group.id)
        }
    }
}
