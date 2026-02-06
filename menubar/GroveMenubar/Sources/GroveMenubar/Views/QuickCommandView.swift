import SwiftUI

struct QuickCommandView: View {
    let server: Server
    @EnvironmentObject var serverManager: ServerManager
    @ObservedObject private var commandHistory = QuickCommandHistory.shared
    @State private var command = ""
    @FocusState private var isCommandFocused: Bool

    private var recentCommands: [String] {
        commandHistory.recentCommands(for: server.name)
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 6) {
            // Command input
            HStack(spacing: 6) {
                Image(systemName: "terminal")
                    .foregroundColor(.secondary)
                    .font(.system(size: 10))

                TextField("Run command...", text: $command)
                    .textFieldStyle(.plain)
                    .font(.system(.caption, design: .monospaced))
                    .focused($isCommandFocused)
                    .onSubmit {
                        runCommand()
                    }

                if !command.isEmpty {
                    Button {
                        runCommand()
                    } label: {
                        Image(systemName: "return")
                            .font(.system(size: 10))
                            .foregroundColor(.grovePrimary)
                    }
                    .buttonStyle(.plain)
                }
            }
            .padding(.horizontal, 8)
            .padding(.vertical, 5)
            .background(Color(NSColor.controlBackgroundColor))
            .cornerRadius(6)

            // Recent commands
            if !recentCommands.isEmpty {
                VStack(alignment: .leading, spacing: 2) {
                    Text("Recent")
                        .font(.system(size: 9))
                        .foregroundColor(.secondary)

                    ForEach(recentCommands, id: \.self) { cmd in
                        Button {
                            command = cmd
                            runCommand()
                        } label: {
                            Text(cmd)
                                .font(.system(.caption2, design: .monospaced))
                                .foregroundColor(.primary)
                                .lineLimit(1)
                        }
                        .buttonStyle(.plain)
                    }
                }
            } else {
                // Show suggestions when no history
                VStack(alignment: .leading, spacing: 2) {
                    Text("Suggestions")
                        .font(.system(size: 9))
                        .foregroundColor(.secondary)

                    HStack(spacing: 4) {
                        ForEach(QuickCommandHistory.suggestions.prefix(3), id: \.self) { suggestion in
                            Button {
                                command = suggestion
                            } label: {
                                Text(suggestion)
                                    .font(.system(size: 9, design: .monospaced))
                                    .padding(.horizontal, 5)
                                    .padding(.vertical, 2)
                                    .background(Color.secondary.opacity(0.1))
                                    .cornerRadius(3)
                            }
                            .buttonStyle(.plain)
                        }
                    }
                }
            }
        }
    }

    private func runCommand() {
        guard !command.isEmpty else { return }
        serverManager.runCommandInWorktree(serverName: server.name, command: command)
        command = ""
    }
}
