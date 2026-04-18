import SwiftUI

struct ShowDetailView: View {
    let post: Post
    @Environment(\.dismiss) private var dismiss
    @State private var selectedSeason: Int = 1

    private var data: MediaData? { post.mediaData }

    private static let showAccent = Color(red: 0.957, green: 0.62, blue: 0.043)

    private var heroURL: URL? {
        if let bd = data?.backdropUrl, !bd.isEmpty { return URL(string: bd) }
        if let p = data?.posterUrl, !p.isEmpty { return URL(string: p) }
        if let i = post.imageURL, !i.isEmpty { return URL(string: i) }
        return nil
    }

    private var isOnAir: Bool {
        data?.onTheAir == true
    }

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 0) {

                // MARK: Full-bleed backdrop hero
                ZStack(alignment: .bottom) {
                    if let url = heroURL {
                        GeometryReader { geo in
                            AsyncImage(url: url) { phase in
                                switch phase {
                                case .success(let img):
                                    img.resizable()
                                        .aspectRatio(contentMode: .fill)
                                        .frame(width: geo.size.width, height: 300)
                                        .clipped()
                                case .failure:
                                    Color.secondary.opacity(0.2)
                                        .frame(height: 300)
                                default:
                                    Color.secondary.opacity(0.2)
                                        .frame(height: 300)
                                        .overlay(ProgressView())
                                }
                            }
                        }
                        .frame(height: 300)
                    } else {
                        Rectangle()
                            .fill(
                                LinearGradient(
                                    colors: [Self.showAccent.opacity(0.4), Color.black.opacity(0.6)],
                                    startPoint: .top,
                                    endPoint: .bottom
                                )
                            )
                            .frame(height: 300)
                    }

                    // Gradient fade at bottom
                    LinearGradient(
                        colors: [.clear, Color(.systemBackground)],
                        startPoint: .top,
                        endPoint: .bottom
                    )
                    .frame(height: 160)
                }

                // MARK: Content
                VStack(alignment: .leading, spacing: 16) {

                    // Status badge + poster row
                    HStack(alignment: .top, spacing: 14) {

                        // Poster thumbnail
                        if let posterStr = data?.posterUrl, !posterStr.isEmpty,
                           let posterURL = URL(string: posterStr) {
                            AsyncImage(url: posterURL) { phase in
                                switch phase {
                                case .success(let img):
                                    img.resizable()
                                        .aspectRatio(contentMode: .fill)
                                        .frame(width: 80, height: 120)
                                        .cornerRadius(8)
                                        .shadow(radius: 6)
                                default:
                                    RoundedRectangle(cornerRadius: 8)
                                        .fill(Color.secondary.opacity(0.2))
                                        .frame(width: 80, height: 120)
                                }
                            }
                        }

                        VStack(alignment: .leading, spacing: 8) {
                            // ON AIR / ENDED badge
                            onAirBadge

                            // Title
                            Text(data?.title ?? post.title)
                                .font(.title2.weight(.bold))
                                .lineLimit(3)

                            // Creator & network
                            VStack(alignment: .leading, spacing: 3) {
                                if let creator = data?.creator, !creator.isEmpty {
                                    Text("Created by \(creator)")
                                        .font(.caption)
                                        .foregroundStyle(.secondary)
                                }
                                if let network = data?.network, !network.isEmpty {
                                    Label(network, systemImage: "tv")
                                        .font(.caption)
                                        .foregroundStyle(.secondary)
                                }
                                if let year = data?.year {
                                    Text(String(year))
                                        .font(.caption)
                                        .foregroundStyle(.secondary)
                                }
                            }

                            // RT scores
                            rtScoresRow
                        }

                        Spacer(minLength: 0)
                    }

                    // Season selector
                    if let seasons = data?.seasons, seasons > 0 {
                        seasonSelector(seasons: seasons)
                    }

                    // Genre pills
                    if !data.flatMap({ $0.genres.isEmpty ? nil : $0.genres }) .isNilOrEmpty {
                        genrePills
                    }

                    // Overview / body
                    if !post.body.isEmpty {
                        Text(post.body)
                            .font(.body)
                            .lineSpacing(4)
                    }

                    // Cast
                    if let cast = data?.cast, !cast.isEmpty {
                        castSection(cast: cast)
                    }

                    // Streaming
                    if let streaming = data?.streaming, !streaming.isEmpty {
                        streamingSection(platforms: streaming)
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

    // MARK: - ON AIR / ENDED Badge

    @ViewBuilder
    private var onAirBadge: some View {
        if isOnAir {
            HStack(spacing: 6) {
                PulsingDot()
                Text("ON AIR")
                    .font(.system(size: 11, weight: .heavy))
                    .tracking(0.8)
            }
            .foregroundStyle(.green)
            .padding(.horizontal, 10)
            .padding(.vertical, 5)
            .background(.green.opacity(0.12), in: Capsule())
        } else {
            Text("ENDED")
                .font(.system(size: 11, weight: .bold))
                .tracking(0.8)
                .foregroundStyle(.secondary)
                .padding(.horizontal, 10)
                .padding(.vertical, 5)
                .background(Color.secondary.opacity(0.1), in: Capsule())
        }
    }

    // MARK: - RT Scores

    @ViewBuilder
    private var rtScoresRow: some View {
        let hasRT = data?.rtScore != nil
        let hasAud = data?.rtAudienceScore != nil
        if hasRT || hasAud {
            HStack(spacing: 14) {
                if let rt = data?.rtScore {
                    HStack(spacing: 4) {
                        Text("🍅")
                        Text("\(rt)%")
                            .font(.subheadline.weight(.bold))
                            .foregroundStyle(rt >= 60 ? .green : .red)
                    }
                }
                if let aud = data?.rtAudienceScore {
                    HStack(spacing: 4) {
                        Text("🍿")
                        Text("\(aud)%")
                            .font(.subheadline.weight(.bold))
                            .foregroundStyle(.secondary)
                    }
                }
            }
        }
    }

    // MARK: - Season Selector

    @ViewBuilder
    private func seasonSelector(seasons: Int) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            Text("SEASONS")
                .font(.caption.weight(.bold))
                .foregroundStyle(.secondary)

            ScrollView(.horizontal, showsIndicators: false) {
                HStack(spacing: 8) {
                    ForEach(1...seasons, id: \.self) { season in
                        Button {
                            withAnimation(.spring(response: 0.3, dampingFraction: 0.7)) {
                                selectedSeason = season
                            }
                        } label: {
                            Text("S\(season)")
                                .font(.caption.weight(.semibold))
                                .padding(.horizontal, 14)
                                .padding(.vertical, 7)
                                .background(
                                    selectedSeason == season
                                        ? Self.showAccent
                                        : Color.secondary.opacity(0.12),
                                    in: Capsule()
                                )
                                .foregroundStyle(selectedSeason == season ? .black : .secondary)
                        }
                        .buttonStyle(.plain)
                    }
                }
            }
        }
    }

    // MARK: - Genre Pills

    @ViewBuilder
    private var genrePills: some View {
        if let genres = data?.genres, !genres.isEmpty {
            ScrollView(.horizontal, showsIndicators: false) {
                HStack(spacing: 8) {
                    ForEach(genres, id: \.self) { genre in
                        Text(genre)
                            .font(.caption.weight(.medium))
                            .padding(.horizontal, 10)
                            .padding(.vertical, 5)
                            .background(
                                Self.showAccent.opacity(0.12),
                                in: Capsule()
                            )
                            .foregroundStyle(Self.showAccent)
                    }
                }
            }
        }
    }

    // MARK: - Cast Section

    @ViewBuilder
    private func castSection(cast: [String]) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            Text("CAST")
                .font(.caption.weight(.bold))
                .foregroundStyle(.secondary)

            ScrollView(.horizontal, showsIndicators: false) {
                HStack(spacing: 12) {
                    ForEach(cast, id: \.self) { name in
                        VStack(spacing: 6) {
                            Circle()
                                .fill(Color.secondary.opacity(0.15))
                                .frame(width: 56, height: 56)
                                .overlay(
                                    Text(name.prefix(1))
                                        .font(.title3.weight(.semibold))
                                        .foregroundStyle(.secondary)
                                )
                            Text(name.components(separatedBy: " ").first ?? name)
                                .font(.caption2)
                                .foregroundStyle(.secondary)
                                .lineLimit(1)
                                .frame(width: 60)
                        }
                    }
                }
            }
        }
    }

    // MARK: - Streaming Section

    @ViewBuilder
    private func streamingSection(platforms: [String]) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            Text("STREAM ON")
                .font(.caption.weight(.bold))
                .foregroundStyle(.secondary)

            ScrollView(.horizontal, showsIndicators: false) {
                HStack(spacing: 8) {
                    ForEach(platforms, id: \.self) { platform in
                        Text(platform)
                            .font(.caption.weight(.semibold))
                            .padding(.horizontal, 12)
                            .padding(.vertical, 6)
                            .background(Color.secondary.opacity(0.15), in: Capsule())
                            .foregroundStyle(.primary)
                    }
                }
            }
        }
    }
}

// MARK: - Pulsing Dot

private struct PulsingDot: View {
    @State private var scale: CGFloat = 1.0

    var body: some View {
        Circle()
            .fill(.green)
            .frame(width: 8, height: 8)
            .scaleEffect(scale)
            .onAppear {
                withAnimation(
                    .easeInOut(duration: 1.0)
                    .repeatForever(autoreverses: true)
                ) {
                    scale = 1.4
                }
            }
    }
}

// MARK: - Optional Array Helper

private extension Optional where Wrapped: Collection {
    var isNilOrEmpty: Bool {
        self?.isEmpty ?? true
    }
}
