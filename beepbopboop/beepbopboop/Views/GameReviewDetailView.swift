import SwiftUI

struct GameReviewDetailView: View {
    let post: Post
    @Environment(\.dismiss) private var dismiss

    private var data: VideoGameData? { post.videoGameData }

    private let reviewPurple = Color(red: 0.58, green: 0.27, blue: 0.96)

    private var heroURL: URL? {
        if let u = data?.coverUrl, !u.isEmpty, let url = URL(string: u) { return url }
        if let u = post.heroImage?.url, !u.isEmpty, let url = URL(string: u) { return url }
        if let u = post.imageURL, !u.isEmpty, let url = URL(string: u) { return url }
        return nil
    }

    // First sentence of post.body used as a verdict pull-quote
    private var verdictQuote: String? {
        let text = post.body.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !text.isEmpty else { return nil }
        // Split on sentence-ending punctuation followed by whitespace or end
        let terminators: [Character] = [".", "!", "?"]
        if let idx = text.firstIndex(where: { terminators.contains($0) }) {
            let sentence = String(text[text.startIndex...idx]).trimmingCharacters(in: .whitespaces)
            if sentence.count > 15 { return sentence }
        }
        // Fall back to first line
        let firstLine = text.components(separatedBy: "\n").first?.trimmingCharacters(in: .whitespaces) ?? ""
        return firstLine.isEmpty ? nil : firstLine
    }

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 0) {

                // MARK: - Hero (240pt)
                heroSection

                // MARK: - Content
                VStack(alignment: .leading, spacing: 20) {

                    // Agent + time
                    HStack(spacing: 6) {
                        Circle()
                            .fill(reviewPurple)
                            .frame(width: 10, height: 10)
                        Text(post.agentName)
                            .font(.subheadline.weight(.medium))
                        Text("·")
                            .foregroundColor(.secondary)
                        Text(post.relativeTime)
                            .font(.subheadline)
                            .foregroundColor(.secondary)
                    }

                    // MARK: - Score Showcase
                    if let d = data {
                        scoreShowcase(data: d)
                    }

                    // MARK: - Verdict pull-quote
                    if let verdict = verdictQuote {
                        verdictCard(verdict)
                    }

                    // MARK: - Platform badges
                    if let d = data, !d.platforms.isEmpty {
                        platformSection(d.platforms)
                    }

                    // MARK: - Genre tags
                    if let d = data, !d.genres.isEmpty {
                        genreSection(d.genres)
                    }

                    // MARK: - Full review body
                    if !post.body.isEmpty {
                        Text(post.body)
                            .font(.body)
                            .lineSpacing(5)
                            .foregroundStyle(.primary)
                    }

                    // MARK: - Developer / publisher / release date
                    if let d = data {
                        devPublisherRow(d)
                    }

                    // MARK: - Store link
                    if let d = data, let storeURL = d.storeUrl, let url = URL(string: storeURL) {
                        storeButton(url: url)
                    }

                    Divider()

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

    // MARK: - Hero Section

    @ViewBuilder
    private var heroSection: some View {
        if let url = heroURL {
            GeometryReader { geo in
                AsyncImage(url: url) { phase in
                    switch phase {
                    case .success(let img):
                        img.resizable()
                            .aspectRatio(contentMode: .fill)
                            .frame(width: geo.size.width, height: 240)
                            .clipped()
                            .overlay {
                                // Darker film-noir overlay
                                LinearGradient(
                                    stops: [
                                        .init(color: Color.black.opacity(0.15), location: 0),
                                        .init(color: Color.black.opacity(0.55), location: 0.55),
                                        .init(color: Color(.systemBackground).opacity(0.95), location: 1.0),
                                    ],
                                    startPoint: .top,
                                    endPoint: .bottom
                                )
                            }
                            .overlay(alignment: .bottomLeading) {
                                heroTitleOverlay
                                    .padding(16)
                            }
                    case .failure:
                        gamecontrollerFallback(width: geo.size.width)
                    default:
                        Color(.systemGroupedBackground)
                            .frame(width: geo.size.width, height: 240)
                            .overlay(ProgressView())
                    }
                }
            }
            .frame(height: 240)
        } else {
            GeometryReader { geo in
                gamecontrollerFallback(width: geo.size.width)
                    .overlay(alignment: .bottomLeading) {
                        heroTitleOverlay.padding(16)
                    }
            }
            .frame(height: 240)
        }
    }

    @ViewBuilder
    private var heroTitleOverlay: some View {
        VStack(alignment: .leading, spacing: 4) {
            Text("REVIEW")
                .font(.system(size: 10, weight: .heavy))
                .tracking(2.5)
                .foregroundStyle(reviewPurple)
            Text(data?.title ?? post.title)
                .font(.title2.weight(.bold))
                .foregroundStyle(.white)
                .lineLimit(2)
                .shadow(color: .black.opacity(0.6), radius: 4)
        }
    }

    @ViewBuilder
    private func gamecontrollerFallback(width: CGFloat) -> some View {
        LinearGradient(
            colors: [
                reviewPurple.opacity(0.85),
                Color(red: 0.18, green: 0.08, blue: 0.35),
            ],
            startPoint: .topLeading,
            endPoint: .bottomTrailing
        )
        .frame(width: width, height: 240)
        .overlay {
            Image(systemName: "gamecontroller.fill")
                .font(.system(size: 72, weight: .ultraLight))
                .foregroundStyle(.white.opacity(0.12))
        }
    }

    // MARK: - Score Showcase

    @ViewBuilder
    private func scoreShowcase(data: VideoGameData) -> some View {
        let hasMeta = data.rating != nil
        let hasSteam = data.steamPositiveRatio != nil

        HStack(alignment: .center, spacing: 0) {
            // Metacritic score circle
            VStack(spacing: 8) {
                ZStack {
                    Circle()
                        .stroke(Color(.systemFill), lineWidth: 8)
                        .frame(width: 100, height: 100)
                    if let r = data.rating {
                        Circle()
                            .trim(from: 0, to: CGFloat(r) / 100)
                            .stroke(data.ratingColor, style: StrokeStyle(lineWidth: 8, lineCap: .round))
                            .frame(width: 100, height: 100)
                            .rotationEffect(.degrees(-90))
                        Text("\(r)")
                            .font(.system(size: 36, weight: .heavy, design: .rounded))
                            .foregroundStyle(data.ratingColor)
                    } else {
                        Image(systemName: "gamecontroller")
                            .font(.system(size: 32))
                            .foregroundStyle(.secondary)
                    }
                }
                Text("METACRITIC")
                    .font(.system(size: 10, weight: .heavy))
                    .tracking(2)
                    .foregroundStyle(.secondary)
                if let count = data.formattedReviewCount, hasMeta {
                    Text("\(count) reviews")
                        .font(.caption2)
                        .foregroundStyle(.tertiary)
                }
            }
            .frame(maxWidth: .infinity)

            // Divider between scores when both present
            if hasMeta && hasSteam {
                Divider()
                    .frame(height: 80)
            }

            // Steam score column
            if let ratio = data.steamPositiveRatio, let pct = data.steamPositivePercent {
                VStack(spacing: 8) {
                    ZStack {
                        Circle()
                            .stroke(Color(.systemFill), lineWidth: 8)
                            .frame(width: 100, height: 100)
                        Circle()
                            .trim(from: 0, to: ratio)
                            .stroke(steamColor(ratio: ratio), style: StrokeStyle(lineWidth: 8, lineCap: .round))
                            .frame(width: 100, height: 100)
                            .rotationEffect(.degrees(-90))
                        Text(pct)
                            .font(.system(size: 30, weight: .heavy, design: .rounded))
                            .foregroundStyle(steamColor(ratio: ratio))
                    }
                    Text("STEAM")
                        .font(.system(size: 10, weight: .heavy))
                        .tracking(2)
                        .foregroundStyle(.secondary)
                    if let count = data.formattedReviewCount {
                        Text("\(count) reviews")
                            .font(.caption2)
                            .foregroundStyle(.tertiary)
                    }
                }
                .frame(maxWidth: .infinity)
            }
        }
        .padding(.vertical, 24)
        .background(Color(.secondarySystemGroupedBackground), in: RoundedRectangle(cornerRadius: 16))

        // Steam community note below the card if steam data exists
        if let pct = data.steamPositivePercent {
            HStack(spacing: 6) {
                Image(systemName: "person.2.fill")
                    .font(.caption2)
                    .foregroundStyle(.secondary)
                Text("\(pct) of Steam reviews positive")
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }
            .padding(.top, -12)
        }
    }

    private func steamColor(ratio: Double) -> Color {
        if ratio >= 0.75 { return Color(red: 0.11, green: 0.44, blue: 0.79) }
        if ratio >= 0.60 { return Color(red: 0.96, green: 0.77, blue: 0.19) }
        return Color(red: 0.93, green: 0.26, blue: 0.21)
    }

    // MARK: - Verdict Card

    @ViewBuilder
    private func verdictCard(_ verdict: String) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack(spacing: 6) {
                Image(systemName: "quote.bubble.fill")
                    .font(.caption)
                    .foregroundStyle(reviewPurple)
                Text("VERDICT")
                    .font(.system(size: 10, weight: .heavy))
                    .tracking(2)
                    .foregroundStyle(reviewPurple)
            }
            Text("\u{201C}\(verdict)\u{201D}")
                .font(.body.italic())
                .foregroundStyle(.primary)
                .lineSpacing(4)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(16)
        .background(reviewPurple.opacity(0.07), in: RoundedRectangle(cornerRadius: 14))
        .overlay(
            RoundedRectangle(cornerRadius: 14)
                .stroke(reviewPurple.opacity(0.18), lineWidth: 1)
        )
    }

    // MARK: - Platform Badges

    @ViewBuilder
    private func platformSection(_ platforms: [String]) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            Text("PLATFORMS")
                .font(.system(size: 10, weight: .heavy))
                .tracking(2)
                .foregroundStyle(.secondary)
            ScrollView(.horizontal, showsIndicators: false) {
                HStack(spacing: 8) {
                    ForEach(platforms, id: \.self) { platform in
                        Text(VideoGameData.platformAbbr(platform))
                            .font(.system(size: 12, weight: .bold))
                            .foregroundStyle(.white)
                            .padding(.horizontal, 12)
                            .padding(.vertical, 6)
                            .background(VideoGameData.platformColor(platform), in: Capsule())
                    }
                }
            }
        }
    }

    // MARK: - Genre Tags

    @ViewBuilder
    private func genreSection(_ genres: [String]) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            Text("GENRES")
                .font(.system(size: 10, weight: .heavy))
                .tracking(2)
                .foregroundStyle(.secondary)
            ScrollView(.horizontal, showsIndicators: false) {
                HStack(spacing: 8) {
                    ForEach(genres, id: \.self) { genre in
                        Text(genre)
                            .font(.caption.weight(.medium))
                            .foregroundStyle(reviewPurple)
                            .padding(.horizontal, 12)
                            .padding(.vertical, 6)
                            .background(reviewPurple.opacity(0.12), in: Capsule())
                    }
                }
            }
        }
    }

    // MARK: - Developer / Publisher / Release Date Row

    @ViewBuilder
    private func devPublisherRow(_ data: VideoGameData) -> some View {
        let hasDev = !(data.developer ?? "").isEmpty
        let hasPub = !(data.publisher ?? "").isEmpty && data.publisher != data.developer
        let hasDate = data.formattedReleaseDate != nil

        if hasDev || hasPub || hasDate {
            HStack(spacing: 0) {
                if hasDev || hasPub {
                    VStack(alignment: .leading, spacing: 3) {
                        Text("DEVELOPER")
                            .font(.system(size: 9, weight: .heavy))
                            .tracking(1.5)
                            .foregroundStyle(.secondary)
                        if let dev = data.developer {
                            Text(dev)
                                .font(.subheadline.weight(.semibold))
                        }
                        if hasPub, let pub = data.publisher {
                            Text(pub)
                                .font(.caption)
                                .foregroundStyle(.secondary)
                        }
                    }
                    .frame(maxWidth: .infinity, alignment: .leading)
                }

                if hasDate {
                    VStack(alignment: .trailing, spacing: 3) {
                        Text("RELEASED")
                            .font(.system(size: 9, weight: .heavy))
                            .tracking(1.5)
                            .foregroundStyle(.secondary)
                        Text(data.formattedReleaseDate ?? "")
                            .font(.subheadline.weight(.semibold))
                    }
                    .frame(maxWidth: .infinity, alignment: .trailing)
                }
            }
            .padding(14)
            .background(Color(.secondarySystemGroupedBackground), in: RoundedRectangle(cornerRadius: 12))
        }
    }

    // MARK: - Store Button

    @ViewBuilder
    private func storeButton(url: URL) -> some View {
        Link(destination: url) {
            HStack(spacing: 8) {
                Image(systemName: "cart.fill")
                Text("View in Store")
                    .fontWeight(.semibold)
                Spacer()
                Image(systemName: "arrow.up.right")
                    .font(.caption)
            }
            .font(.subheadline)
            .foregroundStyle(.white)
            .padding(.horizontal, 18)
            .padding(.vertical, 14)
            .background(reviewPurple, in: RoundedRectangle(cornerRadius: 12))
        }
    }
}
