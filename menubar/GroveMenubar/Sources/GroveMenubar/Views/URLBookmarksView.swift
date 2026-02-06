import SwiftUI

struct URLBookmarksView: View {
    let server: Server
    @ObservedObject private var bookmarkManager = BookmarkManager.shared
    @State private var showAddBookmark = false
    @State private var newPath = ""
    @State private var newLabel = ""

    private var bookmarks: [URLBookmark] {
        bookmarkManager.bookmarks(for: server.name)
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            // Existing bookmarks as chips
            if !bookmarks.isEmpty {
                FlowLayout(spacing: 6) {
                    ForEach(bookmarks) { bookmark in
                        BookmarkChip(
                            bookmark: bookmark,
                            baseURL: server.displayURL,
                            onRemove: {
                                bookmarkManager.removeBookmark(for: server.name, bookmarkId: bookmark.id)
                            }
                        )
                    }
                }
            }

            if showAddBookmark {
                // Add bookmark form
                VStack(alignment: .leading, spacing: 6) {
                    HStack(spacing: 6) {
                        TextField("/path", text: $newPath)
                            .textFieldStyle(.roundedBorder)
                            .font(.system(.caption, design: .monospaced))

                        TextField("Label", text: $newLabel)
                            .textFieldStyle(.roundedBorder)
                            .font(.caption)
                            .frame(width: 70)
                    }

                    // Suggestions
                    HStack(spacing: 4) {
                        ForEach(BookmarkManager.suggestions.prefix(4), id: \.path) { suggestion in
                            Button {
                                newPath = suggestion.path
                                newLabel = suggestion.label
                            } label: {
                                Text(suggestion.label)
                                    .font(.system(size: 9))
                                    .padding(.horizontal, 5)
                                    .padding(.vertical, 2)
                                    .background(Color.secondary.opacity(0.1))
                                    .cornerRadius(3)
                            }
                            .buttonStyle(.plain)
                        }
                    }

                    HStack {
                        Button("Cancel") {
                            showAddBookmark = false
                            newPath = ""
                            newLabel = ""
                        }
                        .controlSize(.small)

                        Button("Add") {
                            let label = newLabel.isEmpty ? newPath : newLabel
                            bookmarkManager.addBookmark(for: server.name, path: newPath, label: label)
                            newPath = ""
                            newLabel = ""
                            showAddBookmark = false
                        }
                        .controlSize(.small)
                        .buttonStyle(.borderedProminent)
                        .tint(.grovePrimary)
                        .disabled(newPath.isEmpty)
                    }
                }
                .padding(8)
                .background(Color.secondary.opacity(0.05))
                .cornerRadius(6)
            } else {
                Button {
                    showAddBookmark = true
                } label: {
                    HStack(spacing: 4) {
                        Image(systemName: "plus")
                            .font(.system(size: 9))
                        Text("Bookmark")
                            .font(.caption2)
                    }
                    .padding(.horizontal, 6)
                    .padding(.vertical, 3)
                    .background(Color.secondary.opacity(0.1))
                    .cornerRadius(4)
                }
                .buttonStyle(.plain)
            }
        }
    }
}

struct BookmarkChip: View {
    let bookmark: URLBookmark
    let baseURL: String
    let onRemove: () -> Void
    @State private var isHovered = false

    var body: some View {
        HStack(spacing: 4) {
            Button {
                let urlString = baseURL + bookmark.path
                if let url = URL(string: urlString) {
                    PreferencesManager.shared.openURL(url)
                }
            } label: {
                HStack(spacing: 3) {
                    Image(systemName: "link")
                        .font(.system(size: 8))
                    Text(bookmark.label)
                        .font(.caption2)
                }
                .padding(.horizontal, 6)
                .padding(.vertical, 3)
                .background(Color.blue.opacity(0.1))
                .foregroundColor(.blue)
                .cornerRadius(4)
            }
            .buttonStyle(.plain)

            if isHovered {
                Button {
                    onRemove()
                } label: {
                    Image(systemName: "xmark")
                        .font(.system(size: 7))
                        .foregroundColor(.secondary)
                }
                .buttonStyle(.plain)
            }
        }
        .onHover { hovering in
            isHovered = hovering
        }
    }
}

/// A simple flow layout that wraps items to the next line
struct FlowLayout: Layout {
    var spacing: CGFloat

    func sizeThatFits(proposal: ProposedViewSize, subviews: Subviews, cache: inout ()) -> CGSize {
        let result = layout(proposal: proposal, subviews: subviews)
        return result.size
    }

    func placeSubviews(in bounds: CGRect, proposal: ProposedViewSize, subviews: Subviews, cache: inout ()) {
        let result = layout(proposal: proposal, subviews: subviews)
        for (index, position) in result.positions.enumerated() {
            subviews[index].place(at: CGPoint(x: bounds.minX + position.x, y: bounds.minY + position.y), proposal: .unspecified)
        }
    }

    private func layout(proposal: ProposedViewSize, subviews: Subviews) -> (size: CGSize, positions: [CGPoint]) {
        let maxWidth = proposal.width ?? .infinity
        var positions: [CGPoint] = []
        var currentX: CGFloat = 0
        var currentY: CGFloat = 0
        var lineHeight: CGFloat = 0
        var maxX: CGFloat = 0

        for subview in subviews {
            let size = subview.sizeThatFits(.unspecified)
            if currentX + size.width > maxWidth && currentX > 0 {
                currentX = 0
                currentY += lineHeight + spacing
                lineHeight = 0
            }
            positions.append(CGPoint(x: currentX, y: currentY))
            lineHeight = max(lineHeight, size.height)
            currentX += size.width + spacing
            maxX = max(maxX, currentX)
        }

        return (CGSize(width: maxX, height: currentY + lineHeight), positions)
    }
}
