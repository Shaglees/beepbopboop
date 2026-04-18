import SwiftUI

// MARK: - MovieCard

struct MovieCard: View {
    let post: Post
    let media: MediaData
    @State private var activeReaction: String?

    private let darkBg = Color(red: 0.059, green: 0.059, blue: 0.059)

    init?(post: Post) {
        guard let md = post.mediaData else { return nil }
        self.post = post
        self.media = md
        self._activeReaction = State(initialValue: post.myReaction)
    }

    var body: some View {
        ZStack {
            darkBg
            mediaBackdrop(url: media.backdropUrl)
            HStack(alignment: .top, spacing: 0) {
                mediaPoster(
                    url: media.posterUrl,
                    placeholder: "film",
                    badge: media.inTheatres ? AnyView(theatresBadge) : nil
                )
                movieMetaColumn
            }
        }
        .frame(height: 220)
    }

    private var movieMetaColumn: some View {
        VStack(alignment: .leading, spacing: 5) {
            Text(media.title)
                .font(.system(size: 18, weight: .semibold))
                .foregroundStyle(.white)
                .lineLimit(2)

            if let tagline = media.tagline, !tagline.isEmpty {
                Text(tagline)
                    .font(.system(size: 12)).italic()
                    .foregroundStyle(.white.opacity(0.5))
                    .lineLimit(1)
            }

            ratingsRow(tmdbRating: media.tmdbRating, rtScore: media.rtScore, rtAudienceScore: media.rtAudienceScore)

            Text(movieMetaText)
                .font(.system(size: 11))
                .foregroundStyle(.white.opacity(0.45))

            if let director = media.director, !director.isEmpty {
                Text("Dir. \(director)")
                    .font(.system(size: 11))
                    .foregroundStyle(.white.opacity(0.4))
            }

            Spacer(minLength: 4)

            if !media.cast.isEmpty { castStrip(cast: media.cast) }
            if !media.streaming.isEmpty { streamingRow(platforms: media.streaming) }

            HStack {
                Spacer()
                ReactionPicker(activeReaction: $activeReaction, postID: post.id, style: .feedDark)
                MediaBookmarkButton(post: post)
            }
        }
        .padding(.top, 12)
        .padding(.bottom, 10)
        .padding(.trailing, 12)
    }

    private var movieMetaText: String {
        var parts: [String] = []
        if let r = media.runtime { parts.append("\(r / 60)h \(r % 60)m") }
        if let d = media.releaseDate, d.count >= 4 { parts.append(String(d.prefix(4))) }
        if let g = media.genres.first { parts.append(g) }
        return parts.joined(separator: " · ")
    }

    private var theatresBadge: some View {
        Text("IN THEATRES")
            .font(.system(size: 7, weight: .heavy))
            .tracking(0.5)
            .foregroundStyle(.black)
            .padding(.horizontal, 5)
            .padding(.vertical, 3)
            .background(Color(red: 0.957, green: 0.62, blue: 0.043))
            .clipShape(Capsule())
    }
}

// MARK: - ShowCard

struct ShowCard: View {
    let post: Post
    let media: MediaData
    @State private var activeReaction: String?

    private let darkBg = Color(red: 0.059, green: 0.059, blue: 0.059)

    init?(post: Post) {
        guard let md = post.mediaData else { return nil }
        self.post = post
        self.media = md
        self._activeReaction = State(initialValue: post.myReaction)
    }

    var body: some View {
        ZStack {
            darkBg
            mediaBackdrop(url: media.backdropUrl)
            HStack(alignment: .top, spacing: 0) {
                mediaPoster(
                    url: media.posterUrl,
                    placeholder: "tv",
                    badge: (media.onTheAir == true) ? AnyView(airingBadge) : nil
                )
                showMetaColumn
            }
        }
        .frame(height: 220)
    }

