import SwiftUI
import MapKit

struct RestaurantDetailView: View {
    let post: Post
    @Environment(\.dismiss) private var dismiss

    private var data: FoodData? { post.foodData }

    private let restaurantRed = Color(red: 0.937, green: 0.267, blue: 0.267)

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 0) {

                // MARK: - Full-bleed hero image
                heroSection

                // MARK: - Content body
                VStack(alignment: .leading, spacing: 20) {

                    // Name + cuisine + neighbourhood
                    headerInfo

                    // Rating + price + open status row
                    statsRow

                    // Must-try dishes
                    if let mustTry = data?.mustTry, !mustTry.isEmpty {
                        mustTrySection(mustTry)
                    }

                    Divider()

                    // Body text
                    if !post.body.isEmpty {
                        Text(post.body)
                            .font(.body)
                            .foregroundColor(.primary)
                            .lineSpacing(4)
                    }

                    // Map
                    if let lat = data?.latitude, let lon = data?.longitude {
                        mapSection(lat: lat, lon: lon)
                    }

                    Divider()

                    // CTA buttons
                    ctaButtons

                    Divider()

                    // Engagement bar
                    RestaurantEngagementBar(post: post)
                }
                .padding()
            }
        }
        .navigationTitle("Restaurant")
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

    // MARK: - Hero

    @ViewBuilder
    private var heroSection: some View {
        let heroURL: URL? = {
            if let raw = data?.imageUrl, let url = URL(string: raw) { return url }
            if let raw = post.imageURL, !raw.isEmpty, let url = URL(string: raw) { return url }
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
                                    colors: [.clear, .black.opacity(0.7)],
                                    startPoint: .center,
                                    endPoint: .bottom
                                )
                                .overlay(alignment: .bottomLeading) {
                                    heroOverlay
                                        .padding(16)
                                }
                            }
                    case .failure:
                        Rectangle()
                            .fill(restaurantRed.opacity(0.15))
                            .frame(width: geo.size.width, height: 300)
                            .overlay {
                                Image(systemName: "fork.knife")
                                    .font(.system(size: 48))
                                    .foregroundStyle(restaurantRed.opacity(0.4))
                            }
                    default:
                        Rectangle()
                            .fill(Color(.systemGroupedBackground))
                            .frame(width: geo.size.width, height: 300)
                            .overlay(ProgressView())
                    }
                }
            }
            .frame(height: 300)
        } else {
            // Colour block fallback
            ZStack(alignment: .bottomLeading) {
                LinearGradient(
                    colors: [restaurantRed.opacity(0.8), restaurantRed.opacity(0.4)],
                    startPoint: .topLeading,
                    endPoint: .bottomTrailing
                )
                .frame(height: 200)
                heroOverlay.padding(16)
            }
        }
    }

    @ViewBuilder
    private var heroOverlay: some View {
        VStack(alignment: .leading, spacing: 6) {
            // "NEW" badge
            if data?.newOpening == true {
                Text("NEW OPENING")
                    .font(.system(size: 10, weight: .black))
                    .tracking(1.5)
                    .foregroundStyle(.white)
                    .padding(.horizontal, 10)
                    .padding(.vertical, 5)
                    .background(restaurantRed, in: Capsule())
            }

            Text(post.title)
                .font(.title.weight(.bold))
                .foregroundStyle(.white)
                .lineLimit(2)
        }
    }

    // MARK: - Header info

    @ViewBuilder
    private var headerInfo: some View {
        VStack(alignment: .leading, spacing: 6) {
            // Agent + time
            HStack(spacing: 6) {
                Circle()
                    .fill(restaurantRed)
                    .frame(width: 10, height: 10)
                Text(post.agentName)
                    .font(.subheadline.weight(.medium))
                Text("·")
                    .foregroundColor(.secondary)
                Text(post.relativeTime)
                    .font(.subheadline)
                    .foregroundColor(.secondary)
            }

            if let cuisines = data?.cuisine, !cuisines.isEmpty {
                Text(cuisines.joined(separator: " · "))
                    .font(.subheadline.weight(.medium))
                    .foregroundColor(restaurantRed)
            }

            if let neighbourhood = data?.neighbourhood, !neighbourhood.isEmpty {
                Label(neighbourhood, systemImage: "mappin.and.ellipse")
                    .font(.subheadline)
                    .foregroundColor(.secondary)
            } else if !(data?.address ?? "").isEmpty {
                Label(data?.address ?? "", systemImage: "mappin.and.ellipse")
                    .font(.subheadline)
                    .foregroundColor(.secondary)
            }
        }
    }

    // MARK: - Stats row

    @ViewBuilder
    private var statsRow: some View {
        HStack(spacing: 16) {
            // Star rating
            if let rating = data?.rating {
                VStack(alignment: .leading, spacing: 2) {
                    starRating(rating)
                    if let count = data?.reviewCount, count > 0 {
                        Text("\(count) reviews")
                            .font(.caption)
                            .foregroundColor(.secondary)
                    }
                }
            }

            Spacer()

            // Price range
            if let price = data?.priceRange, !price.isEmpty {
                Text(price)
                    .font(.subheadline.weight(.semibold))
                    .foregroundColor(.primary)
                    .padding(.horizontal, 10)
                    .padding(.vertical, 5)
                    .background(Color(.secondarySystemGroupedBackground))
                    .cornerRadius(8)
            }

            // Open/closed status
            if let isOpen = data?.isOpenNow {
                HStack(spacing: 4) {
                    Circle()
                        .fill(isOpen ? Color.green : Color.red)
                        .frame(width: 8, height: 8)
                    Text(isOpen ? "Open Now" : "Closed")
                        .font(.subheadline.weight(.medium))
                        .foregroundColor(isOpen ? .green : .red)
                }
                .padding(.horizontal, 10)
                .padding(.vertical, 5)
                .background(
                    (isOpen ? Color.green : Color.red).opacity(0.1)
                )
                .cornerRadius(8)
            }
        }
    }

    @ViewBuilder
    private func starRating(_ rating: Double) -> some View {
        HStack(spacing: 2) {
            ForEach(1...5, id: \.self) { i in
                let filled = Double(i) <= rating
                let halfFilled = !filled && Double(i) - 0.5 <= rating
                Image(systemName: filled ? "star.fill" : (halfFilled ? "star.leadinghalf.filled" : "star"))
                    .font(.system(size: 14))
                    .foregroundColor(.orange)
            }
            Text(String(format: "%.1f", rating))
                .font(.subheadline.weight(.semibold))
                .foregroundColor(.primary)
        }
    }

    // MARK: - Must-try

    @ViewBuilder
    private func mustTrySection(_ dishes: [String]) -> some View {
        VStack(alignment: .leading, spacing: 10) {
            Text("MUST TRY")
                .font(.system(size: 11, weight: .bold))
                .tracking(1.5)
                .foregroundColor(restaurantRed)

            ScrollView(.horizontal, showsIndicators: false) {
                HStack(spacing: 8) {
                    ForEach(dishes, id: \.self) { dish in
                        Text(dish)
                            .font(.subheadline.weight(.medium))
                            .foregroundColor(.primary)
                            .padding(.horizontal, 14)
                            .padding(.vertical, 8)
                            .background(restaurantRed.opacity(0.1))
                            .overlay(
                                Capsule()
                                    .stroke(restaurantRed.opacity(0.3), lineWidth: 1)
                            )
                            .clipShape(Capsule())
                    }
                }
            }
        }
    }

    // MARK: - Map

    @ViewBuilder
    private func mapSection(lat: Double, lon: Double) -> some View {
        let coord = CLLocationCoordinate2D(latitude: lat, longitude: lon)
        VStack(alignment: .leading, spacing: 8) {
            Text("LOCATION")
                .font(.system(size: 11, weight: .bold))
                .tracking(1.5)
                .foregroundColor(.secondary)

            Map(initialPosition: .region(MKCoordinateRegion(
                center: coord,
                span: MKCoordinateSpan(latitudeDelta: 0.008, longitudeDelta: 0.008)
            ))) {
                Marker(data?.name ?? post.title, systemImage: "fork.knife", coordinate: coord)
                    .tint(restaurantRed)
            }
            .frame(height: 200)
            .cornerRadius(12)
        }
    }

    // MARK: - CTA buttons

    @ViewBuilder
    private var ctaButtons: some View {
        HStack(spacing: 12) {
            // Get Directions
            if let lat = data?.latitude, let lon = data?.longitude {
                let name = (data?.name ?? post.title).addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed) ?? ""
                if let mapURL = URL(string: "https://maps.apple.com/?ll=\(lat),\(lon)&q=\(name)") {
                    Link(destination: mapURL) {
                        Label("Get Directions", systemImage: "arrow.triangle.turn.up.right.diamond.fill")
                            .font(.subheadline.weight(.semibold))
                            .foregroundColor(.white)
                            .frame(maxWidth: .infinity)
                            .padding(.vertical, 12)
                            .background(restaurantRed)
                            .cornerRadius(10)
                    }
                }
            }

            // Reserve / View on Yelp
            if let yelpUrl = data?.yelpUrl, let url = URL(string: yelpUrl) {
                Link(destination: url) {
                    Label("View on Yelp", systemImage: "fork.knife.circle.fill")
                        .font(.subheadline.weight(.semibold))
                        .foregroundColor(restaurantRed)
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 12)
                        .background(restaurantRed.opacity(0.1))
                        .cornerRadius(10)
                        .overlay(
                            RoundedRectangle(cornerRadius: 10)
                                .stroke(restaurantRed.opacity(0.3), lineWidth: 1)
                        )
                }
            } else if let extURL = post.externalURL, !extURL.isEmpty, let url = URL(string: extURL) {
                Link(destination: url) {
                    Label("Learn More", systemImage: "arrow.up.right.square")
                        .font(.subheadline.weight(.semibold))
                        .foregroundColor(restaurantRed)
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 12)
                        .background(restaurantRed.opacity(0.1))
                        .cornerRadius(10)
                        .overlay(
                            RoundedRectangle(cornerRadius: 10)
                                .stroke(restaurantRed.opacity(0.3), lineWidth: 1)
                        )
                }
            }
        }
    }
}

