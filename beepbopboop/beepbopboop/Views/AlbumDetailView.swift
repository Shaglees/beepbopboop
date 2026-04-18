import SwiftUI

struct AlbumDetailView: View {
    let post: Post
    @Environment(\.dismiss) private var dismiss

    private var data: MusicData? { post.musicData }

    private static let musicPurple = Color(red: 0.459, green: 0.176, blue: 0.902)

    private var coverURL: URL? {
        if let str = data?.coverUrl, !str.isEmpty { return URL(string: str) }
        if let str = post.imageURL, !str.isEmpty { return URL(string: str) }
        return nil
    }

    private var releaseYear: String? {
        guard let date = data?.releaseDate, date.count >= 4 else { return nil }
        return String(date.prefix(4))
    }

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 0) {

                // MARK: Hero
                heroSection

                // MARK: Content
                VStack(alignment: .leading, spacing: 16) {

                    // Album identity
                    albumIdentitySection

                    // Stats row
                    statsRow

                    // Genre tags
                    if let tags = data?.tags, !tags.isEmpty {
                        tagsRow(tags: tags)
                    }

                    // Body text (review / description)
                    if !post.body.isEmpty {
                        Text(post.body)
                            .font(.body)
                            .lineSpacing(4)
                    }

                    // Spotify button
                    if let spotifyStr = data?.spotifyUrl,
                       !spotifyStr.isEmpty,
                       let spotifyURL = URL(string: spotifyStr) {
                        spotifyButton(url: spotifyURL)
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

    private var heroSection: some View {
        ZStack {
            // Blurred background version of cover art
            if let url = coverURL {
                AsyncImage(url: url) { phase in
                    if case .success(let img) = phase {
                        img.resizable()
                            .aspectRatio(contentMode: .fill)
                            .blur(radius: 40)
                            .saturation(0.8)
                    }
                }
            } else {
                Self.musicPurple.opacity(0.6)
            }

            // Gradient overlay
            LinearGradient(
                colors: [
                    Self.musicPurple.opacity(0.7),
                    Color.black.opacity(0.85)
                ],
                startPoint: .top,
                endPoint: .bottom
            )

            // Centered album art square
            if let url = coverURL {
                AsyncImage(url: url) { phase in
                    switch phase {
                    case .success(let img):
                        img.resizable()
                            .aspectRatio(1, contentMode: .fit)
                            .frame(width: 200, height: 200)
                            .cornerRadius(16)
                            .shadow(color: .black.opacity(0.5), radius: 20, y: 10)
                    case .failure:
                        albumArtPlaceholder
                    default:
                        albumArtPlaceholder
                            .overlay(ProgressView().tint(.white))
                    }
                }
            } else {
                albumArtPlaceholder
            }
        }
        .frame(height: 300)
        .clipped()
    }

    private var albumArtPlaceholder: some View {
        RoundedRectangle(cornerRadius: 16)
            .fill(Color.white.opacity(0.1))
            .frame(width: 200, height: 200)
            .overlay(
                Image(systemName: "music.note")
                    .font(.system(size: 48))
                    .foregroundStyle(.white.opacity(0.6))
            )
    }

    // MARK: - Album Identity

    private var albumIdentitySection: some View {
        VStack(alignment: .leading, spacing: 6) {
            // Title
            Text(data?.title ?? post.title)
                .font(.title2.weight(.bold))
                .lineLimit(3)

            // Artist
            Text(data?.artist ?? "")
                .font(.title3)
                .foregroundStyle(.primary)

            // Type badge + year
            HStack(spacing: 8) {
                Text(data?.albumTypeDisplay ?? "ALBUM")
                    .font(.caption.weight(.bold))
                    .padding(.horizontal, 10)
                    .padding(.vertical, 4)
                    .background(Self.musicPurple.opacity(0.15), in: Capsule())
                    .foregroundStyle(Self.musicPurple)

                if let year = releaseYear {
                    Text(year)
                        .font(.subheadline)
                        .foregroundStyle(.secondary)
                }

                if let relDate = data?.formattedReleaseDate {
                    Text("·")
                        .foregroundStyle(.secondary)
                    Text(relDate)
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
            }

            // Record label
            if let lbl = data?.label, !lbl.isEmpty {
                Text(lbl)
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }
        }
    }

    // MARK: - Stats Row

    @ViewBuilder
    private var statsRow: some View {
        let hasListeners = data?.formattedListeners != nil
        let hasPlays = data?.formattedPlaycount != nil
        let hasTracks = data?.trackCount != nil

        if hasListeners || hasPlays || hasTracks {
            ScrollView(.horizontal, showsIndicators: false) {
                HStack(spacing: 10) {
                    if let listeners = data?.formattedListeners {
                        statPill(icon: "headphones", value: listeners, label: "listeners")
                    }
                    if let plays = data?.formattedPlaycount {
                        statPill(icon: "play.fill", value: plays, label: "plays")
                    }
                    if let tracks = data?.trackCount {
                        statPill(icon: "music.note.list", value: "\(tracks)", label: "tracks")
                    }
                }
            }
        }
    }

    private func statPill(icon: String, value: String, label: String) -> some View {
        HStack(spacing: 6) {
            Image(systemName: icon)
                .font(.caption2)
            Text(value)
                .font(.caption.weight(.semibold))
            Text(label)
                .font(.caption)
                .foregroundStyle(.secondary)
        }
        .padding(.horizontal, 12)
        .padding(.vertical, 7)
        .background(Color.secondary.opacity(0.1), in: Capsule())
    }

    // MARK: - Tags Row

    private func tagsRow(tags: [String]) -> some View {
        ScrollView(.horizontal, showsIndicators: false) {
            HStack(spacing: 8) {
                ForEach(tags, id: \.self) { tag in
                    Text(tag)
                        .font(.caption.weight(.medium))
                        .padding(.horizontal, 10)
                        .padding(.vertical, 5)
                        .background(Self.musicPurple.opacity(0.12), in: Capsule())
                        .foregroundStyle(Self.musicPurple)
                }
            }
        }
    }

    // MARK: - Spotify Button

    private func spotifyButton(url: URL) -> some View {
        Link(destination: url) {
            HStack(spacing: 10) {
                Image(systemName: "music.note")
                    .font(.body.weight(.semibold))
                Text("Listen on Spotify")
                    .font(.body.weight(.semibold))
            }
            .frame(maxWidth: .infinity)
            .padding(.vertical, 14)
            .background(Color(red: 0.118, green: 0.725, blue: 0.333))
            .foregroundStyle(.white)
            .cornerRadius(14)
        }
    }
}