    private var showMetaColumn: some View {
        VStack(alignment: .leading, spacing: 5) {
            Text(media.title)
                .font(.system(size: 18, weight: .semibold))
                .foregroundStyle(.white)
                .lineLimit(2)

            if let network = media.network, !network.isEmpty {
                HStack(spacing: 4) {
                    Text("ON")
                        .font(.system(size: 9, weight: .heavy))
                        .tracking(0.5)
                        .foregroundStyle(.white.opacity(0.45))
                    Text(network.uppercased())
                        .font(.system(size: 10, weight: .bold))
                        .foregroundStyle(Color(red: 0.957, green: 0.62, blue: 0.043))
                }
            }

            ratingsRow(tmdbRating: media.tmdbRating, rtScore: media.rtScore, rtAudienceScore: nil)

            Text(showMetaText)
                .font(.system(size: 11))
                .foregroundStyle(.white.opacity(0.45))

            if let creator = media.creator, !creator.isEmpty {
                Text("Created by \(creator)")
                    .font(.system(size: 11))
                    .foregroundStyle(.white.opacity(0.4))
            } else if let director = media.director, !director.isEmpty {
                Text("Dir. \(director)")
                    .font(.system(size: 11))
                    .foregroundStyle(.white.opacity(0.4))
            }

            Spacer(minLength: 4)

            if !media.cast.isEmpty { castStrip(cast: media.cast) }
            if !media.streaming.isEmpty { streamingRow(platforms: media.streaming) }

            HStack {
                Spacer()
                ReactionPicker(activeReaction: $activeReaction, postID: post.id, style: .feedDark)
                MediaBookmarkButton(post: post)
            }
        }
        .padding(.top, 12)
        .padding(.bottom, 10)
        .padding(.trailing, 12)
    }

    private var showMetaText: String {
        var parts: [String] = []
        if let s = media.seasons { parts.append(s == 1 ? "1 Season" : "\(s) Seasons") }
        if let d = media.releaseDate, d.count >= 4 { parts.append(String(d.prefix(4))) }
        if let g = media.genres.first { parts.append(g) }
        return parts.joined(separator: " · ")
    }

    private var airingBadge: some View {
        Text("AIRING")
            .font(.system(size: 7, weight: .heavy))
            .tracking(0.5)
            .foregroundStyle(.white)
            .padding(.horizontal, 5)
            .padding(.vertical, 3)
            .background(Color.green)
            .clipShape(Capsule())
    }
}

// MARK: - Shared layout helpers

@ViewBuilder
private func mediaBackdrop(url: String?) -> some View {
    if let urlStr = url, let parsedUrl = URL(string: urlStr) {
        AsyncImage(url: parsedUrl) { img in
            img.resizable().aspectRatio(contentMode: .fill)
        } placeholder: { EmptyView() }
        .clipped()
        .overlay(Color.black.opacity(0.65))
    }
}

private func mediaPoster(url: String?, placeholder: String, badge: AnyView?) -> some View {
    ZStack(alignment: .topTrailing) {
        Group {
            if let urlStr = url, let parsedUrl = URL(string: urlStr) {
                AsyncImage(url: parsedUrl) { img in
                    img.resizable().aspectRatio(contentMode: .fill)
                } placeholder: {
                    RoundedRectangle(cornerRadius: 8)
                        .fill(Color.white.opacity(0.08))
                        .overlay(Image(systemName: placeholder).font(.title2).foregroundStyle(.white.opacity(0.2)))
                }
                .clipShape(RoundedRectangle(cornerRadius: 8))
            } else {
                RoundedRectangle(cornerRadius: 8)
                    .fill(Color.white.opacity(0.08))
                    .overlay(Image(systemName: placeholder).font(.title2).foregroundStyle(.white.opacity(0.2)))
            }
        }
        .frame(width: 88, height: 196)

        if let badge = badge {
            badge.offset(x: 4, y: 4)
        }
    }
    .padding(12)
}

