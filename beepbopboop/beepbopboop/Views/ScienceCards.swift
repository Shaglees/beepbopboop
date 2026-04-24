import SwiftUI

// MARK: - Science Card

struct ScienceCard: View {
    let post: Post
    let science: ScienceData
    @State private var activeReaction: String?
    @State var isBookmarked: Bool
    @EnvironmentObject private var apiService: APIService

    init?(post: Post) {
        guard let sd = post.scienceData else { return nil }
        self.post = post
        self.science = sd
        self._activeReaction = State(initialValue: post.myReaction)
        self._isBookmarked = State(initialValue: post.saved ?? false)
    }

    var body: some View {
        VStack(spacing: 0) {
            // Header
            HStack(spacing: 6) {
                ZStack {
                    Circle()
                        .fill(science.categoryAccentColor)
                        .frame(width: 20, height: 20)
                    Text(String(post.agentName.prefix(1)))
                        .font(.caption2.weight(.bold))
                        .foregroundColor(.white)
                }
                Text(post.agentName)
                    .font(.subheadline.weight(.medium))
                    .foregroundStyle(.white.opacity(0.9))
                HStack(spacing: 4) {
                    Circle()
                        .fill(science.categoryAccentColor)
                        .frame(width: 4, height: 4)
                    Text("Science")
                        .font(.system(size: 10, weight: .bold))
                        .tracking(0.8)
                        .textCase(.uppercase)
                }
                .foregroundColor(science.categoryAccentColor)
                .lineLimit(1)
                .fixedSize()
                .padding(.horizontal, 8)
                .padding(.vertical, 4)
                .background(
                    Capsule()
                        .stroke(science.categoryAccentColor.opacity(0.22), lineWidth: 1)
                )
                Spacer()
                Text(post.relativeTime)
                    .font(.caption2.weight(.medium))
                    .monospacedDigit()
                    .foregroundStyle(.white.opacity(0.4))
            }
            .padding(.horizontal, 16)
            .padding(.top, 14)
            .padding(.bottom, 10)

            // Hero image with overlays
            heroSection

            // Body + metadata
            contentSection
        }
        .background(
            LinearGradient(
                stops: [
                    .init(color: science.categoryColor, location: 0),
                    .init(color: science.categoryColor.opacity(0.85), location: 1),
                ],
                startPoint: .top,
                endPoint: .bottom
            )
        )
    }

    // MARK: Hero

    @ViewBuilder
    private var heroSection: some View {
        if let urlStr = science.heroImageUrl, let url = URL(string: urlStr) {
            ZStack(alignment: .bottom) {
                AsyncImage(url: url) { phase in
                    switch phase {
                    case .success(let image):
                        image
                            .resizable()
                            .aspectRatio(contentMode: .fill)
                    case .failure:
                        categoryPlaceholder
                    default:
                        Rectangle()
                            .fill(science.categoryColor.opacity(0.5))
                            .overlay(ProgressView().tint(.white.opacity(0.5)))
                    }
                }
                .frame(height: 220)
                .clipped()

                // Gradient scrim for text legibility
                LinearGradient(
                    stops: [
                        .init(color: .clear, location: 0),
                        .init(color: .black.opacity(0.75), location: 1),
                    ],
                    startPoint: .top,
                    endPoint: .bottom
                )

                // Category badge (top-left) + source (top-right)
                HStack(alignment: .top) {
                    categoryBadge
                    Spacer()
                    sourceLabel
                }
                .padding(.horizontal, 12)
                .padding(.vertical, 10)
                .frame(maxHeight: .infinity, alignment: .top)

                // Headline + institution at bottom
                VStack(alignment: .leading, spacing: 3) {
                    Text(science.headline)
                        .font(.system(size: 18, weight: .bold))
                        .foregroundStyle(.white)
                        .lineLimit(2)
                    if let institution = science.institution {
                        Text(institution)
                            .font(.caption.italic())
                            .foregroundStyle(.white.opacity(0.65))
                            .lineLimit(1)
                    }
                }
                .frame(maxWidth: .infinity, alignment: .leading)
                .padding(.horizontal, 14)
                .padding(.bottom, 12)
            }
            .frame(height: 220)
            .clipped()
        } else {
            // No hero image: fallback to decorated placeholder
            ZStack {
                science.categoryColor.opacity(0.6)
                categoryPlaceholder
                VStack(alignment: .leading, spacing: 4) {
                    categoryBadge
                    Spacer()
                    Text(science.headline)
                        .font(.system(size: 18, weight: .bold))
                        .foregroundStyle(.white)
                        .lineLimit(2)
                    if let institution = science.institution {
                        Text(institution)
                            .font(.caption.italic())
                            .foregroundStyle(.white.opacity(0.65))
                    }
                }
                .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
                .padding(14)
            }
            .frame(height: 160)
            .clipped()
        }
    }

