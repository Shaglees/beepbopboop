import SwiftUI

// MARK: - Scoreboard Card

struct ScoreboardCard: View {
    let post: Post
    let game: GameData
    @State private var activeReaction: String?
    @Environment(\.accessibilityReduceMotion) private var reduceMotion

    init?(post: Post) {
        guard let gd = post.gameData else { return nil }
        self.post = post
        self.game = gd
        self._activeReaction = State(initialValue: post.myReaction)
    }

    private var homeWins: Bool {
        guard let hs = game.home.score, let aws = game.away.score else { return false }
        return hs > aws
    }

    private var awayWins: Bool {
        guard let hs = game.home.score, let aws = game.away.score else { return false }
        return aws > hs
    }

    var body: some View {
        ZStack {
            // Two-tone team gradient: away (left) → home (right), darkened
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

            // Subtle dark overlay for depth
            Color.black.opacity(0.25)

            // Large decorative sport icon
            Image(systemName: game.sportIcon)
                .font(.system(size: 120, weight: .ultraLight))
                .foregroundStyle(.white.opacity(0.08))
                .offset(x: 0, y: -10)
                .modifier(SportIconAnimation(reduceMotion: reduceMotion))

            // Live shimmer overlay
            if game.isLive && !reduceMotion {
                LiveShimmerOverlay()
            }

            // Content
            VStack(spacing: 8) {
                // League + status header
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
                    StatusPill(status: game.status, color: game.statusColor, isLive: game.isLive)
                }

                Spacer()

                // Score display
                HStack(spacing: 0) {
                    // Away team
                    VStack(spacing: 6) {
                        // Team color badge
                        Text(game.away.abbr)
                            .font(.system(size: 20, weight: .heavy, design: .rounded))
                            .foregroundStyle(.white)
                            .padding(.horizontal, 12)
                            .padding(.vertical, 6)
                            .background(
                                RoundedRectangle(cornerRadius: 8)
                                    .fill(game.away.swiftUIColor)
                                    .shadow(color: game.away.swiftUIColor.opacity(0.5), radius: awayWins ? 8 : 0)
                            )
                        if let record = game.away.record {
                            Text(record)
                                .font(.system(size: 10, weight: .medium))
                                .foregroundStyle(.white.opacity(0.45))
                        }
                    }
                    .frame(maxWidth: .infinity)

                    // Scores
                    if let awayScore = game.away.score, let homeScore = game.home.score {
                        HStack(spacing: 10) {
                            Text("\(awayScore)")
                                .font(.system(size: 52, weight: .thin, design: .rounded))
                                .foregroundStyle(.white)
                                .opacity(homeWins ? 0.45 : 1.0)
                                .shadow(color: awayWins ? .white.opacity(0.4) : .clear, radius: 12)
                            Text("–")
                                .font(.system(size: 24, weight: .ultraLight))
                                .foregroundStyle(.white.opacity(0.3))
                            Text("\(homeScore)")
                                .font(.system(size: 52, weight: .thin, design: .rounded))
                                .foregroundStyle(.white)
                                .opacity(awayWins ? 0.45 : 1.0)
                                .shadow(color: homeWins ? .white.opacity(0.4) : .clear, radius: 12)
                        }
                    } else {
                        Text("vs")
                            .font(.title2.weight(.light))
                            .foregroundStyle(.white.opacity(0.5))
                    }

                    // Home team
                    VStack(spacing: 6) {
                        Text(game.home.abbr)
                            .font(.system(size: 20, weight: .heavy, design: .rounded))
                            .foregroundStyle(.white)
                            .padding(.horizontal, 12)
                            .padding(.vertical, 6)
                            .background(
                                RoundedRectangle(cornerRadius: 8)
                                    .fill(game.home.swiftUIColor)
                                    .shadow(color: game.home.swiftUIColor.opacity(0.5), radius: homeWins ? 8 : 0)
                            )
                        if let record = game.home.record {
                            Text(record)
                                .font(.system(size: 10, weight: .medium))
                                .foregroundStyle(.white.opacity(0.45))
                        }
                    }
                    .frame(maxWidth: .infinity)
                }

                // Soccer: goal scorers, matchday, cards
                if game.sport?.lowercased() == "soccer" {
                    SoccerScoreboardExtras(game: game)
                        .padding(.horizontal, 4)
                } else {
                    Spacer()
                }

                // Headline stat line + venue
                VStack(spacing: 6) {
                    if let headline = game.headline, !headline.isEmpty {
                        Text(headline)
                            .font(.caption.weight(.semibold))
                            .foregroundStyle(.white.opacity(0.8))
                            .lineLimit(2)
                            .multilineTextAlignment(.center)
                    }

                    HStack(spacing: 12) {
                        if let venue = game.venue {
                            Label(venue, systemImage: "mappin")
                                .font(.caption2)
                                .foregroundStyle(.white.opacity(0.4))
                                .lineLimit(1)
                        }
                        Spacer()
                        if let broadcast = game.broadcast {
                            Label(broadcast, systemImage: "tv")
                                .font(.caption2)
                                .foregroundStyle(.white.opacity(0.4))
                        }
                        ReactionPicker(
                            activeReaction: $activeReaction,
                            postID: post.id,
                            style: .feedDark
                        )
                        SportsBookmarkButton(post: post, darkMode: true)
                    }
                }
            }
            .padding(16)
        }
        .frame(height: (game.sport?.lowercased() == "soccer" && game.goalScorers?.isEmpty == false) ? 250 : 220)
    }
}

// MARK: - Matchup Card

struct MatchupCard: View {
    let post: Post
    let game: GameData
    @State private var activeReaction: String?
    @Environment(\.accessibilityReduceMotion) private var reduceMotion

