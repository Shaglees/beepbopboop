import SwiftUI

// MARK: - RestaurantCard

struct RestaurantCard: View {
    let post: Post
    let food: FoodData
    @State private var activeReaction: String?

    private let warmBg = Color(red: 0.980, green: 0.980, blue: 0.969)   // #FAFAF7
    private let coral  = Color(red: 0.937, green: 0.267, blue: 0.267)   // #EF4444
    private let sage   = Color(red: 0.518, green: 0.800, blue: 0.086)   // #84CC16

    init?(post: Post) {
        guard let fd = post.foodData else { return nil }
        self.post = post
        self.food = fd
        self._activeReaction = State(initialValue: post.myReaction)
    }

    var body: some View {
        VStack(spacing: 0) {
            heroSection
            infoSection
            RestaurantFooter(post: post, coral: coral, activeReaction: $activeReaction)
        }
        .background(warmBg)
    }

    // MARK: Hero

    private var heroSection: some View {
        ZStack(alignment: .top) {
            heroImage
                .frame(height: 180)
                .clipped()

            // Header overlay
            HStack(spacing: 6) {
                Circle()
                    .fill(coral)
                    .frame(width: 8, height: 8)
                Text(post.agentName)
                    .font(.subheadline.weight(.medium))
                    .foregroundColor(.white)
                Text("Restaurant")
                    .font(.caption2.weight(.semibold))
                    .foregroundColor(.white)
                    .padding(.horizontal, 7)
                    .padding(.vertical, 3)
                    .background(.white.opacity(0.2))
                    .cornerRadius(4)
                Spacer()
                Text(post.relativeTime)
                    .font(.subheadline)
                    .foregroundColor(.white.opacity(0.7))
            }
            .padding(.horizontal, 16)
            .padding(.top, 14)
            .padding(.bottom, 32)
            .background(
                LinearGradient(
                    colors: [.black.opacity(0.35), .clear],
                    startPoint: .top,
                    endPoint: .bottom
                )
            )

            if food.newOpening {
                newBanner
            }

            if let isOpen = food.isOpenNow {
                openPill(isOpen: isOpen)
                    .padding(.top, 14)
                    .padding(.trailing, 16)
                    .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topTrailing)
            }
        }
        .frame(height: 180)
    }

    @ViewBuilder
    private var heroImage: some View {
        if let urlStr = food.imageUrl, let url = URL(string: urlStr) {
            AsyncImage(url: url) { phase in
                switch phase {
                case .success(let image):
                    image.resizable().aspectRatio(contentMode: .fill)
                case .failure:
                    placeholderHero
                default:
                    placeholderHero.overlay(ProgressView())
                }
            }
        } else {
            placeholderHero
        }
    }

    private var placeholderHero: some View {
        Rectangle()
            .fill(
                LinearGradient(
                    colors: [coral.opacity(0.3), coral.opacity(0.15)],
                    startPoint: .topLeading,
                    endPoint: .bottomTrailing
                )
            )
            .overlay(
                Image(systemName: "fork.knife")
                    .font(.system(size: 40))
                    .foregroundColor(coral.opacity(0.5))
            )
    }

    private var newBanner: some View {
        Text("NEW")
            .font(.system(size: 9, weight: .black))
            .tracking(1.5)
            .foregroundColor(.white)
            .padding(.horizontal, 20)
            .padding(.vertical, 4)
            .background(Color(red: 0.133, green: 0.773, blue: 0.369))
            .rotationEffect(.degrees(-45))
            .offset(x: -22, y: 18)
            .clipped()
            .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
    }

    private func openPill(isOpen: Bool) -> some View {
        HStack(spacing: 4) {
            Circle()
                .fill(isOpen ? Color(red: 0.133, green: 0.773, blue: 0.369) : .red)
                .frame(width: 6, height: 6)
            Text(isOpen ? "Open Now" : "Closed")
                .font(.caption2.weight(.semibold))
                .foregroundColor(.white)
        }
        .padding(.horizontal, 8)
        .padding(.vertical, 4)
        .background(.black.opacity(0.5))
        .clipShape(Capsule())
    }

    // MARK: Info

    private var infoSection: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(food.name)
                .font(.system(size: 18, weight: .semibold))
                .foregroundColor(Color(red: 0.1, green: 0.1, blue: 0.1))
                .lineLimit(1)

            if !food.cuisine.isEmpty {
                cuisineChips
            }

            ratingRow
            distancePriceRow

            if !food.mustTry.isEmpty {
                mustTryStrip
            }

            if let pricePerHead = food.pricePerHead {
                Text("~\(pricePerHead)/person")
                    .font(.caption.weight(.medium))
                    .foregroundColor(sage)
            }
        }
        .padding(.horizontal, 16)
        .padding(.top, 12)
        .padding(.bottom, 10)
    }

    private var cuisineChips: some View {
        ScrollView(.horizontal, showsIndicators: false) {
            HStack(spacing: 6) {
                ForEach(food.cuisine, id: \.self) { tag in
                    Text(tag)
                        .font(.caption2.weight(.medium))
                        .foregroundColor(coral)
                        .padding(.horizontal, 8)
                        .padding(.vertical, 3)
                        .background(coral.opacity(0.1))
                        .clipShape(Capsule())
                }
            }
        }
    }

    private var ratingRow: some View {
        HStack(spacing: 6) {
            HStack(spacing: 2) {
                ForEach(0..<5) { i in
                    let filled = Double(i) < food.rating
                    let halfFilled = !filled && Double(i) < food.rating + 0.5
                    Image(systemName: filled ? "star.fill" : (halfFilled ? "star.leadinghalf.filled" : "star"))
                        .font(.system(size: 11))
                        .foregroundColor(coral)
                }
            }
            Text(String(format: "%.1f", food.rating))
                .font(.caption.weight(.semibold))
                .foregroundColor(Color(red: 0.2, green: 0.2, blue: 0.2))
            Text("(\(food.reviewCount.formatted()) reviews)")
                .font(.caption)
                .foregroundColor(.secondary)
        }
    }

    private var distancePriceRow: some View {
        HStack(spacing: 4) {
            Image(systemName: "location.fill")
                .font(.caption2)
                .foregroundColor(.secondary)
            if let distM = food.distanceM {
                Text(formatDistance(distM))
                    .font(.caption)
                    .foregroundColor(.secondary)
            }
            if let price = food.priceRange {
                Text("·")
                    .font(.caption)
                    .foregroundColor(.secondary)
                Text(price)
                    .font(.caption.weight(.semibold))
                    .foregroundColor(Color(red: 0.2, green: 0.2, blue: 0.2))
            }
            if let neighbourhood = food.neighbourhood, !neighbourhood.isEmpty {
                Text("·")
                    .font(.caption)
                    .foregroundColor(.secondary)
                Text(neighbourhood)
                    .font(.caption)
                    .foregroundColor(.secondary)
                    .lineLimit(1)
            }
        }
    }

    private var mustTryStrip: some View {
        HStack(spacing: 4) {
            Text("Try:")
                .font(.caption.weight(.semibold))
                .foregroundColor(.secondary)
            Text(food.mustTry.joined(separator: " · "))
                .font(.caption)
                .italic()
                .foregroundColor(.secondary)
                .lineLimit(1)
        }
    }

    private func formatDistance(_ metres: Double) -> String {
        if metres < 1000 {
            return "\(Int(metres))m away"
        } else {
            return String(format: "%.1fkm away", metres / 1000)
        }
    }
}

