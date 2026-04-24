import SwiftUI

// MARK: - Album Card

struct AlbumCard: View {
    let post: Post
    let music: MusicData
    @State private var activeReaction: String?

    init?(post: Post) {
        guard let md = post.musicData, md.isAlbum else { return nil }
        self.post = post
        self.music = md
        self._activeReaction = State(initialValue: post.myReaction)
    }

    private let accentColor = Color(red: 0.459, green: 0.176, blue: 0.902)
    private let spotifyGreen = Color(red: 0.114, green: 0.729, blue: 0.333)
    private let cardBg = Color(red: 0.039, green: 0.039, blue: 0.039)

    var body: some View {
        ZStack {
            // Background
            cardBg

            // Vinyl ring watermark
            Circle()
                .stroke(Color.white.opacity(0.04), lineWidth: 28)
                .frame(width: 280, height: 280)
                .offset(x: 90, y: 60)

            Circle()
                .stroke(Color.white.opacity(0.03), lineWidth: 14)
                .frame(width: 200, height: 200)
                .offset(x: 90, y: 60)

            VStack(spacing: 0) {
                // Main content row
                HStack(alignment: .top, spacing: 14) {
                    // Album art
                    ZStack(alignment: .topLeading) {
                        if let coverUrl = music.coverUrl, let url = URL(string: coverUrl) {
                            AsyncImage(url: url) { phase in
                                switch phase {
                                case .success(let image):
                                    image
                                        .resizable()
                                        .scaledToFill()
                                case .failure, .empty:
                                    albumArtPlaceholder
                                @unknown default:
                                    albumArtPlaceholder
                                }
                            }
                        } else {
                            albumArtPlaceholder
                        }
                    }
                    .frame(width: 130, height: 130)
                    .clipShape(RoundedRectangle(cornerRadius: 8))
                    .overlay(
                        RoundedRectangle(cornerRadius: 8)
                            .stroke(Color.white.opacity(0.1), lineWidth: 1)
                    )

                    // Metadata
                    VStack(alignment: .leading, spacing: 6) {
                        // Album type badge
                        Text(music.albumTypeDisplay)
                            .font(.system(size: 9, weight: .bold))
                            .foregroundStyle(.white)
                            .padding(.horizontal, 6)
                            .padding(.vertical, 3)
                            .background(accentColor, in: Capsule())

                        // Artist name
                        Text(music.artist.uppercased())
                            .font(.system(size: 11, weight: .semibold))
                            .tracking(1.0)
                            .foregroundStyle(.white.opacity(0.55))
                            .lineLimit(1)

                        // Album title
                        Text(music.title ?? post.title)
                            .font(.system(size: 18, design: .serif))
                            .italic()
                            .foregroundStyle(.white)
                            .lineLimit(2)

                        if let label = music.label {
                            Text(label)
                                .font(.system(size: 10))
                                .foregroundStyle(.white.opacity(0.35))
                                .lineLimit(1)
                        }

                        // Track/duration info
                        if let tracks = music.trackCount {
                            Text("\(tracks) tracks")
                                .font(.system(size: 11, design: .monospaced))
                                .foregroundStyle(.white.opacity(0.45))
                        }

                        Spacer(minLength: 0)

                        // Last.fm stats
                        if let listeners = music.formattedListeners,
                           let plays = music.formattedPlaycount {
                            HStack(spacing: 4) {
                                Image(systemName: "person.2")
                                    .font(.system(size: 9))
                                Text("\(listeners) listeners · \(plays) plays")
                                    .font(.system(size: 10))
                            }
                            .foregroundStyle(.white.opacity(0.45))
                        } else if let listeners = music.formattedListeners {
                            HStack(spacing: 4) {
                                Image(systemName: "person.2")
                                    .font(.system(size: 9))
                                Text("\(listeners) listeners")
                                    .font(.system(size: 10))
                            }
                            .foregroundStyle(.white.opacity(0.45))
                        }
                    }
                    .frame(maxWidth: .infinity, alignment: .leading)
                }
                .padding(.horizontal, 16)
                .padding(.top, 16)

                // Genre tags
                if let tags = music.tags, !tags.isEmpty {
                    ScrollView(.horizontal, showsIndicators: false) {
                        HStack(spacing: 6) {
                            ForEach(Array(tags.prefix(4)), id: \.self) { tag in
                                Text(tag)
                                    .font(.system(size: 10, weight: .medium))
                                    .foregroundStyle(.white.opacity(0.7))
                                    .padding(.horizontal, 8)
                                    .padding(.vertical, 4)
                                    .background(Color.white.opacity(0.08), in: Capsule())
                            }
                        }
                        .padding(.horizontal, 16)
                    }
                    .padding(.top, 10)
                }

                Spacer(minLength: 0)

                // Footer row
                HStack(spacing: 10) {
                    // Spotify button
                    if let spotifyUrl = music.spotifyUrl, let url = URL(string: spotifyUrl) {
                        Link(destination: url) {
                            HStack(spacing: 5) {
                                Image(systemName: "music.note")
                                    .font(.system(size: 11, weight: .semibold))
                                Text("Listen on Spotify")
                                    .font(.system(size: 11, weight: .semibold))
                            }
                            .foregroundStyle(.black)
                            .padding(.horizontal, 12)
                            .padding(.vertical, 7)
                            .background(spotifyGreen, in: Capsule())
                        }
                    }

                    Spacer()

                    ReactionPicker(
                        activeReaction: $activeReaction,
                        postID: post.id,
                        style: .feedDark
                    )
                    MusicBookmarkButton(post: post)
                }
                .padding(.horizontal, 16)
                .padding(.bottom, 14)
                .padding(.top, 10)
            }
        }
        .frame(height: 200)
        .clipShape(RoundedRectangle(cornerRadius: 16))
    }

