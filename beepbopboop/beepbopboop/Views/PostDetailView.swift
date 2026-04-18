import SwiftUI
import MapKit
import UIKit

struct PostDetailView: View {
    let post: Post
    @AppStorage private var isBookmarked: Bool
    @Environment(\.dismiss) private var dismiss
    @State private var activeReaction: String?
    @EnvironmentObject private var apiService: APIService
    @EnvironmentObject private var eventTracker: EventTracker

    init(post: Post) {
        self.post = post
        self._isBookmarked = AppStorage(wrappedValue: false, "bookmark_\(post.id)")
        self._activeReaction = State(initialValue: post.myReaction)
    }

    private var bodyLines: [String] {
        post.body.components(separatedBy: "\n").filter { !$0.trimmingCharacters(in: .whitespaces).isEmpty }
    }

    var body: some View {
        detailContent
            .onAppear { eventTracker.fireEvent(postID: post.id, type: "expand") }
    }

    @ViewBuilder
    private var detailContent: some View {
        switch post.displayHintValue {
        case .outfit:
            outfitDetailBody
        case .scoreboard:
            scoreboardDetailBody
        case .matchup:
            matchupDetailBody
        case .standings:
            standingsDetailBody
        case .deal:
            DealDetailView(post: post)
        case .brief, .digest:
            BriefDetailView(post: post)
        case .weather:
            WeatherDetailView(post: post)
        case .calendar, .event:
            CalendarDetailView(post: post)
        case .place:
            PlaceDetailView(post: post)
        case .entertainment:
            EntertainmentDetailView(post: post)
        case .show:
            ShowDetailView(post: post)
        case .restaurant:
            RestaurantDetailView(post: post)
        case .movie:
            MovieDetailView(post: post)
        case .playerSpotlight:
            PlayerSpotlightDetailView(post: post)
        case .gameRelease:
            GameReleaseDetailView(post: post)
        default:
            standardDetailBody
        }
    }

