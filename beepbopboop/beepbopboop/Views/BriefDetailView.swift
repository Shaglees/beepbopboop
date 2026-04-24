import SwiftUI

struct BriefDetailView: View {
    let post: Post
    @State private var isBookmarked: Bool
    @State private var activeReaction: String?
    @Environment(\.dismiss) private var dismiss
    @State private var expandedSections: Set<Int> = [0]
    @EnvironmentObject private var apiService: APIService

    init(post: Post) {
        self.post = post
        self._isBookmarked = State(initialValue: post.saved ?? false)
        self._activeReaction = State(initialValue: post.myReaction)
    }

    private var isDigest: Bool { post.displayHintValue == .digest }

    private var sections: [(title: String, body: String)] {
        let lines = post.body.components(separatedBy: "\n\n")
            .map { $0.trimmingCharacters(in: .whitespacesAndNewlines) }
            .filter { !$0.isEmpty }

        return lines.enumerated().map { (idx, text) -> (title: String, body: String) in
            let parts = text.components(separatedBy: "\n")
            if parts.count > 1 {
                let title = parts[0]
                    .replacingOccurrences(of: #"^\d+\.\s*"#, with: "", options: .regularExpression)
                    .replacingOccurrences(of: "**", with: "")
                    .trimmingCharacters(in: .whitespaces)
                let body = parts.dropFirst().joined(separator: "\n")
                if title.count < 80 && !body.isEmpty {
                    return (title: title, body: body)
                }
            }
            return (title: "Section \(idx + 1)", body: text)
        }
    }

    private var readTimeMinutes: Int {
        let wordCount = post.body.split(separator: " ").count
        return max(1, wordCount / 200)
    }

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 0) {
                headerCard
                tableOfContents
                sectionsList
                Divider().padding(.top, 16)
                engagementBar
            }
        }
        .navigationTitle(isDigest ? "Digest" : "Brief")
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button { dismiss() } label: {
                    Image(systemName: "xmark.circle.fill").foregroundStyle(.secondary)
                }
            }
        }
    }

    // MARK: - Header Card

    private var headerCard: some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack {
                Image(systemName: isDigest ? "list.bullet.rectangle" : "checklist")
                    .font(.title2)
                    .foregroundStyle(post.hintColor)
                Text(isDigest ? "DIGEST" : "BRIEF")
                    .font(.caption.weight(.black))
                    .kerning(2)
                    .foregroundStyle(post.hintColor)
                Spacer()
                Label("\(readTimeMinutes) min read", systemImage: "clock")
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }

            Text(post.title)
                .font(.title2.weight(.bold))

            HStack(spacing: 6) {
                Circle().fill(post.hintColor).frame(width: 8, height: 8)
                Text(post.agentName).font(.caption).foregroundStyle(.secondary)
                Text("·").foregroundStyle(.secondary)
                Text(post.relativeTime).font(.caption).foregroundStyle(.secondary)
                Spacer()
                Text("\(sections.count) sections")
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }
        }
        .padding(16)
        .background(post.hintColor.opacity(0.08), in: Rectangle())
    }

    // MARK: - Table of Contents

    @ViewBuilder
    private var tableOfContents: some View {
        if sections.count > 2 {
            VStack(alignment: .leading, spacing: 6) {
                Text("CONTENTS")
                    .font(.caption.weight(.bold))
                    .kerning(1.5)
                    .foregroundStyle(.secondary)
                    .padding(.horizontal, 16)
                    .padding(.top, 16)

                ScrollView(.horizontal, showsIndicators: false) {
                    HStack(spacing: 8) {
                        ForEach(sections.indices, id: \.self) { idx in
                            tocChip(idx: idx)
                        }
                    }
                    .padding(.horizontal, 16)
                    .padding(.bottom, 12)
                }
            }
            Divider()
        }
    }

    private func tocChip(idx: Int) -> some View {
        Button {
            withAnimation(.spring(response: 0.3)) {
                var s = expandedSections
                s.insert(idx)
                expandedSections = s
            }
        } label: {
            Text("\(idx + 1). \(sections[idx].title)")
                .font(.caption.weight(.medium))
                .lineLimit(1)
                .padding(.horizontal, 10)
                .padding(.vertical, 6)
                .background(
                    expandedSections.contains(idx)
                        ? post.hintColor.opacity(0.15)
                        : Color.secondary.opacity(0.1),
                    in: Capsule()
                )
                .foregroundStyle(
                    expandedSections.contains(idx)
                        ? post.hintColor
                        : Color.secondary
                )
        }
        .buttonStyle(.plain)
    }

    // MARK: - Sections List

    private var sectionsList: some View {
        VStack(alignment: .leading, spacing: 0) {
            ForEach(sections.indices, id: \.self) { idx in
                sectionRow(idx: idx)
                if idx < sections.count - 1 {
                    Divider().padding(.leading, 16)
                }
            }
        }
        .padding(.top, 8)
    }

    private func sectionRow(idx: Int) -> some View {
        let section = sections[idx]
        let isExpanded = expandedSections.contains(idx)
        return VStack(alignment: .leading, spacing: 0) {
            Button {
                withAnimation(.spring(response: 0.35, dampingFraction: 0.8)) {
                    if isExpanded {
                        _ = expandedSections.remove(idx)
                    } else {
                        _ = expandedSections.insert(idx)
                    }
                }
            } label: {
                HStack(spacing: 12) {
                    Text("\(idx + 1)")
                        .font(.caption.weight(.black))
                        .foregroundStyle(.white)
                        .frame(width: 22, height: 22)
                        .background(post.hintColor, in: Circle())

                    Text(section.title)
                        .font(.subheadline.weight(.semibold))
                        .foregroundStyle(.primary)
                        .multilineTextAlignment(.leading)

                    Spacer()

                    Image(systemName: isExpanded ? "chevron.up" : "chevron.down")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
                .padding(.horizontal, 16)
                .padding(.vertical, 14)
            }
            .buttonStyle(.plain)

            if isExpanded {
                Text(section.body)
                    .font(.body)
                    .foregroundStyle(.primary)
                    .lineSpacing(5)
                    .padding(.horizontal, 16)
                    .padding(.bottom, 16)
                    .padding(.leading, 34)
                    .transition(.move(edge: .top).combined(with: .opacity))
            }
        }
    }

    // MARK: - Engagement Bar

    private var engagementBar: some View {
        HStack(spacing: 12) {
            Button {
                let wasSaved = isBookmarked
                withAnimation(.bouncy) { isBookmarked.toggle() }
                UIImpactFeedbackGenerator(style: .light).impactOccurred()
                Task {
                    await apiService.trackEvent(
                        postID: post.id,
                        eventType: wasSaved ? "unsave" : "save"
                    )
                }
            } label: {
                Label(
                    isBookmarked ? "Bookmarked" : "Bookmark",
                    systemImage: isBookmarked ? "bookmark.fill" : "bookmark"
                )
                .font(.subheadline)
                .foregroundColor(isBookmarked ? post.hintColor : .secondary)
                .symbolEffect(.bounce, value: isBookmarked)
                .contentTransition(.symbolEffect(.replace))
            }
            .buttonStyle(.plain)

            ReactionPicker(
                activeReaction: $activeReaction,
                postID: post.id,
                style: .detailBar
            )

            Spacer()

            ShareLink(
                item: post.shareURL,
                subject: Text(post.title),
                message: Text(post.body.prefix(100))
            ) {
                Label("Share", systemImage: "square.and.arrow.up")
                    .font(.subheadline)
                    .foregroundColor(.secondary)
            }
            .simultaneousGesture(TapGesture().onEnded {
                Task { await apiService.trackEvent(postID: post.id, type: "share") }
            })
        }
        .padding(16)
    }
}
