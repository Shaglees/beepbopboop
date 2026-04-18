import SwiftUI

struct PlayerSpotlightDetailView: View {
    let post: Post
    @Environment(\.dismiss) private var dismiss

    private var data: PlayerData? { post.playerData }

    private var accentColor: Color {
        data?.teamSwiftUIColor ?? Color(red: 0.0, green: 0.478, blue: 0.757)
    }

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 0) {

                // MARK: - Hero gradient with headshot
                heroSection

                // MARK: - Content
                VStack(alignment: .leading, spacing: 20) {

                    // Game result banner
                    if let data = data, let result = data.gameResult {
                        gameResultBanner(result: result, opponent: data.opponent, gameDate: data.gameDate)
                    }

                    // Last game stats — the big card
                    if let data = data {
                        lastGameStatsCard(data: data)
                    }

                    // Shooting efficiency
                    if let data = data {
                        let fg = data.lastGameStats.fieldGoalPct
                        let tp = data.lastGameStats.threePointPct
                        if fg != nil || tp != nil {
                            shootingEfficiencyRow(fg: fg, tp: tp)
                        }
                    }

                    // Season averages
                    if let data = data {
                        seasonAveragesCard(data: data)
                    }

                    // Series context pill
                    if let series = data?.seriesContext, !series.isEmpty {
                        seriesContextBadge(series)
                    }

                    // Storyline / body
                    let narrativeText = data?.storyline ?? (post.body.isEmpty ? nil : post.body)
                    if let text = narrativeText, !text.isEmpty {
                        Text(text)
                            .font(.body)
                            .foregroundStyle(.primary)
                            .lineSpacing(4)
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
                        .foregroundStyle(.white.opacity(0.8))
                        .shadow(color: .black.opacity(0.3), radius: 4)
                }
            }
        }
    }

    // MARK: - Hero

    @ViewBuilder
    private var heroSection: some View {
        let heroHeight: CGFloat = 260

        ZStack(alignment: .bottomLeading) {
            // Background gradient
            LinearGradient(
                colors: [accentColor, accentColor.opacity(0.6), Color.black.opacity(0.85)],
                startPoint: .topLeading,
                endPoint: .bottomTrailing
            )
            .frame(height: heroHeight)

            // Faded sport icon watermark
            if let data = data {
                Image(systemName: data.sportIcon)
                    .font(.system(size: 200, weight: .ultraLight))
                    .foregroundStyle(.white.opacity(0.06))
                    .frame(maxWidth: .infinity, maxHeight: heroHeight, alignment: .trailing)
                    .clipped()
            }

            // Player headshot
            if let data = data, let headshotStr = data.playerHeadshotUrl,
               !headshotStr.isEmpty, let url = URL(string: headshotStr) {
                HStack {
                    Spacer()
                    AsyncImage(url: url) { phase in
                        switch phase {
                        case .success(let img):
                            img.resizable()
                                .aspectRatio(contentMode: .fill)
                                .frame(width: 160, height: heroHeight)
                                .clipped()
                                .mask(
                                    LinearGradient(
                                        colors: [.clear, .black.opacity(0.8), .black],
                                        startPoint: .leading,
                                        endPoint: .trailing
                                    )
                                )
                                .overlay(alignment: .bottom) {
                                    LinearGradient(
                                        colors: [.clear, .black.opacity(0.5)],
                                        startPoint: .center,
                                        endPoint: .bottom
                                    )
                                }
                        case .failure:
                            // Fallback circle avatar
                            Circle()
                                .fill(.white.opacity(0.1))
                                .frame(width: 100, height: 100)
                                .overlay(
                                    Image(systemName: "person.fill")
                                        .font(.system(size: 44))
                                        .foregroundStyle(.white.opacity(0.4))
                                )
                                .padding(.trailing, 20)
                        default:
                            ProgressView()
                                .tint(.white)
                                .frame(width: 160, height: heroHeight)
                        }
                    }
                }
            }

            // Player info overlay — bottom-left
            if let data = data {
                VStack(alignment: .leading, spacing: 4) {
                    // League + sport icon
                    HStack(spacing: 6) {
                        Image(systemName: data.sportIcon)
                            .font(.caption2.weight(.semibold))
                            .foregroundStyle(.white.opacity(0.7))
                        Text(data.league)
                            .font(.caption.weight(.bold))
                            .tracking(0.5)
                            .foregroundStyle(.white.opacity(0.7))
                    }

                    Text(data.playerName)
                        .font(.system(size: 28, weight: .heavy, design: .rounded))
                        .foregroundStyle(.white)
                        .shadow(color: .black.opacity(0.4), radius: 4)

                    HStack(spacing: 6) {
                        Text(data.team)
                            .font(.subheadline.weight(.semibold))
                            .foregroundStyle(.white.opacity(0.85))
                        if let position = data.position, !position.isEmpty {
                            Text("·")
                                .foregroundStyle(.white.opacity(0.5))
                            Text(position)
                                .font(.subheadline.weight(.medium))
                                .foregroundStyle(.white.opacity(0.75))
                        }
                    }
                }
                .padding(.horizontal, 20)
                .padding(.bottom, 20)
            }
        }
        .frame(height: heroHeight)
    }

    // MARK: - Game result banner

    @ViewBuilder
    private func gameResultBanner(result: String, opponent: String?, gameDate: String?) -> some View {
        let isWin = result.uppercased().hasPrefix("W")
        let isLoss = result.uppercased().hasPrefix("L")
        let resultColor: Color = isWin ? .green : (isLoss ? .red : .orange)

        HStack(spacing: 12) {
            // W/L badge
            Text(isWin ? "W" : (isLoss ? "L" : "–"))
                .font(.system(size: 18, weight: .black, design: .rounded))
                .foregroundStyle(.white)
                .frame(width: 36, height: 36)
                .background(resultColor, in: RoundedRectangle(cornerRadius: 8))

            VStack(alignment: .leading, spacing: 2) {
                Text(result)
                    .font(.subheadline.weight(.bold))
                    .foregroundStyle(.primary)
                HStack(spacing: 6) {
                    if let opponent = opponent, !opponent.isEmpty {
                        Text("vs \(opponent)")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }
                    if let dateStr = gameDate, !dateStr.isEmpty {
                        if opponent != nil { Text("·").font(.caption).foregroundStyle(.secondary) }
                        Text(formattedGameDate(dateStr))
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }
                }
            }

            Spacer()
        }
        .padding(.horizontal, 14)
        .padding(.vertical, 12)
        .background(resultColor.opacity(0.08), in: RoundedRectangle(cornerRadius: 12))
        .overlay(
            RoundedRectangle(cornerRadius: 12)
                .stroke(resultColor.opacity(0.2), lineWidth: 1)
        )
    }

    // MARK: - Last game stats card

    @ViewBuilder
    private func lastGameStatsCard(data: PlayerData) -> some View {
        let stats = data.lastGameStats

        VStack(alignment: .leading, spacing: 12) {
            Text("LAST GAME")
                .font(.system(size: 11, weight: .bold))
                .tracking(1.5)
                .foregroundStyle(accentColor)

            // Primary stats: PTS, REB, AST
            HStack(spacing: 0) {
                ForEach(
                    [(stats.points, "PTS"), (stats.rebounds, "REB"), (stats.assists, "AST")],
                    id: \.1
                ) { value, label in
                    VStack(spacing: 4) {
                        Text("\(value)")
                            .font(.system(size: 48, weight: .bold, design: .rounded))
                            .foregroundStyle(accentColor)
                        Text(label)
                            .font(.caption.weight(.semibold))
                            .foregroundStyle(.secondary)
                    }
                    .frame(maxWidth: .infinity)
                }
            }
            .padding(.vertical, 20)

            // Secondary stats: STL, BLK, +/-
            let hasSecondary = stats.steals != nil || stats.blocks != nil || stats.plusMinus != nil
            if hasSecondary {
                Divider()
                HStack(spacing: 0) {
                    if let steals = stats.steals {
                        secondaryStat(value: "\(steals)", label: "STL")
                    }
                    if let blocks = stats.blocks {
                        secondaryStat(value: "\(blocks)", label: "BLK")
                    }
                    if let plusMinus = stats.plusMinus {
                        let prefix = plusMinus >= 0 ? "+" : ""
                        secondaryStat(value: "\(prefix)\(plusMinus)", label: "+/-",
                                      color: plusMinus > 0 ? .green : (plusMinus < 0 ? .red : .secondary))
                    }
                }
                .padding(.bottom, 8)
            }
        }
        .padding(.horizontal, 16)
        .padding(.top, 16)
        .padding(.bottom, hasSecondaryStats(data) ? 4 : 16)
        .background(Color(.secondarySystemGroupedBackground), in: RoundedRectangle(cornerRadius: 16))
    }

    private func hasSecondaryStats(_ data: PlayerData) -> Bool {
        let s = data.lastGameStats
        return s.steals != nil || s.blocks != nil || s.plusMinus != nil
    }

    @ViewBuilder
    private func secondaryStat(value: String, label: String, color: Color = .primary) -> some View {
        VStack(spacing: 2) {
            Text(value)
                .font(.system(size: 22, weight: .bold, design: .rounded))
                .foregroundStyle(color)
            Text(label)
                .font(.caption2.weight(.semibold))
                .foregroundStyle(.secondary)
        }
        .frame(maxWidth: .infinity)
        .padding(.vertical, 8)
    }

    // MARK: - Shooting efficiency

    @ViewBuilder
    private func shootingEfficiencyRow(fg: Double?, tp: Double?) -> some View {
        HStack(spacing: 16) {
            if let fg = fg {
                shootingBadge(value: fg, label: "FG%")
            }
            if let tp = tp {
                shootingBadge(value: tp, label: "3P%")
            }
            Spacer()
        }
    }

    @ViewBuilder
    private func shootingBadge(value: Double, label: String) -> some View {
        let pct = Int((value * 100).rounded())
        let isGood = (label == "FG%" && value >= 0.45) || (label == "3P%" && value >= 0.35)
        HStack(spacing: 6) {
            Text(label)
                .font(.caption.weight(.semibold))
                .foregroundStyle(.secondary)
            Text("\(pct)%")
                .font(.subheadline.weight(.bold))
                .foregroundStyle(isGood ? .green : .primary)
        }
        .padding(.horizontal, 12)
        .padding(.vertical, 7)
        .background(Color(.secondarySystemGroupedBackground), in: RoundedRectangle(cornerRadius: 8))
    }

    // MARK: - Season averages card

    @ViewBuilder
    private func seasonAveragesCard(data: PlayerData) -> some View {
        let avgs = data.seasonAverages

        VStack(alignment: .leading, spacing: 10) {
            Text("SEASON AVERAGES")
                .font(.system(size: 11, weight: .bold))
                .tracking(1.5)
                .foregroundStyle(.secondary)

            HStack(spacing: 0) {
                ForEach(
                    [(avgs.points, "PPG"), (avgs.rebounds, "RPG"), (avgs.assists, "APG")],
                    id: \.1
                ) { value, label in
                    VStack(spacing: 3) {
                        Text(String(format: value.truncatingRemainder(dividingBy: 1) == 0 ? "%.0f" : "%.1f", value))
                            .font(.system(size: 22, weight: .bold, design: .rounded))
                            .foregroundStyle(.primary)
                        Text(label)
                            .font(.caption2.weight(.semibold))
                            .foregroundStyle(.secondary)
                    }
                    .frame(maxWidth: .infinity)
                }
            }
            .padding(.vertical, 14)
        }
        .padding(.horizontal, 16)
        .padding(.top, 14)
        .background(Color(.secondarySystemGroupedBackground), in: RoundedRectangle(cornerRadius: 16))
    }

    // MARK: - Series context badge

    @ViewBuilder
    private func seriesContextBadge(_ text: String) -> some View {
        HStack(spacing: 6) {
            Image(systemName: "trophy.fill")
                .font(.caption2)
                .foregroundStyle(accentColor)
            Text(text)
                .font(.caption.weight(.semibold))
                .foregroundStyle(accentColor)
        }
        .padding(.horizontal, 14)
        .padding(.vertical, 8)
        .background(accentColor.opacity(0.1), in: Capsule())
        .overlay(Capsule().stroke(accentColor.opacity(0.25), lineWidth: 1))
    }

    // MARK: - Helpers

    private func formattedGameDate(_ dateStr: String) -> String {
        let f = DateFormatter()
        f.dateFormat = "yyyy-MM-dd"
        f.timeZone = TimeZone(identifier: "UTC")
        guard let date = f.date(from: dateStr) else { return dateStr }
        var utcCal = Calendar.current
        utcCal.timeZone = TimeZone(identifier: "UTC")!
        let today = Date()
        if utcCal.isDate(date, inSameDayAs: today) { return "Today" }
        if let yesterday = utcCal.date(byAdding: .day, value: -1, to: today),
           utcCal.isDate(date, inSameDayAs: yesterday) { return "Yesterday" }
        f.timeZone = .current
        f.dateFormat = "MMM d"
        return f.string(from: date)
    }
}