    init?(post: Post) {
        guard let gd = post.gameData else { return nil }
        self.post = post
        self.game = gd
        self._activeReaction = State(initialValue: post.myReaction)
    }

    var body: some View {
        ZStack {
            // Angular/diagonal split gradient
            GeometryReader { geo in
                ZStack {
                    // Away team color (full background)
                    game.away.swiftUIColor

                    // Home team color (right triangle)
                    game.home.swiftUIColor
                        .clipShape(DiagonalShape())

                    // Dark overlay for readability
                    LinearGradient(
                        colors: [.black.opacity(0.45), .black.opacity(0.3), .black.opacity(0.45)],
                        startPoint: .top,
                        endPoint: .bottom
                    )

                    // Diagonal divider line
                    DiagonalLine()
                        .stroke(.white.opacity(0.15), lineWidth: 1.5)

                    // Subtle noise texture
                    if !reduceMotion {
                        ShimmerLine()
                    }
                }
            }

            // Large watermark sport icon
            Image(systemName: game.sportIcon)
                .font(.system(size: 140, weight: .ultraLight))
                .foregroundStyle(.white.opacity(0.06))
                .modifier(SportIconAnimation(reduceMotion: reduceMotion))

            // Content
            VStack(spacing: 0) {
                // League + series header
                HStack {
                    if let league = game.league {
                        HStack(spacing: 5) {
                            Image(systemName: game.sportIcon)
                                .font(.caption2)
                            Text(league)
                                .font(.caption.weight(.bold))
                        }
                        .foregroundStyle(.white.opacity(0.7))
                    }
                    Spacer()
                    if game.sport?.lowercased() == "soccer", game.matchday != nil || game.leagueShortName != nil {
                        SoccerMatchupHeader(game: game)
                    } else if let series = game.series {
                        Text(series)
                            .font(.caption2.weight(.semibold))
                            .foregroundStyle(.white)
                            .padding(.horizontal, 8)
                            .padding(.vertical, 4)
                            .background(.white.opacity(0.15))
                            .cornerRadius(6)
                    }
                }
                .padding(.bottom, 12)

                Spacer()

                // Teams + VS
                HStack(spacing: 0) {
                    // Away team
                    VStack(spacing: 8) {
                        Text(game.away.abbr)
                            .font(.system(size: 32, weight: .heavy, design: .rounded))
                            .foregroundStyle(.white)
                            .shadow(color: game.away.swiftUIColor.opacity(0.6), radius: 8)
                        Text(game.away.name)
                            .font(.caption.weight(.semibold))
                            .foregroundStyle(.white.opacity(0.8))
                        if let record = game.away.record {
                            Text(record)
                                .font(.system(size: 10, weight: .medium))
                                .foregroundStyle(.white.opacity(0.5))
                        }
                    }
                    .frame(maxWidth: .infinity)

                    // VS divider
                    ZStack {
                        Circle()
                            .fill(.ultraThinMaterial)
                            .frame(width: 44, height: 44)
                        Circle()
                            .stroke(.white.opacity(0.2), lineWidth: 1)
                            .frame(width: 44, height: 44)
                        Text("VS")
                            .font(.system(size: 16, weight: .black, design: .rounded))
                            .foregroundStyle(.white)
                    }

                    // Home team
                    VStack(spacing: 8) {
                        Text(game.home.abbr)
                            .font(.system(size: 32, weight: .heavy, design: .rounded))
                            .foregroundStyle(.white)
                            .shadow(color: game.home.swiftUIColor.opacity(0.6), radius: 8)
                        Text(game.home.name)
                            .font(.caption.weight(.semibold))
                            .foregroundStyle(.white.opacity(0.8))
                        if let record = game.home.record {
                            Text(record)
                                .font(.system(size: 10, weight: .medium))
                                .foregroundStyle(.white.opacity(0.5))
                        }
                    }
                    .frame(maxWidth: .infinity)
                }

                Spacer()

                // Game time + countdown
                VStack(spacing: 6) {
                    if let countdown = game.countdown {
                        Text(countdown)
                            .font(.system(size: 11, weight: .heavy))
                            .tracking(2)
                            .foregroundStyle(.white)
                            .padding(.horizontal, 12)
                            .padding(.vertical, 4)
                            .background(
                                Capsule()
                                    .fill(.white.opacity(0.15))
                            )
                    }
                    if let time = game.formattedGameTimeShort {
                        Text(time)
                            .font(.system(size: 28, weight: .bold, design: .rounded))
                            .foregroundStyle(.white)
                            .shadow(color: .white.opacity(0.2), radius: 6)
                    }
                    HStack(spacing: 12) {
                        if let date = game.formattedGameDate {
                            Text(date)
                                .font(.caption.weight(.medium))
                                .foregroundStyle(.white.opacity(0.6))
                        }
                        if let venue = game.venue {
                            Text("·")
                                .foregroundStyle(.white.opacity(0.3))
                            Label(venue, systemImage: "mappin")
                                .font(.caption2)
                                .foregroundStyle(.white.opacity(0.5))
                                .lineLimit(1)
                        }
                    }
                }
                .padding(.bottom, 4)

                // Broadcast + bookmark
                HStack {
                    if let broadcast = game.broadcast {
                        Label(broadcast, systemImage: "tv")
                            .font(.caption2)
                            .foregroundStyle(.white.opacity(0.4))
                    }
                    Spacer()
                    ReactionPicker(
                        activeReaction: $activeReaction,
                        postID: post.id,
                        style: .feedDark
                    )
                    SportsBookmarkButton(post: post, darkMode: true)
                }

                // Football-specific details
                if game.sport == "football" {
                    FootballMatchupDetails(game: game)
                        .padding(.top, 4)
                }
            }
            .padding(16)
        }
        .frame(minHeight: 260)
    }
}