    private var albumArtPlaceholder: some View {
        ZStack {
            LinearGradient(
                colors: [accentColor.opacity(0.6), accentColor.opacity(0.3)],
                startPoint: .topLeading,
                endPoint: .bottomTrailing
            )
            Image(systemName: "music.note")
                .font(.system(size: 36, weight: .ultraLight))
                .foregroundStyle(.white.opacity(0.5))
        }
    }
}

// MARK: - Concert Card

struct ConcertCard: View {
    let post: Post
    let music: MusicData
    @State private var activeReaction: String?

    init?(post: Post) {
        guard let md = post.musicData, md.isConcert else { return nil }
        self.post = post
        self.music = md
        self._activeReaction = State(initialValue: post.myReaction)
    }

    private let accentColor = Color(red: 0.984, green: 0.729, blue: 0.012)
    private let cardBg = Color(red: 0.039, green: 0.039, blue: 0.039)

    var body: some View {
        ZStack {
            // Background
            LinearGradient(
                colors: [Color(red: 0.06, green: 0.04, blue: 0.10), cardBg],
                startPoint: .top,
                endPoint: .bottom
            )

            // Large mic watermark
            Image(systemName: "music.mic")
                .font(.system(size: 140, weight: .ultraLight))
                .foregroundStyle(.white.opacity(0.03))
                .offset(x: 80, y: -10)

            VStack(spacing: 0) {
                // Header: date badge + artist
                HStack(alignment: .top, spacing: 14) {
                    // Calendar date badge
                    VStack(spacing: 0) {
                        Text(music.monthAbbrev ?? "")
                            .font(.system(size: 10, weight: .bold))
                            .foregroundStyle(.black)
                            .frame(maxWidth: .infinity)
                            .padding(.vertical, 4)
                            .background(accentColor)

                        Text(music.dayNumber ?? "")
                            .font(.system(size: 26, weight: .bold))
                            .foregroundStyle(.white)
                            .frame(maxWidth: .infinity)
                            .padding(.vertical, 6)
                            .background(Color.white.opacity(0.08))
                    }
                    .frame(width: 52)
                    .clipShape(RoundedRectangle(cornerRadius: 8))
                    .overlay(
                        RoundedRectangle(cornerRadius: 8)
                            .stroke(accentColor.opacity(0.4), lineWidth: 1)
                    )

                    VStack(alignment: .leading, spacing: 5) {
                        // On sale badge
                        if music.onSale == true {
                            Text("ON SALE NOW")
                                .font(.system(size: 9, weight: .bold))
                                .foregroundStyle(.white)
                                .padding(.horizontal, 7)
                                .padding(.vertical, 3)
                                .background(Color.green, in: Capsule())
                        }

                        // Artist name
                        Text(music.artist)
                            .font(.system(size: 20, weight: .bold))
                            .foregroundStyle(.white)
                            .lineLimit(2)

                        // Venue
                        if let venue = music.venue {
                            HStack(spacing: 4) {
                                Image(systemName: "mappin")
                                    .font(.system(size: 10))
                                    .foregroundStyle(accentColor)
                                Text(venue)
                                    .font(.system(size: 12))
                                    .foregroundStyle(.white.opacity(0.7))
                                    .lineLimit(1)
                            }
                        }
                    }

                    Spacer(minLength: 0)
                }
                .padding(.horizontal, 16)
                .padding(.top, 16)

                // Time + price row
                HStack(spacing: 16) {
                    if let doors = music.doorsTime {
                        Label("Doors \(doors)", systemImage: "door.left.hand.open")
                            .font(.system(size: 11))
                            .foregroundStyle(.white.opacity(0.5))
                    }
                    if let start = music.startTime {
                        Label("Show \(start)", systemImage: "clock")
                            .font(.system(size: 11))
                            .foregroundStyle(.white.opacity(0.5))
                    }
                    Spacer()
                    if let price = music.priceRange {
                        Text(price)
                            .font(.system(size: 12, weight: .semibold))
                            .foregroundStyle(music.onSale == true ? .green : .white.opacity(0.6))
                    }
                }
                .padding(.horizontal, 16)
                .padding(.top, 12)

                Spacer(minLength: 0)

                // Ticket button + reactions
                HStack(spacing: 10) {
                    if let ticketUrlStr = music.ticketUrl, let ticketUrl = URL(string: ticketUrlStr) {
                        Link(destination: ticketUrl) {
                            Text("Get Tickets")
                                .font(.system(size: 13, weight: .semibold))
                                .foregroundStyle(.black)
                                .frame(maxWidth: .infinity)
                                .padding(.vertical, 10)
                                .background(accentColor, in: RoundedRectangle(cornerRadius: 10))
                        }
                    } else {
                        Text("Get Tickets")
                            .font(.system(size: 13, weight: .semibold))
                            .foregroundStyle(.black.opacity(0.5))
                            .frame(maxWidth: .infinity)
                            .padding(.vertical, 10)
                            .background(accentColor.opacity(0.4), in: RoundedRectangle(cornerRadius: 10))
                    }

                    ReactionPicker(
                        activeReaction: $activeReaction,
                        postID: post.id,
                        style: .feedDark
                    )
                    MusicBookmarkButton(post: post)
                }
                .padding(.horizontal, 16)
                .padding(.bottom, 14)
                .padding(.top, 8)
            }
        }
        .frame(height: 240)
        .clipShape(RoundedRectangle(cornerRadius: 16))
    }
}

// MARK: - Shared Bookmark Button

private struct MusicBookmarkButton: View {
    let post: Post
    @State var isBookmarked: Bool

    @EnvironmentObject private var apiService: APIService

    init(post: Post) {
        self.post = post
        self._isBookmarked = State(initialValue: post.saved ?? false)
    }

    var body: some View {
        Button {
            let wasSaved = isBookmarked
            UIImpactFeedbackGenerator(style: .light).impactOccurred()
            isBookmarked.toggle()
            Task {
                do { try await apiService.trackEvent(postID: post.id, eventType: wasSaved ? "unsave" : "save") }
                catch { isBookmarked = wasSaved }
            }
        } label: {
            Image(systemName: isBookmarked ? "bookmark.fill" : "bookmark")
                .font(.caption)
                .foregroundColor(isBookmarked ? .orange : .white.opacity(0.4))
                .contentTransition(.symbolEffect(.replace))
        }
        .buttonStyle(.plain)
    }
}
