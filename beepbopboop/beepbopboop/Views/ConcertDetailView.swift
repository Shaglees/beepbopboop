import SwiftUI
import MapKit

struct ConcertDetailView: View {
    let post: Post
    @Environment(\.dismiss) private var dismiss

    private var data: MusicData? { post.musicData }

    private let accentColor = Color(red: 0.984, green: 0.729, blue: 0.012)

    private var heroURL: URL? {
        if let raw = data?.coverUrl, !raw.isEmpty, let url = URL(string: raw) { return url }
        if let hero = post.heroImage?.url, !hero.isEmpty, let url = URL(string: hero) { return url }
        if let raw = post.imageURL, !raw.isEmpty, let url = URL(string: raw) { return url }
        return nil
    }

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 0) {

                // MARK: - Hero
                heroSection

                // MARK: - Content
                VStack(alignment: .leading, spacing: 20) {

                    // Date + Venue block
                    dateVenueBlock

                    // Time row
                    if data?.doorsTime != nil || data?.startTime != nil {
                        timeRow
                    }

                    // Price + on-sale badge
                    if data?.priceRange != nil || data?.onSale == true {
                        priceRow
                    }

                    // Genre tags
                    if let tags = data?.tags, !tags.isEmpty {
                        genreTags(tags)
                    }

                    // Map preview
                    if let lat = data?.latitude, let lon = data?.longitude {
                        mapSection(lat: lat, lon: lon)
                    }

                    // Body description
                    if !post.body.isEmpty {
                        Text(post.body)
                            .font(.body)
                            .foregroundStyle(.primary)
                            .lineSpacing(4)
                    }

                    // Get Tickets button
                    if let ticketUrlStr = data?.ticketUrl, let ticketURL = URL(string: ticketUrlStr) {
                        getTicketsButton(url: ticketURL)
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
    private var heroSection: some View {
        if let url = heroURL {
            GeometryReader { geo in
                AsyncImage(url: url) { phase in
                    switch phase {
                    case .success(let img):
                        img.resizable()
                            .aspectRatio(contentMode: .fill)
                            .frame(width: geo.size.width, height: 260)
                            .clipped()
                            .overlay(alignment: .bottom) {
                                LinearGradient(
                                    colors: [.clear, Color(.systemBackground)],
                                    startPoint: .center,
                                    endPoint: .bottom
                                )
                                .frame(height: 140)
                            }
                            .overlay(alignment: .bottomLeading) {
                                heroOverlay.padding(16)
                            }
                    case .failure:
                        noImageFallback(width: geo.size.width, height: 260)
                    default:
                        Color(.systemGroupedBackground)
                            .frame(width: geo.size.width, height: 260)
                            .overlay(ProgressView())
                    }
                }
            }
            .frame(height: 260)
        } else {
            GeometryReader { geo in
                noImageFallback(width: geo.size.width, height: 260)
                    .overlay(alignment: .bottomLeading) {
                        heroOverlay.padding(16)
                    }
            }
            .frame(height: 260)
        }
    }

    @ViewBuilder
    private var heroOverlay: some View {
        VStack(alignment: .leading, spacing: 6) {
            // LIVE badge
            Text("LIVE")
                .font(.system(size: 10, weight: .black))
                .tracking(2)
                .foregroundStyle(.black)
                .padding(.horizontal, 10)
                .padding(.vertical, 5)
                .background(accentColor, in: Capsule())

            Text(data?.artist ?? post.title)
                .font(.system(size: 28, weight: .bold))
                .foregroundStyle(.primary)
                .lineLimit(2)
                .shadow(color: Color(.systemBackground).opacity(0.4), radius: 4)
        }
    }

    @ViewBuilder
    private func noImageFallback(width: CGFloat, height: CGFloat) -> some View {
        LinearGradient(
            colors: [accentColor.opacity(0.9), Color(.systemBackground)],
            startPoint: .top,
            endPoint: .bottom
        )
        .frame(width: width, height: height)
        .overlay {
            Image(systemName: "music.mic")
                .font(.system(size: 72, weight: .ultraLight))
                .foregroundStyle(.black.opacity(0.2))
        }
    }

    // MARK: - Date / Venue Block

    @ViewBuilder
    private var dateVenueBlock: some View {
        if data?.date != nil || data?.venue != nil {
            HStack(spacing: 16) {
                // Calendar tile
                if data?.monthAbbrev != nil || data?.dayNumber != nil {
                    VStack(spacing: 2) {
                        if let month = data?.monthAbbrev {
                            Text(month)
                                .font(.system(size: 11, weight: .bold))
                                .foregroundStyle(accentColor)
                        }
                        if let day = data?.dayNumber {
                            Text(day)
                                .font(.system(size: 36, weight: .bold, design: .rounded))
                                .foregroundStyle(.primary)
                        }
                    }
                    .frame(width: 64, height: 68)
                    .background(accentColor.opacity(0.1), in: RoundedRectangle(cornerRadius: 12))
                }

                VStack(alignment: .leading, spacing: 4) {
                    if let venue = data?.venue {
                        Text(venue)
                            .font(.headline.weight(.bold))
                    }
                    if let addr = data?.venueAddress {
                        Text(addr)
                            .font(.subheadline)
                            .foregroundStyle(.secondary)
                    }
                    if let formattedDate = data?.formattedDate {
                        Text(formattedDate)
                            .font(.subheadline.weight(.medium))
                            .foregroundStyle(accentColor)
                    }
                }
                Spacer()
            }
            .padding(14)
            .background(Color(.secondarySystemGroupedBackground), in: RoundedRectangle(cornerRadius: 14))
        }
    }

    // MARK: - Time Row

    @ViewBuilder
    private var timeRow: some View {
        HStack(spacing: 20) {
            if let doors = data?.doorsTime {
                HStack(spacing: 6) {
                    Image(systemName: "clock")
                        .font(.subheadline)
                        .foregroundStyle(accentColor)
                    VStack(alignment: .leading, spacing: 1) {
                        Text("DOORS")
                            .font(.system(size: 9, weight: .bold))
                            .tracking(1)
                            .foregroundStyle(.secondary)
                        Text(doors)
                            .font(.subheadline.weight(.semibold))
                            .foregroundStyle(.primary)
                    }
                }
            }

            if let show = data?.startTime {
                HStack(spacing: 6) {
                    Image(systemName: "music.mic")
                        .font(.subheadline)
                        .foregroundStyle(accentColor)
                    VStack(alignment: .leading, spacing: 1) {
                        Text("SHOW")
                            .font(.system(size: 9, weight: .bold))
                            .tracking(1)
                            .foregroundStyle(.secondary)
                        Text(show)
                            .font(.subheadline.weight(.semibold))
                            .foregroundStyle(.primary)
                    }
                }
            }

            Spacer()
        }
    }

    // MARK: - Price Row

    @ViewBuilder
    private var priceRow: some View {
        HStack(spacing: 12) {
            if let price = data?.priceRange {
                Text(price)
                    .font(.subheadline.weight(.semibold))
                    .foregroundStyle(.primary)
                    .padding(.horizontal, 12)
                    .padding(.vertical, 6)
                    .background(Color(.secondarySystemGroupedBackground), in: RoundedRectangle(cornerRadius: 8))
            }

            if data?.onSale == true {
                HStack(spacing: 5) {
                    Circle()
                        .fill(Color.green)
                        .frame(width: 6, height: 6)
                    Text("ON SALE")
                        .font(.system(size: 11, weight: .black))
                        .tracking(1.5)
                        .foregroundStyle(.white)
                }
                .padding(.horizontal, 12)
                .padding(.vertical, 6)
                .background(Color.green, in: RoundedRectangle(cornerRadius: 8))
                .shadow(color: Color.green.opacity(0.5), radius: 6, x: 0, y: 2)
            }

            Spacer()
        }
    }

    // MARK: - Genre Tags

    @ViewBuilder
    private func genreTags(_ tags: [String]) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            Text("GENRE")
                .font(.system(size: 10, weight: .bold))
                .tracking(1.5)
                .foregroundStyle(.secondary)

            ScrollView(.horizontal, showsIndicators: false) {
                HStack(spacing: 8) {
                    ForEach(tags, id: \.self) { tag in
                        Text(tag)
                            .font(.caption.weight(.semibold))
                            .foregroundStyle(accentColor)
                            .padding(.horizontal, 12)
                            .padding(.vertical, 6)
                            .background(accentColor.opacity(0.12), in: Capsule())
                            .overlay(
                                Capsule()
                                    .stroke(accentColor.opacity(0.25), lineWidth: 1)
                            )
                    }
                }
            }
        }
    }

    // MARK: - Map Section

    @ViewBuilder
    private func mapSection(lat: Double, lon: Double) -> some View {
        let coord = CLLocationCoordinate2D(latitude: lat, longitude: lon)
        VStack(alignment: .leading, spacing: 8) {
            Text("VENUE LOCATION")
                .font(.system(size: 10, weight: .bold))
                .tracking(1.5)
                .foregroundStyle(.secondary)

            Map(initialPosition: .region(MKCoordinateRegion(
                center: coord,
                span: MKCoordinateSpan(latitudeDelta: 0.008, longitudeDelta: 0.008)
            ))) {
                Marker(data?.venue ?? post.title, systemImage: "music.mic", coordinate: coord)
                    .tint(accentColor)
            }
            .frame(height: 150)
            .clipShape(RoundedRectangle(cornerRadius: 12))
        }
    }

    // MARK: - Get Tickets Button

    @ViewBuilder
    private func getTicketsButton(url: URL) -> some View {
        Link(destination: url) {
            HStack(spacing: 8) {
                Text("Get Tickets")
                    .font(.headline.weight(.bold))
                Image(systemName: "arrow.up.right")
                    .font(.subheadline.weight(.bold))
            }
            .foregroundStyle(.black)
            .frame(maxWidth: .infinity)
            .padding(.vertical, 16)
            .background(accentColor, in: RoundedRectangle(cornerRadius: 14))
            .shadow(color: accentColor.opacity(0.4), radius: 8, x: 0, y: 4)
        }
    }
}