    private var standardDetailBody: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 0) {
                // Hint-specific header
                hintHeader
                    .frame(maxWidth: .infinity, alignment: .leading)

                VStack(alignment: .leading, spacing: 16) {
                    // Agent + relative time
                    HStack(spacing: 6) {
                        Circle()
                            .fill(post.hintColor)
                            .frame(width: 10, height: 10)
                        Text(post.agentName)
                            .font(.subheadline.weight(.medium))
                        Text("·")
                            .foregroundColor(.secondary)
                        Text(post.relativeTime)
                            .font(.subheadline)
                            .foregroundColor(.secondary)
                    }

                    // Title
                    Text(post.title)
                        .font(.title2)
                        .fontWeight(.bold)

                    // Body — hint-aware rendering
                    bodyContent

                    // Image (if available)
                    if let imageURL = post.imageURL, !imageURL.isEmpty, let imgSrc = URL(string: imageURL) {
                        if let externalURL = post.externalURL, !externalURL.isEmpty, let dest = URL(string: externalURL) {
                            Link(destination: dest) {
                                postImage(url: imgSrc)
                            }
                        } else {
                            postImage(url: imgSrc)
                        }
                    }

                    // Map (if coordinates available)
                    if let lat = post.latitude, let lon = post.longitude {
                        Map(initialPosition: .region(MKCoordinateRegion(
                            center: CLLocationCoordinate2D(latitude: lat, longitude: lon),
                            span: MKCoordinateSpan(latitudeDelta: 0.01, longitudeDelta: 0.01)
                        ))) {
                            Marker(post.markerLabel, systemImage: post.typeIcon, coordinate: CLLocationCoordinate2D(latitude: lat, longitude: lon))
                                .tint(post.typeColor)
                        }
                        .frame(height: 200)
                        .cornerRadius(12)
                    }

                    // Metadata
                    GlassEffectContainer(spacing: 8) {
                        HStack(spacing: 8) {
                            Text(post.typeLabel)
                                .font(.caption2.weight(.semibold))
                                .foregroundColor(post.typeColor)
                                .padding(.horizontal, 8)
                                .padding(.vertical, 4)
                                .glassEffect(.regular.tint(post.typeColor), in: .capsule)

                            if post.displayHintValue != .card && post.hintLabel.lowercased() != (post.postType ?? "").lowercased() {
                                Label(post.hintLabel, systemImage: post.hintIcon)
                                    .font(.caption2.weight(.semibold))
                                    .foregroundColor(post.hintColor)
                                    .padding(.horizontal, 8)
                                    .padding(.vertical, 4)
                                    .glassEffect(.regular.tint(post.hintColor), in: .capsule)
                            }

                            if let locality = post.locality, !locality.isEmpty {
                                localityLink
                            }
                        }

                        Label(formattedDate, systemImage: "clock")
                            .font(.caption)
                            .foregroundColor(.secondary)
                    }

                    Divider()

                    // Engagement bar
                    engagementBar
                }
                .padding()
            }
        }
        .navigationTitle(post.hintLabel)
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

    // MARK: - Outfit Detail

    private let outfitMauve = Color(red: 0.878, green: 0.251, blue: 0.984)
    private let warmCream = Color(red: 0.98, green: 0.97, blue: 0.95)

    private var outfitDetailBody: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 0) {
                // Top collage
                outfitHeader

                VStack(alignment: .leading, spacing: 20) {
                    let content = post.outfitContent

                    // Trend subtitle
                    if let trend = content.trend, !trend.isEmpty {
                        Text(trend.uppercased())
                            .font(.system(size: 9, weight: .semibold))
                            .tracking(3)
                            .foregroundColor(Color(red: 0.54, green: 0.49, blue: 0.45))
                    }

                    // Serif title
                    Text(post.title)
                        .font(.system(size: 26, weight: .bold, design: .serif))
                        .foregroundColor(Color(red: 0.1, green: 0.1, blue: 0.1))
                        .lineSpacing(4)

                    // Agent line
                    HStack(spacing: 6) {
                        Circle()
                            .fill(outfitMauve)
                            .frame(width: 10, height: 10)
                        Text(post.agentName)
                            .font(.subheadline.weight(.medium))
                        Text("·")
                            .foregroundColor(.secondary)
                        Text(post.relativeTime)
                            .font(.subheadline)
                            .foregroundColor(.secondary)
                    }

                    // Body text
                    if !content.body.isEmpty {
                        Text(content.body)
                            .font(.system(size: 15))
                            .foregroundColor(Color(red: 0.29, green: 0.29, blue: 0.29))
                            .lineSpacing(6)
                    }

                    // Inline detail image (between body and styled-for-you)
                    outfitInlineImage(slot: 0)

                    // "Styled for you" callout
                    if let forYou = content.forYou, !forYou.isEmpty {
                        VStack(alignment: .leading, spacing: 8) {
                            Text("STYLED FOR YOU")
                                .font(.system(size: 9, weight: .heavy))
                                .tracking(1.5)
                                .foregroundColor(outfitMauve)
                            Text(forYou)
                                .font(.system(size: 13))
                                .foregroundColor(Color(red: 0.227, green: 0.227, blue: 0.227))
                                .lineSpacing(4)
                        }
                        .frame(maxWidth: .infinity, alignment: .leading)
                        .padding(16)
                        .background(
                            LinearGradient(
                                colors: [outfitMauve.opacity(0.05), outfitMauve.opacity(0.02)],
                                startPoint: .topLeading,
                                endPoint: .bottomTrailing
                            )
                        )
                        .overlay(
                            RoundedRectangle(cornerRadius: 12)
                                .stroke(outfitMauve.opacity(0.1), lineWidth: 1)
                        )
                        .cornerRadius(12)
                    }

                    // Second inline detail image
                    outfitInlineImage(slot: 1)

                    // "Shop the look" section
                    if !content.products.isEmpty {
                        VStack(alignment: .leading, spacing: 12) {
                            Text("SHOP THE LOOK")
                                .font(.system(size: 9, weight: .heavy))
                                .tracking(1.5)
                                .foregroundColor(Color(red: 0.1, green: 0.1, blue: 0.1))

                            VStack(spacing: 0) {
                                ForEach(Array(content.products.enumerated()), id: \.offset) { index, product in
                                    outfitProductRow(product: product, index: index)

                                    if index < content.products.count - 1 {
                                        Divider()
                                            .background(Color(red: 0.94, green: 0.93, blue: 0.9))
                                    }
                                }
                            }
                            .overlay(
                                RoundedRectangle(cornerRadius: 12)
                                    .stroke(Color(red: 0.91, green: 0.886, blue: 0.859), lineWidth: 1)
                            )
                            .cornerRadius(12)
                        }
                    }

                    // Budget pick
                    if let alt = content.budgetAlt {
                        VStack(alignment: .leading, spacing: 6) {
                            Text("BUDGET PICK")
                                .font(.system(size: 9, weight: .bold))
                                .tracking(1)
                                .foregroundColor(Color(red: 0.54, green: 0.49, blue: 0.45))
                            HStack {
                                Text(alt.name)
                                    .font(.system(size: 13, weight: .semibold))
                                Spacer()
                                Text(alt.price)
                                    .font(.system(size: 14, weight: .bold))
                                    .foregroundColor(Color(red: 0.1, green: 0.1, blue: 0.1))
                            }
                        }
                        .padding(14)
                        .background(Color(red: 0.94, green: 0.93, blue: 0.9))
                        .cornerRadius(10)
                    }

                    Divider()

                    // Engagement bar
                    engagementBar
                }
                .padding()
            }
        }
        .background(warmCream)
        .navigationTitle("Outfit")
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

    // MARK: - Scoreboard Detail

    private var scoreboardDetailBody: some View {
        let game = post.gameData
        return ScrollView {
            VStack(alignment: .leading, spacing: 0) {
                // Hero header — expanded version of the card
                if let game = game {
                    ZStack {
                        LinearGradient(
                            stops: [
                                .init(color: game.away.swiftUIColor.opacity(0.9), location: 0),
                                .init(color: Color.black.opacity(0.7), location: 0.45),
                                .init(color: Color.black.opacity(0.7), location: 0.55),
                                .init(color: game.home.swiftUIColor.opacity(0.9), location: 1),
                            ],
                            startPoint: .leading,
                            endPoint: .trailing
                        )
                        Color.black.opacity(0.2)

                        Image(systemName: game.sportIcon)
                            .font(.system(size: 160, weight: .ultraLight))
                            .foregroundStyle(.white.opacity(0.05))

                        VStack(spacing: 16) {
                            // League + status
                            HStack {
                                if let league = game.league {
                                    HStack(spacing: 4) {
                                        Image(systemName: game.sportIcon)
                                            .font(.caption2)
                                        Text(league)
                                            .font(.caption.weight(.bold))
                                    }
                                    .foregroundStyle(.white.opacity(0.6))
                                }
                                Spacer()
                                Text(game.status.uppercased())
                                    .font(.system(size: 11, weight: .heavy))
                                    .tracking(0.5)
                                    .foregroundStyle(.white)
                                    .padding(.horizontal, 10)
                                    .padding(.vertical, 5)
                                    .background(Capsule().fill(game.statusColor.opacity(0.8)))
                            }

                            Spacer()

                            // Score
                            HStack(spacing: 0) {
                                let homeWins = (game.home.score ?? 0) > (game.away.score ?? 0)
                                let awayWins = (game.away.score ?? 0) > (game.home.score ?? 0)

                                VStack(spacing: 8) {
                                    Text(game.away.abbr)
                                        .font(.system(size: 24, weight: .heavy, design: .rounded))
                                        .foregroundStyle(.white)
                                        .padding(.horizontal, 14)
                                        .padding(.vertical, 8)
                                        .background(
                                            RoundedRectangle(cornerRadius: 10)
                                                .fill(game.away.swiftUIColor)
                                                .shadow(color: game.away.swiftUIColor.opacity(awayWins ? 0.6 : 0), radius: 10)
                                        )
                                    Text(game.away.name)
                                        .font(.caption.weight(.semibold))
                                        .foregroundStyle(.white.opacity(0.8))
                                    if let record = game.away.record {
                                        Text(record)
                                            .font(.caption2)
                                            .foregroundStyle(.white.opacity(0.45))
                                    }
                                }
                                .frame(maxWidth: .infinity)

                                if let awayScore = game.away.score, let homeScore = game.home.score {
                                    HStack(spacing: 12) {
                                        Text("\(awayScore)")
                                            .font(.system(size: 56, weight: .thin, design: .rounded))
                                            .foregroundStyle(.white)
                                            .opacity(homeWins ? 0.4 : 1.0)
                                            .shadow(color: awayWins ? .white.opacity(0.4) : .clear, radius: 14)
                                        Text("–")
                                            .font(.system(size: 28, weight: .ultraLight))
                                            .foregroundStyle(.white.opacity(0.3))
                                        Text("\(homeScore)")
                                            .font(.system(size: 56, weight: .thin, design: .rounded))
                                            .foregroundStyle(.white)
                                            .opacity(awayWins ? 0.4 : 1.0)
                                            .shadow(color: homeWins ? .white.opacity(0.4) : .clear, radius: 14)
                                    }
                                }

                                VStack(spacing: 8) {
                                    Text(game.home.abbr)
                                        .font(.system(size: 24, weight: .heavy, design: .rounded))
                                        .foregroundStyle(.white)
                                        .padding(.horizontal, 14)
                                        .padding(.vertical, 8)
                                        .background(
                                            RoundedRectangle(cornerRadius: 10)
                                                .fill(game.home.swiftUIColor)
                                                .shadow(color: game.home.swiftUIColor.opacity(homeWins ? 0.6 : 0), radius: 10)
                                        )
                                    Text(game.home.name)
                                        .font(.caption.weight(.semibold))
                                        .foregroundStyle(.white.opacity(0.8))
                                    if let record = game.home.record {
                                        Text(record)
                                            .font(.caption2)
                                            .foregroundStyle(.white.opacity(0.45))
                                    }
                                }
                                .frame(maxWidth: .infinity)
                            }

                            Spacer()

                            // Headline
                            if let headline = game.headline, !headline.isEmpty {
                                Text(headline)
                                    .font(.subheadline.weight(.semibold))
                                    .foregroundStyle(.white.opacity(0.85))
                                    .multilineTextAlignment(.center)
                                    .frame(maxWidth: .infinity)
                            }

                            // Venue + broadcast
                            HStack(spacing: 16) {
                                if let venue = game.venue {
                                    Label(venue, systemImage: "mappin")
                                        .font(.caption2)
                                        .foregroundStyle(.white.opacity(0.5))
                                }
                                Spacer()
                                if let broadcast = game.broadcast {
                                    Label(broadcast, systemImage: "tv")
                                        .font(.caption2)
                                        .foregroundStyle(.white.opacity(0.5))
                                }
                            }
                        }
                        .padding(20)
                    }
                    .frame(height: 300)
                }

                // Recap body
                VStack(alignment: .leading, spacing: 16) {
                    Text(post.title)
                        .font(.title3.weight(.bold))

                    LinkableText(post.body, font: .preferredFont(forTextStyle: .body))
                        .frame(maxWidth: .infinity, alignment: .leading)

                    Divider()
                    sportsEngagementBar
                }
                .padding()
            }
        }
        .navigationTitle(game?.league ?? "Score")
        .navigationBarTitleDisplayMode(.inline)
        .toolbar { closeButton }
    }

    // MARK: - Matchup Detail

    private var matchupDetailBody: some View {
        let game = post.gameData
        return ScrollView {
            VStack(alignment: .leading, spacing: 0) {
                // Hero header
                if let game = game {
                    ZStack {
                        // Diagonal split
                        GeometryReader { _ in
                            ZStack {
                                game.away.swiftUIColor
                                game.home.swiftUIColor
                                    .clipShape(MatchupDiagonalShape())
                                LinearGradient(
                                    colors: [.black.opacity(0.4), .black.opacity(0.25), .black.opacity(0.4)],
                                    startPoint: .top,
                                    endPoint: .bottom
                                )
                            }
                        }

                        Image(systemName: game.sportIcon)
                            .font(.system(size: 160, weight: .ultraLight))
                            .foregroundStyle(.white.opacity(0.05))

                        VStack(spacing: 16) {
                            // League + series
                            HStack {
                                if let league = game.league {
                                    HStack(spacing: 4) {
                                        Image(systemName: game.sportIcon)
                                            .font(.caption2)
                                        Text(league)
                                            .font(.caption.weight(.bold))
                                    }
                                    .foregroundStyle(.white.opacity(0.7))
                                }
                                Spacer()
                                if let series = game.series {
                                    Text(series)
                                        .font(.caption2.weight(.semibold))
                                        .foregroundStyle(.white)
                                        .padding(.horizontal, 10)
                                        .padding(.vertical, 5)
                                        .background(.white.opacity(0.15))
                                        .cornerRadius(6)
                                }
                            }

                            Spacer()

                            // Teams
                            HStack(spacing: 0) {
                                VStack(spacing: 8) {
                                    Text(game.away.abbr)
                                        .font(.system(size: 36, weight: .heavy, design: .rounded))
                                        .foregroundStyle(.white)
                                        .shadow(color: game.away.swiftUIColor.opacity(0.6), radius: 10)
                                    Text(game.away.name)
                                        .font(.subheadline.weight(.semibold))
                                        .foregroundStyle(.white.opacity(0.85))
                                    if let record = game.away.record {
                                        Text(record)
                                            .font(.caption2)
                                            .foregroundStyle(.white.opacity(0.5))
                                    }
                                }
                                .frame(maxWidth: .infinity)

                                ZStack {
                                    Circle()
                                        .fill(.ultraThinMaterial)
                                        .frame(width: 52, height: 52)
                                    Circle()
                                        .stroke(.white.opacity(0.2), lineWidth: 1)
                                        .frame(width: 52, height: 52)
                                    Text("VS")
                                        .font(.system(size: 18, weight: .black, design: .rounded))
                                        .foregroundStyle(.white)
                                }

                                VStack(spacing: 8) {
                                    Text(game.home.abbr)
                                        .font(.system(size: 36, weight: .heavy, design: .rounded))
                                        .foregroundStyle(.white)
                                        .shadow(color: game.home.swiftUIColor.opacity(0.6), radius: 10)
                                    Text(game.home.name)
                                        .font(.subheadline.weight(.semibold))
                                        .foregroundStyle(.white.opacity(0.85))
                                    if let record = game.home.record {
                                        Text(record)
                                            .font(.caption2)
                                            .foregroundStyle(.white.opacity(0.5))
                                    }
                                }
                                .frame(maxWidth: .infinity)
                            }

                            Spacer()

                            // Game time
                            VStack(spacing: 6) {
                                if let countdown = game.countdown {
                                    Text(countdown)
                                        .font(.system(size: 11, weight: .heavy))
                                        .tracking(2)
                                        .foregroundStyle(.white)
                                        .padding(.horizontal, 14)
                                        .padding(.vertical, 5)
                                        .background(Capsule().fill(.white.opacity(0.15)))
                                }
                                if let time = game.formattedGameTime {
                                    Text(time)
                                        .font(.system(size: 32, weight: .bold, design: .rounded))
                                        .foregroundStyle(.white)
                                        .shadow(color: .white.opacity(0.2), radius: 8)
                                }
                                if let date = game.formattedGameDate {
                                    Text(date)
                                        .font(.subheadline.weight(.medium))
                                        .foregroundStyle(.white.opacity(0.65))
                                }
                            }

                            // Venue + broadcast
                            HStack(spacing: 16) {
                                if let venue = game.venue {
                                    Label(venue, systemImage: "mappin")
                                        .font(.caption2)
                                        .foregroundStyle(.white.opacity(0.5))
                                }
                                Spacer()
                                if let broadcast = game.broadcast {
                                    Label(broadcast, systemImage: "tv")
                                        .font(.caption2)
                                        .foregroundStyle(.white.opacity(0.5))
                                }
                            }
                        }
                        .padding(20)
                    }
                    .frame(height: 340)
                }

                // Preview body
                VStack(alignment: .leading, spacing: 16) {
                    Text(post.title)
                        .font(.title3.weight(.bold))

                    LinkableText(post.body, font: .preferredFont(forTextStyle: .body))
                        .frame(maxWidth: .infinity, alignment: .leading)

                    Divider()
                    sportsEngagementBar
                }
                .padding()
            }
        }
        .navigationTitle(game?.league ?? "Matchup")
        .navigationBarTitleDisplayMode(.inline)
        .toolbar { closeButton }
    }

    // MARK: - Standings Detail

    private let standingsDarkBg = Color(red: 0.1, green: 0.09, blue: 0.08)

    private var standingsDetailBody: some View {
        let standings = post.standingsData
        return ScrollView {
            VStack(alignment: .leading, spacing: 0) {
                if let standings = standings {
                    let accent: Color = {
                        guard let hex = standings.leagueColor else { return .blue }
                        let c = Color(hexString: hex)
                        return c == .gray ? .blue : c
                    }()

                    // League header
                    HStack(spacing: 8) {
                        Text(standings.league)
                            .font(.system(size: 20, weight: .heavy, design: .rounded))
                            .foregroundStyle(.white)
                        Text("SCORES")
                            .font(.system(size: 11, weight: .bold))
                            .tracking(2)
                            .foregroundStyle(.white.opacity(0.5))
                        Spacer()
                        Text(standingsFormattedDate(standings.date))
                            .font(.subheadline.weight(.semibold))
                            .foregroundStyle(.white.opacity(0.7))
                    }
                    .padding(.horizontal, 20)
                    .padding(.vertical, 16)
                    .background(
                        LinearGradient(
                            colors: [accent.opacity(0.8), accent.opacity(0.4)],
                            startPoint: .leading,
                            endPoint: .trailing
                        )
                    )

                    // Game rows — expanded for detail
                    VStack(spacing: 0) {
                        ForEach(Array(standings.games.enumerated()), id: \.element.id) { index, game in
                            standingsDetailRow(game)
                            if index < standings.games.count - 1 {
                                Divider()
                                    .overlay(Color.white.opacity(0.06))
                            }
                        }
                    }
                    .padding(.vertical, 8)
                    .background(standingsDarkBg)

                    // Headline banner
                    if let headline = standings.headline, !headline.isEmpty {
                        HStack(spacing: 8) {
                            RoundedRectangle(cornerRadius: 2)
                                .fill(accent)
                                .frame(width: 4, height: 18)
                            Text(headline)
                                .font(.subheadline.weight(.bold))
                                .foregroundStyle(.white.opacity(0.9))
                            Spacer()
                        }
                        .padding(.horizontal, 20)
                        .padding(.vertical, 14)
                        .background(standingsDarkBg)
                    }
                }

                // Summary body
                VStack(alignment: .leading, spacing: 16) {
                    Text(post.title)
                        .font(.title3.weight(.bold))

                    LinkableText(post.body, font: .preferredFont(forTextStyle: .body))
                        .frame(maxWidth: .infinity, alignment: .leading)

                    Divider()
                    sportsEngagementBar
                }
                .padding()
            }
        }
        .navigationTitle(standings?.league ?? "Scores")
        .navigationBarTitleDisplayMode(.inline)
        .toolbar { closeButton }
    }

    @ViewBuilder
    private func standingsDetailRow(_ game: StandingsGame) -> some View {
        let homeWins = game.homeScore > game.awayScore
        let awayWins = game.awayScore > game.homeScore

        HStack(spacing: 0) {
            // Away team
            HStack(spacing: 8) {
                RoundedRectangle(cornerRadius: 2)
                    .fill(game.awaySwiftUIColor)
                    .frame(width: 4, height: 24)
                Text(game.away)
                    .font(.system(size: 16, weight: awayWins ? .bold : .medium, design: .rounded))
                    .foregroundStyle(awayWins ? .white : .white.opacity(0.5))
                    .frame(width: 42, alignment: .leading)
                Text("\(game.awayScore)")
                    .font(.system(size: 18, weight: awayWins ? .bold : .regular, design: .rounded))
                    .foregroundStyle(awayWins ? game.awaySwiftUIColor : .white.opacity(0.4))
                    .frame(width: 26, alignment: .trailing)
            }

            Text("@")
                .font(.caption.weight(.medium))
                .foregroundStyle(.white.opacity(0.2))
                .frame(width: 32)

            // Home team
            HStack(spacing: 8) {
                RoundedRectangle(cornerRadius: 2)
                    .fill(game.homeSwiftUIColor)
                    .frame(width: 4, height: 24)
                Text(game.home)
                    .font(.system(size: 16, weight: homeWins ? .bold : .medium, design: .rounded))
                    .foregroundStyle(homeWins ? .white : .white.opacity(0.5))
                    .frame(width: 42, alignment: .leading)
                Text("\(game.homeScore)")
                    .font(.system(size: 18, weight: homeWins ? .bold : .regular, design: .rounded))
                    .foregroundStyle(homeWins ? game.homeSwiftUIColor : .white.opacity(0.4))
                    .frame(width: 26, alignment: .trailing)
            }

            Spacer()

            Text(game.status.uppercased())
                .font(.system(size: 10, weight: .semibold))
                .foregroundStyle(.white.opacity(0.35))
        }
        .padding(.horizontal, 20)
        .padding(.vertical, 10)
    }

    private func standingsFormattedDate(_ dateStr: String) -> String {
        let f = DateFormatter()
        f.dateFormat = "yyyy-MM-dd"
        // Parse in UTC since bare dates have no timezone offset
        f.timeZone = TimeZone(identifier: "UTC")
        guard let date = f.date(from: dateStr) else { return dateStr }
        var utcCal = Calendar.current
        utcCal.timeZone = TimeZone(identifier: "UTC")!
        let today = Date()
        if utcCal.isDate(date, inSameDayAs: today) { return "Today" }
        if let yesterday = utcCal.date(byAdding: .day, value: -1, to: today),
           utcCal.isDate(date, inSameDayAs: yesterday) { return "Yesterday" }
        f.timeZone = .current
        f.dateFormat = "EEEE, MMM d"
        return f.string(from: date)
    }

    // MARK: - Sports Shared Components

    private var sportsEngagementBar: some View {
        HStack(spacing: 12) {
            Button {
                let wasSaved = isBookmarked
                withAnimation(.bouncy) { isBookmarked.toggle() }
                UIImpactFeedbackGenerator(style: .light).impactOccurred()
                Task {
                    do {
                        try await apiService.trackEvent(
                            postID: post.id,
                            eventType: wasSaved ? "unsave" : "save"
                        )
                    } catch {
                        withAnimation(.bouncy) { isBookmarked = wasSaved }
                    }
                }
            } label: {
                Label(
                    isBookmarked ? "Bookmarked" : "Bookmark",
                    systemImage: isBookmarked ? "bookmark.fill" : "bookmark"
                )
                .font(.subheadline)
                .foregroundColor(isBookmarked ? .orange : .secondary)
                .symbolEffect(.bounce, value: isBookmarked)
                .contentTransition(.symbolEffect(.replace))
            }
            .buttonStyle(.plain)

            ReactionPicker(
                activeReaction: $activeReaction,
                postID: post.id,
                style: .detailBarDark
            )

            Spacer()

            ShareLink(
                item: post.shareURL,
                subject: Text(post.title),
                message: Text(post.body.prefix(100))
            ) {
                Label("Share", systemImage: "square.and.arrow.up")
                    .font(.subheadline)
                    .foregroundColor(.secondary)
            }
            .simultaneousGesture(TapGesture().onEnded {
                Task { await apiService.trackEvent(postID: post.id, type: "share") }
            })
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 12)
        .glassEffect(.regular, in: .rect(cornerRadius: 16))
    }

    private var closeButton: some ToolbarContent {
        ToolbarItem(placement: .topBarTrailing) {
            Button { dismiss() } label: {
                Image(systemName: "xmark.circle.fill")
                    .foregroundStyle(.secondary)
            }
        }
    }

    @ViewBuilder
    private func outfitInlineImage(slot: Int) -> some View {
        let detailImages = post.imagesByRole("detail")
        // Slot 0 = first remaining detail image (after the one used in top collage)
        // Slot 1 = second remaining detail image
        let startIndex = 1 // first detail image goes to top collage
        let imageIndex = startIndex + slot
        if imageIndex < detailImages.count, let url = URL(string: detailImages[imageIndex].url) {
            AsyncImage(url: url) { phase in
                switch phase {
                case .success(let image):
                    image
                        .resizable()
                        .aspectRatio(contentMode: .fill)
                        .frame(maxWidth: .infinity)
                        .frame(maxHeight: 240)
                        .clipped()
                        .cornerRadius(8)
                default:
                    EmptyView()
                }
            }
        }
    }

    private func outfitProductRow(product: OutfitContent.Product, index: Int) -> some View {
        let productImages = post.imagesByRole("product")
        return Button {
            // Open product link if available, otherwise search
            if index < productImages.count,
               let link = productImages[index].link,
               let url = URL(string: link) {
                UIApplication.shared.open(url)
            } else {
                let query = product.name.addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed) ?? product.name
                if let url = URL(string: "https://duckduckgo.com/?q=\(query)") {
                    UIApplication.shared.open(url)
                }
            }
        } label: {
            HStack(spacing: 12) {
                if index < productImages.count, let url = URL(string: productImages[index].url) {
                    AsyncImage(url: url) { phase in
                        switch phase {
                        case .success(let image):
                            image.resizable().aspectRatio(contentMode: .fill)
                        default:
                            RoundedRectangle(cornerRadius: 8)
                                .fill(Color(red: 0.94, green: 0.93, blue: 0.9))
                        }
                    }
                    .frame(width: 44, height: 44)
                    .clipShape(RoundedRectangle(cornerRadius: 8))
                } else {
                    RoundedRectangle(cornerRadius: 8)
                        .fill(Color(red: 0.94, green: 0.93, blue: 0.9))
                        .frame(width: 44, height: 44)
                        .overlay(
                            Image(systemName: "tshirt")
                                .font(.caption2)
                                .foregroundColor(.secondary)
                        )
                }

                VStack(alignment: .leading, spacing: 2) {
                    Text(product.name)
                        .font(.system(size: 13, weight: .semibold))
                        .foregroundColor(.primary)
                    Text(product.price)
                        .font(.system(size: 12))
                        .foregroundColor(Color(red: 0.53, green: 0.53, blue: 0.53))
                }

                Spacer()

                Image(systemName: "chevron.right")
                    .font(.caption2)
                    .foregroundColor(.secondary)
            }
            .padding(12)
        }
        .buttonStyle(.plain)
    }

    // MARK: - Body Content (hint-aware)

    @ViewBuilder
    private var bodyContent: some View {
        switch post.displayHintValue {
        case .digest:
            digestBody
        case .brief:
            briefBody
        case .weather:
            weatherBody
        default:
            LinkableText(post.body, font: .preferredFont(forTextStyle: .body))
                .frame(maxWidth: .infinity, alignment: .leading)
        }
    }

    private var digestBody: some View {
        VStack(alignment: .leading, spacing: 0) {
            ForEach(Array(bodyLines.enumerated()), id: \.offset) { index, line in
                HStack(alignment: .top, spacing: 12) {
                    Text("\(index + 1)")
                        .font(.title3.weight(.bold).monospacedDigit())
                        .foregroundColor(.teal)
                        .frame(width: 28, alignment: .trailing)

                    Text(line.trimmingCharacters(in: .whitespaces))
                        .font(.body)
                        .foregroundColor(.primary)
                }
                .padding(.vertical, 10)

                if index < bodyLines.count - 1 {
                    Divider()
                        .padding(.leading, 40)
                }
            }
        }
        .padding(16)
        .background(Color(.secondarySystemGroupedBackground))
        .cornerRadius(12)
    }

    private var briefBody: some View {
        VStack(alignment: .leading, spacing: 0) {
            ForEach(Array(bodyLines.enumerated()), id: \.offset) { index, line in
                HStack(alignment: .top, spacing: 12) {
                    Image(systemName: "circle")
                        .font(.system(size: 8))
                        .foregroundColor(.secondary)
                        .frame(width: 20, alignment: .center)
                        .padding(.top, 6)

                    Text(line.trimmingCharacters(in: .whitespaces))
                        .font(.body)
                        .foregroundColor(.primary)
                }
                .padding(.vertical, 10)

                if index < bodyLines.count - 1 {
                    Divider()
                        .padding(.leading, 32)
                }
            }
        }
        .padding(16)
        .background(Color(.secondarySystemGroupedBackground))
        .cornerRadius(12)
    }

    private var weatherBody: some View {
        VStack(alignment: .leading, spacing: 0) {
            ForEach(Array(bodyLines.enumerated()), id: \.offset) { index, line in
                let parts = line.split(separator: ":", maxSplits: 1)
                if parts.count == 2 {
                    HStack {
                        Text(parts[0].trimmingCharacters(in: .whitespaces))
                            .font(.subheadline)
                            .foregroundColor(.secondary)
                        Spacer()
                        Text(parts[1].trimmingCharacters(in: .whitespaces))
                            .font(.subheadline.weight(.medium))
                            .foregroundColor(.primary)
                    }
                    .padding(.vertical, 8)
                } else {
                    Text(line.trimmingCharacters(in: .whitespaces))
                        .font(.body)
                        .foregroundColor(.primary)
                        .padding(.vertical, 8)
                }

                if index < bodyLines.count - 1 {
                    Divider()
                }
            }
        }
        .padding(16)
        .background(Color(.secondarySystemGroupedBackground))
        .cornerRadius(12)
    }

    // MARK: - Hint-Specific Headers

    @ViewBuilder
    private var hintHeader: some View {
        switch post.displayHintValue {
        case .weather:
            weatherHeader
        case .deal:
            dealHeader
        case .calendar, .event:
            dateHeader.padding()
        case .comparison:
            comparisonHeader.padding(.horizontal).padding(.top)
        case .digest:
            digestHeader.padding(.horizontal).padding(.top)
        case .brief:
            briefHeader.padding(.horizontal).padding(.top)
        default:
            EmptyView()
        }
    }

    private var weatherHeader: some View {
        let weather = DetailWeatherInfo.detect(from: post.title + " " + post.body)
        return HStack(spacing: 16) {
            Image(systemName: weather.icon)
                .font(.system(size: 48))
                .foregroundStyle(weather.primaryColor, weather.secondaryColor)

            VStack(alignment: .leading, spacing: 4) {
                Text(weather.label)
                    .font(.title3.weight(.semibold))
                    .foregroundColor(.primary)
                if let locality = post.locality, !locality.isEmpty {
                    Label(locality, systemImage: "location")
                        .font(.subheadline)
                        .foregroundColor(.white.opacity(0.8))
                }
            }
            Spacer()
        }
        .padding(20)
        .background(
            LinearGradient(
                colors: [weather.primaryColor.opacity(0.25), weather.secondaryColor.opacity(0.15)],
                startPoint: .topLeading,
                endPoint: .bottomTrailing
            )
        )
    }

    private var dealHeader: some View {
        HStack(spacing: 10) {
            Image(systemName: "tag.fill")
                .font(.title2)
            Text("DEAL")
                .font(.title3.weight(.black))
            Spacer()
            if let locality = post.locality, !locality.isEmpty {
                Text(locality)
                    .font(.subheadline)
                    .foregroundColor(.white.opacity(0.8))
            }
        }
        .foregroundColor(.white)
        .padding(20)
        .background(
            LinearGradient(
                colors: [.pink, .orange],
                startPoint: .leading,
                endPoint: .trailing
            )
        )
    }

    private var dateHeader: some View {
        let parts = extractDateParts()
        return HStack(spacing: 16) {
            VStack(spacing: 2) {
                Text(parts.month)
                    .font(.caption.weight(.bold))
                    .foregroundColor(post.hintColor)
                    .textCase(.uppercase)
                Text(parts.day)
                    .font(.system(size: 36, weight: .bold))
                    .foregroundColor(.primary)
            }
            .frame(width: 72, height: 76)
            .background(post.hintColor.opacity(0.1))
            .cornerRadius(14)

            VStack(alignment: .leading, spacing: 6) {
                Text(post.displayHintValue == .event ? "Event" : "Calendar")
                    .font(.subheadline.weight(.semibold))
                    .foregroundColor(post.hintColor)
                if let locality = post.locality, !locality.isEmpty {
                    Label(locality, systemImage: "location")
                        .font(.subheadline)
                        .foregroundColor(.secondary)
                }
                if post.displayHintValue == .event, let extURL = post.externalURL, !extURL.isEmpty, let url = URL(string: extURL) {
                    Link(destination: url) {
                        Label("Get Tickets", systemImage: "arrow.up.right.square")
                            .font(.subheadline.weight(.medium))
                            .foregroundColor(post.hintColor)
                    }
                }
            }
            Spacer()
        }
    }

    private var comparisonHeader: some View {
        HStack(spacing: 10) {
            Image(systemName: "arrow.left.arrow.right")
                .font(.title3)
            Text("Comparison")
                .font(.headline.weight(.semibold))
            Spacer()
        }
        .foregroundColor(.mint)
        .padding(14)
        .background(.mint.opacity(0.1))
        .cornerRadius(12)
    }

    private var digestHeader: some View {
        HStack(spacing: 10) {
            Image(systemName: "list.bullet.rectangle.fill")
                .font(.title3)
            Text("Digest")
                .font(.headline.weight(.semibold))
            Spacer()
            Text("\(bodyLines.count) items")
                .font(.subheadline)
                .foregroundColor(.teal.opacity(0.7))
        }
        .foregroundColor(.teal)
        .padding(14)
        .background(.teal.opacity(0.1))
        .cornerRadius(12)
    }

    private var briefHeader: some View {
        HStack(spacing: 10) {
            Image(systemName: "checklist")
                .font(.title3)
            Text("Brief")
                .font(.headline.weight(.semibold))
            Spacer()
            Text("\(bodyLines.count) items")
                .font(.subheadline)
                .foregroundColor(.secondary)
        }
        .foregroundColor(.primary)
        .padding(14)
        .background(Color(.secondarySystemGroupedBackground))
        .cornerRadius(12)
    }

    // MARK: - Outfit Header (Collage)

    @ViewBuilder
    private var outfitHeader: some View {
        let allImages = post.images ?? []
        let heroImages = post.imagesByRole("hero")
        let detailImages = post.imagesByRole("detail")

        // Top collage: hero + first detail (max 2 images)
        let collageImages: [PostImage] = {
            var imgs: [PostImage] = []
            if let hero = heroImages.first { imgs.append(hero) }
            else if let first = allImages.first { imgs.append(first) }
            if let firstDetail = detailImages.first { imgs.append(firstDetail) }
            return imgs
        }()

        if !collageImages.isEmpty {
            OutfitCollageView(images: collageImages, postID: post.id)
        } else if let imageURL = post.imageURL, !imageURL.isEmpty, let url = URL(string: imageURL) {
            // Fallback to single imageURL
            AsyncImage(url: url) { phase in
                switch phase {
                case .success(let image):
                    image
                        .resizable()
                        .aspectRatio(contentMode: .fill)
                        .frame(maxWidth: .infinity)
                        .frame(height: 320)
                        .clipped()
                default:
                    Rectangle()
                        .fill(Color(red: 0.94, green: 0.93, blue: 0.9))
                        .frame(height: 320)
                        .overlay(ProgressView())
                }
            }
        }
    }

    // MARK: - Engagement Bar

    private var engagementBar: some View {
        HStack(spacing: 12) {
            Button {
                let wasSaved = isBookmarked
                withAnimation(.bouncy) {
                    isBookmarked.toggle()
                }
                UIImpactFeedbackGenerator(style: .light).impactOccurred()
                Task {
                    do {
                        try await apiService.trackEvent(
                            postID: post.id,
                            eventType: wasSaved ? "unsave" : "save"
                        )
                    } catch {
                        withAnimation(.bouncy) { isBookmarked = wasSaved }
                    }
                }
            } label: {
                Label(
                    isBookmarked ? "Bookmarked" : "Bookmark",
                    systemImage: isBookmarked ? "bookmark.fill" : "bookmark"
                )
                .font(.subheadline)
                .foregroundColor(isBookmarked ? post.typeColor : .secondary)
                .symbolEffect(.bounce, value: isBookmarked)
                .contentTransition(.symbolEffect(.replace))
            }
            .buttonStyle(.plain)

            ReactionPicker(
                activeReaction: $activeReaction,
                postID: post.id,
                style: .detailBar
            )

            Spacer()

            ShareLink(
                item: post.shareURL,
                subject: Text(post.title),
                message: Text(post.body.prefix(100))
            ) {
                Label("Share", systemImage: "square.and.arrow.up")
                    .font(.subheadline)
                    .foregroundColor(.secondary)
            }
            .simultaneousGesture(TapGesture().onEnded {
                Task { await apiService.trackEvent(postID: post.id, type: "share") }
            })

            if let externalURL = post.externalURL, !externalURL.isEmpty, let url = URL(string: externalURL) {
                Link(destination: url) {
                    Label("Open", systemImage: "arrow.up.right.square")
                        .font(.subheadline)
                }
            }
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 12)
        .glassEffect(.regular, in: .rect(cornerRadius: 16))
    }

    // MARK: - Helpers

    @ViewBuilder
    private func postImage(url: URL) -> some View {
        AsyncImage(url: url) { phase in
            switch phase {
            case .success(let image):
                image
                    .resizable()
                    .aspectRatio(contentMode: .fit)
                    .frame(maxWidth: .infinity)
                    .cornerRadius(12)
            case .failure:
                EmptyView()
            default:
                ProgressView()
                    .frame(height: 200)
                    .frame(maxWidth: .infinity)
            }
        }
    }

    private var formattedDate: String {
        let formatters: [ISO8601DateFormatter] = {
            let f1 = ISO8601DateFormatter()
            f1.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
            let f2 = ISO8601DateFormatter()
            f2.formatOptions = [.withInternetDateTime]
            return [f1, f2]
        }()
        for f in formatters {
            if let date = f.date(from: post.createdAt) {
                let df = DateFormatter()
                df.dateFormat = "MMM d, yyyy 'at' h:mm a zzz"
                return df.string(from: date)
            }
        }
        return post.createdAt
    }

    private func extractDateParts() -> (month: String, day: String) {
        let detector = try? NSDataDetector(types: NSTextCheckingResult.CheckingType.date.rawValue)
        let range = NSRange(post.title.startIndex..., in: post.title)
        if let match = detector?.firstMatch(in: post.title, range: range), let date = match.date {
            let monthF = DateFormatter()
            monthF.dateFormat = "MMM"
            return (monthF.string(from: date), "\(Calendar.current.component(.day, from: date))")
        }

        let formatters: [ISO8601DateFormatter] = {
            let f1 = ISO8601DateFormatter()
            f1.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
            let f2 = ISO8601DateFormatter()
            f2.formatOptions = [.withInternetDateTime]
            return [f1, f2]
        }()
        var date = Date()
        for f in formatters {
            if let d = f.date(from: post.createdAt) { date = d; break }
        }
        let monthF = DateFormatter()
        monthF.dateFormat = "MMM"
        return (monthF.string(from: date), "\(Calendar.current.component(.day, from: date))")
    }

    @ViewBuilder
    private var localityLink: some View {
        if let locality = post.locality, !locality.isEmpty {
            if post.isSourceAttribution {
                if let ext = post.externalURL, !ext.isEmpty, let url = URL(string: ext) {
                    Link(destination: url) {
                        Label(locality, systemImage: "link")
                            .font(.subheadline)
                    }
                } else {
                    Label(locality, systemImage: "link")
                        .font(.subheadline)
                }
            } else if let lat = post.latitude, let lon = post.longitude,
                      let mapURL = URL(string: "https://maps.apple.com/?ll=\(lat),\(lon)&q=\(locality.addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed) ?? locality)") {
                Link(destination: mapURL) {
                    Label(locality, systemImage: "location")
                        .font(.subheadline)
                }
            } else {
                Label(locality, systemImage: "location")
                    .font(.subheadline)
            }
        }
    }
}

