import SwiftUI

// MARK: - Scoreboard Card

struct ScoreboardCard: View {
    let post: Post
    let game: GameData

    init?(post: Post) {
        guard let gd = post.gameData else { return nil }
        self.post = post
        self.game = gd
    }

    private var homeWins: Bool {
        guard let hs = game.home.score, let as_ = game.away.score else { return false }
        return hs > as_
    }

    var body: some View {
        VStack(spacing: 0) {
            // Hero: team-color gradient with scores
            ZStack {
                LinearGradient(
                    colors: [game.home.swiftUIColor, .black.opacity(0.85)],
                    startPoint: .topLeading,
                    endPoint: .bottomTrailing
                )

                // Large decorative sport icon
                Image(systemName: game.sportIcon)
                    .font(.system(size: 100, weight: .thin))
                    .foregroundStyle(.white.opacity(0.06))
                    .offset(x: 120, y: -10)

                VStack(spacing: 12) {
                    // League + status header
                    HStack {
                        if let league = game.league {
                            Text(league)
                                .font(.caption.weight(.semibold))
                                .foregroundStyle(.white.opacity(0.6))
                        }
                        Spacer()
                        StatusPill(status: game.status, color: game.statusColor, isLive: game.isLive)
                    }

                    Spacer()

                    // Score display
                    HStack(spacing: 0) {
                        // Away team
                        VStack(spacing: 4) {
                            Text(game.away.abbr)
                                .font(.system(size: 22, weight: .bold, design: .rounded))
                                .foregroundStyle(.white)
                            if let record = game.away.record {
                                Text(record)
                                    .font(.caption2)
                                    .foregroundStyle(.white.opacity(0.5))
                            }
                        }
                        .frame(maxWidth: .infinity)

                        // Scores
                        if let awayScore = game.away.score, let homeScore = game.home.score {
                            HStack(spacing: 12) {
                                Text("\(awayScore)")
                                    .font(.system(size: 48, weight: .thin, design: .rounded))
                                    .foregroundStyle(.white.opacity(homeWins ? 0.5 : 1.0))
                                Text("—")
                                    .font(.system(size: 28, weight: .thin))
                                    .foregroundStyle(.white.opacity(0.4))
                                Text("\(homeScore)")
                                    .font(.system(size: 48, weight: .thin, design: .rounded))
                                    .foregroundStyle(.white.opacity(homeWins ? 1.0 : 0.5))
                            }
                        } else {
                            Text("vs")
                                .font(.title2.weight(.light))
                                .foregroundStyle(.white.opacity(0.5))
                        }

                        // Home team
                        VStack(spacing: 4) {
                            Text(game.home.abbr)
                                .font(.system(size: 22, weight: .bold, design: .rounded))
                                .foregroundStyle(.white)
                            if let record = game.home.record {
                                Text(record)
                                    .font(.caption2)
                                    .foregroundStyle(.white.opacity(0.5))
                            }
                        }
                        .frame(maxWidth: .infinity)
                    }

                    Spacer()

                    // Headline stat line
                    if let headline = game.headline, !headline.isEmpty {
                        Text(headline)
                            .font(.caption.weight(.medium))
                            .foregroundStyle(.white.opacity(0.7))
                            .lineLimit(2)
                            .multilineTextAlignment(.center)
                    }
                }
                .padding(16)
            }
            .frame(height: 200)

            // Footer with venue + broadcast
            HStack(spacing: 8) {
                if let venue = game.venue {
                    Label(venue, systemImage: "mappin")
                        .font(.caption2)
                        .foregroundStyle(.secondary)
                        .lineLimit(1)
                }
                Spacer()
                if let broadcast = game.broadcast {
                    Label(broadcast, systemImage: "tv")
                        .font(.caption2)
                        .foregroundStyle(.secondary)
                }
                SportsBookmarkButton(post: post)
            }
            .padding(.horizontal, 16)
            .padding(.vertical, 10)
            .background(Color(.secondarySystemGroupedBackground))
        }
    }
}

// MARK: - Matchup Card

struct MatchupCard: View {
    let post: Post
    let game: GameData