// MARK: - Football Matchup Details

private struct FootballMatchupDetails: View {
    let game: GameData

    var body: some View {
        VStack(spacing: 6) {
            // Key matchup strip
            if let km = game.keyMatchup {
                Text(km)
                    .font(.system(size: 10, weight: .medium))
                    .italic()
                    .foregroundStyle(.white.opacity(0.7))
                    .multilineTextAlignment(.center)
                    .lineLimit(2)
                    .frame(maxWidth: .infinity)
                    .padding(.horizontal, 8)
                    .padding(.vertical, 5)
                    .background(.white.opacity(0.08))
                    .cornerRadius(6)
            }

            // Weather note
            if let weather = game.weatherNote {
                Text(weather)
                    .font(.system(size: 10, weight: .medium))
                    .foregroundStyle(.white.opacity(0.55))
                    .frame(maxWidth: .infinity, alignment: .center)
            }

            // Injury flags
            if let injuries = game.injuries, !injuries.isEmpty {
                injuryRow(injuries)
            }

            // Fantasy projections
            if let fantasy = game.fantasyPlayers, !fantasy.isEmpty {
                fantasySection(fantasy)
            }
        }
    }

    @ViewBuilder
    private func injuryRow(_ injuries: [InjuryNote]) -> some View {
        ScrollView(.horizontal, showsIndicators: false) {
            HStack(spacing: 6) {
                ForEach(injuries, id: \.player) { injury in
                    HStack(spacing: 4) {
                        Circle()
                            .fill(injuryColor(injury.status))
                            .frame(width: 5, height: 5)
                        Text("\(injury.status): \(injury.player) (\(injury.position))")
                            .font(.system(size: 9, weight: .semibold))
                            .foregroundStyle(.white.opacity(0.85))
                    }
                    .padding(.horizontal, 7)
                    .padding(.vertical, 4)
                    .background(injuryColor(injury.status).opacity(0.2))
                    .cornerRadius(4)
                }
            }
        }
    }

    @ViewBuilder
    private func fantasySection(_ players: [FantasyPlayer]) -> some View {
        VStack(alignment: .leading, spacing: 4) {
            Text("FANTASY")
                .font(.system(size: 8, weight: .heavy))
                .tracking(1.5)
                .foregroundStyle(.white.opacity(0.4))

            let limited = Array(players.prefix(4))
            let columns = Array(repeating: GridItem(.flexible(), spacing: 6), count: min(2, limited.count))
            LazyVGrid(columns: columns, spacing: 4) {
                ForEach(limited, id: \.name) { fp in
                    HStack(spacing: 4) {
                        Circle()
                            .fill(adviceColor(fp.startSitAdvice))
                            .frame(width: 5, height: 5)
                        Text("\(fp.name) \(fp.position)")
                            .font(.system(size: 9, weight: .medium))
                            .foregroundStyle(.white.opacity(0.75))
                            .lineLimit(1)
                        Spacer()
                        Text(String(format: "%.1f", fp.projectedPoints))
                            .font(.system(size: 9, weight: .bold, design: .rounded))
                            .foregroundStyle(adviceColor(fp.startSitAdvice))
                    }
                    .padding(.horizontal, 6)
                    .padding(.vertical, 3)
                    .background(.white.opacity(0.06))
                    .cornerRadius(4)
                }
            }
        }
    }

    private func injuryColor(_ status: String) -> Color {
        switch status.lowercased() {
        case "out", "ir":    return .red
        default:             return Color(red: 1.0, green: 0.7, blue: 0.0) // amber
        }
    }

    private func adviceColor(_ advice: String) -> Color {
        switch advice.lowercased() {
        case "start":  return .green
        case "sit":    return .red
        default:       return Color(red: 1.0, green: 0.7, blue: 0.0) // amber for flex
        }
    }
}

// MARK: - Standings Card

struct StandingsCard: View {
    let post: Post
    let standings: StandingsData
    @State private var activeReaction: String?

    private let darkBg = Color(red: 0.1, green: 0.09, blue: 0.08)

    init?(post: Post) {
        guard let sd = post.standingsData else { return nil }
        self.post = post
        self.standings = sd
        self._activeReaction = State(initialValue: post.myReaction)
    }

