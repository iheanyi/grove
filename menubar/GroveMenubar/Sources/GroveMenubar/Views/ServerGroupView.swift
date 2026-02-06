import SwiftUI

struct ServerGroupView: View {
    @EnvironmentObject var serverManager: ServerManager
    let group: ServerGroup
    var searchText: String = ""
    @State private var isCollapsed: Bool = false
    @State private var showStopAllConfirmation = false
    @State private var showRemoveAllConfirmation = false
    @State private var showNewWorktree = false

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
                Button {
                    showNewWorktree = true
                } label: {
                    Label("New Worktree", systemImage: "arrow.triangle.branch")
                }

                if group.runningCount > 0 {
                    Divider()

                    Button {
                        showStopAllConfirmation = true
                    } label: {
                        Label("Stop All in \(group.name)", systemImage: "stop.fill")
                    }
                }

                Divider()

                Button(role: .destructive) {
                    showRemoveAllConfirmation = true
                } label: {
                    Label("Remove All from Grove", systemImage: "xmark.circle")
                }
            }
            .confirmationDialog(
                "Stop All Servers in \(group.name)?",
                isPresented: $showStopAllConfirmation,
                titleVisibility: .visible
            ) {
                Button("Stop All", role: .destructive) {
                    serverManager.stopAllServersInGroup(group)
                }
                Button("Cancel", role: .cancel) {}
            } message: {
                Text("This will stop \(group.runningCount) running server\(group.runningCount == 1 ? "" : "s") in this group.")
            }
            .confirmationDialog(
                "Remove All Servers in \(group.name)?",
                isPresented: $showRemoveAllConfirmation,
                titleVisibility: .visible
            ) {
                Button("Remove All", role: .destructive) {
                    serverManager.removeAllServersInGroup(group)
                }
                Button("Cancel", role: .cancel) {}
            } message: {
                Text("This will remove \(group.totalCount) server\(group.totalCount == 1 ? "" : "s") from Grove. Running servers will be stopped first.")
            }
            .popover(isPresented: $showNewWorktree) {
                NewWorktreeView()
                    .environmentObject(serverManager)
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
