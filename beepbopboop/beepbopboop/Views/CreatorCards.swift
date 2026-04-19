import SwiftUI

// MARK: - Palette

private let creatorIndigo = Color(red: 0.541, green: 0.169, blue: 0.886)  // #8A2BE2
private let creatorPurple = Color(red: 0.686, green: 0.400, blue: 0.961)  // #AF66F5
private let creatorCream  = Color(red: 0.976, green: 0.969, blue: 1.0)    // #F9F7FF

// MARK: - Feed Card

struct CreatorSpotlightCard: View {
    let post: Post
    let creator: CreatorData

    init?(post: Post) {
        guard post.displayHintValue == .creatorSpotlight,
              let cd = post.creatorData else { return nil }
        self.post = post
        self.creator = cd
    }

    var body: some View {
        VStack(spacing: 0) {
            photoSection
            infoSection
        }
        .background(creatorCream)
    }

    // MARK: Photo

    private var photoSection: some View {
        ZStack(alignment: .bottomLeading) {
            creatorPhoto
            designationBadge
                .padding(10)
        }
        .frame(height: 200)
        .clipped()
    }

    @ViewBuilder
    private var creatorPhoto: some View {
        if let urlStr = post.imageURL, let url = URL(string: urlStr) {
            AsyncImage(url: url) { phase in
                switch phase {
                case .success(let image):
                    image
                        .resizable()
                        .aspectRatio(contentMode: .fill)
                        .frame(maxWidth: .infinity, maxHeight: 200)
                        .clipped()
                case .failure:
                    placeholderBackground
                default:
                    Color(red: 0.93, green: 0.90, blue: 0.97)
                        .overlay(ProgressView().tint(creatorIndigo))
                }
            }
        } else {
            placeholderBackground
        }
    }

    private var placeholderBackground: some View {
        creatorIndigo.opacity(0.15)
            .overlay(
                Image(systemName: "paintpalette")
                    .font(.system(size: 48))
                    .foregroundColor(creatorIndigo.opacity(0.4))
            )
    }

    private var designationBadge: some View {
        Text(creator.designation)
            .font(.caption.weight(.semibold))
            .foregroundColor(.white)
            .padding(.horizontal, 10)
            .padding(.vertical, 5)
            .background(creatorIndigo)
            .clipShape(Capsule())
    }

    // MARK: Info

    private var infoSection: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(post.title)
                .font(.headline)
                .foregroundColor(.primary)
                .lineLimit(1)

            if !post.body.isEmpty {
                Text(post.body)
                    .font(.subheadline)
                    .foregroundColor(.secondary)
                    .lineLimit(3)
            }

            HStack(spacing: 12) {
                if let locality = post.locality, !locality.isEmpty {
                    HStack(spacing: 4) {
                        Image(systemName: "mappin.circle.fill")
                            .font(.caption)
                            .foregroundColor(creatorPurple)
                        Text(locality)
                            .font(.caption)
                            .foregroundColor(.secondary)
                    }
                }

                if let source = creator.source, !source.isEmpty {
                    Text("via \(source)")
                        .font(.caption2)
                        .foregroundColor(creatorPurple)
                }
            }
        }
        .padding(12)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(creatorCream)
    }
}

// MARK: - Detail View

struct CreatorSpotlightDetailView: View {
    let post: Post
    @Environment(\.openURL) private var openURL

