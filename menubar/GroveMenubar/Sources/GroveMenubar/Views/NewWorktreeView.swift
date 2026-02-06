import SwiftUI

struct NewWorktreeView: View {
    @EnvironmentObject var serverManager: ServerManager
    @Environment(\.dismiss) private var dismiss
    @State private var branchName = ""
    @State private var baseBranch = "main"
    @State private var selectedRepo = ""
    @State private var isCreating = false
    @State private var errorMessage: String?

    private let commonBaseBranches = ["main", "master", "develop"]

    var body: some View {
        VStack(alignment: .leading, spacing: 16) {
            // Header
            HStack {
                Image(systemName: "arrow.triangle.branch")
                    .foregroundColor(.grovePrimary)
                Text("New Worktree")
                    .font(.headline)
                Spacer()
            }

            // Repository picker
            if serverManager.mainRepoPaths.count > 1 {
                VStack(alignment: .leading, spacing: 4) {
                    Text("Repository")
                        .font(.caption)
                        .foregroundColor(.secondary)
                    Picker("Repository", selection: $selectedRepo) {
                        ForEach(serverManager.mainRepoPaths, id: \.self) { path in
                            Text(URL(fileURLWithPath: path).lastPathComponent)
                                .tag(path)
                        }
                    }
                    .labelsHidden()
                }
            }

            // Branch name
            VStack(alignment: .leading, spacing: 4) {
                Text("Branch Name")
                    .font(.caption)
                    .foregroundColor(.secondary)
                TextField("feature/my-branch", text: $branchName)
                    .textFieldStyle(.roundedBorder)
            }

            // Base branch
            VStack(alignment: .leading, spacing: 4) {
                Text("Base Branch")
                    .font(.caption)
                    .foregroundColor(.secondary)
                Picker("Base Branch", selection: $baseBranch) {
                    ForEach(commonBaseBranches, id: \.self) { branch in
                        Text(branch).tag(branch)
                    }
                }
                .labelsHidden()
                .pickerStyle(.segmented)
            }

            // Error message
            if let errorMessage = errorMessage {
                HStack(spacing: 4) {
                    Image(systemName: "exclamationmark.triangle.fill")
                        .foregroundColor(.orange)
                        .font(.caption)
                    Text(errorMessage)
                        .font(.caption)
                        .foregroundColor(.orange)
                }
            }

            Divider()

            // Actions
            HStack {
                Button("Cancel") {
                    dismiss()
                }
                .keyboardShortcut(.cancelAction)

                Spacer()

                Button {
                    createWorktree()
                } label: {
                    if isCreating {
                        ProgressView()
                            .controlSize(.small)
                            .padding(.horizontal, 8)
                    } else {
                        Text("Create")
                    }
                }
                .buttonStyle(.borderedProminent)
                .tint(.grovePrimary)
                .disabled(branchName.isEmpty || selectedRepo.isEmpty || isCreating)
                .keyboardShortcut(.defaultAction)
            }
        }
        .padding()
        .frame(width: 340)
        .onAppear {
            // Default to first repo
            if selectedRepo.isEmpty, let first = serverManager.mainRepoPaths.first {
                selectedRepo = first
            }
        }
    }

    private func createWorktree() {
        isCreating = true
        errorMessage = nil

        serverManager.createWorktree(
            branch: branchName,
            baseBranch: baseBranch,
            repoPath: selectedRepo
        ) { result in
            isCreating = false
            switch result {
            case .success:
                dismiss()
            case .failure(let error):
                errorMessage = error.localizedDescription
            }
        }
    }
}