    private var accentColor: Color {
        guard let hex = standings.leagueColor else { return .gray }
        let c = Color(hexString: hex)
        return c == .gray ? .blue : c  // fallback black→blue for visibility
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            // League header bar
            HStack(spacing: 8) {
                Text(standings.league)
                    .font(.system(size: 15, weight: .heavy, design: .rounded))
                    .foregroundStyle(.white)
                Text("SCORES")
                    .font(.system(size: 10, weight: .bold))
                    .tracking(1.5)
                    .foregroundStyle(.white.opacity(0.5))
                Spacer()
                Text(formattedDate)
                    .font(.caption2.weight(.semibold))
                    .foregroundStyle(.white.opacity(0.6))
            }
            .padding(.horizontal, 16)
            .padding(.vertical, 12)
            .background(
                LinearGradient(
                    colors: [accentColor.opacity(0.8), accentColor.opacity(0.4)],
                    startPoint: .leading,
                    endPoint: .trailing
                )
            )

            // Game rows
            VStack(spacing: 0) {
                ForEach(Array(standings.games.enumerated()), id: \.element.id) { index, game in
                    gameRow(game)
                    if index < standings.games.count - 1 {
                        Divider()
                            .overlay(Color.white.opacity(0.06))
                    }
                }
            }
            .padding(.vertical, 4)
            .background(darkBg)

            // Headline footer
            if let headline = standings.headline, !headline.isEmpty {
                HStack(spacing: 6) {
                    RoundedRectangle(cornerRadius: 1)
                        .fill(accentColor)
                        .frame(width: 3, height: 14)
                    Text(headline)
                        .font(.caption.weight(.semibold))
                        .foregroundStyle(.white.opacity(0.8))
                        .lineLimit(1)
                    Spacer()
                    ReactionPicker(
                        activeReaction: $activeReaction,
                        postID: post.id,
                        style: .feedDark
                    )
                    SportsBookmarkButton(post: post, darkMode: true)
                }
                .padding(.horizontal, 16)
                .padding(.vertical, 10)
                .background(darkBg)
            } else {
                HStack {
                    Spacer()
                    ReactionPicker(
                        activeReaction: $activeReaction,
                        postID: post.id,
                        style: .feedDark
                    )
                    SportsBookmarkButton(post: post, darkMode: true)
                }
                .padding(.horizontal, 16)
                .padding(.vertical, 8)
                .background(darkBg)
            }
        }
    }

    @ViewBuilder
    private func gameRow(_ game: StandingsGame) -> some View {
        let homeWins = game.homeScore > game.awayScore
        let awayWins = game.awayScore > game.homeScore

        HStack(spacing: 0) {
            // Away team block
            HStack(spacing: 6) {
                RoundedRectangle(cornerRadius: 2)
                    .fill(game.awaySwiftUIColor)
                    .frame(width: 4, height: 20)
                Text(game.away)
                    .font(.system(size: 14, weight: awayWins ? .bold : .medium, design: .rounded))
                    .foregroundStyle(awayWins ? .white : .white.opacity(0.5))
                    .frame(width: 38, alignment: .leading)
                Text("\(game.awayScore)")
                    .font(.system(size: 16, weight: awayWins ? .bold : .regular, design: .rounded))
                    .foregroundStyle(awayWins ? game.awaySwiftUIColor : .white.opacity(0.4))
                    .frame(width: 22, alignment: .trailing)
            }

            Text("@")
                .font(.caption2.weight(.medium))
                .foregroundStyle(.white.opacity(0.2))
                .frame(width: 28)

            // Home team block
            HStack(spacing: 6) {
                RoundedRectangle(cornerRadius: 2)
                    .fill(game.homeSwiftUIColor)
                    .frame(width: 4, height: 20)
                Text(game.home)
                    .font(.system(size: 14, weight: homeWins ? .bold : .medium, design: .rounded))
                    .foregroundStyle(homeWins ? .white : .white.opacity(0.5))
                    .frame(width: 38, alignment: .leading)
                Text("\(game.homeScore)")
                    .font(.system(size: 16, weight: homeWins ? .bold : .regular, design: .rounded))
                    .foregroundStyle(homeWins ? game.homeSwiftUIColor : .white.opacity(0.4))
                    .frame(width: 22, alignment: .trailing)
            }

            Spacer()

            // Status
            Text(game.status)
                .font(.system(size: 10, weight: .semibold))
                .foregroundStyle(.white.opacity(0.35))
                .textCase(.uppercase)
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 8)
    }

    private var formattedDate: String {
        let f = DateFormatter()
        f.dateFormat = "yyyy-MM-dd"
        // Parse in UTC since bare dates have no timezone offset
        f.timeZone = TimeZone(identifier: "UTC")
        guard let date = f.date(from: standings.date) else { return standings.date }
        // Compare using a calendar set to UTC to avoid day-boundary drift
        var utcCal = Calendar.current
        utcCal.timeZone = TimeZone(identifier: "UTC")!
        let today = Date()
        if utcCal.isDate(date, inSameDayAs: today) { return "Today" }
        if let yesterday = utcCal.date(byAdding: .day, value: -1, to: today),
           utcCal.isDate(date, inSameDayAs: yesterday) { return "Yesterday" }
        f.timeZone = .current
        f.dateFormat = "EEE, MMM d"
        return f.string(from: date)
    }
}

// MARK: - Player Spotlight Card

struct PlayerSpotlightCard: View {
    let post: Post
    let player: PlayerData
    @State private var activeReaction: String?
    @Environment(\.accessibilityReduceMotion) private var reduceMotion

    init?(post: Post) {
        guard let pd = post.playerData else { return nil }
        self.post = post
        self.player = pd
        self._activeReaction = State(initialValue: post.myReaction)
    }

    private var teamColor: Color { player.teamSwiftUIColor }

    private var plusMinusColor: Color {
        guard let pm = player.lastGameStats.plusMinus else { return .white.opacity(0.5) }
        return pm > 0 ? .green : pm < 0 ? .red : .white.opacity(0.5)
    }

