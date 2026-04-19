import SwiftUI
import MapKit
import CoreLocation

struct CreatorDetailView: View {
    let post: Post
    @Environment(\.dismiss) private var dismiss

    private var creator: CreatorData? { post.creatorData }

    private let creatorIndigo = Color(red: 0.380, green: 0.333, blue: 0.933)
    private let creatorAmber  = Color(red: 0.969, green: 0.706, blue: 0.118)

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 0) {
                if let c = creator {
                    creatorDetailBody(c)
                } else {
                    fallbackBody
                }
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

    // MARK: - Creator Detail Body

    @ViewBuilder
    private func creatorDetailBody(_ c: CreatorData) -> some View {
        // Hero banner
        heroBanner(c)

        VStack(alignment: .leading, spacing: 20) {
            // Agent + time
            HStack(spacing: 6) {
                Circle()
                    .fill(creatorIndigo)
                    .frame(width: 10, height: 10)
                Text(post.agentName)
                    .font(.subheadline.weight(.medium))
                Text("·")
                    .foregroundColor(.secondary)
                Text(post.relativeTime)
                    .font(.subheadline)
                    .foregroundColor(.secondary)
            }

            // Designation + area badges
            infoBadgesRow(c)

            // Bio
            if !c.bio.isEmpty {
                aboutSection(c.bio)
            }

            // Notable works
            if let works = c.notableWorks, !works.isEmpty {
                notableWorksSection(works)
            }

            // Tags
            if let tags = c.tags, !tags.isEmpty {
                tagsSection(tags)
            }

            // Links
            if !c.links.allLinks.isEmpty {
                linksSection(c.links)
            }

            // Map
            mapSection(c)

            // Source
            if let source = c.source, !source.isEmpty {
                sourceRow(source)
            }

            Divider()

            PostDetailEngagementBar(post: post)
        }
        .padding()
    }

    // MARK: - Hero Banner

    @ViewBuilder
    private func heroBanner(_ c: CreatorData) -> some View {
        GeometryReader { geo in
            ZStack(alignment: .bottomLeading) {
                LinearGradient(
                    colors: [creatorIndigo, creatorIndigo.opacity(0.6)],
                    startPoint: .topLeading,
                    endPoint: .bottomTrailing
                )
                .frame(width: geo.size.width, height: 240)

                // Large background icon
                Image(systemName: c.designationSymbol)
                    .font(.system(size: 120, weight: .ultraLight))
                    .foregroundStyle(.white.opacity(0.07))
                    .frame(maxWidth: .infinity, alignment: .trailing)
                    .padding(.trailing, 24)
                    .padding(.bottom, 24)

                // Name + designation
                VStack(alignment: .leading, spacing: 8) {
                    HStack(spacing: 6) {
                        Text(c.designationIcon)
                            .font(.title3)
                        Text(c.designation.capitalized)
                            .font(.subheadline.weight(.semibold))
                            .foregroundStyle(creatorAmber)
                    }
                    .padding(.horizontal, 10)
                    .padding(.vertical, 5)
                    .background(creatorAmber.opacity(0.15))
                    .clipShape(Capsule())

                    Text(c.name)
                        .font(.system(size: 34, weight: .bold))
                        .foregroundStyle(.white)
                        .shadow(color: .black.opacity(0.3), radius: 4)
                        .lineLimit(2)
                }
                .padding(20)
            }
        }
        .frame(height: 240)
    }

    // MARK: - Info Badges

    @ViewBuilder
    private func infoBadgesRow(_ c: CreatorData) -> some View {
        ScrollView(.horizontal, showsIndicators: false) {
            HStack(spacing: 8) {
                Label(c.areaName, systemImage: "location.fill")
                    .font(.caption.weight(.semibold))
                    .foregroundStyle(creatorIndigo)
                    .padding(.horizontal, 12)
                    .padding(.vertical, 7)
                    .background(creatorIndigo.opacity(0.1), in: Capsule())
                    .overlay(Capsule().stroke(creatorIndigo.opacity(0.2), lineWidth: 1))

                if let tags = c.tags, let firstTag = tags.first {
                    Label(firstTag, systemImage: "tag")
                        .font(.caption.weight(.semibold))
                        .foregroundStyle(.secondary)
                        .padding(.horizontal, 12)
                        .padding(.vertical, 7)
                        .background(Color(.systemGray6), in: Capsule())
                }
            }
        }
    }

    // MARK: - About

    @ViewBuilder
    private func aboutSection(_ text: String) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            Text("ABOUT")
                .font(.system(size: 11, weight: .bold))
                .tracking(1.5)
                .foregroundStyle(.secondary)

            Text(text)
                .font(.body)
                .foregroundStyle(.primary)
                .lineSpacing(4)
        }
    }

    // MARK: - Notable Works

    @ViewBuilder
    private func notableWorksSection(_ works: String) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            Text("NOTABLE WORKS")
                .font(.system(size: 11, weight: .bold))
                .tracking(1.5)
                .foregroundStyle(.secondary)

            Text(works)
                .font(.subheadline)
                .foregroundStyle(.primary)
                .lineSpacing(3)
        }
    }

    // MARK: - Tags

    @ViewBuilder
    private func tagsSection(_ tags: [String]) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            Text("TAGS")
                .font(.system(size: 11, weight: .bold))
                .tracking(1.5)
                .foregroundStyle(.secondary)

            FlowTagRow(tags: tags, accentColor: creatorIndigo)
        }
    }

    // MARK: - Links

    @ViewBuilder
    private func linksSection(_ links: CreatorLinks) -> some View {
        VStack(alignment: .leading, spacing: 10) {
            Text("FIND THEM ONLINE")
                .font(.system(size: 11, weight: .bold))
                .tracking(1.5)
                .foregroundStyle(.secondary)

            VStack(spacing: 8) {
                ForEach(links.allLinks, id: \.label) { label, url in
                    Link(destination: url) {
                        HStack(spacing: 10) {
                            Image(systemName: linkIcon(for: label))
                                .font(.subheadline)
                                .foregroundStyle(creatorIndigo)
                                .frame(width: 28)
                            Text(label)
                                .font(.subheadline.weight(.medium))
                                .foregroundStyle(.primary)
                            Spacer()
                            Image(systemName: "arrow.up.right")
                                .font(.caption)
                                .foregroundStyle(.tertiary)
                        }
                        .padding(.horizontal, 14)
                        .padding(.vertical, 12)
                        .background(Color(.secondarySystemGroupedBackground),
                                    in: RoundedRectangle(cornerRadius: 10))
                    }
                    .buttonStyle(.plain)
                }
            }
        }
    }

    private func linkIcon(for label: String) -> String {
        switch label.lowercased() {
        case "instagram":    return "camera"
        case "bandcamp":     return "music.note"
        case "etsy":         return "cart"
        case "substack":     return "doc.text"
        case "soundcloud":   return "headphones"
        case "behance":      return "pencil.and.ruler"
        default:             return "link"
        }
    }

    // MARK: - Map

    @ViewBuilder
    private func mapSection(_ c: CreatorData) -> some View {
        let coord = CLLocationCoordinate2D(latitude: c.lat, longitude: c.lon)
        Map(initialPosition: .region(MKCoordinateRegion(
            center: coord,
            span: MKCoordinateSpan(latitudeDelta: 0.04, longitudeDelta: 0.04)
        ))) {
            Marker(c.areaName, systemImage: c.designationSymbol, coordinate: coord)
                .tint(creatorIndigo)
        }
        .frame(height: 160)
        .clipShape(RoundedRectangle(cornerRadius: 12))
        .allowsHitTesting(false)
    }

    // MARK: - Source

    @ViewBuilder
    private func sourceRow(_ source: String) -> some View {
        HStack(spacing: 4) {
            Image(systemName: "magnifyingglass")
                .font(.caption2)
            Text("Discovered via \(source)")
                .font(.caption2)
        }
        .foregroundStyle(.tertiary)
    }

    // MARK: - Fallback

    private var fallbackBody: some View {
        VStack(alignment: .leading, spacing: 16) {
            Text(post.title)
                .font(.title2.weight(.bold))
                .padding(.top, 60)
            Text(post.body)
                .font(.body)
                .foregroundStyle(.primary)
                .lineSpacing(4)
            Divider()
            PostDetailEngagementBar(post: post)
        }
        .padding()
    }
}

// MARK: - Flow Tag Row (wraps tags onto multiple lines)

private struct FlowTagRow: View {
    let tags: [String]
    let accentColor: Color

    var body: some View {
        // Simple horizontal scroll for up to 6 tags.
        ScrollView(.horizontal, showsIndicators: false) {
            HStack(spacing: 6) {
                ForEach(tags.prefix(6), id: \.self) { tag in
                    Text(tag)
                        .font(.caption.weight(.medium))
                        .foregroundStyle(accentColor)
                        .padding(.horizontal, 10)
                        .padding(.vertical, 5)
                        .background(accentColor.opacity(0.1), in: Capsule())
                        .overlay(Capsule().stroke(accentColor.opacity(0.2), lineWidth: 1))
                }
            }
        }
    }
}