// MARK: - Outfit Collage View

private struct OutfitCollageView: View {
    let images: [PostImage]
    let postID: String
    private let gap: CGFloat = 3

    private var templateIndex: Int {
        abs(postID.hashValue)
    }

    var body: some View {
        GeometryReader { geo in
            let width = geo.size.width
            collageLayout(width: width)
        }
        .aspectRatio(collageAspectRatio, contentMode: .fit)
    }

    private var collageAspectRatio: CGFloat {
        switch images.count {
        case 1: return 16.0 / 10.0
        case 2:
            let variant = templateIndex % 3
            switch variant {
            case 0: return 16.0 / 9.0   // 2A side-by-side
            case 1: return 9.0 / 14.0   // 2B stacked
            default: return 16.0 / 10.0 // 2C offset
            }
        default: return 16.0 / 10.0
        }
    }

    @ViewBuilder
    private func collageLayout(width: CGFloat) -> some View {
        switch images.count {
        case 1:
            collageImage(images[0], width: width, height: width / (16.0 / 10.0))
        case 2:
            layout2(width: width)
        default:
            collageImage(images[0], width: width, height: width / (16.0 / 10.0))
        }
    }

    @ViewBuilder
    private func layout2(width: CGFloat) -> some View {
        let variant = templateIndex % 3
        switch variant {
        case 0:
            // 2A: Side by side 60/40
            let leftW = (width - gap) * 0.6
            let rightW = (width - gap) * 0.4
            let height = width / (16.0 / 9.0)
            HStack(spacing: gap) {
                collageImage(images[0], width: leftW, height: height)
                collageImage(images[1], width: rightW, height: height)
            }
        case 1:
            // 2B: Stacked vertically
            let topH = width * 0.55
            let bottomH = width * 0.35
            VStack(spacing: gap) {
                collageImage(images[0], width: width, height: topH)
                collageImage(images[1], width: width, height: bottomH)
            }
        default:
            // 2C: Side by side with offset heights
            let halfW = (width - gap) / 2
            let tallH = width / (16.0 / 10.0)
            let shortH = tallH * 0.75
            HStack(alignment: .top, spacing: gap) {
                collageImage(images[0], width: halfW, height: tallH)
                collageImage(images[1], width: halfW, height: shortH)
                    .padding(.top, tallH - shortH)
            }
        }
    }

