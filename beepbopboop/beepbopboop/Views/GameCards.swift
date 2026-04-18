import SwiftUI

// MARK: - Game Release Card

struct GameReleaseCard: View {
    let post: Post
    let game: VideoGameData
    @State private var activeReaction: String?

    init?(post: Post) {
        guard let g = post.videoGameData else { return nil }
        self.post = post
        self.game = g
        self._activeReaction = State(initialValue: post.myReaction)
    }

    var body: some View {
        ZStack(alignment: .bottom) {
            // Cover art hero
            heroBackground

            // Dark gradient overlay for text legibility
            LinearGradient(
                stops: [
                    .init(color: .clear, location: 0.2),
                    .init(color: Color(white: 0.05).opacity(0.85), location: 0.65),
                    .init(color: Color(white: 0.05).opacity(0.98), location: 1.0),
                ],
                startPoint: .top,
                endPoint: .bottom
            )

            VStack(alignment: .leading, spacing: 0) {
                // Platform badges — top left overlay
                HStack(spacing: 6) {
                    ForEach(game.platforms.prefix(4), id: \.self) { platform in
                        PlatformBadge(platform: platform)
                    }
                    Spacer()
                }
                .padding(.horizontal, 14)
                .padding(.top, 14)

                Spacer()

                // Bottom content block
                VStack(alignment: .leading, spacing: 8) {
                    // Countdown strip
                    if let countdown = game.releaseCountdown {
                        Text(countdown)
                            .font(.system(size: 11, weight: .heavy))
                            .tracking(1.5)
                            .foregroundStyle(Color(red: 0.96, green: 0.62, blue: 0.04))
                    }

                    // Title
                    Text(game.title)
                        .font(.system(size: 20, weight: .bold, design: .rounded))
                        .foregroundStyle(.white)
                        .lineLimit(2)

                    // Developer credit
                    if let dev = game.developer {
                        let pub = game.publisher.flatMap { $0 != dev ? $0 : nil }
                        Text([dev, pub].compactMap { $0 }.joined(separator: " · "))
                            .font(.caption.weight(.medium))
                            .foregroundStyle(.white.opacity(0.55))
                    }

                    // Genre tags
                    if !game.genres.isEmpty {
                        HStack(spacing: 6) {
                            ForEach(game.genres.prefix(3), id: \.self) { genre in
                                GenreTag(genre: genre)
                            }
                        }
                    }

                    // Footer: reactions
                    HStack {
                        Spacer()
                        ReactionPicker(
                            activeReaction: $activeReaction,
                            postID: post.id,
                            style: .feedDark
                        )
                    }
                }
                .padding(.horizontal, 14)
                .padding(.bottom, 14)
            }
        }
        .frame(height: 280)
        .background(Color(white: 0.07))
    }

    @ViewBuilder
    private var heroBackground: some View {
        if let urlStr = game.coverUrl, let url = URL(string: urlStr) {
            AsyncImage(url: url) { phase in
                switch phase {
                case .success(let image):
                    image
                        .resizable()
                        .aspectRatio(contentMode: .fill)
                default:
                    Rectangle().fill(game.accentColor.opacity(0.3))
                }
            }
            .frame(maxWidth: .infinity, maxHeight: .infinity)
            .clipped()
        } else {
            Rectangle().fill(game.accentColor.opacity(0.3))
        }
    }
}

// MARK: - Game Review Card

struct GameReviewCard: View {
    let post: Post
    let game: VideoGameData
    @State private var activeReaction: String?

    init?(post: Post) {
        guard let g = post.videoGameData else { return nil }
        self.post = post
        self.game = g
        self._activeReaction = State(initialValue: post.myReaction)
    }