    init?(post: Post) {
        guard let gd = post.gameData else { return nil }
        self.post = post
        self.game = gd
    }

    var body: some View {
        VStack(spacing: 0) {
            // Split gradient hero
            ZStack {
                // Diagonal split: away (left) / home (right)
                GeometryReader { geo in
                    ZStack {
                        game.away.swiftUIColor
                        LinearGradient(
                            stops: [
                                .init(color: game.away.swiftUIColor, location: 0),
                                .init(color: game.away.swiftUIColor, location: 0.45),
                                .init(color: game.home.swiftUIColor, location: 0.55),
                                .init(color: game.home.swiftUIColor, location: 1),
                            ],
                            startPoint: .leading,
                            endPoint: .trailing
                        )
                        // Dark overlay for readability
                        Color.black.opacity(0.3)
                    }
                }

                VStack(spacing: 14) {
                    // League + sport badge
                    HStack {
                        if let league = game.league {
                            HStack(spacing: 4) {
                                Image(systemName: game.sportIcon)
                                    .font(.caption2)
                                Text(league)
                                    .font(.caption.weight(.semibold))
                            }
                            .foregroundStyle(.white.opacity(0.7))
                        }
                        Spacer()
                        if let series = game.series {
                            Text(series)
                                .font(.caption2.weight(.medium))
                                .foregroundStyle(.white.opacity(0.7))
                                .padding(.horizontal, 8)
                                .padding(.vertical, 3)
                                .background(.white.opacity(0.15))
                                .cornerRadius(4)
                        }
                    }

                    Spacer()

                    // Teams + VS
                    HStack(spacing: 0) {
                        VStack(spacing: 6) {
                            Text(game.away.abbr)
                                .font(.system(size: 28, weight: .bold, design: .rounded))
                                .foregroundStyle(.white)
                            Text(game.away.name)
                                .font(.caption.weight(.medium))
                                .foregroundStyle(.white.opacity(0.7))
                            if let record = game.away.record {
                                Text(record)
                                    .font(.caption2)
                                    .foregroundStyle(.white.opacity(0.5))
                            }
                        }
                        .frame(maxWidth: .infinity)

                        VStack(spacing: 2) {
                            Text("VS")
                                .font(.system(size: 14, weight: .heavy, design: .rounded))
                                .foregroundStyle(.white.opacity(0.4))
                        }
                        .frame(width: 40)

                        VStack(spacing: 6) {
                            Text(game.home.abbr)
                                .font(.system(size: 28, weight: .bold, design: .rounded))
                                .foregroundStyle(.white)
                            Text(game.home.name)
                                .font(.caption.weight(.medium))
                                .foregroundStyle(.white.opacity(0.7))
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
                    VStack(spacing: 4) {
                        if let time = game.formattedGameTime {
                            Text(time)
                                .font(.system(size: 24, weight: .bold, design: .rounded))
                                .foregroundStyle(.white)
                        }
                        if let date = game.formattedGameDate {
                            Text(date)
                                .font(.caption.weight(.medium))
                                .foregroundStyle(.white.opacity(0.6))
                        }
                    }
                }
                .padding(16)
            }
            .frame(height: 220)

            // Footer
            HStack(spacing: 8) {
                if let venue = game.venue {
                    Label(venue, systemImage: "mappin")
                        .font(.caption2)
                        .foregroundStyle(.secondary)
                        .lineLimit(1)
                }
                Spacer()
                if let broadcast = game.broadcast {
                    Label(broadcast, systemImage: "tv")
                        .font(.caption2)
                        .foregroundStyle(.secondary)
                }
                SportsBookmarkButton(post: post)
            }
            .padding(.horizontal, 16)
            .padding(.vertical, 10)
            .background(Color(.secondarySystemGroupedBackground))
        }
    }
}

// MARK: - Standings Card

struct StandingsCard: View {
    let post: Post
    let standings: StandingsData

    init?(post: Post) {
        guard let sd = post.standingsData else { return nil }
        self.post = post
        self.standings = sd
    }

    private var accentColor: Color {
        guard let hex = standings.leagueColor else { return .gray }
        return Color(hexString: hex)
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            // Accent stripe
            Rectangle()
                .fill(accentColor)
                .frame(height: 4)

            VStack(alignment: .leading, spacing: 12) {
                // League + date header
                HStack {
                    Text(standings.league)
                        .font(.headline.weight(.bold))
                    Spacer()
                    Text(formattedDate)
                        .font(.caption.weight(.medium))
                        .foregroundStyle(.secondary)
                    SportsBookmarkButton(post: post)
                }

                // Game rows
                VStack(spacing: 0) {
                    ForEach(standings.games) { game in
                        gameRow(game)
                        if game.id != standings.games.last?.id {
                            Divider()
                                .padding(.vertical, 2)
                        }
                    }
                }

                // Headline
                if let headline = standings.headline, !headline.isEmpty {
                    Text(headline)
                        .font(.caption.weight(.semibold))
                        .foregroundStyle(.primary)
                        .padding(.top, 4)
                }
            }
            .padding(16)
            .background(Color(.secondarySystemGroupedBackground))
        }
    }

    @ViewBuilder
    private func gameRow(_ game: StandingsGame) -> some View {
        HStack(spacing: 6) {
            // Away team
            Circle()
                .fill(game.awaySwiftUIColor)
                .frame(width: 8, height: 8)
            Text(game.away)
                .font(.subheadline.weight(.semibold))
                .frame(width: 36, alignment: .leading)
            Text("\(game.awayScore)")
                .font(.subheadline.weight(game.awayScore > game.homeScore ? .bold : .regular))
                .foregroundStyle(game.awayScore > game.homeScore ? .primary : .secondary)
                .frame(width: 20, alignment: .trailing)

            Text("@")
                .font(.caption2)
                .foregroundStyle(.tertiary)
                .frame(width: 16)

            // Home team
            Circle()
                .fill(game.homeSwiftUIColor)
                .frame(width: 8, height: 8)
            Text(game.home)
                .font(.subheadline.weight(.semibold))
                .frame(width: 36, alignment: .leading)
            Text("\(game.homeScore)")
                .font(.subheadline.weight(game.homeScore > game.awayScore ? .bold : .regular))
                .foregroundStyle(game.homeScore > game.awayScore ? .primary : .secondary)
                .frame(width: 20, alignment: .trailing)

            Spacer()

            Text(game.status)
                .font(.caption2.weight(.medium))
                .foregroundStyle(.secondary)
        }
        .padding(.vertical, 6)
    }

    private var formattedDate: String {
        let f = DateFormatter()
        f.dateFormat = "yyyy-MM-dd"
        guard let date = f.date(from: standings.date) else { return standings.date }
        f.dateFormat = "EEEE, MMM d"
        return f.string(from: date)
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
                .font(.caption2.weight(.heavy))
        }
        .foregroundStyle(color)
        .padding(.horizontal, 8)
        .padding(.vertical, 4)
        .background(color.opacity(0.15))
        .cornerRadius(6)
    }
}

private struct PulseModifier: ViewModifier {
    @State private var pulse = false

    func body(content: Content) -> some View {
        content
            .scaleEffect(pulse ? 1.4 : 1.0)
            .opacity(pulse ? 0.6 : 1.0)
            .animation(.easeInOut(duration: 0.8).repeatForever(autoreverses: true), value: pulse)
            .onAppear { pulse = true }
    }
}

private struct SportsBookmarkButton: View {
    let post: Post
    @AppStorage var isBookmarked: Bool

    init(post: Post) {
        self.post = post
        self._isBookmarked = AppStorage(wrappedValue: false, "bookmark_\(post.id)")
    }

    var body: some View {
        Button {
            UIImpactFeedbackGenerator(style: .light).impactOccurred()
            isBookmarked.toggle()
        } label: {
            Image(systemName: isBookmarked ? "bookmark.fill" : "bookmark")
                .font(.caption)
                .foregroundColor(isBookmarked ? .orange : .secondary)
                .contentTransition(.symbolEffect(.replace))
        }
        .buttonStyle(.plain)
    }
}
