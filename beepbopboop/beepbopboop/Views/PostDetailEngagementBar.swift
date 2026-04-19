import SwiftUI

/// Reusable engagement bar for custom detail views.
/// Provides bookmark, reactions, share, and external link actions.
struct PostDetailEngagementBar: View {
    let post: Post
    @State private var isBookmarked: Bool
    @State private var activeReaction: String?
    @EnvironmentObject private var apiService: APIService
    @EnvironmentObject private var eventTracker: EventTracker

    init(post: Post) {
        self.post = post
        self._isBookmarked = State(initialValue: post.saved ?? false)
        self._activeReaction = State(initialValue: post.myReaction)
    }

    var body: some View {
        HStack(spacing: 12) {
            Button {
                let wasSaved = isBookmarked
                withAnimation(.bouncy) { isBookmarked.toggle() }
                UIImpactFeedbackGenerator(style: .light).impactOccurred()
                Task {
                    do {
                        try await apiService.trackEvent(postID: post.id, eventType: wasSaved ? "unsave" : "save")
                    } catch {
                        withAnimation(.bouncy) { isBookmarked = wasSaved }
                    }
                }
            } label: {
                Label(
                    isBookmarked ? "Bookmarked" : "Bookmark",
                    systemImage: isBookmarked ? "bookmark.fill" : "bookmark"
                )
                .font(.subheadline)
                .foregroundColor(isBookmarked ? post.typeColor : .secondary)
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

            if let externalURL = post.externalURL,
               !externalURL.isEmpty,
               let url = URL(string: externalURL) {
                Link(destination: url) {
                    Label("Open", systemImage: "arrow.up.right.square")
                        .font(.subheadline)
                }
            }
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 12)
        .glassEffect(.regular, in: .rect(cornerRadius: 16))
    }
}
