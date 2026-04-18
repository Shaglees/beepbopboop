import SwiftUI

struct ScienceDetailView: View {
    let post: Post
    @Environment(\.dismiss) private var dismiss

    private var data: ScienceData? { post.scienceData }

    private var heroURL: URL? {
        if let urlStr = data?.heroImageUrl, !urlStr.isEmpty { return URL(string: urlStr) }
        if let heroImg = post.heroImage { return URL(string: heroImg.url) }
        if let imgURL = post.imageURL, !imgURL.isEmpty { return URL(string: imgURL) }
        return nil
    }

    var body: some View {
        guard let data = data else {
            // Fallback: should not normally reach here
            return AnyView(Text("No science data available").foregroundStyle(.secondary).padding())
        }
        return AnyView(scienceBody(data: data))
    }

    private func scienceBody(data: ScienceData) -> some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 0) {
                // Hero
                heroSection(data: data)

                // Content
                VStack(alignment: .leading, spacing: 20) {

                    // Headline card
                    headlineCard(data: data)

                    // Institution + source row
                    metaRow(data: data)

                    // Tags
                    if !data.tags.isEmpty {
                        tagsRow(data: data)
                    }

                    Divider()

                    // Key findings body
                    keyFindingsSection(data: data)

                    // Research identifiers
                    let hasDOI = data.doi != nil
                    let hasArXiv = data.arxivId != nil
                    if hasDOI || hasArXiv {
                        researchIdentifiers(data: data)
                    }

                    // Read More button
                    if let readURL = data.primaryReadUrl {
                        readMoreButton(url: readURL, data: data)
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
    private func heroSection(data: ScienceData) -> some View {
        ZStack(alignment: .bottomLeading) {
            // Background
            if let url = heroURL {
                GeometryReader { geo in
                    AsyncImage(url: url) { phase in
                        switch phase {
                        case .success(let img):
                            img.resizable()
                                .aspectRatio(contentMode: .fill)
                                .frame(width: geo.size.width, height: 280)
                                .clipped()
                        case .failure:
                            gradientFallback(data: data, width: geo.size.width)
                        default:
                            data.categoryColor
                                .frame(height: 280)
                                .overlay(ProgressView().tint(data.categoryAccentColor))
                        }
                    }
                }
                .frame(height: 280)
            } else {
                GeometryReader { geo in
                    gradientFallback(data: data, width: geo.size.width)
                }
                .frame(height: 280)
            }

            // Bottom gradient fade to systemBackground
            LinearGradient(
                colors: [.clear, Color(.systemBackground)],
                startPoint: UnitPoint(x: 0.5, y: 0.4),
                endPoint: .bottom
            )
            .frame(height: 280)

            // Category badge (bottom-left)
            categoryBadge(data: data)
                .padding(.horizontal, 16)
                .padding(.bottom, 16)
        }
        .frame(height: 280)
    }

    @ViewBuilder
    private func gradientFallback(data: ScienceData, width: CGFloat) -> some View {
        ZStack {
            LinearGradient(
                colors: [data.categoryColor, Color.black],
                startPoint: .top,
                endPoint: .bottom
            )
            .frame(width: width, height: 280)

            Image(systemName: data.categoryIcon)
                .font(.system(size: 140, weight: .ultraLight))
                .foregroundStyle(data.categoryAccentColor.opacity(0.12))
        }
    }

    private func categoryBadge(data: ScienceData) -> some View {
        HStack(spacing: 6) {
            Image(systemName: data.categoryIcon)
                .font(.caption.weight(.semibold))
            Text(data.categoryLabel)
                .font(.caption.weight(.semibold))
        }
        .foregroundStyle(data.categoryColor)
        .padding(.horizontal, 12)
        .padding(.vertical, 6)
        .background(data.categoryAccentColor, in: Capsule())
    }

    // MARK: - Headline Card

    private func headlineCard(data: ScienceData) -> some View {
        HStack(alignment: .top, spacing: 12) {
            RoundedRectangle(cornerRadius: 2)
                .fill(data.categoryAccentColor)
                .frame(width: 4)
            VStack(alignment: .leading, spacing: 6) {
                Text(data.headline)
                    .font(.title3.weight(.semibold))
                    .lineSpacing(3)
                HStack {
                    Text(data.source)
                        .font(.caption.weight(.semibold))
                        .foregroundStyle(data.categoryAccentColor)
                    Spacer()
                    if let date = data.formattedDate {
                        Text(date)
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }
                }
            }
        }
        .padding(16)
        .background(data.categoryAccentColor.opacity(0.08), in: RoundedRectangle(cornerRadius: 12))
    }

    // MARK: - Meta Row

    @ViewBuilder
    private func metaRow(data: ScienceData) -> some View {
        VStack(alignment: .leading, spacing: 10) {
            if let institution = data.institution, !institution.isEmpty {
                Label(institution, systemImage: "building.2.fill")
                    .font(.subheadline.weight(.medium))
                    .foregroundStyle(.primary)
            }

            HStack(spacing: 20) {
                Label(data.source, systemImage: "newspaper")
                    .font(.subheadline)
                    .foregroundStyle(.secondary)

                if let date = data.formattedDate {
                    Label(date, systemImage: "calendar")
                        .font(.subheadline)
                        .foregroundStyle(.secondary)
                }
            }
        }
    }

    // MARK: - Tags Row

    private func tagsRow(data: ScienceData) -> some View {
        ScrollView(.horizontal, showsIndicators: false) {
            HStack(spacing: 8) {
                ForEach(data.tags, id: \.self) { tag in
                    Text(tag)
                        .font(.caption.weight(.medium))
                        .padding(.horizontal, 12)
                        .padding(.vertical, 6)
                        .background(data.categoryAccentColor.opacity(0.12), in: Capsule())
                        .foregroundStyle(data.categoryAccentColor)
                }
            }
        }
    }

    // MARK: - Key Findings

    @ViewBuilder
    private func keyFindingsSection(data: ScienceData) -> some View {
        let paragraphs = post.body
            .components(separatedBy: "\n\n")
            .map { $0.trimmingCharacters(in: .whitespacesAndNewlines) }
            .filter { !$0.isEmpty }

        if paragraphs.count > 1 {
            VStack(alignment: .leading, spacing: 0) {
                ForEach(Array(paragraphs.enumerated()), id: \.offset) { index, paragraph in
                    Text(paragraph)
                        .font(.body)
                        .lineSpacing(4)
                        .frame(maxWidth: .infinity, alignment: .leading)

                    if index < paragraphs.count - 1 {
                        Divider()
                            .padding(.vertical, 12)
                    }
                }
            }
        } else {
            Text(post.body)
                .font(.body)
                .lineSpacing(4)
                .frame(maxWidth: .infinity, alignment: .leading)
        }
    }

    // MARK: - Research Identifiers

    private func researchIdentifiers(data: ScienceData) -> some View {
        HStack(spacing: 12) {
            if let doi = data.doi {
                researchBadge(
                    prefix: "DOI",
                    value: doi.count > 20 ? String(doi.prefix(20)) + "…" : doi,
                    url: data.doiUrl,
                    accentColor: data.categoryAccentColor
                )
            }
            if let arxivId = data.arxivId {
                researchBadge(
                    prefix: "arXiv",
                    value: arxivId,
                    url: data.arxivUrl,
                    accentColor: data.categoryAccentColor
                )
            }
            Spacer()
        }
    }

    @ViewBuilder
    private func researchBadge(prefix: String, value: String, url: URL?, accentColor: Color) -> some View {
        let content = HStack(spacing: 6) {
            Text(prefix)
                .font(.caption2.weight(.bold))
                .foregroundStyle(accentColor)
            Text(value)
                .font(.caption2)
                .foregroundStyle(.secondary)
        }
        .padding(.horizontal, 10)
        .padding(.vertical, 6)
        .background(accentColor.opacity(0.08), in: RoundedRectangle(cornerRadius: 8))
        .overlay(
            RoundedRectangle(cornerRadius: 8)
                .stroke(accentColor.opacity(0.2), lineWidth: 1)
        )

        if let url = url {
            Link(destination: url) {
                content
            }
        } else {
            content
        }
    }

    // MARK: - Read More Button

    private func readMoreButton(url: URL, data: ScienceData) -> some View {
        Link(destination: url) {
            HStack(spacing: 8) {
                Image(systemName: "arrow.up.right.square")
                    .font(.subheadline.weight(.semibold))
                Text(data.doi != nil || data.arxivId != nil ? "Read Full Paper" : "Read More")
                    .font(.subheadline.weight(.semibold))
            }
            .frame(maxWidth: .infinity)
            .padding(.vertical, 14)
            .background(data.categoryAccentColor, in: RoundedRectangle(cornerRadius: 12))
            .foregroundStyle(data.categoryColor)
        }
    }
}
