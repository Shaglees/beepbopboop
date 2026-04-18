import SwiftUI

struct EntertainmentDetailView: View {
    let post: Post
    @Environment(\.dismiss) private var dismiss

    private var data: EntertainmentData? { post.entertainmentData }

    private func categoryIcon(_ category: String) -> String {
        switch category.lowercased() {
        case "award":      return "trophy.fill"
        case "project":    return "film.fill"
        case "appearance": return "star.fill"
        case "social":     return "bubble.fill"
        default:           return "newspaper.fill"
        }
    }

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 0) {

                // MARK: - Hero image with gradient scrim
                let heroURL: URL? = {
                    if let raw = data?.subjectImageUrl, let url = URL(string: raw) { return url }
                    if let raw = post.imageURL, let url = URL(string: raw) { return url }
                    return nil
                }()

                if let url = heroURL {
                    GeometryReader { geo in
                        AsyncImage(url: url) { phase in
                            switch phase {
                            case .success(let img):
                                img.resizable()
                                    .aspectRatio(contentMode: .fill)
                                    .frame(width: geo.size.width, height: 300)
                                    .clipped()
                                    .overlay(alignment: .bottomLeading) {
                                        LinearGradient(
                                            colors: [.clear, .black.opacity(0.75)],
                                            startPoint: .center,
                                            endPoint: .bottom
                                        )
                                        .overlay(alignment: .bottomLeading) {
                                            VStack(alignment: .leading, spacing: 6) {
                                                if let subject = data?.subject, !subject.isEmpty {
                                                    Text(subject.uppercased())
                                                        .font(.title.weight(.black))
                                                        .foregroundStyle(.white)
                                                        .kerning(1)
                                                }
                                                if let cat = data?.category {
                                                    Label(data?.categoryLabel ?? cat, systemImage: categoryIcon(cat))
                                                        .font(.caption.weight(.bold))
                                                        .padding(.horizontal, 10)
                                                        .padding(.vertical, 5)
                                                        .background(.white.opacity(0.2), in: Capsule())
                                                        .foregroundStyle(.white)
                                                }
                                            }
                                            .padding(16)
                                        }
                                    }
                            case .failure:
                                EmptyView()
                            default:
                                Color.secondary.opacity(0.2)
                                    .frame(width: geo.size.width, height: 300)
                                    .overlay(ProgressView())
                            }
                        }
                    }
                    .frame(height: 300)
                } else {
                    // Fallback gradient header when no image
                    ZStack(alignment: .bottomLeading) {
                        LinearGradient(
                            colors: [post.hintColor.opacity(0.7), post.hintColor.opacity(0.4)],
                            startPoint: .topLeading,
                            endPoint: .bottomTrailing
                        )
                        .frame(height: 180)

                        VStack(alignment: .leading, spacing: 6) {
                            if let subject = data?.subject, !subject.isEmpty {
                                Text(subject.uppercased())
                                    .font(.title.weight(.black))
                                    .foregroundStyle(.white)
                                    .kerning(1)
                            }
                            if let cat = data?.category {
                                Label(data?.categoryLabel ?? cat, systemImage: categoryIcon(cat))
                                    .font(.caption.weight(.bold))
                                    .padding(.horizontal, 10)
                                    .padding(.vertical, 5)
                                    .background(.white.opacity(0.2), in: Capsule())
                                    .foregroundStyle(.white)
                            }
                        }
                        .padding(16)
                    }
                }

                // MARK: - Content
                VStack(alignment: .leading, spacing: 18) {

                    // Headline
                    Text(post.title)
                        .font(.title2.weight(.bold))

                    // Pull quote
                    if let quote = data?.quote, !quote.isEmpty {
                        HStack(spacing: 12) {
                            Rectangle()
                                .fill(post.hintColor)
                                .frame(width: 4)
                                .cornerRadius(2)
                            Text("\u{201C}\(quote)\u{201D}")
                                .font(.body.italic())
                                .foregroundStyle(.secondary)
                        }
                        .padding(12)
                        .background(post.hintColor.opacity(0.08), in: RoundedRectangle(cornerRadius: 10))
                    }

                    // Body text
                    if !post.body.isEmpty {
                        Text(post.body)
                            .font(.body)
                            .lineSpacing(4)
                    }

                    // Related project chip
                    if let project = data?.relatedProject, !project.isEmpty {
                        HStack {
                            Image(systemName: "film")
                                .foregroundStyle(post.hintColor)
                            Text(project)
                                .font(.subheadline.weight(.medium))
                        }
                        .padding(.horizontal, 12)
                        .padding(.vertical, 8)
                        .background(post.hintColor.opacity(0.1), in: RoundedRectangle(cornerRadius: 8))
                    }

                    // Tags
                    if let tags = data?.tags, !tags.isEmpty {
                        ScrollView(.horizontal, showsIndicators: false) {
                            HStack(spacing: 8) {
                                ForEach(tags, id: \.self) { tag in
                                    Text(tag)
                                        .font(.caption.weight(.medium))
                                        .padding(.horizontal, 10)
                                        .padding(.vertical, 5)
                                        .background(Color.secondary.opacity(0.12), in: Capsule())
                                        .foregroundStyle(.secondary)
                                }
                            }
                        }
                    }

                    // Source attribution
                    HStack(spacing: 6) {
                        let sourceName = data?.source ?? post.agentName
                        Text("Via \(sourceName)")
                            .font(.caption)
                            .foregroundStyle(.secondary)

                        if let published = data?.publishedAt, !published.isEmpty {
                            Text("·").foregroundStyle(.secondary)
                            Text(String(published.prefix(10)))
                                .font(.caption)
                                .foregroundStyle(.secondary)
                        } else {
                            Text("·").foregroundStyle(.secondary)
                            Text(post.relativeTime)
                                .font(.caption)
                                .foregroundStyle(.secondary)
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
