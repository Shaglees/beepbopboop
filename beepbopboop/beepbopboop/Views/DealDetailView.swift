import SwiftUI

struct DealDetailView: View {
    let post: Post
    @AppStorage private var isBookmarked: Bool
    @Environment(\.dismiss) private var dismiss
    @State private var activeReaction: String?
    @EnvironmentObject private var apiService: APIService
    @EnvironmentObject private var eventTracker: EventTracker

    init(post: Post) {
        self.post = post
        self._isBookmarked = AppStorage(wrappedValue: post.mySaved, "bookmark_\(post.id)")
        self._activeReaction = State(initialValue: post.myReaction)
    }

    // Try to extract price info from title/body using simple regex patterns
    private var currentPrice: String? {
        let pattern = #"\$[\d,]+\.?\d*"#
        let text = post.title + " " + post.body
        guard let regex = try? NSRegularExpression(pattern: pattern),
              let match = regex.firstMatch(in: text, range: NSRange(text.startIndex..., in: text)),
              let range = Range(match.range, in: text) else { return nil }
        return String(text[range])
    }

    private var savings: String? {
        let pattern = #"(\d+)%\s*off"#
        let text = post.title + " " + post.body
        guard let regex = try? NSRegularExpression(pattern: pattern, options: .caseInsensitive),
              let match = regex.firstMatch(in: text, range: NSRange(text.startIndex..., in: text)),
              let range = Range(match.range(at: 1), in: text) else { return nil }
        return "\(text[range])% OFF"
    }

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 0) {
                // Hero image
                if let imageURL = post.imageURL, !imageURL.isEmpty, let url = URL(string: imageURL) {
                    AsyncImage(url: url) { phase in
                        switch phase {
                        case .success(let img):
                            img.resizable()
                                .aspectRatio(contentMode: .fill)
                                .frame(maxWidth: .infinity)
                                .frame(height: 260)
                                .clipped()
                        case .failure: EmptyView()
                        default: ProgressView().frame(height: 260).frame(maxWidth: .infinity)
                        }
                    }
                }

                VStack(alignment: .leading, spacing: 20) {
                    // Price + savings badge
                    HStack(alignment: .top) {
                        VStack(alignment: .leading, spacing: 4) {
                            if let price = currentPrice {
                                Text(price)
                                    .font(.system(size: 42, weight: .bold))
                                    .foregroundStyle(Color(red: 0.937, green: 0.267, blue: 0.267))
                            }
                        }
                        Spacer()
                        if let save = savings {
                            Text(save)
                                .font(.headline.weight(.black))
                                .foregroundStyle(.white)
                                .padding(.horizontal, 14)
                                .padding(.vertical, 8)
                                .background(
                                    LinearGradient(
                                        colors: [Color(red: 0.937, green: 0.267, blue: 0.267), Color(red: 1.0, green: 0.5, blue: 0.0)],
                                        startPoint: .topLeading, endPoint: .bottomTrailing
                                    ),
                                    in: Capsule()
                                )
                        }
                    }

                    // Title
                    Text(post.title)
                        .font(.title2.weight(.bold))

                    // Body
                    if !post.body.isEmpty {
                        Text(post.body)
                            .font(.body)
                            .foregroundStyle(.secondary)
                            .lineSpacing(4)
                    }

                    // CTA button
                    if let extURL = post.externalURL, !extURL.isEmpty, let url = URL(string: extURL) {
                        Link(destination: url) {
                            HStack {
                                Image(systemName: "cart.fill")
                                Text("View Deal")
                                    .fontWeight(.semibold)
                            }
                            .frame(maxWidth: .infinity)
                            .padding(.vertical, 16)
                            .background(
                                LinearGradient(
                                    colors: [Color(red: 0.937, green: 0.267, blue: 0.267), Color(red: 1.0, green: 0.5, blue: 0.0)],
                                    startPoint: .leading, endPoint: .trailing
                                ),
                                in: RoundedRectangle(cornerRadius: 14)
                            )
                            .foregroundStyle(.white)
                            .font(.headline)
                        }
                    }

                    // Metadata badges
                    HStack(spacing: 8) {
                        Circle().fill(post.hintColor).frame(width: 8, height: 8)
                        Text(post.agentName).font(.caption).foregroundStyle(.secondary)
                        Text("·").foregroundStyle(.secondary)
                        Text(post.relativeTime).font(.caption).foregroundStyle(.secondary)
                    }

                    Divider()
                    engagementBar
                }
                .padding(16)
            }
        }
        .navigationTitle("Deal")
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button { dismiss() } label: {
                    Image(systemName: "xmark.circle.fill").foregroundStyle(.secondary)
                }
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
                    do {
                        if wasSaved {
                            try await apiService.unsavePost(postID: post.id)
                        } else {
                            try await apiService.savePost(postID: post.id)
                        }
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
                .foregroundColor(isBookmarked ? Color(red: 0.937, green: 0.267, blue: 0.267) : .secondary)
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
        .padding(.horizontal, 16)
        .padding(.vertical, 12)
        .glassEffect(.regular, in: .rect(cornerRadius: 16))
    }
}