    var body: some View {
        VStack(spacing: 0) {
            // Header: cover art + score
            HStack(alignment: .top, spacing: 14) {
                // Cover art
                coverArtView
                    .frame(width: 110, height: 148)
                    .clipShape(RoundedRectangle(cornerRadius: 8))

                // Right column: score + meta
                VStack(alignment: .leading, spacing: 10) {
                    // Score circle
                    if let rating = game.rating {
                        HStack(spacing: 10) {
                            ScoreCircle(score: rating, color: game.ratingColor)
                            VStack(alignment: .leading, spacing: 2) {
                                Text("Metacritic")
                                    .font(.caption2.weight(.semibold))
                                    .foregroundStyle(.secondary)
                                if let count = game.formattedReviewCount {
                                    Text("\(count) reviews")
                                        .font(.caption2)
                                        .foregroundStyle(.tertiary)
                                }
                            }
                        }
                    }

                    // Platform badges
                    HStack(spacing: 5) {
                        ForEach(game.platforms.prefix(3), id: \.self) { platform in
                            PlatformBadge(platform: platform)
                        }
                    }

                    // Release date
                    if let dateStr = game.formattedReleaseDate {
                        Label(dateStr, systemImage: "calendar")
                            .font(.caption2)
                            .foregroundStyle(.secondary)
                    }

                    // Developer
                    if let dev = game.developer {
                        Label(dev, systemImage: "person.fill")
                            .font(.caption2)
                            .foregroundStyle(.secondary)
                            .lineLimit(1)
                    }

                    Spacer()
                }
                .frame(maxWidth: .infinity, alignment: .leading)
            }
            .padding(14)

            // Divider
            Divider()
                .padding(.horizontal, 14)

            // Body section
            VStack(alignment: .leading, spacing: 8) {
                Text(game.title)
                    .font(.system(size: 15, weight: .semibold, design: .rounded))
                    .foregroundStyle(.primary)
                    .lineLimit(2)

                if !post.body.isEmpty {
                    Text(post.body)
                        .font(.caption)
                        .foregroundStyle(.secondary)
                        .lineLimit(3)
                }

                // Steam bar
                if let pct = game.steamPositivePercent, let count = game.formattedReviewCount {
                    SteamRatingBar(ratio: game.steamPositiveRatio ?? 0, percent: pct, count: count)
                }

                // Footer
                HStack {
                    // Price
                    if let price = game.displayPrice {
                        Text(price)
                            .font(.caption.weight(.semibold))
                            .foregroundStyle(game.steamDiscount ?? 0 > 0 ? Color(red: 0.2, green: 0.78, blue: 0.35) : .primary)
                    }
                    Spacer()
                    ReactionPicker(
                        activeReaction: $activeReaction,
                        postID: post.id,
                        style: .feedCompact
                    )
                }
            }
            .padding(.horizontal, 14)
            .padding(.vertical, 10)
        }
        .frame(minHeight: 240)
        .background(Color(.secondarySystemGroupedBackground))
    }

    @ViewBuilder
    private var coverArtView: some View {
        if let urlStr = game.coverUrl, let url = URL(string: urlStr) {
            AsyncImage(url: url) { phase in
                switch phase {
                case .success(let image):
                    image.resizable().aspectRatio(contentMode: .fill)
                default:
                    Rectangle().fill(game.accentColor.opacity(0.25))
                }
            }
        } else {
            ZStack {
                Rectangle().fill(game.accentColor.opacity(0.25))
                Image(systemName: "gamecontroller")
                    .font(.title)
                    .foregroundStyle(game.accentColor.opacity(0.6))
            }
        }
    }
}

// MARK: - Shared Components

private struct PlatformBadge: View {
    let platform: String

    var body: some View {
        Text(VideoGameData.platformAbbr(platform))
            .font(.system(size: 10, weight: .bold))
            .foregroundStyle(.white)
            .padding(.horizontal, 7)
            .padding(.vertical, 3)
            .background(
                Capsule()
                    .fill(VideoGameData.platformColor(platform))
            )
    }
}

private struct GenreTag: View {
    let genre: String

    var body: some View {
        Text(genre)
            .font(.system(size: 10, weight: .medium))
            .foregroundStyle(.white.opacity(0.75))
            .padding(.horizontal, 8)
            .padding(.vertical, 3)
            .background(
                Capsule()
                    .fill(Color(white: 0.2))
            )
    }
}

private struct ScoreCircle: View {
    let score: Int
    let color: Color

    var body: some View {
        ZStack {
            Circle()
                .stroke(color.opacity(0.25), lineWidth: 3)
            Circle()
                .trim(from: 0, to: CGFloat(score) / 100)
                .stroke(color, style: StrokeStyle(lineWidth: 3, lineCap: .round))
                .rotationEffect(.degrees(-90))
            VStack(spacing: 0) {
                Text("\(score)")
                    .font(.system(size: 18, weight: .black, design: .rounded))
                    .foregroundStyle(color)
                Text("/100")
                    .font(.system(size: 8, weight: .medium))
                    .foregroundStyle(.tertiary)
            }
        }
        .frame(width: 58, height: 58)
    }
}

private struct SteamRatingBar: View {
    let ratio: Double
    let percent: String
    let count: String

    var body: some View {
        VStack(alignment: .leading, spacing: 4) {
            GeometryReader { geo in
                ZStack(alignment: .leading) {
                    Capsule()
                        .fill(Color(white: 0.15))
                        .frame(height: 6)
                    Capsule()
                        .fill(Color(red: 0.11, green: 0.44, blue: 0.79))
                        .frame(width: geo.size.width * ratio, height: 6)
                }
            }
            .frame(height: 6)
            HStack(spacing: 4) {
                Text("\(percent) Positive")
                    .font(.caption2.weight(.semibold))
                    .foregroundStyle(Color(red: 0.11, green: 0.44, blue: 0.79))
                Text("· \(count) reviews")
                    .font(.caption2)
                    .foregroundStyle(.tertiary)
            }
        }
    }
}