// MARK: - Restaurant Footer

private struct RestaurantFooter: View {
    let post: Post
    let coral: Color
    @Binding var activeReaction: String?
    @AppStorage var isBookmarked: Bool
    @EnvironmentObject private var apiService: APIService

    init(post: Post, coral: Color, activeReaction: Binding<String?>) {
        self.post = post
        self.coral = coral
        self._activeReaction = activeReaction
        self._isBookmarked = AppStorage(wrappedValue: false, "bookmark_\(post.id)")
    }

    var body: some View {
        HStack(spacing: 8) {
            if let yelpUrlStr = post.foodData?.yelpUrl, let yelpUrl = URL(string: yelpUrlStr) {
                Link(destination: yelpUrl) {
                    HStack(spacing: 4) {
                        Image(systemName: "arrow.up.right.square")
                            .font(.caption2)
                        Text("Open in Yelp")
                            .font(.caption2.weight(.medium))
                    }
                    .foregroundColor(coral)
                }
            }

            Spacer()

            ReactionPicker(
                activeReaction: $activeReaction,
                postID: post.id,
                style: .feedCompact
            )

            Button {
                UIImpactFeedbackGenerator(style: .light).impactOccurred()
                isBookmarked.toggle()
            } label: {
                Image(systemName: isBookmarked ? "bookmark.fill" : "bookmark")
                    .font(.caption)
                    .foregroundColor(isBookmarked ? coral : .secondary)
                    .contentTransition(.symbolEffect(.replace))
            }
            .buttonStyle(.plain)
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 10)
        .background(Color(red: 0.980, green: 0.980, blue: 0.969))
    }
}