    private var creator: CreatorData? { post.creatorData }

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 0) {
                heroSection
                    .frame(height: 300)
                    .clipped()

                VStack(alignment: .leading, spacing: 20) {
                    nameSection
                    if !post.body.isEmpty { bioSection }
                    if let links = creator?.links { linksSection(links) }
                    if let tags = creator?.tags, !tags.isEmpty { tagsSection(tags) }
                    if let works = creator?.notableWorks, !works.isEmpty { worksSection(works) }
                    sourceSection
                }
                .padding(20)
            }
        }
        .ignoresSafeArea(edges: .top)
    }

    // MARK: Hero

    @ViewBuilder
    private var heroSection: some View {
        if let urlStr = post.imageURL, let url = URL(string: urlStr) {
            AsyncImage(url: url) { phase in
                switch phase {
                case .success(let image):
                    image.resizable().aspectRatio(contentMode: .fill)
                default:
                    creatorIndigo.opacity(0.2)
                        .overlay(Image(systemName: "paintpalette").font(.system(size: 64)).foregroundColor(creatorIndigo.opacity(0.4)))
                }
            }
        } else {
            creatorIndigo.opacity(0.2)
                .overlay(Image(systemName: "paintpalette").font(.system(size: 64)).foregroundColor(creatorIndigo.opacity(0.4)))
        }
    }

    // MARK: Name + Designation

    private var nameSection: some View {
        VStack(alignment: .leading, spacing: 6) {
            HStack(alignment: .firstTextBaseline, spacing: 10) {
                Text(post.title)
                    .font(.title2.bold())
                if let d = creator?.designation {
                    Text(d)
                        .font(.subheadline.weight(.medium))
                        .foregroundColor(.white)
                        .padding(.horizontal, 10)
                        .padding(.vertical, 4)
                        .background(creatorIndigo)
                        .clipShape(Capsule())
                }
            }
            if let locality = post.locality, !locality.isEmpty {
                HStack(spacing: 4) {
                    Image(systemName: "mappin.circle.fill").foregroundColor(creatorPurple)
                    Text(locality).font(.subheadline).foregroundColor(.secondary)
                }
            }
        }
    }

    // MARK: Bio

    private var bioSection: some View {
        VStack(alignment: .leading, spacing: 6) {
            sectionHeader("About")
            Text(post.body).font(.body)
        }
    }

    // MARK: Links

    private func linksSection(_ links: CreatorLinks) -> some View {
        VStack(alignment: .leading, spacing: 10) {
            sectionHeader("Links")
            CreatorFlowLayout(spacing: 8) {
                if let url = links.website   { linkChip("Website",    "globe",                  url) }
                if let ig  = links.instagram { linkChip("Instagram",  "camera",                 ig) }
                if let bc  = links.bandcamp  { linkChip("Bandcamp",   "music.note",             bc) }
                if let et  = links.etsy      { linkChip("Etsy",       "bag",                    et) }
                if let sub = links.substack  { linkChip("Substack",   "envelope",               sub) }
                if let sc  = links.soundcloud { linkChip("SoundCloud", "waveform",              sc) }
                if let bh  = links.behance   { linkChip("Behance",    "photo.on.rectangle",     bh) }
            }
        }
    }

    private func linkChip(_ label: String, _ icon: String, _ urlString: String) -> some View {
        Button {
            if let url = URL(string: urlString) { openURL(url) }
        } label: {
            HStack(spacing: 5) {
                Image(systemName: icon).font(.caption)
                Text(label).font(.subheadline.weight(.medium))
            }
            .foregroundColor(creatorIndigo)
            .padding(.horizontal, 12)
            .padding(.vertical, 7)
            .background(creatorIndigo.opacity(0.1))
            .clipShape(Capsule())
        }
    }

    // MARK: Tags

    private func tagsSection(_ tags: [String]) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            sectionHeader("Tags")
            CreatorFlowLayout(spacing: 6) {
                ForEach(tags, id: \.self) { tag in
                    Text(tag)
                        .font(.caption.weight(.medium))
                        .foregroundColor(creatorPurple)
                        .padding(.horizontal, 10)
                        .padding(.vertical, 5)
                        .background(creatorPurple.opacity(0.12))
                        .clipShape(Capsule())
                }
            }
        }
    }

    // MARK: Notable Works

    private func worksSection(_ works: String) -> some View {
        VStack(alignment: .leading, spacing: 6) {
            sectionHeader("Notable Works")
            Text(works).font(.body)
        }
    }

    // MARK: Source

    @ViewBuilder
    private var sourceSection: some View {
        if let source = creator?.source, !source.isEmpty {
            HStack(spacing: 4) {
                Image(systemName: "magnifyingglass").font(.caption)
                Text("Discovered via \(source)").font(.caption)
            }
            .foregroundColor(.secondary)
        }
    }

    private func sectionHeader(_ title: String) -> some View {
        Text(title)
            .font(.subheadline.weight(.semibold))
            .foregroundColor(.secondary)
    }
}

// MARK: - Flow Layout

/// Left-to-right wrapping chip layout.
struct CreatorFlowLayout: Layout {
    var spacing: CGFloat = 8

    func sizeThatFits(proposal: ProposedViewSize, subviews: Subviews, cache: inout ()) -> CGSize {
        let width = proposal.width ?? .infinity
        var x: CGFloat = 0
        var y: CGFloat = 0
        var rowHeight: CGFloat = 0
        for subview in subviews {
            let size = subview.sizeThatFits(.unspecified)
            if x + size.width > width, x > 0 {
                x = 0; y += rowHeight + spacing; rowHeight = 0
            }
            x += size.width + spacing
            rowHeight = max(rowHeight, size.height)
        }
        return CGSize(width: width, height: y + rowHeight)
    }

    func placeSubviews(in bounds: CGRect, proposal: ProposedViewSize, subviews: Subviews, cache: inout ()) {
        var x = bounds.minX
        var y = bounds.minY
        var rowHeight: CGFloat = 0
        for subview in subviews {
            let size = subview.sizeThatFits(.unspecified)
            if x + size.width > bounds.maxX, x > bounds.minX {
                x = bounds.minX; y += rowHeight + spacing; rowHeight = 0
            }
            subview.place(at: CGPoint(x: x, y: y), proposal: ProposedViewSize(size))
            x += size.width + spacing
            rowHeight = max(rowHeight, size.height)
        }
    }
}
