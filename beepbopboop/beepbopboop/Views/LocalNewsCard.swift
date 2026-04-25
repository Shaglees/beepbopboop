import SwiftUI

// MARK: - LocalNewsData

struct LocalNewsData: Codable {
    let contentKind: String       // "article", "video", "hybrid"
    let sourceName: String
    let sourceURL: String
    let sourceLogoURL: String?
    let thumbnailURL: String?
    let articleURL: String?
    let embedURL: String?
    let durationSeconds: Int?
    let locality: String?
    let publishedAt: String?      // ISO 8601 string — decoded manually
    let trustScore: Int?

    enum CodingKeys: String, CodingKey {
        case contentKind = "content_kind"
        case sourceName = "source_name"
        case sourceURL = "source_url"
        case sourceLogoURL = "source_logo_url"
        case thumbnailURL = "thumbnail_url"
        case articleURL = "article_url"
        case embedURL = "embed_url"
        case durationSeconds = "duration_seconds"
        case locality
        case publishedAt = "published_at"
        case trustScore = "trust_score"
    }

    /// Parses the ISO 8601 `publishedAt` string into a `Date`.
    var publishedDate: Date? {
        guard let s = publishedAt else { return nil }
        let f1 = ISO8601DateFormatter()
        f1.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        if let d = f1.date(from: s) { return d }
        let f2 = ISO8601DateFormatter()
        f2.formatOptions = [.withInternetDateTime]
        return f2.date(from: s)
    }

    /// Human-readable relative timestamp for the publish date.
    var relativePublishedAt: String? {
        guard let date = publishedDate else { return nil }
        let formatter = RelativeDateTimeFormatter()
        formatter.unitsStyle = .abbreviated
        return formatter.localizedString(for: date, relativeTo: Date())
    }

    /// Duration string formatted as M:SS.
    var durationLabel: String? {
        guard let secs = durationSeconds, secs > 0 else { return nil }
        let m = secs / 60
        let s = secs % 60
        return String(format: "%d:%02d", m, s)
    }
}

// MARK: - LocalNewsCard

struct LocalNewsCard: View {
    let post: Post

    private var newsData: LocalNewsData? {
        guard let json = post.externalURL,
              let data = json.data(using: .utf8) else { return nil }
        return try? JSONDecoder().decode(LocalNewsData.self, from: data)
    }

    var body: some View {
        if let news = newsData {
            switch news.contentKind {
            case "video":
                LocalNewsVideoLayout(post: post, news: news)
            case "hybrid":
                LocalNewsHybridLayout(post: post, news: news)
            default:
                LocalNewsArticleLayout(post: post, news: news)
            }
        } else {
            LocalNewsFallback(post: post)
        }
    }
}

// MARK: - Fallback (passes through to StandardCard pattern)

private struct LocalNewsFallback: View {
    let post: Post

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            CardHeader(post: post)

            Text(post.title)
                .font(.system(size: 15, weight: .semibold))
                .tracking(-0.2)
                .foregroundColor(BBBDesign.ink)
                .lineLimit(2)

            Text(post.body)
                .font(.system(size: 13))
                .lineSpacing(2)
                .foregroundColor(BBBDesign.ink2)
                .lineLimit(3)

            CardFooter(post: post)
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 14)
    }
}

// MARK: - Source Badge

private struct SourceBadge: View {
    let news: LocalNewsData

    private let localNewsBlue = Color(red: 0.165, green: 0.478, blue: 0.839)

    var body: some View {
        HStack(spacing: 6) {
            if let logoURLString = news.sourceLogoURL, let url = URL(string: logoURLString) {
                AsyncImage(url: url) { phase in
                    switch phase {
                    case .success(let image):
                        image
                            .resizable()
                            .scaledToFill()
                            .frame(width: 16, height: 16)
                            .clipShape(RoundedRectangle(cornerRadius: 3))
                    default:
                        RoundedRectangle(cornerRadius: 3)
                            .fill(localNewsBlue.opacity(0.2))
                            .frame(width: 16, height: 16)
                    }
                }
            }

            Text(news.sourceName)
                .font(.system(size: 11, weight: .semibold))
                .foregroundColor(localNewsBlue)
                .lineLimit(1)

            if let locality = news.locality, !locality.isEmpty {
                Text(locality)
                    .font(.system(size: 10, weight: .medium))
                    .foregroundColor(BBBDesign.ink3)
                    .padding(.horizontal, 6)
                    .padding(.vertical, 2)
                    .background(BBBDesign.sunken)
                    .clipShape(Capsule())
                    .lineLimit(1)
            }

            if let relTime = news.relativePublishedAt {
                Text(relTime)
                    .font(.system(size: 10, design: .monospaced))
                    .foregroundColor(BBBDesign.ink3)
            }
        }
    }
}

// MARK: - Trust Indicator

private struct TrustDot: View {
    let trustScore: Int?

