import SwiftUI

// MARK: - Destination Card

struct DestinationCard: View {
    let post: Post
    let travel: TravelData

    private static let darkBg = Color(hex: 0x1a1a2e)
    private static let accent = Color(hex: 0x06B6D4)

    @State private var activeReaction: String?
    @EnvironmentObject private var apiService: APIService

    init?(post: Post) {
        guard let td = post.travelData else { return nil }
        self.post = post
        self.travel = td
        self._activeReaction = State(initialValue: post.myReaction)
    }

    var body: some View {
        VStack(spacing: 0) {
            heroSection
            statStrip
            knownForChips
            if let forecast = travel.weekendForecast {
                forecastStrip(forecast)
            }
            destinationFooter
        }
    }

    // MARK: - Hero

    private var heroSection: some View {
        ZStack(alignment: .top) {
            heroImage

            // Top: header gradient
            VStack {
                destinationHeader
                    .padding(.horizontal, 16)
                    .padding(.top, 14)
                    .padding(.bottom, 32)
                    .background(
                        LinearGradient(
                            colors: [.black.opacity(0.35), .clear],
                            startPoint: .top, endPoint: .bottom
                        )
                    )
                Spacer()
            }

            // Bottom: location + weather badge
            VStack {
                Spacer()
                HStack(alignment: .bottom) {
                    locationLabel
                    Spacer()
                    weatherBadge
                }
                .padding(.horizontal, 14)
                .padding(.bottom, 14)
                .padding(.top, 48)
                .background(
                    LinearGradient(
                        colors: [.clear, .black.opacity(0.55)],
                        startPoint: .top, endPoint: .bottom
                    )
                )
            }
        }
        .frame(height: 280)
        .clipped()
    }

    @ViewBuilder
    private var heroImage: some View {
        let heroURL: URL? = {
            if let url = travel.heroImageUrl { return URL(string: url) }
            if let img = post.heroImage { return URL(string: img.url) }
            return nil
        }()

        if let url = heroURL {
            AsyncImage(url: url) { phase in
                switch phase {
                case .success(let image):
                    image.resizable().aspectRatio(contentMode: .fill)
                        .frame(height: 280)
                        .clipped()
                default:
                    placeholderHero
                }
            }
        } else {
            placeholderHero
        }
    }

    private var placeholderHero: some View {
        ZStack {
            Self.darkBg
            Image(systemName: "airplane")
                .font(.system(size: 64, weight: .ultraLight))
                .foregroundStyle(Self.accent.opacity(0.3))
        }
        .frame(height: 280)
    }

    private var destinationHeader: some View {
        HStack(spacing: 6) {
            Circle()
                .fill(Self.accent)
                .frame(width: 8, height: 8)
            Text(post.agentName)
                .font(.subheadline.weight(.medium))
                .foregroundStyle(.white)
            Text("Destination")
                .font(.caption2.weight(.semibold))
                .foregroundStyle(.white)
                .padding(.horizontal, 7)
                .padding(.vertical, 3)
                .background(.white.opacity(0.2))
                .cornerRadius(4)
            Spacer()
            Text(post.relativeTime)
                .font(.subheadline)
                .foregroundStyle(.white.opacity(0.6))
        }
    }

    private var locationLabel: some View {
        VStack(alignment: .leading, spacing: 2) {
            Text(countryFlag(travel.country))
                .font(.system(size: 22))
            Text("\(travel.city), \(travel.country)")
                .font(.system(size: 22, weight: .bold))
                .foregroundStyle(.white)
                .shadow(color: .black.opacity(0.4), radius: 2, x: 0, y: 1)
                .lineLimit(1)
                .minimumScaleFactor(0.8)
        }
    }

    @ViewBuilder
    private var weatherBadge: some View {
        if let code = travel.currentConditionCode, let temp = travel.currentTempC {
            let icon = WeatherData.icon(for: code, isDay: true)
            HStack(spacing: 4) {
                Image(systemName: icon)
                    .font(.system(size: 13, weight: .semibold))
                    .foregroundStyle(Self.accent)
                Text("\(Int(temp.rounded()))°")
                    .font(.system(size: 13, weight: .bold))
                    .foregroundStyle(.white)
            }
            .padding(.horizontal, 10)
            .padding(.vertical, 6)
            .background(.ultraThinMaterial, in: Capsule())
        }
    }

    // MARK: - Stat Strip

    private var statStrip: some View {
        HStack(spacing: 0) {
            statCell(
                icon: "airplane",
                label: travel.flightPriceFrom ?? "—",
                color: flightPriceColor
            )
            Divider()
                .frame(height: 28)
                .background(.white.opacity(0.15))
            statCell(
                icon: "calendar",
                label: travel.bestTimeToVisit ?? "—",
                color: .white.opacity(0.75)
            )
            Divider()
                .frame(height: 28)
                .background(.white.opacity(0.15))
            visaCell
        }
        .frame(maxWidth: .infinity)
        .padding(.vertical, 10)
        .background(Self.darkBg)
    }

