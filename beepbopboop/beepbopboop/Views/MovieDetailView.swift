import SwiftUI

struct MovieDetailView: View {
    let post: Post
    @Environment(\.dismiss) private var dismiss

    private var data: MediaData? { post.mediaData }

    private var heroURL: URL? {
        if let bd = data?.backdropUrl, !bd.isEmpty { return URL(string: bd) }
        if let p = data?.posterUrl, !p.isEmpty { return URL(string: p) }
        if let i = post.imageURL, !i.isEmpty { return URL(string: i) }
        return nil
    }

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 0) {
                // Backdrop hero
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
                                        LinearGradient(
                                            colors: [.clear, Color(.systemBackground)],
                                            startPoint: .center,
                                            endPoint: .bottom
                                        )
                                    }
                            case .failure:
                                EmptyView()
                            default:
                                Color.secondary.opacity(0.2)
                                    .frame(height: 240)
                                    .overlay(ProgressView())
                            }
                        }
                    }
                    .frame(height: 240)
                }

                VStack(alignment: .leading, spacing: 16) {
                    // Poster + metadata row
                    HStack(alignment: .top, spacing: 14) {
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

                        VStack(alignment: .leading, spacing: 6) {
                            Text(data?.title ?? post.title)
                                .font(.title2.weight(.bold))
                                .lineLimit(3)

                            HStack(spacing: 8) {
                                if let year = data?.year {
                                    Text(String(year))
                                        .font(.subheadline)
                                        .foregroundStyle(.secondary)
                                }
                                if let runtime = data?.runtime, runtime > 0 {
                                    Text("·")
                                    let h = runtime / 60
                                    let m = runtime % 60
                                    Text(h > 0 ? "\(h)h \(m)m" : "\(m)m")
                                        .font(.subheadline)
                                        .foregroundStyle(.secondary)
                                }
                            }

                            if let director = data?.director, !director.isEmpty {
                                Text("Dir. \(director)")
                                    .font(.caption)
                                    .foregroundStyle(.secondary)
                            }

                            // RT scores
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
                        Spacer()
                    }

                    // Status banner
                    if let status = data?.status, status != "available" {
                        HStack {
                            Image(systemName: status == "upcoming" ? "clock.badge.fill" : "theatermasks.fill")
                            Text(status == "upcoming" ? "Coming Soon" : "In Theatres")
                                .fontWeight(.semibold)
                        }
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 10)
                        .background(
                            Color(red: 0.957, green: 0.62, blue: 0.043).opacity(0.15),
                            in: RoundedRectangle(cornerRadius: 10)
                        )
                        .foregroundStyle(Color(red: 0.957, green: 0.62, blue: 0.043))
                        .font(.subheadline)
                    }

                    // Tagline
                    if let tagline = data?.tagline, !tagline.isEmpty {
                        Text("\u{201C}\(tagline)\u{201D}")
                            .font(.body.italic())
                            .foregroundStyle(.secondary)
                    }

                    // Body / overview
                    if !post.body.isEmpty {
                        Text(post.body)
                            .font(.body)
                            .lineSpacing(4)
                    }

                    // Genres
                    if let genres = data?.genres, !genres.isEmpty {
                        ScrollView(.horizontal, showsIndicators: false) {
                            HStack(spacing: 8) {
                                ForEach(genres, id: \.self) { genre in
                                    Text(genre)
                                        .font(.caption.weight(.medium))
                                        .padding(.horizontal, 10)
                                        .padding(.vertical, 5)
                                        .background(
                                            Color(red: 0.957, green: 0.62, blue: 0.043).opacity(0.12),
                                            in: Capsule()
                                        )
                                        .foregroundStyle(Color(red: 0.957, green: 0.62, blue: 0.043))
                                }
                            }
                        }
                    }

                    // Cast
                    if let cast = data?.cast, !cast.isEmpty {
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

                    // Streaming
                    if let streaming = data?.streaming, !streaming.isEmpty {
                        VStack(alignment: .leading, spacing: 8) {
                            Text("WATCH ON")
                                .font(.caption.weight(.bold))
                                .foregroundStyle(.secondary)
                            ScrollView(.horizontal, showsIndicators: false) {
                                HStack(spacing: 8) {
                                    ForEach(streaming, id: \.self) { platform in
                                        Text(platform)
                                            .font(.caption.weight(.semibold))
                                            .padding(.horizontal, 12)
                                            .padding(.vertical, 6)
                                            .background(Color.secondary.opacity(0.15), in: Capsule())
                                    }
                                }
                            }
                        }
                    }

                    // Rent / Buy
                    if let rentBuy = data?.rentBuy, !rentBuy.isEmpty {
                        VStack(alignment: .leading, spacing: 8) {
                            Text("RENT / BUY")
                                .font(.caption.weight(.bold))
                                .foregroundStyle(.secondary)
                            ScrollView(.horizontal, showsIndicators: false) {
                                HStack(spacing: 8) {
                                    ForEach(rentBuy, id: \.self) { platform in
                                        Text(platform)
                                            .font(.caption.weight(.semibold))
                                            .padding(.horizontal, 12)
                                            .padding(.vertical, 6)
                                            .background(Color.secondary.opacity(0.1), in: Capsule())
                                    }
                                }
                            }
                        }
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
}