// MARK: - Engagement Bar

private struct RestaurantEngagementBar: View {
    let post: Post
    @AppStorage private var isBookmarked: Bool
    @State private var activeReaction: String?
    @EnvironmentObject private var apiService: APIService
    @EnvironmentObject private var eventTracker: EventTracker
    init(post: Post) {
        self.post = post
        self._isBookmarked = AppStorage(wrappedValue: false, "bookmark_\(post.id)")
        self._activeReaction = State(initialValue: post.myReaction)
    }
    var body: some View {
        HStack(spacing: 12) {
            Button {
                let wasSaved = isBookmarked
                withAnimation(.bouncy) { isBookmarked.toggle() }
                UIImpactFeedbackGenerator(style: .light).impactOccurred()
                Task {
                    do { try await apiService.trackEvent(postID: post.id, eventType: wasSaved ? "unsave" : "save") }
                    catch { withAnimation(.bouncy) { isBookmarked = wasSaved } }
                }
            } label: {
                Label(isBookmarked ? "Bookmarked" : "Bookmark", systemImage: isBookmarked ? "bookmark.fill" : "bookmark")
                    .font(.subheadline).foregroundColor(isBookmarked ? post.typeColor : .secondary)
                    .symbolEffect(.bounce, value: isBookmarked).contentTransition(.symbolEffect(.replace))
            }.buttonStyle(.plain)
            ReactionPicker(activeReaction: $activeReaction, postID: post.id, style: .detailBar)
            Spacer()
            ShareLink(item: post.shareURL, subject: Text(post.title), message: Text(post.body.prefix(100))) {
                Label("Share", systemImage: "square.and.arrow.up").font(.subheadline).foregroundColor(.secondary)
            }
            if let ext = post.externalURL, !ext.isEmpty, let url = URL(string: ext) {
                Link(destination: url) { Label("Open", systemImage: "arrow.up.right.square").font(.subheadline) }
            }
        }
        .padding(.horizontal, 16).padding(.vertical, 12)
        .glassEffect(.regular, in: .rect(cornerRadius: 16))
    }
}