    var body: some View {
        if let score = trustScore, score > 70 {
            Circle()
                .fill(Color(red: 0.133, green: 0.773, blue: 0.369))
                .frame(width: 6, height: 6)
                .overlay(
                    Circle()
                        .stroke(Color(red: 0.133, green: 0.773, blue: 0.369).opacity(0.3), lineWidth: 2)
                        .frame(width: 10, height: 10)
                )
                .accessibilityLabel("Trusted source")
        }
    }
}

// MARK: - Article Layout

private struct LocalNewsArticleLayout: View {
    let post: Post
    let news: LocalNewsData
    @Environment(\.openURL) private var openURL

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            VStack(alignment: .leading, spacing: 10) {
                CardHeader(post: post)

                // Source + trust row
                HStack(spacing: 6) {
                    SourceBadge(news: news)
                    Spacer(minLength: 0)
                    TrustDot(trustScore: news.trustScore)
                }

                // Headline + thumbnail row
                HStack(alignment: .top, spacing: 10) {
                    VStack(alignment: .leading, spacing: 6) {
                        Text(post.title)
                            .font(.system(size: 15, weight: .semibold))
                            .tracking(-0.2)
                            .foregroundColor(BBBDesign.ink)
                            .lineLimit(3)

                        if !post.body.isEmpty {
                            Text(post.body)
                                .font(.system(size: 13))
                                .lineSpacing(2)
                                .foregroundColor(BBBDesign.ink2)
                                .lineLimit(3)
                        }
                    }

                    if let thumbURLString = news.thumbnailURL, let url = URL(string: thumbURLString) {
                        AsyncImage(url: url) { phase in
                            switch phase {
                            case .success(let image):
                                image
                                    .resizable()
                                    .scaledToFill()
                                    .frame(width: 80, height: 80)
                                    .clipShape(RoundedRectangle(cornerRadius: 10))
                            case .failure:
                                EmptyView()
                            default:
                                RoundedRectangle(cornerRadius: 10)
                                    .fill(BBBDesign.sunken)
                                    .frame(width: 80, height: 80)
                                    .overlay(ProgressView().scaleEffect(0.6))
                            }
                        }
                        .frame(width: 80, height: 80)
                    }
                }

                // Read article CTA
                if let articleURLString = news.articleURL, let url = URL(string: articleURLString) {
                    Button {
                        openURL(url)
                    } label: {
                        Label("Read Article", systemImage: "arrow.up.right")
                            .font(.caption.weight(.semibold))
                            .foregroundColor(post.hintColor)
                            .padding(.horizontal, 10)
                            .padding(.vertical, 5)
                            .background(post.hintColor.opacity(0.1))
                            .clipShape(Capsule())
                    }
                    .buttonStyle(.plain)
                }

                CardFooter(post: post)
            }
            .padding(.horizontal, 16)
            .padding(.vertical, 14)
        }
    }
}

// MARK: - Video Layout

private struct LocalNewsVideoLayout: View {
    let post: Post
    let news: LocalNewsData
    @Environment(\.openURL) private var openURL

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            // Full-width 16:9 thumbnail with play overlay
            ZStack(alignment: .bottomLeading) {
                if let thumbURLString = news.thumbnailURL, let url = URL(string: thumbURLString) {
                    AsyncImage(url: url) { phase in
                        switch phase {
                        case .success(let image):
                            image
                                .resizable()
                                .scaledToFill()
                                .frame(maxWidth: .infinity)
                                .aspectRatio(16.0 / 9.0, contentMode: .fill)
                                .clipped()
                        case .failure:
                            Rectangle()
                                .fill(BBBDesign.sunken)
                                .aspectRatio(16.0 / 9.0, contentMode: .fit)
                                .frame(maxWidth: .infinity)
                        default:
                            Rectangle()
                                .fill(BBBDesign.sunken)
                                .aspectRatio(16.0 / 9.0, contentMode: .fit)
                                .frame(maxWidth: .infinity)
                                .overlay(ProgressView())
                        }
                    }
                } else {
                    Rectangle()
                        .fill(BBBDesign.sunken)
                        .aspectRatio(16.0 / 9.0, contentMode: .fit)
                        .frame(maxWidth: .infinity)
                }

                // Gradient overlay for readability
                LinearGradient(
                    colors: [.black.opacity(0.55), .clear],
                    startPoint: .bottom,
                    endPoint: .center
                )

                // Play button overlay
                Circle()
                    .fill(.white.opacity(0.9))
                    .frame(width: 48, height: 48)
                    .overlay(
                        Image(systemName: "play.fill")
                            .font(.system(size: 18))
                            .foregroundColor(.black)
                            .offset(x: 2)
                    )
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
                    .background(.clear)

                // Duration badge (bottom-right)
                if let duration = news.durationLabel {
                    HStack {
                        Spacer()
                        Text(duration)
                            .font(.system(size: 11, weight: .bold, design: .monospaced))
                            .foregroundColor(.white)
                            .padding(.horizontal, 7)
                            .padding(.vertical, 3)
                            .background(.black.opacity(0.7))
                            .clipShape(RoundedRectangle(cornerRadius: 4))
                            .padding(8)
                    }
                    .frame(maxHeight: .infinity, alignment: .bottom)
                }
            }
            .frame(maxWidth: .infinity)
            .clipped()
            .onTapGesture {
                if let embedURLString = news.embedURL, let url = URL(string: embedURLString) {
                    openURL(url)
                } else if let articleURLString = news.articleURL, let url = URL(string: articleURLString) {
                    openURL(url)
                }
            }

