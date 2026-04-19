import SwiftUI

// MARK: - Palette

private let creatorIndigo  = Color(red: 0.380, green: 0.333, blue: 0.933) // #6155EE
private let creatorAmber   = Color(red: 0.969, green: 0.706, blue: 0.118) // #F7B41E
private let creatorCream   = Color(red: 0.992, green: 0.984, blue: 0.969) // #FDFBF7

// MARK: - CreatorSpotlightCard

struct CreatorSpotlightCard: View {
    let post: Post
    let creator: CreatorData

    init?(post: Post) {
        guard post.displayHintValue == .creatorSpotlight,
              let cd = post.creatorData else { return nil }
        self.post = post
        self.creator = cd
    }

    var body: some View {
        VStack(spacing: 0) {
            headerSection
            bodySection
        }
        .background(creatorCream)
    }

    // MARK: Header

    private var headerSection: some View {
        ZStack(alignment: .bottomLeading) {
            // Gradient background with subtle pattern.
            LinearGradient(
                colors: [creatorIndigo, creatorIndigo.opacity(0.7)],
                startPoint: .topLeading,
                endPoint: .bottomTrailing
            )
            .frame(height: 120)

            // Large designation symbol as decorative backdrop.
            Image(systemName: creator.designationSymbol)
                .font(.system(size: 72, weight: .ultraLight))
                .foregroundStyle(.white.opacity(0.08))
                .frame(maxWidth: .infinity, alignment: .trailing)
                .padding(.trailing, 16)
                .padding(.bottom, 16)

            // Creator name + designation badge.
            VStack(alignment: .leading, spacing: 6) {
                cardHeader
                Text(creator.name)
                    .font(.system(size: 20, weight: .bold))
                    .foregroundStyle(.white)
                    .lineLimit(1)
            }
            .padding(14)
        }
        .frame(height: 120)
        .clipped()
    }

    private var cardHeader: some View {
        HStack(spacing: 6) {
            Circle()
                .fill(creatorAmber)
                .frame(width: 8, height: 8)
            Text(post.agentName)
                .font(.subheadline.weight(.medium))
                .foregroundStyle(.white)

            designationBadge

            Spacer()
            Text(post.relativeTime)
                .font(.subheadline)
                .foregroundStyle(.white.opacity(0.5))
        }
    }

    private var designationBadge: some View {
        HStack(spacing: 4) {
            Text(creator.designationIcon)
                .font(.caption2)
            Text(creator.designation.capitalized)
                .font(.caption2.weight(.semibold))
        }
        .foregroundStyle(creatorAmber)
        .padding(.horizontal, 7)
        .padding(.vertical, 3)
        .background(creatorAmber.opacity(0.18))
        .cornerRadius(4)
    }

    // MARK: Body

    private var bodySection: some View {
        VStack(alignment: .leading, spacing: 10) {
            // Area + distance row.
            HStack(spacing: 4) {
                Image(systemName: "location.fill")
                    .font(.caption2)
                    .foregroundStyle(creatorIndigo)
                Text(creator.areaName)
                    .font(.caption.weight(.medium))
                    .foregroundStyle(creatorIndigo)
                    .lineLimit(1)
            }

            // Bio snippet.
            if !creator.bio.isEmpty {
                Text(creator.bio)
                    .font(.subheadline)
                    .foregroundStyle(.secondary)
                    .lineLimit(3)
            }

            // Tags.
            if let tags = creator.tags, !tags.isEmpty {
                tagRow(tags: tags)
            }

            // Source attribution.
            if let source = creator.source, !source.isEmpty {
                HStack(spacing: 4) {
                    Image(systemName: "magnifyingglass")
                        .font(.caption2)
                    Text("Found via \(source)")
                        .font(.caption2)
                }
                .foregroundStyle(.tertiary)
            }

            CreatorCardFooter(post: post, accentColor: creatorIndigo)
        }
        .padding(14)
    }

    private func tagRow(tags: [String]) -> some View {
        ScrollView(.horizontal, showsIndicators: false) {
            HStack(spacing: 6) {
                ForEach(tags.prefix(4), id: \.self) { tag in
                    Text(tag)
                        .font(.caption2.weight(.medium))
                        .foregroundStyle(.secondary)
                        .padding(.horizontal, 8)
                        .padding(.vertical, 4)
                        .background(Color(.systemGray6), in: RoundedRectangle(cornerRadius: 6))
                }
            }
        }
    }
}

// MARK: - Card Footer

struct CreatorCardFooter: View {
    let post: Post
    let accentColor: Color
    @AppStorage var isBookmarked: Bool
    @State private var activeReaction: String?
    @EnvironmentObject private var apiService: APIService
    @EnvironmentObject private var eventTracker: EventTracker

    init(post: Post, accentColor: Color) {
        self.post = post
        self.accentColor = accentColor
        self._isBookmarked = AppStorage(wrappedValue: false, "bookmark_\(post.id)")
        self._activeReaction = State(initialValue: post.myReaction)
    }

    var body: some View {
        HStack(spacing: 6) {
            if let locality = post.locality, !locality.isEmpty {
                Label(locality, systemImage: "location")
                    .font(.caption2)
                    .foregroundStyle(.secondary)
                    .lineLimit(1)
            }
            Spacer()
            ReactionPicker(activeReaction: $activeReaction, postID: post.id, style: .feedCompact)
            Button {
                UIImpactFeedbackGenerator(style: .light).impactOccurred()
                let wasSaved = isBookmarked
                isBookmarked.toggle()
                Task {
                    do {
                        try await apiService.trackEvent(postID: post.id, eventType: wasSaved ? "unsave" : "save")
                    } catch {
                        isBookmarked = wasSaved
                    }
                }
            } label: {
                Image(systemName: isBookmarked ? "bookmark.fill" : "bookmark")
                    .font(.caption)
                    .foregroundStyle(isBookmarked ? accentColor : .secondary)
                    .contentTransition(.symbolEffect(.replace))
            }
            .buttonStyle(.plain)
        }
    }
}
