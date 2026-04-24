import SwiftUI

// MARK: - EntertainmentCard

struct EntertainmentCard: View {
    let post: Post

    var body: some View {
        if let data = post.entertainmentData {
            EntertainmentCardContent(post: post, data: data)
        } else {
            EntertainmentFallbackCard(post: post)
        }
    }
}

// MARK: - Full card

private struct EntertainmentCardContent: View {
    let post: Post
    let data: EntertainmentData

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            heroSection
            contentSection
        }
    }

    // MARK: Hero image with overlay badges and subject name

    private var heroSection: some View {
        ZStack(alignment: .bottom) {
            heroImage

            // Warm gradient overlay at bottom
            LinearGradient(
                colors: [.clear, .black.opacity(0.6)],
                startPoint: .center,
                endPoint: .bottom
            )

            // Subject name at bottom of image
            HStack {
                Text(data.subject)
                    .font(.system(size: 20, weight: .semibold))
                    .foregroundColor(.white)
                    .lineLimit(1)
                    .shadow(radius: 2)
                Spacer()
            }
            .padding(.horizontal, 12)
            .padding(.bottom, 10)

            // Floating badges pinned to top
            VStack {
                HStack(alignment: .top) {
                    categoryBadge
                    Spacer()
                    sourceBadge
                }
                .padding(.horizontal, 10)
                .padding(.top, 10)
                Spacer()
            }
        }
        .frame(height: 200)
        .clipped()
    }

    @ViewBuilder
    private var heroImage: some View {
        if let urlString = data.subjectImageUrl, let url = URL(string: urlString) {
            AsyncImage(url: url) { phase in
                switch phase {
                case .success(let img): img.resizable().scaledToFill()
                default: fallbackBackground
                }
            }
            .frame(height: 200)
            .clipped()
        } else if let urlString = post.imageURL, let url = URL(string: urlString) {
            AsyncImage(url: url) { phase in
                switch phase {
                case .success(let img): img.resizable().scaledToFill()
                default: fallbackBackground
                }
            }
            .frame(height: 200)
            .clipped()
        } else {
            fallbackBackground.frame(height: 200)
        }
    }

    private var fallbackBackground: some View {
        LinearGradient(
            colors: [Color(hexString: "#2D1B69"), Color(hexString: "#1A1035")],
            startPoint: .topLeading,
            endPoint: .bottomTrailing
        )
    }

    private var categoryBadge: some View {
        Text(data.categoryLabel)
            .font(.system(size: 10, weight: .bold))
            .foregroundColor(.white)
            .padding(.horizontal, 8)
            .padding(.vertical, 4)
            .background(data.categoryBadgeColor)
            .clipShape(Capsule())
    }

    private var sourceBadge: some View {
        Text(data.source)
            .font(.system(size: 10, weight: .semibold))
            .foregroundColor(Color(.label))
            .padding(.horizontal, 8)
            .padding(.vertical, 4)
            .background(Color(.systemBackground).opacity(0.92))
            .clipShape(Capsule())
    }

    // MARK: Content below the image

    private var contentSection: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(data.headline)
                .font(.system(size: 16, weight: .semibold))
                .foregroundColor(Color(.label))
                .lineLimit(2)

            if let quote = data.quote, !quote.isEmpty {
                quoteStrip(quote)
            }

            if let project = data.relatedProject, !project.isEmpty {
                Text("re: \(project)")
                    .font(.system(size: 12))
                    .foregroundColor(Color(.secondaryLabel))
                    .padding(.horizontal, 8)
                    .padding(.vertical, 4)
                    .background(Color(.tertiarySystemFill))
                    .clipShape(Capsule())
            }

            Text("\(post.relativeTime) · \(data.source)")
                .font(.caption)
                .foregroundColor(Color(.tertiaryLabel))

            if let tags = data.tags, !tags.isEmpty {
                HStack(spacing: 6) {
                    ForEach(tags.prefix(3), id: \.self) { tag in
                        Text("#\(tag)")
                            .font(.system(size: 11))
                            .foregroundColor(Color(.secondaryLabel))
                            .padding(.horizontal, 7)
                            .padding(.vertical, 3)
                            .background(Color(.systemFill))
                            .clipShape(Capsule())
                    }
                }
            }

            EntertainmentFooter(post: post)
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 12)
        .background(Color(.systemBackground))
    }

    private func quoteStrip(_ quote: String) -> some View {
        HStack(alignment: .top, spacing: 8) {
            RoundedRectangle(cornerRadius: 2)
                .fill(data.categoryBadgeColor)
                .frame(width: 4)
            Text(quote)
                .font(.system(size: 13))
                .italic()
                .foregroundColor(Color(.secondaryLabel))
                .lineLimit(3)
        }
    }
}

// MARK: - Footer (replicated from CardFooter pattern)

private struct EntertainmentFooter: View {
    let post: Post
    @State var isBookmarked: Bool
    @State private var activeReaction: String?
    @EnvironmentObject private var apiService: APIService

    init(post: Post) {
        self.post = post
        self._isBookmarked = State(initialValue: post.saved ?? false)
        self._activeReaction = State(initialValue: post.myReaction)
    }

    var body: some View {
        HStack(spacing: 6) {
            if let locality = post.locality, !locality.isEmpty {
                Label(locality, systemImage: "link")
                    .font(.caption2)
                    .foregroundColor(.secondary)
                    .lineLimit(1)
            }
            Spacer()
            ReactionPicker(
                activeReaction: $activeReaction,
                postID: post.id,
                style: .feedCompact
            )
            Button {
                let wasSaved = isBookmarked
                UIImpactFeedbackGenerator(style: .light).impactOccurred()
                isBookmarked.toggle()
                Task {
                    await apiService.trackEvent(postID: post.id, eventType: wasSaved ? "unsave" : "save")
                }
            } label: {
                Image(systemName: isBookmarked ? "bookmark.fill" : "bookmark")
                    .font(.caption)
                    .foregroundColor(isBookmarked ? Color(hexString: "#F59E0B") : .secondary)
                    .contentTransition(.symbolEffect(.replace))
            }
            .buttonStyle(.plain)
        }
    }
}

// MARK: - Fallback when JSON is missing/malformed

private struct EntertainmentFallbackCard: View {
    let post: Post
    @State var isBookmarked: Bool
    @State private var activeReaction: String?
    @EnvironmentObject private var apiService: APIService

    init(post: Post) {
        self.post = post
        self._isBookmarked = State(initialValue: post.saved ?? false)
        self._activeReaction = State(initialValue: post.myReaction)
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(post.title)
                .font(.headline)
                .foregroundColor(.primary)
            Text(post.body)
                .font(.subheadline)
                .foregroundColor(.secondary)
                .lineLimit(3)
            HStack(spacing: 6) {
                Spacer()
                ReactionPicker(
                    activeReaction: $activeReaction,
                    postID: post.id,
                    style: .feedCompact
                )
                Button {
                    let wasSaved = isBookmarked
                    UIImpactFeedbackGenerator(style: .light).impactOccurred()
                    isBookmarked.toggle()
                    Task {
                        await apiService.trackEvent(postID: post.id, eventType: wasSaved ? "unsave" : "save")
                    }
                } label: {
                    Image(systemName: isBookmarked ? "bookmark.fill" : "bookmark")
                        .font(.caption)
                        .foregroundColor(isBookmarked ? Color(hexString: "#F59E0B") : .secondary)
                        .contentTransition(.symbolEffect(.replace))
                }
                .buttonStyle(.plain)
            }
        }
        .padding(16)
        .background(Color(.secondarySystemGroupedBackground))
    }
}