    @ViewBuilder
    private func collageImage(_ img: PostImage, width: CGFloat, height: CGFloat) -> some View {
        if let url = URL(string: img.url) {
            AsyncImage(url: url) { phase in
                switch phase {
                case .success(let image):
                    image
                        .resizable()
                        .aspectRatio(contentMode: .fill)
                        .frame(width: width, height: height)
                        .clipped()
                case .failure:
                    Rectangle()
                        .fill(Color(red: 0.94, green: 0.93, blue: 0.9))
                        .frame(width: width, height: height)
                default:
                    Rectangle()
                        .fill(Color(red: 0.94, green: 0.93, blue: 0.9))
                        .frame(width: width, height: height)
                        .overlay(ProgressView())
                }
            }
        }
    }
}

// MARK: - Detail Weather Info

private struct DetailWeatherInfo {
    let icon: String
    let label: String
    let primaryColor: Color
    let secondaryColor: Color

    static func detect(from text: String) -> DetailWeatherInfo {
        let lower = text.lowercased()

        if lower.contains("snow") || lower.contains("blizzard") {
            return DetailWeatherInfo(icon: "cloud.snow.fill", label: "Snow", primaryColor: .gray, secondaryColor: .white)
        }
        if lower.contains("thunder") || lower.contains("lightning") || lower.contains("storm") {
            return DetailWeatherInfo(icon: "cloud.bolt.rain.fill", label: "Thunderstorm", primaryColor: .gray, secondaryColor: .yellow)
        }
        if lower.contains("heavy rain") || lower.contains("downpour") || lower.contains("torrential") {
            return DetailWeatherInfo(icon: "cloud.heavyrain.fill", label: "Heavy Rain", primaryColor: .gray, secondaryColor: .blue)
        }
        if lower.contains("rain") || lower.contains("drizzle") || lower.contains("shower") {
            return DetailWeatherInfo(icon: "cloud.rain.fill", label: "Rain", primaryColor: .gray, secondaryColor: .cyan)
        }
        if lower.contains("partly cloudy") || lower.contains("partly sunny") || lower.contains("mix of sun") {
            return DetailWeatherInfo(icon: "cloud.sun.fill", label: "Partly Cloudy", primaryColor: .cyan, secondaryColor: .yellow)
        }
        if lower.contains("overcast") || lower.contains("cloudy") {
            return DetailWeatherInfo(icon: "cloud.fill", label: "Cloudy", primaryColor: .gray, secondaryColor: .gray)
        }
        if lower.contains("fog") || lower.contains("mist") || lower.contains("haze") {
            return DetailWeatherInfo(icon: "cloud.fog.fill", label: "Fog", primaryColor: .gray, secondaryColor: .secondary)
        }
        if lower.contains("clear") || lower.contains("sunny") {
            return DetailWeatherInfo(icon: "sun.max.fill", label: "Clear", primaryColor: .yellow, secondaryColor: .orange)
        }
        if lower.contains("wind") || lower.contains("gusty") || lower.contains("breezy") {
            return DetailWeatherInfo(icon: "wind", label: "Windy", primaryColor: .cyan, secondaryColor: .gray)
        }
        return DetailWeatherInfo(icon: "cloud.sun.fill", label: "Partly Cloudy", primaryColor: .cyan, secondaryColor: .yellow)
    }
}

// MARK: - Matchup Diagonal Shape (Detail)

private struct MatchupDiagonalShape: Shape {
    func path(in rect: CGRect) -> Path {
        Path { p in
            p.move(to: CGPoint(x: rect.maxX * 0.35, y: 0))
            p.addLine(to: CGPoint(x: rect.maxX, y: 0))
            p.addLine(to: CGPoint(x: rect.maxX, y: rect.maxY))
            p.addLine(to: CGPoint(x: rect.maxX * 0.65, y: rect.maxY))
            p.closeSubpath()
        }
    }
}