    var body: some View {
        ZStack {
            // Team color gradient background
            LinearGradient(
                stops: [
                    .init(color: teamColor.opacity(0.95), location: 0),
                    .init(color: teamColor.opacity(0.6), location: 0.5),
                    .init(color: Color.black.opacity(0.85), location: 1),
                ],
                startPoint: .topLeading,
                endPoint: .bottomTrailing
            )

            // Dark overlay for readability
            Color.black.opacity(0.3)

            // Basketball watermark icon
            Image(systemName: post.hintIcon)
                .font(.system(size: 130, weight: .ultraLight))
                .foregroundStyle(.white.opacity(0.05))
                .offset(x: 40, y: 0)
                .modifier(SportIconAnimation(reduceMotion: reduceMotion))

            // Content
            HStack(alignment: .top, spacing: 0) {
                // Left: metadata + stats
                VStack(alignment: .leading, spacing: 0) {
                    // Header: team + league
                    HStack(spacing: 6) {
                        Text(player.league)
                            .font(.system(size: 10, weight: .heavy))
                            .tracking(1.5)
                            .foregroundStyle(.white.opacity(0.5))
                        Text("·")
                            .foregroundStyle(.white.opacity(0.3))
                        Text(player.team)
                            .font(.caption2.weight(.semibold))
                            .foregroundStyle(.white.opacity(0.6))
                            .lineLimit(1)
                    }
                    .padding(.bottom, 6)

                    // Player name
                    Text(player.playerName)
                        .font(.system(size: 20, weight: .bold, design: .rounded))
                        .foregroundStyle(.white)
                        .lineLimit(2)
                        .padding(.bottom, 4)

                    // Position badge + opponent context
                    HStack(spacing: 6) {
                        if let position = player.position {
                            Text(position)
                                .font(.system(size: 9, weight: .semibold))
                                .foregroundStyle(.white)
                                .padding(.horizontal, 7)
                                .padding(.vertical, 3)
                                .background(teamColor.opacity(0.5))
                                .cornerRadius(4)
                        }
                        if let opponent = player.opponent, let result = player.gameResult {
                            Text("vs \(opponent) · \(result)")
                                .font(.system(size: 10, weight: .medium))
                                .foregroundStyle(.white.opacity(0.6))
                                .lineLimit(1)
                        }
                    }
                    .padding(.bottom, 4)

                    // Series context
                    if let series = player.seriesContext {
                        Text(series)
                            .font(.system(size: 10, weight: .semibold))
                            .foregroundStyle(.white)
                            .padding(.horizontal, 8)
                            .padding(.vertical, 3)
                            .background(.white.opacity(0.12))
                            .cornerRadius(5)
                            .padding(.bottom, 6)
                    }

                    Spacer()

                    // Stat trio: PTS / REB / AST
                    HStack(spacing: 16) {
                        statBlock(value: "\(player.lastGameStats.points)", label: "PTS")
                        statBlock(value: "\(player.lastGameStats.rebounds)", label: "REB")
                        statBlock(value: "\(player.lastGameStats.assists)", label: "AST")
                    }
                    .padding(.bottom, 6)

                    // Shooting splits + +/-
                    HStack(spacing: 8) {
                        if let fgPct = player.lastGameStats.fieldGoalPct {
                            Text("FG \(Int((fgPct * 100).rounded()))%")
                                .font(.system(size: 11, weight: .medium))
                                .foregroundStyle(.white.opacity(0.65))
                        }
                        if let threePct = player.lastGameStats.threePointPct {
                            Text("· 3P \(Int((threePct * 100).rounded()))%")
                                .font(.system(size: 11, weight: .medium))
                                .foregroundStyle(.white.opacity(0.65))
                        }
                        if let pm = player.lastGameStats.plusMinus {
                            let pmText = pm >= 0 ? "+\(pm)" : "\(pm)"
                            Text(pmText)
                                .font(.system(size: 11, weight: .bold))
                                .foregroundStyle(plusMinusColor)
                                .padding(.horizontal, 6)
                                .padding(.vertical, 2)
                                .background(plusMinusColor.opacity(0.15))
                                .cornerRadius(4)
                        }
                    }
                    .padding(.bottom, 6)

                    // Season averages footer
                    Text("Season: \(String(format: "%.1f", player.seasonAverages.points)) / \(String(format: "%.1f", player.seasonAverages.rebounds)) / \(String(format: "%.1f", player.seasonAverages.assists)) PPG/RPG/APG")
                        .font(.system(size: 9, weight: .medium))
                        .foregroundStyle(.white.opacity(0.4))
                        .padding(.bottom, 8)

                    // Storyline
                    if let storyline = player.storyline, !storyline.isEmpty {
                        Text(storyline)
                            .font(.system(size: 10, weight: .regular))
                            .italic()
                            .foregroundStyle(.white.opacity(0.6))
                            .lineLimit(2)
                            .padding(.bottom, 6)
                    }

                    // Footer: reaction + bookmark
                    HStack {
                        ReactionPicker(
                            activeReaction: $activeReaction,
                            postID: post.id,
                            style: .feedDark
                        )
                        SportsBookmarkButton(post: post, darkMode: true)
                    }
                }
                .frame(maxWidth: .infinity, alignment: .leading)

                // Right: player headshot
                if let urlStr = player.playerHeadshotUrl, let url = URL(string: urlStr) {
                    AsyncImage(url: url) { phase in
                        switch phase {
                        case .success(let image):
                            image
                                .resizable()
                                .aspectRatio(contentMode: .fill)
                                .frame(width: 130, height: 130)
                                .clipShape(Circle())
                                .overlay(
                                    Circle()
                                        .stroke(teamColor.opacity(0.8), lineWidth: 2)
                                )
                                .shadow(color: teamColor.opacity(0.4), radius: 8)
                        case .failure:
                            playerHeadshotFallback
                        default:
                            Circle()
                                .fill(.white.opacity(0.08))
                                .frame(width: 130, height: 130)
                                .overlay(ProgressView().tint(.white))
                        }
                    }
                    .frame(width: 130, height: 130)
                    .padding(.leading, 12)
                    .padding(.top, 4)
                } else {
                    playerHeadshotFallback
                        .padding(.leading, 12)
                        .padding(.top, 4)
                }
            }
            .padding(16)
        }
        .frame(height: 260)
    }

    private var playerHeadshotFallback: some View {
        Circle()
            .fill(teamColor.opacity(0.3))
            .frame(width: 130, height: 130)
            .overlay(
                Image(systemName: player.sportIcon)
                    .font(.system(size: 40, weight: .ultraLight))
                    .foregroundStyle(.white.opacity(0.4))
            )
            .overlay(Circle().stroke(teamColor.opacity(0.5), lineWidth: 2))
    }