            // Info below the video
            VStack(alignment: .leading, spacing: 8) {
                // Source badge row
                HStack(spacing: 6) {
                    SourceBadge(news: news)
                    Spacer(minLength: 0)
                    TrustDot(trustScore: news.trustScore)
                }

                Text(post.title)
                    .font(.system(size: 15, weight: .semibold))
                    .tracking(-0.2)
                    .foregroundColor(BBBDesign.ink)
                    .lineLimit(2)

                CardFooter(post: post)
            }
            .padding(.horizontal, 16)
            .padding(.top, 12)
            .padding(.bottom, 14)
        }
    }
}

// MARK: - Hybrid Layout (article + secondary video row)

private struct LocalNewsHybridLayout: View {
    let post: Post
    let news: LocalNewsData
    @Environment(\.openURL) private var openURL

    private let localNewsBlue = Color(red: 0.165, green: 0.478, blue: 0.839)

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            // Primary: article section
            VStack(alignment: .leading, spacing: 10) {
                CardHeader(post: post)

                HStack(spacing: 6) {
                    SourceBadge(news: news)
                    Spacer(minLength: 0)
                    TrustDot(trustScore: news.trustScore)
                }

                HStack(alignment: .top, spacing: 10) {
                    VStack(alignment: .leading, spacing: 6) {
                        Text(post.title)
                            .font(.system(size: 15, weight: .semibold))
                            .tracking(-0.2)
                            .foregroundColor(BBBDesign.ink)
                            .lineLimit(3)

                        if !post.body.isEmpty {
                            Text(post.body)
                                .font(.system(size: 13))
                                .lineSpacing(2)
                                .foregroundColor(BBBDesign.ink2)
                                .lineLimit(2)
                        }
                    }

                    if let thumbURLString = news.thumbnailURL, let url = URL(string: thumbURLString) {
                        AsyncImage(url: url) { phase in
                            switch phase {
                            case .success(let image):
                                image
                                    .resizable()
                                    .scaledToFill()
                                    .frame(width: 80, height: 80)
                                    .clipShape(RoundedRectangle(cornerRadius: 10))
                            case .failure:
                                EmptyView()
                            default:
                                RoundedRectangle(cornerRadius: 10)
                                    .fill(BBBDesign.sunken)
                                    .frame(width: 80, height: 80)
                                    .overlay(ProgressView().scaleEffect(0.6))
                            }
                        }
                        .frame(width: 80, height: 80)
                    }
                }
            }
            .padding(.horizontal, 16)
            .padding(.top, 14)
            .padding(.bottom, 10)

            // Secondary: video thumbnail strip
            if let embedURLString = news.embedURL, let embedURL = URL(string: embedURLString) {
                Divider()
                    .padding(.horizontal, 16)

                Button {
                    openURL(embedURL)
                } label: {
                    HStack(spacing: 10) {
                        // Small video thumbnail
                        ZStack {
                            if let thumbURLString = news.thumbnailURL, let url = URL(string: thumbURLString) {
                                AsyncImage(url: url) { phase in
                                    switch phase {
                                    case .success(let image):
                                        image
                                            .resizable()
                                            .scaledToFill()
                                    default:
                                        Rectangle().fill(BBBDesign.sunken)
                                    }
                                }
                            } else {
                                Rectangle().fill(BBBDesign.sunken)
                            }

                            Circle()
                                .fill(.white.opacity(0.85))
                                .frame(width: 24, height: 24)
                                .overlay(
                                    Image(systemName: "play.fill")
                                        .font(.system(size: 9))
                                        .foregroundColor(.black)
                                        .offset(x: 1)
                                )
                        }
                        .frame(width: 72, height: 48)
                        .clipShape(RoundedRectangle(cornerRadius: 8))

                        VStack(alignment: .leading, spacing: 3) {
                            Text("Watch Video")
                                .font(.system(size: 12, weight: .semibold))
                                .foregroundColor(localNewsBlue)
                            if let duration = news.durationLabel {
                                Text(duration)
                                    .font(.system(size: 10, design: .monospaced))
                                    .foregroundColor(BBBDesign.ink3)
                            }
                        }

                        Spacer()

                        Image(systemName: "arrow.up.right.circle")
                            .font(.system(size: 16))
                            .foregroundColor(localNewsBlue.opacity(0.6))
                    }
                    .padding(.horizontal, 16)
                    .padding(.vertical, 10)
                }
                .buttonStyle(.plain)

                Divider()
                    .padding(.horizontal, 16)
            }

            // Footer
            CardFooter(post: post)
                .padding(.horizontal, 16)
                .padding(.bottom, 14)
                .padding(.top, 4)
        }
    }
}