    private func statCell(icon: String, label: String, color: Color) -> some View {
        HStack(spacing: 5) {
            Image(systemName: icon)
                .font(.system(size: 11))
                .foregroundStyle(color.opacity(0.7))
            Text(label)
                .font(.system(size: 12, weight: .semibold))
                .foregroundStyle(color)
                .lineLimit(1)
        }
        .frame(maxWidth: .infinity)
    }

    private var visaCell: some View {
        HStack(spacing: 5) {
            Circle()
                .fill(visaColor)
                .frame(width: 7, height: 7)
            Text(visaLabel)
                .font(.system(size: 12, weight: .semibold))
                .foregroundStyle(.white.opacity(0.75))
                .lineLimit(1)
        }
        .frame(maxWidth: .infinity)
    }

    private var flightPriceColor: Color {
        guard let price = travel.flightPriceFrom,
              let digits = price.components(separatedBy: CharacterSet.decimalDigits.inverted).joined().isEmpty ? nil : Int(price.components(separatedBy: CharacterSet.decimalDigits.inverted).joined()) else {
            return .white.opacity(0.75)
        }
        return digits < 500 ? Color(hex: 0xFBBF24) : .white.opacity(0.75)
    }

    private var visaColor: Color {
        guard let required = travel.visaRequired else { return .gray }
        return required ? .red : .green
    }

    private var visaLabel: String {
        guard let required = travel.visaRequired else { return "Visa info" }
        return required ? "Visa required" : "No visa"
    }

    // MARK: - Known For Chips

    private var knownForChips: some View {
        ScrollView(.horizontal, showsIndicators: false) {
            HStack(spacing: 8) {
                ForEach(Array(travel.knownFor.prefix(4).enumerated()), id: \.offset) { _, fact in
                    Text(fact)
                        .font(.system(size: 12, weight: .medium))
                        .foregroundStyle(Self.accent)
                        .padding(.horizontal, 10)
                        .padding(.vertical, 5)
                        .background(Self.accent.opacity(0.12))
                        .clipShape(Capsule())
                        .overlay(Capsule().stroke(Self.accent.opacity(0.25), lineWidth: 0.5))
                }
            }
            .padding(.horizontal, 14)
            .padding(.vertical, 8)
        }
        .background(Self.darkBg)
    }

    // MARK: - Weekend Forecast Strip

    private func forecastStrip(_ forecast: String) -> some View {
        let code = travel.currentConditionCode ?? 0
        let icon = WeatherData.icon(for: code, isDay: true)
        return HStack(spacing: 6) {
            Image(systemName: icon)
                .font(.system(size: 11))
                .foregroundStyle(Self.accent)
            Text("This weekend: \(forecast)")
                .font(.system(size: 12))
                .foregroundStyle(.white.opacity(0.6))
            Spacer()
        }
        .padding(.horizontal, 14)
        .padding(.vertical, 7)
        .background(Color(hex: 0x141425))
    }

    // MARK: - Footer

    private var destinationFooter: some View {
        VStack(alignment: .leading, spacing: 10) {
            Text(post.body)
                .font(.subheadline)
                .foregroundStyle(.white.opacity(0.75))
                .lineLimit(2)

            HStack(spacing: 8) {
                if let wikiURL = travel.wikiUrl, let url = URL(string: wikiURL) {
                    Link(destination: url) {
                        Label("Wikipedia", systemImage: "globe")
                            .font(.caption2.weight(.semibold))
                            .foregroundStyle(Self.accent)
                    }
                    .buttonStyle(.plain)
                }

                Spacer()

                ReactionPicker(
                    activeReaction: $activeReaction,
                    postID: post.id,
                    style: .feedDark
                )

                DestinationBookmark(post: post, accent: Self.accent)
            }
        }
        .padding(.horizontal, 14)
        .padding(.vertical, 12)
        .background(Self.darkBg)
    }

    // MARK: - Helpers

    private func countryFlag(_ countryName: String) -> String {
        for region in Locale.Region.isoRegions {
            let code = region.identifier
            let locale = Locale(identifier: "en_\(code)")
            if locale.localizedString(forRegionCode: code)?.caseInsensitiveCompare(countryName) == .orderedSame {
                return code.unicodeScalars.compactMap {
                    UnicodeScalar(127397 + $0.value)
                }.reduce("") { $0 + String($1) }
            }
        }
        return "🌍"
    }
}

// MARK: - Bookmark (separate to isolate @AppStorage)

private struct DestinationBookmark: View {
    let post: Post
    let accent: Color
    @AppStorage var isBookmarked: Bool

    init(post: Post, accent: Color) {
        self.post = post
        self.accent = accent
        self._isBookmarked = AppStorage(wrappedValue: false, "bookmark_\(post.id)")
    }

    var body: some View {
        Button {
            UIImpactFeedbackGenerator(style: .light).impactOccurred()
            isBookmarked.toggle()
        } label: {
            Image(systemName: isBookmarked ? "bookmark.fill" : "bookmark")
                .font(.caption)
                .foregroundStyle(isBookmarked ? accent : .white.opacity(0.5))
                .contentTransition(.symbolEffect(.replace))
        }
        .buttonStyle(.plain)
    }
}