    @ViewBuilder
    private func statBlock(value: String, label: String) -> some View {
        VStack(spacing: 2) {
            Text(value)
                .font(.system(size: 28, weight: .bold, design: .rounded))
                .foregroundStyle(.white)
            Text(label)
                .font(.system(size: 9, weight: .semibold))
                .tracking(1)
                .foregroundStyle(.white.opacity(0.5))
        }
    }
}

// MARK: - Soccer Extras

private struct SoccerScoreboardExtras: View {
    let game: GameData

    var body: some View {
        VStack(alignment: .leading, spacing: 5) {
            // Matchday strip with league accent bar
            if let matchday = game.matchday {
                HStack(spacing: 6) {
                    RoundedRectangle(cornerRadius: 1)
                        .fill(game.leagueAccentColor)
                        .frame(width: 3, height: 10)
                    Text(matchday.uppercased())
                        .font(.system(size: 9, weight: .bold))
                        .tracking(1.2)
                        .foregroundStyle(.white.opacity(0.5))
                }
            }

            // Goal scorers (away left, home right)
            if let scorers = game.goalScorers, !scorers.isEmpty {
                HStack(alignment: .top, spacing: 8) {
                    scorerLine(scorers.filter { $0.team == game.away.abbr }, align: .leading)
                    Spacer(minLength: 0)
                    scorerLine(scorers.filter { $0.team == game.home.abbr }, align: .trailing)
                }
            }

            // Cards indicator
            let yellows = game.yellowCards ?? 0
            let reds = game.redCards ?? 0
            if yellows > 0 || reds > 0 {
                HStack(spacing: 6) {
                    if yellows > 0 { Text("🟨×\(yellows)").font(.system(size: 10)) }
                    if reds > 0   { Text("🟥×\(reds)").font(.system(size: 10)) }
                    Spacer()
                }
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
    }

    @ViewBuilder
    private func scorerLine(_ scorers: [GoalScorer], align: TextAlignment) -> some View {
        if !scorers.isEmpty {
            Text(
                scorers.map { s in
                    let lastName = s.player.components(separatedBy: " ").last ?? s.player
                    let assistStr = s.assist.map { " (\($0.components(separatedBy: " ").last ?? $0))" } ?? ""
                    return "\(s.minute)' \(lastName)\(assistStr)"
                }.joined(separator: " · ")
            )
            .font(.system(size: 9, weight: .medium))
            .foregroundStyle(.white.opacity(0.65))
            .multilineTextAlignment(align)
            .lineLimit(2)
        }
    }
}

private struct SoccerMatchupHeader: View {
    let game: GameData

    var body: some View {
        HStack(spacing: 6) {
            RoundedRectangle(cornerRadius: 1)
                .fill(game.leagueAccentColor)
                .frame(width: 3, height: 12)
            if let shortName = game.leagueShortName {
                Text(shortName)
                    .font(.system(size: 9, weight: .black))
                    .tracking(1.5)
                    .foregroundStyle(game.leagueAccentColor)
            }
            if let matchday = game.matchday {
                Text("·")
                    .foregroundStyle(.white.opacity(0.3))
                Text(matchday.uppercased())
                    .font(.system(size: 9, weight: .bold))
                    .tracking(1)
                    .foregroundStyle(.white.opacity(0.5))
            }
            Spacer()
        }
    }
}

// MARK: - Shared Components

private struct StatusPill: View {
    let status: String
    let color: Color
    let isLive: Bool

    var body: some View {
        HStack(spacing: 4) {
            if isLive {
                Circle()
                    .fill(color)
                    .frame(width: 6, height: 6)
                    .modifier(PulseModifier())
            }
            Text(status.uppercased())
                .font(.system(size: 10, weight: .heavy))
                .tracking(0.5)
        }
        .foregroundStyle(isLive ? .white : color)
        .padding(.horizontal, 10)
        .padding(.vertical, 5)
        .background(
            Capsule()
                .fill(color.opacity(isLive ? 0.9 : 0.15))
        )
        .shadow(color: isLive ? color.opacity(0.5) : .clear, radius: 8)
    }
}

private struct PulseModifier: ViewModifier {
    @State private var pulse = false

    func body(content: Content) -> some View {
        content
            .scaleEffect(pulse ? 1.5 : 1.0)
            .opacity(pulse ? 0.5 : 1.0)
            .animation(.easeInOut(duration: 0.8).repeatForever(autoreverses: true), value: pulse)
            .onAppear { pulse = true }
    }
}

private struct SportsBookmarkButton: View {
    let post: Post
    let darkMode: Bool
    @AppStorage var isBookmarked: Bool
    @EnvironmentObject private var apiService: APIService

    init(post: Post, darkMode: Bool = false) {
        self.post = post
        self.darkMode = darkMode
        self._isBookmarked = AppStorage(wrappedValue: false, "bookmark_\(post.id)")
    }

    var body: some View {
        Button {
            UIImpactFeedbackGenerator(style: .light).impactOccurred()
            let wasSaved = isBookmarked
            isBookmarked.toggle()
            Task {
                do {
                    try await apiService.trackEvent(
                        postID: post.id,
                        eventType: wasSaved ? "unsave" : "save"
                    )
                } catch {
                    isBookmarked = wasSaved
                }
            }
        } label: {
            Image(systemName: isBookmarked ? "bookmark.fill" : "bookmark")
                .font(.caption)
                .foregroundColor(isBookmarked ? .orange : (darkMode ? .white.opacity(0.4) : .secondary))
                .contentTransition(.symbolEffect(.replace))
        }
        .buttonStyle(.plain)
    }
}

// MARK: - Sport Icon Animation

private struct SportIconAnimation: ViewModifier {
    let reduceMotion: Bool

    func body(content: Content) -> some View {
        if reduceMotion {
            content
        } else {
            content.symbolEffect(.breathe, isActive: true)
        }
    }
}

// MARK: - Diagonal Shapes

/// Clips to a diagonal triangle (top-right to bottom-left).
private struct DiagonalShape: Shape {
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

/// Draws a diagonal divider line.
private struct DiagonalLine: Shape {
    func path(in rect: CGRect) -> Path {
        Path { p in
            p.move(to: CGPoint(x: rect.maxX * 0.35, y: 0))
            p.addLine(to: CGPoint(x: rect.maxX * 0.65, y: rect.maxY))
        }
    }
}

/// Subtle animated shimmer along the diagonal.
private struct ShimmerLine: View {
    @State private var phase: CGFloat = -1

    var body: some View {
        GeometryReader { geo in
            DiagonalLine()
                .stroke(
                    LinearGradient(
                        stops: [
                            .init(color: .clear, location: max(0, phase - 0.1)),
                            .init(color: .white.opacity(0.25), location: phase),
                            .init(color: .clear, location: min(1, phase + 0.1)),
                        ],
                        startPoint: .top,
                        endPoint: .bottom
                    ),
                    lineWidth: 2
                )
                .onAppear {
                    withAnimation(.easeInOut(duration: 2.5).repeatForever(autoreverses: false)) {
                        phase = 2
                    }
                }
        }
        .allowsHitTesting(false)
    }
}

/// Canvas-based shimmer for live scoreboard games.
private struct LiveShimmerOverlay: View {
    var body: some View {
        TimelineView(.animation(minimumInterval: 0.05)) { timeline in
            Canvas { context, size in
                let t = timeline.date.timeIntervalSinceReferenceDate
                // Subtle horizontal scan line
                let y = (t * 40).truncatingRemainder(dividingBy: Double(size.height))
                context.opacity = 0.04
                context.fill(
                    Path(CGRect(x: 0, y: y, width: size.width, height: 2)),
                    with: .color(.white)
                )
                // Faint edge glow
                let pulse = (sin(t * 3) + 1) / 2 * 0.06
                context.opacity = pulse
                context.fill(
                    Path(CGRect(x: 0, y: 0, width: size.width, height: size.height)),
                    with: .linearGradient(
                        Gradient(colors: [.red.opacity(0.3), .clear, .clear, .red.opacity(0.3)]),
                        startPoint: .zero,
                        endPoint: CGPoint(x: size.width, y: 0)
                    )
                )
            }
        }
        .allowsHitTesting(false)
    }
}

// MARK: - Box Score Card

struct BoxScoreCard: View {
    let post: Post
    let game: BaseballData
    @State private var activeReaction: String?
    @Environment(\.accessibilityReduceMotion) private var reduceMotion

    init?(post: Post) {
        guard let bd = post.baseballData else { return nil }
        self.post = post
        self.game = bd
        self._activeReaction = State(initialValue: post.myReaction)
    }

    private var homeWins: Bool {
        guard let hs = game.home.score, let aws = game.away.score else { return false }
        return hs > aws
    }

    private var awayWins: Bool {
        guard let hs = game.home.score, let aws = game.away.score else { return false }
        return aws > hs
    }

    var body: some View {
        ZStack(alignment: .top) {
            // Background: dark baseball-green
            Color(red: 0.039, green: 0.086, blue: 0.157)

            // Sport icon watermark
            Image(systemName: "figure.baseball")
                .font(.system(size: 120, weight: .ultraLight))
                .foregroundStyle(.white.opacity(0.05))
                .offset(x: 0, y: 30)

            // Team gradient strip at top
            LinearGradient(
                stops: [
                    .init(color: game.away.swiftUIColor.opacity(0.7), location: 0),
                    .init(color: Color.black.opacity(0.5), location: 0.45),
                    .init(color: Color.black.opacity(0.5), location: 0.55),
                    .init(color: game.home.swiftUIColor.opacity(0.7), location: 1),
                ],
                startPoint: .leading,
                endPoint: .trailing
            )
            .frame(height: 4)
            .frame(maxHeight: .infinity, alignment: .top)

            VStack(spacing: 0) {
                // League + status header
                HStack {
                    HStack(spacing: 4) {
                        Image(systemName: "figure.baseball")
                            .font(.caption2)
                        Text(game.league)
                            .font(.caption.weight(.bold))
                    }
                    .foregroundStyle(.white.opacity(0.5))
                    Spacer()
                    Text(game.status.uppercased())
                        .font(.system(size: 10, weight: .heavy))
                        .tracking(0.5)
                        .foregroundStyle(.green)
                        .padding(.horizontal, 10)
                        .padding(.vertical, 5)
                        .background(Capsule().fill(Color.green.opacity(0.15)))
                }
                .padding(.top, 14)
                .padding(.horizontal, 16)

                Spacer(minLength: 8)

                // Score row
                HStack(spacing: 0) {
                    // Away team
                    VStack(spacing: 4) {
                        Text(game.away.abbr)
                            .font(.system(size: 18, weight: .heavy, design: .rounded))
                            .foregroundStyle(.white)
                            .padding(.horizontal, 10)
                            .padding(.vertical, 5)
                            .background(
                                RoundedRectangle(cornerRadius: 7)
                                    .fill(game.away.swiftUIColor)
                                    .shadow(color: game.away.swiftUIColor.opacity(0.5), radius: awayWins ? 8 : 0)
                            )
                        if let record = game.away.record {
                            Text(record)
                                .font(.system(size: 10, weight: .medium))
                                .foregroundStyle(.white.opacity(0.4))
                        }
                    }
                    .frame(maxWidth: .infinity)

                    // Scores
                    if let awayScore = game.away.score, let homeScore = game.home.score {
                        HStack(spacing: 10) {
                            Text("\(awayScore)")
                                .font(.system(size: 48, weight: .thin, design: .rounded))
                                .foregroundStyle(.white.opacity(homeWins ? 0.4 : 1.0))
                                .shadow(color: awayWins ? .white.opacity(0.35) : .clear, radius: 10)
                            Text("–")
                                .font(.system(size: 22, weight: .ultraLight))
                                .foregroundStyle(.white.opacity(0.3))
                            Text("\(homeScore)")
                                .font(.system(size: 48, weight: .thin, design: .rounded))
                                .foregroundStyle(.white.opacity(awayWins ? 0.4 : 1.0))
                                .shadow(color: homeWins ? .white.opacity(0.35) : .clear, radius: 10)
                        }
                    }

                    // Home team
                    VStack(spacing: 4) {
                        Text(game.home.abbr)
                            .font(.system(size: 18, weight: .heavy, design: .rounded))
                            .foregroundStyle(.white)
                            .padding(.horizontal, 10)
                            .padding(.vertical, 5)
                            .background(
                                RoundedRectangle(cornerRadius: 7)
                                    .fill(game.home.swiftUIColor)
                                    .shadow(color: game.home.swiftUIColor.opacity(0.5), radius: homeWins ? 8 : 0)
                            )
                        if let record = game.home.record {
                            Text(record)
                                .font(.system(size: 10, weight: .medium))
                                .foregroundStyle(.white.opacity(0.4))
                        }
                    }
                    .frame(maxWidth: .infinity)
                }
                .padding(.horizontal, 16)

                Spacer(minLength: 10)

                // Divider
                Rectangle()
                    .fill(Color.white.opacity(0.08))
                    .frame(height: 1)
                    .padding(.horizontal, 16)

                Spacer(minLength: 8)

                // Pitching section
                VStack(alignment: .leading, spacing: 4) {
                    Text("⚾  PITCHING")
                        .font(.system(size: 9, weight: .heavy))
                        .tracking(1)
                        .foregroundStyle(.white.opacity(0.35))
                        .padding(.bottom, 2)

                    if let wp = game.winningPitcher {
                        Text("W: \(wp.name) (\(wp.record), \(wp.era) ERA) · \(wp.formattedIP) IP · \(wp.strikeouts)K")
                            .font(.system(size: 12, weight: .medium, design: .monospaced))
                            .foregroundStyle(.white.opacity(0.85))
                            .lineLimit(1)
                            .minimumScaleFactor(0.75)
                    }
                    if let lp = game.losingPitcher {
                        Text("L: \(lp.name) (\(lp.record), \(lp.era) ERA) · \(lp.formattedIP) IP · \(lp.strikeouts)K")
                            .font(.system(size: 12, weight: .medium, design: .monospaced))
                            .foregroundStyle(.white.opacity(0.5))
                            .lineLimit(1)
                            .minimumScaleFactor(0.75)
                    }
                    if let sv = game.savePitcher {
                        Text("SV: \(sv.name) (\(sv.saves))")
                            .font(.system(size: 12, weight: .medium, design: .monospaced))
                            .foregroundStyle(Color(red: 0.878, green: 0.133, blue: 0.243).opacity(0.9))
                            .lineLimit(1)
                    }
                }
                .frame(maxWidth: .infinity, alignment: .leading)
                .padding(.horizontal, 16)

                // Key batter section (conditional)
                if let batter = game.keyBatter, !batter.summaryText.isEmpty {
                    Spacer(minLength: 6)
                    Rectangle()
                        .fill(Color.white.opacity(0.06))
                        .frame(height: 1)
                        .padding(.horizontal, 16)
                    Spacer(minLength: 6)

                    VStack(alignment: .leading, spacing: 4) {
                        Text("🏆  KEY PERFORMANCE")
                            .font(.system(size: 9, weight: .heavy))
                            .tracking(1)
                            .foregroundStyle(.white.opacity(0.35))
                        Text("\(batter.name) (\(batter.team)): \(batter.summaryText)")
                            .font(.system(size: 12, weight: .semibold))
                            .foregroundStyle(.white.opacity(0.85))
                            .lineLimit(1)
                            .minimumScaleFactor(0.8)
                    }
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .padding(.horizontal, 16)
                }

                // Headline strip
                if let headline = game.headline, !headline.isEmpty {
                    Spacer(minLength: 6)
                    Text("\u{201C}\(headline)\u{201D}")
                        .font(.system(size: 11, weight: .regular).italic())
                        .foregroundStyle(.white.opacity(0.55))
                        .lineLimit(2)
                        .multilineTextAlignment(.center)
                        .padding(.horizontal, 16)
                }

                Spacer(minLength: 10)

                // Footer: venue + reactions
                HStack(spacing: 12) {
                    if let venue = game.venue {
                        Label(venue, systemImage: "mappin")
                            .font(.caption2)
                            .foregroundStyle(.white.opacity(0.35))
                            .lineLimit(1)
                    }
                    Spacer()
                    ReactionPicker(
                        activeReaction: $activeReaction,
                        postID: post.id,
                        style: .feedDark
                    )
                    SportsBookmarkButton(post: post, darkMode: true)
                }
                .padding(.horizontal, 16)
                .padding(.bottom, 14)
            }
        }
        .frame(height: 280)
    }
}