    private var categoryPlaceholder: some View {
        Image(systemName: science.categoryIcon)
            .font(.system(size: 90, weight: .ultraLight))
            .foregroundStyle(science.categoryAccentColor.opacity(0.15))
    }

    private var categoryBadge: some View {
        HStack(spacing: 5) {
            Image(systemName: science.categoryIcon)
                .font(.caption2.weight(.semibold))
            Text(science.categoryLabel)
                .font(.system(size: 10, weight: .bold, design: .monospaced))
                .tracking(0.6)
                .textCase(.uppercase)
        }
        .foregroundStyle(.white)
        .padding(.horizontal, 10)
        .padding(.vertical, 5)
        .background(.ultraThinMaterial.opacity(0.85))
        .clipShape(Capsule())
    }

    private var sourceLabel: some View {
        Text(science.source)
            .font(.caption2.weight(.medium))
            .foregroundStyle(.white.opacity(0.75))
            .padding(.horizontal, 8)
            .padding(.vertical, 4)
            .background(.black.opacity(0.3))
            .clipShape(Capsule())
    }

    // MARK: Content

    private var contentSection: some View {
        VStack(alignment: .leading, spacing: 10) {
            // Body text
            Text(post.body)
                .font(.subheadline)
                .foregroundStyle(.white.opacity(0.85))
                .lineLimit(3)
                .fixedSize(horizontal: false, vertical: true)

            // Tags
            if !science.tags.isEmpty {
                ScrollView(.horizontal, showsIndicators: false) {
                    HStack(spacing: 6) {
                        ForEach(science.tags.prefix(3), id: \.self) { tag in
                            Text(tag)
                                .font(.caption2.weight(.medium))
                                .foregroundStyle(science.categoryAccentColor)
                                .padding(.horizontal, 8)
                                .padding(.vertical, 4)
                                .background(science.categoryAccentColor.opacity(0.15))
                                .clipShape(Capsule())
                        }
                    }
                }
            }

            // Date + Read More
            HStack {
                if let dateStr = science.formattedDate {
                    Text(dateStr)
                        .font(.caption2)
                        .foregroundStyle(.white.opacity(0.45))
                }
                Spacer()
                if let readUrl = science.primaryReadUrl {
                    Link(destination: readUrl) {
                        HStack(spacing: 3) {
                            Text("Read full study")
                                .font(.caption2.weight(.medium))
                            Image(systemName: "arrow.right")
                                .font(.caption2)
                        }
                        .foregroundStyle(science.categoryAccentColor)
                    }
                }
            }

            // Footer
            HStack(spacing: 6) {
                if let locality = post.locality, !locality.isEmpty {
                    Label(locality, systemImage: "link")
                        .font(.caption2)
                        .foregroundColor(.white.opacity(0.4))
                        .lineLimit(1)
                }
                Spacer()
                ReactionPicker(
                    activeReaction: $activeReaction,
                    postID: post.id,
                    style: .feedDark
                )
                Button {
                    UIImpactFeedbackGenerator(style: .light).impactOccurred()
                    let wasSaved = isBookmarked
                    isBookmarked.toggle()
                    Task {
                        do { try await apiService.trackEvent(postID: post.id, eventType: wasSaved ? "unsave" : "save") }
                        catch { isBookmarked = wasSaved }
                    }
                } label: {
                    Image(systemName: isBookmarked ? "bookmark.fill" : "bookmark")
                        .font(.caption)
                        .foregroundColor(isBookmarked ? science.categoryAccentColor : .white.opacity(0.4))
                        .contentTransition(.symbolEffect(.replace))
                }
                .buttonStyle(.plain)
            }
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 14)
    }
}
