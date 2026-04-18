import SwiftUI

struct GameReleaseDetailView: View {
    let post: Post
    @Environment(\.dismiss) private var dismiss

    private var data: VideoGameData? { post.videoGameData }

    private let fallbackAccent = Color(red: 0.96, green: 0.62, blue: 0.04)

    private var accentColor: Color {
        data?.accentColor ?? fallbackAccent
    }

    private var heroURL: URL? {
        if let cover = data?.coverUrl, !cover.isEmpty { return URL(string: cover) }
        if let hero = post.heroImage?.url, !hero.isEmpty { return URL(string: hero) }
        if let img = post.imageURL, !img.isEmpty { return URL(string: img) }
        return nil
    }

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 0) {

                // 1. Hero section
                heroSection

                VStack(alignment: .leading, spacing: 20) {

                    // 2. Release countdown banner
                    if let data = data, let countdown = data.releaseCountdown {
                        countdownBanner(countdown, accent: accentColor)
                    }

                    // 3. Platform badges
                    if let data = data, !data.platforms.isEmpty {
                        platformSection(data.platforms)
                    }

                    // 4. Genre tags
                    if let data = data, !data.genres.isEmpty {
                        genreSection(data.genres, accent: accentColor)
                    }

                    // 5. Stats row
                    if let data = data {
                        statsRow(data: data)
                    }

                    // 6. Developer / publisher card
                    if let data = data, data.developer != nil || data.publisher != nil {
                        devPublisherCard(data: data)
                    }

                    // 7. Body text
                    if !post.body.isEmpty {
                        Text(post.body)
                            .font(.body)
                            .foregroundStyle(.primary)
                            .lineSpacing(4)
                    }

                    // 8. Steam store button
                    if let data = data, let storeUrl = data.storeUrl, !storeUrl.isEmpty,
                       let url = URL(string: storeUrl) {
                        storeButton(url: url)
                    }

                    Divider()

                    // 9. Engagement bar
                    PostDetailEngagementBar(post: post)
                }
                .padding(16)
            }
        }
        .ignoresSafeArea(edges: .top)
        .navigationTitle("")
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button { dismiss() } label: {
                    Image(systemName: "xmark.circle.fill")
                        .foregroundStyle(.secondary)
                }
            }
        }
    }

    // MARK: - Hero

    @ViewBuilder
    private var heroSection: some View {
        if let url = heroURL {
            GeometryReader { geo in
                AsyncImage(url: url) { phase in
                    switch phase {
                    case .success(let img):
                        ZStack(alignment: .bottomLeading) {
                            img.resizable()
                                .aspectRatio(contentMode: .fill)
                                .frame(width: geo.size.width, height: 260)
                                .clipped()

                            // Gradient overlay
                            LinearGradient(
                                colors: [.clear, Color(.systemBackground)],
                                startPoint: .center,
                                endPoint: .bottom
                            )
                            .frame(width: geo.size.width, height: 260)

                            // Title + developer overlay
                            if let data = data {
                                VStack(alignment: .leading, spacing: 4) {
                                    Text(data.title)
                                        .font(.system(size: 22, weight: .bold, design: .rounded))
                                        .foregroundStyle(.white)
                                        .shadow(color: .black.opacity(0.6), radius: 4, x: 0, y: 2)
                                        .lineLimit(2)

                                    if let dev = data.developer {
                                        Text(dev)
                                            .font(.caption.weight(.medium))
                                            .foregroundStyle(.white.opacity(0.75))
                                            .shadow(color: .black.opacity(0.5), radius: 2)
                                    }
                                }
                                .padding(.horizontal, 16)
                                .padding(.bottom, 20)
                            }
                        }
                    case .failure:
                        noHeroFallback(width: geo.size.width)
                    default:
                        Color.secondary.opacity(0.2)
                            .frame(width: geo.size.width, height: 260)
                            .overlay(ProgressView())
                    }
                }
            }
            .frame(height: 260)
        } else {
            GeometryReader { geo in
                noHeroFallback(width: geo.size.width)
            }
            .frame(height: 260)
        }
    }

    @ViewBuilder
    private func noHeroFallback(width: CGFloat) -> some View {
        ZStack(alignment: .bottomLeading) {
            LinearGradient(
                colors: [accentColor.opacity(0.8), accentColor.opacity(0.4)],
                startPoint: .topLeading,
                endPoint: .bottomTrailing
            )
            .frame(width: width, height: 260)

            Image(systemName: "gamecontroller.fill")
                .font(.system(size: 80, weight: .ultraLight))
                .foregroundStyle(.white.opacity(0.15))
                .frame(width: width, height: 260)

            if let data = data {
                VStack(alignment: .leading, spacing: 4) {
                    Text(data.title)
                        .font(.system(size: 22, weight: .bold, design: .rounded))
                        .foregroundStyle(.white)
                        .lineLimit(2)

                    if let dev = data.developer {
                        Text(dev)
                            .font(.caption.weight(.medium))
                            .foregroundStyle(.white.opacity(0.75))
                    }
                }
                .padding(.horizontal, 16)
                .padding(.bottom, 20)
            }
        }
        .clipped()
    }

    // MARK: - Countdown Banner

    @ViewBuilder
    private func countdownBanner(_ text: String, accent: Color) -> some View {
        HStack {
            Image(systemName: "clock.badge.fill")
                .font(.subheadline)
                .foregroundStyle(accent)

            Text(text)
                .font(.system(size: 14, weight: .heavy))
                .tracking(1)
                .foregroundStyle(accent)

            Spacer()
        }
        .frame(maxWidth: .infinity)
        .padding(.horizontal, 14)
        .padding(.vertical, 12)
        .background(
            RoundedRectangle(cornerRadius: 12)
                .fill(accent.opacity(0.12))
        )
        .overlay(
            RoundedRectangle(cornerRadius: 12)
                .strokeBorder(accent.opacity(0.25), lineWidth: 1)
        )
    }

    // MARK: - Platform Badges

    @ViewBuilder
    private func platformSection(_ platforms: [String]) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            Text("PLATFORMS")
                .font(.caption.weight(.bold))
                .foregroundStyle(.secondary)

            ScrollView(.horizontal, showsIndicators: false) {
                HStack(spacing: 8) {
                    ForEach(platforms, id: \.self) { platform in
                        Text(VideoGameData.platformAbbr(platform))
                            .font(.system(size: 12, weight: .bold))
                            .foregroundStyle(.white)
                            .padding(.horizontal, 12)
                            .padding(.vertical, 6)
                            .background(
                                Capsule()
                                    .fill(VideoGameData.platformColor(platform))
                            )
                    }
                }
                .padding(.horizontal, 1)
            }
        }
    }

    // MARK: - Genre Tags

    @ViewBuilder
    private func genreSection(_ genres: [String], accent: Color) -> some View {
        ScrollView(.horizontal, showsIndicators: false) {
            HStack(spacing: 8) {
                ForEach(genres, id: \.self) { genre in
                    Text(genre)
                        .font(.caption.weight(.medium))
                        .foregroundStyle(accent)
                        .padding(.horizontal, 10)
                        .padding(.vertical, 5)
                        .background(
                            Capsule()
                                .fill(accent.opacity(0.12))
                        )
                }
            }
            .padding(.horizontal, 1)
        }
    }

    // MARK: - Stats Row

    @ViewBuilder
    private func statsRow(data: VideoGameData) -> some View {
        let hasRating = data.rating != nil
        let hasPrice = data.displayPrice != nil
        let hasSteam = data.steamPositivePercent != nil

        if hasRating || hasPrice || hasSteam {
            HStack(spacing: 16) {

                // Metacritic score circle
                if let rating = data.rating {
                    VStack(spacing: 4) {
                        ZStack {
                            Circle()
                                .stroke(Color.secondary.opacity(0.2), lineWidth: 4)
                                .frame(width: 60, height: 60)
                            Circle()
                                .trim(from: 0, to: CGFloat(rating) / 100)
                                .stroke(
                                    data.ratingColor,
                                    style: StrokeStyle(lineWidth: 4, lineCap: .round)
                                )
                                .frame(width: 60, height: 60)
                                .rotationEffect(.degrees(-90))
                            Text("\(rating)")
                                .font(.system(size: 18, weight: .bold, design: .rounded))
                                .foregroundStyle(data.ratingColor)
                        }

                        Text("Metacritic")
                            .font(.caption2)
                            .foregroundStyle(.secondary)

                        if let count = data.formattedReviewCount {
                            Text("\(count) reviews")
                                .font(.caption2)
                                .foregroundStyle(.tertiary)
                        }
                    }
                }

                if hasRating && (hasPrice || hasSteam) {
                    Divider()
                        .frame(height: 56)
                }

                // Price info
                if let price = data.displayPrice {
                    VStack(spacing: 4) {
                        Image(systemName: "tag.fill")
                            .font(.title3)
                            .foregroundStyle(
                                (data.steamDiscount ?? 0) > 0
                                    ? Color(red: 0.2, green: 0.78, blue: 0.35)
                                    : accentColor
                            )
                        Text(price)
                            .font(.subheadline.weight(.bold))
                            .foregroundStyle(
                                (data.steamDiscount ?? 0) > 0
                                    ? Color(red: 0.2, green: 0.78, blue: 0.35)
                                    : .primary
                            )
                        Text("Steam")
                            .font(.caption2)
                            .foregroundStyle(.secondary)
                    }
                }

                if hasPrice && hasSteam {
                    Divider()
                        .frame(height: 56)
                }

                // Steam positive reviews
                if let pct = data.steamPositivePercent {
                    VStack(spacing: 4) {
                        Image(systemName: "hand.thumbsup.fill")
                            .font(.title3)
                            .foregroundStyle(Color(red: 0.11, green: 0.44, blue: 0.79))
                        Text(pct)
                            .font(.subheadline.weight(.bold))
                            .foregroundStyle(Color(red: 0.11, green: 0.44, blue: 0.79))
                        Text("Positive")
                            .font(.caption2)
                            .foregroundStyle(.secondary)
                        if let count = data.formattedReviewCount, data.rating == nil {
                            Text("\(count) reviews")
                                .font(.caption2)
                                .foregroundStyle(.tertiary)
                        }
                    }
                }

                Spacer()
            }
            .padding(14)
            .background(
                RoundedRectangle(cornerRadius: 14)
                    .fill(Color(.secondarySystemGroupedBackground))
            )
        }
    }

    // MARK: - Dev/Publisher Card

    @ViewBuilder
    private func devPublisherCard(data: VideoGameData) -> some View {
        VStack(alignment: .leading, spacing: 10) {
            if let dev = data.developer {
                HStack(spacing: 8) {
                    Image(systemName: "hammer.fill")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                        .frame(width: 20)
                    VStack(alignment: .leading, spacing: 1) {
                        Text("DEVELOPER")
                            .font(.caption2.weight(.bold))
                            .foregroundStyle(.secondary)
                        Text(dev)
                            .font(.subheadline)
                            .foregroundStyle(.primary)
                    }
                }
            }

            if let pub = data.publisher, pub != data.developer {
                HStack(spacing: 8) {
                    Image(systemName: "building.2.fill")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                        .frame(width: 20)
                    VStack(alignment: .leading, spacing: 1) {
                        Text("PUBLISHER")
                            .font(.caption2.weight(.bold))
                            .foregroundStyle(.secondary)
                        Text(pub)
                            .font(.subheadline)
                            .foregroundStyle(.primary)
                    }
                }
            }

            if let releaseDate = data.formattedReleaseDate {
                HStack(spacing: 8) {
                    Image(systemName: "calendar")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                        .frame(width: 20)
                    VStack(alignment: .leading, spacing: 1) {
                        Text("RELEASE DATE")
                            .font(.caption2.weight(.bold))
                            .foregroundStyle(.secondary)
                        Text(releaseDate)
                            .font(.subheadline)
                            .foregroundStyle(.primary)
                    }
                }
            }
        }
        .padding(14)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(
            RoundedRectangle(cornerRadius: 14)
                .fill(Color(.secondarySystemGroupedBackground))
        )
    }

    // MARK: - Store Button

    @ViewBuilder
    private func storeButton(url: URL) -> some View {
        Link(destination: url) {
            HStack(spacing: 8) {
                Image(systemName: "bag.fill")
                    .font(.subheadline)
                Text("View on Steam")
                    .font(.subheadline.weight(.semibold))
            }
            .frame(maxWidth: .infinity)
            .padding(.vertical, 13)
            .background(
                RoundedRectangle(cornerRadius: 12)
                    .fill(Color(red: 0.11, green: 0.44, blue: 0.79))
            )
            .foregroundStyle(.white)
        }
    }
}