@ViewBuilder
private func ratingsRow(tmdbRating: Double?, rtScore: Int?, rtAudienceScore: Int?) -> some View {
    if tmdbRating != nil || rtScore != nil {
        HStack(spacing: 8) {
            if let rating = tmdbRating {
                HStack(spacing: 4) {
                    Text(String(format: "%.1f", rating))
                        .font(.system(size: 13, weight: .bold, design: .rounded))
                    Image(systemName: "star.fill")
                        .font(.system(size: 10))
                }
                .foregroundStyle(.white)
                .padding(.horizontal, 8)
                .padding(.vertical, 4)
                .background(Color(red: 0.051, green: 0.58, blue: 0.533))
                .clipShape(Capsule())
            }
            if let rt = rtScore {
                HStack(spacing: 3) {
                    Text("🍅").font(.system(size: 10))
                    Text("\(rt)%")
                        .font(.system(size: 12, weight: .semibold))
                        .foregroundStyle(Color(red: 0.86, green: 0.2, blue: 0.2))
                }
                if let audience = rtAudienceScore {
                    HStack(spacing: 3) {
                        Text("🍿").font(.system(size: 10))
                        Text("\(audience)%")
                            .font(.system(size: 11, weight: .medium))
                            .foregroundStyle(.white.opacity(0.6))
                    }
                }
            }
        }
    }
}

private func castStrip(cast: [String]) -> some View {
    HStack(spacing: 4) {
        ForEach(Array(cast.prefix(3)), id: \.self) { name in
            Text(name.components(separatedBy: " ").last ?? name)
                .font(.system(size: 10, weight: .medium))
                .foregroundStyle(.white.opacity(0.7))
                .padding(.horizontal, 6)
                .padding(.vertical, 3)
                .background(Color.white.opacity(0.1))
                .clipShape(Capsule())
                .lineLimit(1)
        }
    }
}

private func streamingRow(platforms: [String]) -> some View {
    HStack(spacing: 4) {
        ForEach(Array(platforms.prefix(3)), id: \.self) { platform in
            Text(platform)
                .font(.system(size: 9, weight: .semibold))
                .foregroundStyle(.white)
                .padding(.horizontal, 6)
                .padding(.vertical, 3)
                .background(streamingColor(for: platform))
                .clipShape(Capsule())
                .lineLimit(1)
        }
    }
}

private func streamingColor(for platform: String) -> Color {
    let lower = platform.lowercased()
    if lower.contains("netflix") { return Color(red: 0.9, green: 0.1, blue: 0.1) }
    if lower.contains("disney") { return Color(red: 0.11, green: 0.27, blue: 0.74) }
    if lower.contains("prime") || lower.contains("amazon") { return Color(red: 0.0, green: 0.61, blue: 0.71) }
    if lower.contains("apple") { return Color(red: 0.15, green: 0.15, blue: 0.15) }
    if lower.contains("max") || lower.contains("hbo") { return Color(red: 0.29, green: 0.18, blue: 0.55) }
    if lower.contains("paramount") { return Color(red: 0.06, green: 0.28, blue: 0.64) }
    if lower.contains("hulu") { return Color(red: 0.07, green: 0.78, blue: 0.47) }
    if lower.contains("peacock") { return Color(red: 0.0, green: 0.32, blue: 0.96) }
    return Color.white.opacity(0.15)
}

// MARK: - Bookmark button

private struct MediaBookmarkButton: View {
    let post: Post
    @AppStorage var isBookmarked: Bool

    init(post: Post) {
        self.post = post
        self._isBookmarked = AppStorage(wrappedValue: false, "bookmark_\(post.id)")
    }

    var body: some View {
        Button {
            UIImpactFeedbackGenerator(style: .light).impactOccurred()
            isBookmarked.toggle()
        } label: {
            Image(systemName: isBookmarked ? "bookmark.fill" : "bookmark")
                .font(.caption)
                .foregroundColor(isBookmarked ? .orange : .white.opacity(0.4))
                .contentTransition(.symbolEffect(.replace))
        }
        .buttonStyle(.plain)
    }
}
