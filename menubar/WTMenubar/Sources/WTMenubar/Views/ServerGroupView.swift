import SwiftUI

struct ServerGroupView: View {
    @EnvironmentObject var serverManager: ServerManager
    let group: ServerGroup
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

            // Group servers
            if !isCollapsed {
                ForEach(group.servers) { server in
                    ServerRowView(server: server)
                }
            }
        }
        .onAppear {
            isCollapsed = CollapsedGroupsManager.shared.isCollapsed(group.id)
        }
    }
}
