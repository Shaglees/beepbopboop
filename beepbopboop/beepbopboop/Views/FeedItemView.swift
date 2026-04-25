import SwiftUI
import MapKit

struct FeedItemView: View {
    let post: Post
    @EnvironmentObject private var eventTracker: EventTracker

    var body: some View {
        styledContent
            .onAppear { eventTracker.cardAppeared(postID: post.id) }
            .onDisappear { eventTracker.cardDisappeared(postID: post.id) }
    }

    @ViewBuilder
    private var styledContent: some View {
        cardContent
            .bbbCardChassis()
    }

    @ViewBuilder
    private var cardContent: some View {
        switch post.displayHintValue {
        case .weather:
            if let liveCard = LiveWeatherCard(post: post) {
                liveCard
            } else {
                WeatherCard(post: post)
            }
        case .brief, .digest:
            CompactCard(post: post)
        case .calendar, .event:
            DateCard(post: post)
        case .deal:
            DealCard(post: post)
        case .place:
            PlaceCard(post: post)
        case .outfit:
            OutfitCard(post: post)
        case .scoreboard:
            if let card = ScoreboardCard(post: post) {
                card
            } else {
                StandardCard(post: post)
            }
        case .matchup:
            if let card = MatchupCard(post: post) {
                card
            } else {
                StandardCard(post: post)
            }
        case .standings:
            if let card = StandingsCard(post: post) {
                card
            } else {
                StandardCard(post: post)
            }
        case .boxScore:
            if let card = BoxScoreCard(post: post) {
                card
            } else {
                StandardCard(post: post)
            }
        case .movie:
            if let card = MovieCard(post: post) {
                card
            } else {
                StandardCard(post: post)
            }
        case .show:
            if let card = ShowCard(post: post) {
                card
            } else {
                StandardCard(post: post)
            }
        case .restaurant:
            if let card = RestaurantCard(post: post) {
                card
            } else {
                StandardCard(post: post)
            }
        case .destination:
            if let card = DestinationCard(post: post) {
                card
            } else {
                StandardCard(post: post)
            }
        case .playerSpotlight:
            if let card = PlayerSpotlightCard(post: post) {
                card
            } else {
                StandardCard(post: post)
            }
        case .entertainment:
            EntertainmentCard(post: post)
        case .album:
            if let card = AlbumCard(post: post) {
                card
            } else {
                StandardCard(post: post)
            }
        case .concert:
            if let card = ConcertCard(post: post) {
                card
            } else {
                StandardCard(post: post)
            }
        case .gameRelease:
            if let card = GameReleaseCard(post: post) {
                card
            } else {
                StandardCard(post: post)
            }
        case .gameReview:
            if let card = GameReviewCard(post: post) {
                card
            } else {
                StandardCard(post: post)
            }
        case .science:
            if let card = ScienceCard(post: post) {
                card
            } else {
                StandardCard(post: post)
            }
        case .petSpotlight:
            if let card = PetSpotlightCard(post: post) {
                card
            } else {
                StandardCard(post: post)
            }
        case .fitness:
            if let card = FitnessCard(post: post) {
                card
            } else {
                StandardCard(post: post)
            }
        case .feedback:
            if let card = FeedbackCard(post: post) {
                card
            } else {
                StandardCard(post: post)
            }
        case .creatorSpotlight:
            if let card = CreatorSpotlightCard(post: post) {
                card
            } else {
                StandardCard(post: post)
            }
        case .videoEmbed:
            if let card = VideoEmbedCard(post: post) {
                card
            } else {
                StandardCard(post: post)
            }
        default:
            StandardCard(post: post)
        }
    }
}

// MARK: - Shared Components

struct CardHeader: View {
    let post: Post
    @State private var showAgentProfile = false

    var body: some View {
        HStack(spacing: 8) {
            Button {
                showAgentProfile = true
            } label: {
                ZStack {
                    Circle()
                        .fill(post.hintColor)
                        .frame(width: 20, height: 20)
                    Text(String(post.agentName.prefix(1)))
                        .font(.system(size: 9, weight: .bold))
                        .foregroundColor(.white)
                }
            }
            .buttonStyle(.plain)
            Button {
                showAgentProfile = true
            } label: {
                Text(post.agentName)
                    .font(.system(size: 13, weight: .semibold))
                    .tracking(-0.1)
                    .foregroundStyle(BBBDesign.ink)
            }
            .buttonStyle(.plain)
            HStack(spacing: 4) {
                Circle()
                    .fill(post.hintColor)
                    .frame(width: 4, height: 4)
                Text(post.hintLabel)
                    .font(.system(size: 10, weight: .bold))
                    .tracking(0.9)
                    .textCase(.uppercase)
                    .foregroundColor(post.hintColor)
            }
            .lineLimit(1)
            .fixedSize()
            .padding(.horizontal, 8)
            .padding(.vertical, 4)
            .background(
                Capsule()
                    .stroke(post.hintColor.opacity(0.22), lineWidth: 1)
            )
            Spacer()
            Text(post.relativeTime)
                .font(.system(size: 11, weight: .medium, design: .monospaced))
                .monospacedDigit()
                .foregroundStyle(BBBDesign.ink3)
        }
        .sheet(isPresented: $showAgentProfile) {
            NavigationStack {
                AgentProfileView(agentID: post.agentID, agentName: post.agentName)
            }
            .presentationDragIndicator(.visible)
        }
    }
}

struct CardFooter: View {
    let post: Post
    @State var isBookmarked: Bool
    @State private var activeReaction: String?
    @EnvironmentObject private var apiService: APIService
    @EnvironmentObject private var eventTracker: EventTracker

    init(post: Post) {
        self.post = post
        self._isBookmarked = State(initialValue: post.saved ?? false)
        self._activeReaction = State(initialValue: post.myReaction)
    }

    var body: some View {
        HStack(spacing: 6) {
            if let locality = post.locality, !locality.isEmpty {
                Label(locality, systemImage: post.isSourceAttribution ? "link" : "location")
                    .font(.system(size: 11))
                    .foregroundColor(BBBDesign.ink3)
                    .lineLimit(1)
            }

            Spacer()

            // Swipe-to-tune label or active reaction pill
            if let active = activeReaction,
               let reaction = ReactionPicker.reactionDefs.first(where: { $0.key == active }) {
                // Active reaction: show colored pill (tappable to open full menu)
                Menu {
                    ForEach(ReactionPicker.reactionDefs) { r in
                        Button { setReaction(r.key) } label: {
                            Label(r.label, systemImage: r.icon)
                        }
                    }
                    Divider()
                    Button(role: .destructive) { clearReaction() } label: {
                        Label("Clear reaction", systemImage: "xmark.circle")
                    }
                } label: {
                    HStack(spacing: 4) {
                        Image(systemName: reaction.icon + ".fill")
                            .font(.caption2)
                        Text(reaction.label)
                            .font(.caption2.weight(.semibold))
                    }
                    .foregroundColor(reaction.color)
                    .padding(.horizontal, 8)
                    .padding(.vertical, 5)
                    .background(reaction.color.opacity(0.12))
                    .clipShape(Capsule())
                }
                .menuStyle(.button)
                .buttonStyle(.plain)
            } else {
                // No reaction: passive "swipe to tune" label
                Text("swipe to tune")
                    .font(.system(size: 11, design: .monospaced))
                    .foregroundColor(BBBDesign.ink3)
            }

            ShareLink(
                item: post.shareURL,
                subject: Text(post.title),
                message: Text(post.body.prefix(100))
            ) {
                Image(systemName: "square.and.arrow.up")
                    .font(.caption)
                    .foregroundColor(BBBDesign.ink3)
                    .frame(minWidth: 44, minHeight: 44)
                    .contentShape(Rectangle())
            }
            .buttonStyle(.plain)
            .simultaneousGesture(TapGesture().onEnded {
                Task { await apiService.trackEvent(postID: post.id, type: "share") }
            })

            Button {
                UIImpactFeedbackGenerator(style: .light).impactOccurred()
                let wasSaved = isBookmarked
                isBookmarked.toggle()
                Task {
                    await apiService.trackEvent(
                        postID: post.id,
                        eventType: wasSaved ? "unsave" : "save"
                    )
                }
            } label: {
                Image(systemName: isBookmarked ? "bookmark.fill" : "bookmark")
                    .font(.caption)
                    .foregroundColor(isBookmarked ? BBBDesign.clay : BBBDesign.ink3)
                    .contentTransition(.symbolEffect(.replace))
                    .frame(minWidth: 44, minHeight: 44)
                    .contentShape(Rectangle())
            }
            .buttonStyle(.plain)
        }
        .onChange(of: post.saved) { _, newValue in
            isBookmarked = newValue ?? false
        }
        .onChange(of: post.myReaction) { _, newValue in
            activeReaction = newValue
        }
    }

    private func setReaction(_ key: String) {
        UIImpactFeedbackGenerator(style: .medium).impactOccurred()
        let previous = activeReaction
        withAnimation(.spring(response: 0.35, dampingFraction: 0.75)) {
            activeReaction = key
        }
        Task {
            do {
                try await apiService.setReaction(postID: post.id, reaction: key)
            } catch {
                activeReaction = previous
            }
        }
    }

    private func clearReaction() {
        guard activeReaction != nil else { return }
        UIImpactFeedbackGenerator(style: .light).impactOccurred()
        let previous = activeReaction
        withAnimation(.spring(response: 0.35, dampingFraction: 0.75)) {
            activeReaction = nil
        }
        Task {
            do {
                try await apiService.removeReaction(postID: post.id)
            } catch {
                activeReaction = previous
            }
        }
    }
}

// MARK: - Reaction Picker

enum ReactionPickerStyle {
    case feedCompact    // standard card footer
    case feedDark       // outfit/sports card footer (white-on-dark)
    case detailBar      // standard detail engagement bar
    case detailBarDark  // sports detail engagement bar

    var isDark: Bool { self == .feedDark || self == .detailBarDark }
    var isFeed: Bool { self == .feedCompact || self == .feedDark }
}

struct ReactionPicker: View {
    @Binding var activeReaction: String?
    let postID: String
    var style: ReactionPickerStyle
    @EnvironmentObject private var apiService: APIService

    struct ReactionDef: Identifiable {
        let key: String
        let icon: String
        let label: String
        let color: Color
        var id: String { key }
    }

    static let reactionDefs: [ReactionDef] = [
        ReactionDef(key: "more", icon: "arrow.up.circle", label: "More", color: BBBDesign.reactionMore),
        ReactionDef(key: "less", icon: "arrow.down.circle", label: "Less", color: BBBDesign.reactionLess),
        ReactionDef(key: "stale", icon: "repeat.circle", label: "Stale", color: BBBDesign.reactionStale),
        ReactionDef(key: "not_for_me", icon: "xmark.circle", label: "Not for me", color: BBBDesign.reactionNotForMe),
    ]

    var body: some View {
        if style.isFeed {
            feedLayout
        } else {
            detailLayout
        }
    }

    // MARK: Feed Layout (compact trigger → Menu)
    //
    // Previously this used a manual overlay with a hand-rolled dismiss layer
    // offset by (100, 100) — it didn't reliably dismiss on outside tap and the
    // floating picker was clipped by the card's rounded-rect shape. A native
    // Menu handles positioning, outside-tap dismissal, and keyboard focus for
    // us, and it renders in its own window so it's never clipped by the row.

    @ViewBuilder
    private var feedLayout: some View {
        Menu {
            ForEach(Self.reactionDefs) { reaction in
                Button {
                    selectReaction(reaction)
                } label: {
                    Label(reaction.label, systemImage: reaction.icon)
                }
            }
            if activeReaction != nil {
                Divider()
                Button(role: .destructive) {
                    clearReaction()
                } label: {
                    Label("Clear reaction", systemImage: "xmark.circle")
                }
            }
        } label: {
            feedTriggerLabel
        } primaryAction: {
            // Tapping the trigger when no reaction is set opens the menu
            // (default). When a reaction is already set, we want a quick
            // "remove" affordance — handled by long-press-to-open-menu; the
            // primary tap here toggles the active reaction off.
            if activeReaction != nil {
                clearReaction()
            }
        }
        .menuStyle(.button)
        .buttonStyle(.plain)
    }

    @ViewBuilder
    private var feedTriggerLabel: some View {
        if let active = activeReaction,
           let reaction = Self.reactionDefs.first(where: { $0.key == active }) {
            HStack(spacing: 4) {
                Image(systemName: reaction.icon + ".fill")
                    .font(.caption2)
                Text(reaction.label)
                    .font(.caption2.weight(.semibold))
            }
            .foregroundColor(reaction.color)
            .padding(.horizontal, 8)
            .padding(.vertical, 5)
            .background(reaction.color.opacity(0.12))
            .clipShape(Capsule())
            .frame(minHeight: 44)
            .contentShape(Rectangle())
            .accessibilityLabel("Reacted with \(reaction.label). Double tap to change.")
        } else {
            Text("swipe to tune")
                .font(.system(size: 11, design: .monospaced))
                .foregroundColor(style.isDark ? .white.opacity(0.45) : BBBDesign.ink3)
                .frame(minHeight: 44)
                .contentShape(Rectangle())
                .accessibilityLabel("Swipe or double tap to react to this post")
        }
    }

    // MARK: Detail Layout (always-visible inline circles)

    private var detailLayout: some View {
        HStack(spacing: 6) {
            ForEach(Self.reactionDefs) { reaction in
                let isActive = activeReaction == reaction.key
                Button {
                    selectReaction(reaction)
                } label: {
                    Image(systemName: isActive ? reaction.icon + ".fill" : reaction.icon)
                        .font(.subheadline)
                        .foregroundColor(isActive ? .white : reaction.color)
                        .frame(width: 34, height: 34)
                        .background(isActive ? reaction.color : reaction.color.opacity(style.isDark ? 0.15 : 0.1))
                        .clipShape(Circle())
                        .contentTransition(.symbolEffect(.replace))
                }
                .buttonStyle(.plain)
                .accessibilityLabel(isActive ? "Reacted with \(reaction.label). Double tap to change." : reaction.label)
            }
        }
    }

    // MARK: Action

    private func selectReaction(_ reaction: ReactionDef) {
        UIImpactFeedbackGenerator(style: .medium).impactOccurred()
        let wasActive = activeReaction == reaction.key
        let previous = activeReaction
        withAnimation(.spring(response: 0.35, dampingFraction: 0.75)) {
            activeReaction = wasActive ? nil : reaction.key
        }

        Task {
            do {
                if wasActive {
                    try await apiService.removeReaction(postID: postID)
                } else {
                    try await apiService.setReaction(postID: postID, reaction: reaction.key)
                }
            } catch {
                activeReaction = previous
            }
        }
    }

    private func clearReaction() {
        guard activeReaction != nil else { return }
        UIImpactFeedbackGenerator(style: .light).impactOccurred()
        let previous = activeReaction
        withAnimation(.spring(response: 0.35, dampingFraction: 0.75)) {
            activeReaction = nil
        }
        Task {
            do {
                try await apiService.removeReaction(postID: postID)
            } catch {
                activeReaction = previous
            }
        }
    }
}

// MARK: - Standard Card (card, article, comparison)

private struct StandardCard: View {
    let post: Post

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            CardHeader(post: post)

            Text(post.title)
                .font(.system(size: 15, weight: .semibold))
                .tracking(-0.2)
                .foregroundColor(BBBDesign.ink)
                .lineLimit(2)

            Text(post.body)
                .font(.system(size: 13))
                .lineSpacing(2)
                .foregroundColor(BBBDesign.ink2)
                .lineLimit(3)

            // Article hint: hero image
            if let imageURL = post.imageURL, !imageURL.isEmpty, let url = URL(string: imageURL) {
                AsyncImage(url: url) { phase in
                    switch phase {
                    case .success(let image):
                        image
                            .resizable()
                            .aspectRatio(contentMode: .fill)
                            .frame(maxHeight: 200)
                            .clipped()
                            .cornerRadius(10)
                    case .failure:
                        EmptyView()
                    default:
                        ProgressView()
                            .frame(height: 120)
                            .frame(maxWidth: .infinity)
                    }
                }
            } else if post.postTypeValue == .place,
                      let lat = post.latitude, let lon = post.longitude {
                Map(initialPosition: .region(MKCoordinateRegion(
                    center: CLLocationCoordinate2D(latitude: lat, longitude: lon),
                    span: MKCoordinateSpan(latitudeDelta: 0.005, longitudeDelta: 0.005)
                ))) {
                    Marker(post.markerLabel, systemImage: post.typeIcon, coordinate: CLLocationCoordinate2D(latitude: lat, longitude: lon))
                        .tint(post.typeColor)
                }
                .frame(height: 120)
                .cornerRadius(10)
                .allowsHitTesting(false)
            }

            // Comparison badge
            if post.displayHintValue == .comparison {
                HStack(spacing: 4) {
                    Image(systemName: "arrow.left.arrow.right")
                    Text("Comparison")
                }
                .font(.caption2.weight(.medium))
                .foregroundColor(BBBDesign.reactionMore)
                .padding(.horizontal, 8)
                .padding(.vertical, 4)
                .background(BBBDesign.reactionMore.opacity(0.1))
                .cornerRadius(6)
            }

            CardFooter(post: post)
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 14)
    }
}

// MARK: - Weather Card

private struct WeatherCard: View {
    let post: Post

    // Try to parse structured weather from external_url JSON
    private var structuredWeather: StructuredWeather? {
        guard let json = post.externalURL,
              let data = json.data(using: .utf8),
              let dict = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
              let current = dict["current"] as? [String: Any],
              let tempC = current["temp_c"] as? Double else { return nil }
        let condition = current["condition"] as? String ?? ""
        let hiLo: (hi: Double, lo: Double)? = {
            if let daily = dict["daily"] as? [[String: Any]], let first = daily.first,
               let hi = first["high_c"] as? Double, let lo = first["low_c"] as? Double {
                return (hi, lo)
            }
            return nil
        }()
        let hourly: [HourlyEntry] = {
            guard let arr = dict["hourly"] as? [[String: Any]] else { return [] }
            return arr.compactMap { h in
                guard let time = h["time"] as? String,
                      let temp = h["temp_c"] as? Double else { return nil }
                let code = h["condition_code"] as? Int ?? 0
                return HourlyEntry(time: time, tempC: temp, conditionCode: code)
            }
        }()
        return StructuredWeather(tempC: tempC, condition: condition, hiLo: hiLo, hourly: hourly)
    }

    private struct StructuredWeather {
        let tempC: Double
        let condition: String
        let hiLo: (hi: Double, lo: Double)?
        let hourly: [HourlyEntry]
    }

    private struct HourlyEntry {
        let time: String
        let tempC: Double
        let conditionCode: Int

        var hourLabel: String {
            if time.count >= 13 {
                let hourString = String(time.dropFirst(11).prefix(2))
                if let hour = Int(hourString) {
                    if hour == 0 { return "12am" }
                    if hour == 12 { return "12pm" }
                    return hour < 12 ? "\(hour)am" : "\(hour - 12)pm"
                }
            }

            if time.count >= 5 {
                return String(time.prefix(5))
            }

            return time
        }
    }

    var body: some View {
        if let sw = structuredWeather {
            structuredBody(sw)
        } else {
            fallbackBody
        }
    }

    // MARK: Structured weather layout (matches PDF)
    private func structuredBody(_ sw: StructuredWeather) -> some View {
        VStack(alignment: .leading, spacing: 0) {
            VStack(alignment: .leading, spacing: 8) {
                CardHeader(post: post)

                HStack(alignment: .bottom, spacing: 14) {
                    Text("\(Int(sw.tempC.rounded()))°")
                        .font(.system(size: 52, weight: .regular, design: .serif))
                        .tracking(-1.6)
                        .foregroundColor(BBBDesign.ink)

                    VStack(alignment: .leading, spacing: 4) {
                        Text(sw.condition.isEmpty ? post.title : sw.condition)
                            .font(.system(size: 14, weight: .semibold))
                            .foregroundColor(BBBDesign.ink)
                            .lineLimit(2)
                        if let hiLo = sw.hiLo {
                            Text("H \(Int(hiLo.hi.rounded()))° · L \(Int(hiLo.lo.rounded()))°")
                                .font(.system(size: 12, design: .monospaced))
                                .foregroundColor(BBBDesign.ink3)
                        }
                    }
                    .padding(.top, 8)
                }

                Text(post.body)
                    .font(.system(size: 13))
                    .lineSpacing(2)
                    .foregroundColor(BBBDesign.ink2)
                    .lineLimit(3)

                CardFooter(post: post)
            }
            .padding(.horizontal, 16)
            .padding(.vertical, 12)

            // Hourly strip with sunken background
            if !sw.hourly.isEmpty {
                ScrollView(.horizontal, showsIndicators: false) {
                    HStack(spacing: 6) {
                        ForEach(Array(sw.hourly.enumerated()), id: \.offset) { _, hour in
                            VStack(spacing: 4) {
                                Text(hour.hourLabel)
                                    .font(.system(size: 10, weight: .medium, design: .monospaced))
                                    .foregroundColor(BBBDesign.ink3)
                                Image(systemName: WeatherData.icon(for: hour.conditionCode, isDay: true))
                                    .font(.system(size: 17))
                                    .foregroundStyle(BBBDesign.ink)
                                    .frame(height: 20)
                                Text("\(Int(hour.tempC.rounded()))°")
                                    .font(.system(size: 12, weight: .semibold))
                                    .foregroundColor(BBBDesign.ink)
                            }
                            .frame(minWidth: 40)
                            .padding(.vertical, 4)
                            .padding(.horizontal, 5)
                        }
                    }
                    .padding(.horizontal, 8)
                    .padding(.vertical, 10)
                }
                .background(BBBDesign.sunken)
            }
        }
        .background(BBBDesign.surface)
    }

    // MARK: Fallback layout (no structured data)
    private var fallbackBody: some View {
        VStack(alignment: .leading, spacing: 8) {
            CardHeader(post: post)

            HStack(alignment: .top, spacing: 12) {
                let weather = WeatherInfo.detect(from: post.title + " " + post.body)
                Image(systemName: weather.icon)
                    .font(.system(size: 34))
                    .foregroundStyle(weather.primaryColor, weather.secondaryColor)
                    .frame(width: 44)

                VStack(alignment: .leading, spacing: 4) {
                    Text(post.title)
                        .font(.system(size: 22, weight: .regular, design: .serif))
                        .tracking(-0.7)
                        .foregroundColor(BBBDesign.ink)
                        .lineLimit(2)
                    Text(post.body)
                        .font(.system(size: 13))
                        .lineSpacing(2)
                        .foregroundColor(BBBDesign.ink2)
                        .lineLimit(4)
                }
            }

            CardFooter(post: post)
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 12)
        .background(BBBDesign.surface)
    }
}

// MARK: - Compact Card (brief, digest)

private struct CompactCard: View {
    let post: Post

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            CardHeader(post: post)

            Text(post.title)
                .font(.system(size: 22, weight: .regular, design: .serif))
                .tracking(-0.7)
                .foregroundColor(BBBDesign.ink)
                .lineLimit(2)

            // Show body as compact bullets — split on newlines
            let lines = post.body.components(separatedBy: "\n").filter { !$0.trimmingCharacters(in: .whitespaces).isEmpty }
            VStack(alignment: .leading, spacing: 8) {
                ForEach(Array(lines.prefix(5).enumerated()), id: \.offset) { index, line in
                    HStack(alignment: .top, spacing: 8) {
                        if post.displayHintValue == .digest {
                            Text(String(format: "%02d", index + 1))
                                .font(.system(size: 11, weight: .medium, design: .monospaced))
                                .foregroundColor(post.hintColor)
                                .frame(width: 18, alignment: .trailing)
                        } else {
                            Text(String(format: "%02d", index + 1))
                                .font(.system(size: 11, weight: .medium, design: .monospaced))
                                .foregroundColor(BBBDesign.ink3)
                                .frame(width: 18, alignment: .trailing)
                        }
                        Text(line.trimmingCharacters(in: .whitespaces))
                            .font(.system(size: 13))
                            .foregroundColor(BBBDesign.ink2)
                            .lineSpacing(2)
                            .lineLimit(2)
                    }
                }
                if lines.count > 5 {
                    Text("+\(lines.count - 5) more")
                        .font(.caption2)
                        .foregroundStyle(.tertiary)
                }
            }

            CardFooter(post: post)
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 14)
    }
}

// MARK: - Date Card (calendar, event)

private struct DateCard: View {
    let post: Post
    // Use openURL from an explicit Button action rather than SwiftUI's Link.
    // The outer FeedItemView wraps the whole card in a Button for navigation,
    // and nesting a Link inside produces gesture conflicts: sometimes the tap
    // activates the outer button (goes to post detail) instead of opening the
    // URL. An inner Button with its own action reliably consumes the tap.
    @Environment(\.openURL) private var openURL

    /// The real date a skill attached to this post, used to render the badge.
    /// Priority order:
    ///   1. `scheduled_at` (structured field) — the server's authoritative answer.
    ///   2. Date token in the title (e.g. "April 16").
    ///   3. Date token in the body (e.g. "Saturday May 10 at 2pm").
    /// Returns `nil` for evergreen content so the badge is suppressed rather
    /// than fabricated from `createdAt`.
    private var parsedDate: Date? {
        if let iso = post.scheduledAt, let d = Self.parseISODate(iso) { return d }
        if let d = Self.extractDate(from: post.title) { return d }
        if let d = Self.extractDate(from: post.body) { return d }
        return nil
    }

    /// Parse an ISO-8601 timestamp with or without fractional seconds, matching
    /// the two formats the backend emits for `scheduled_at` / `created_at`.
    private static func parseISODate(_ s: String) -> Date? {
        let f1 = ISO8601DateFormatter()
        f1.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        if let d = f1.date(from: s) { return d }
        let f2 = ISO8601DateFormatter()
        f2.formatOptions = [.withInternetDateTime]
        return f2.date(from: s)
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            CardHeader(post: post)

            HStack(alignment: .top, spacing: 12) {
                if let date = parsedDate {
                    let parts = Self.formatBadge(for: date)
                    VStack(spacing: 2) {
                        Text(parts.month)
                            .font(.caption2.weight(.bold))
                            .foregroundColor(post.hintColor)
                            .textCase(.uppercase)
                        Text(parts.day)
                            .font(.title2.weight(.bold))
                            .foregroundColor(.primary)
                    }
                    .frame(width: 48, height: 52)
                    .background(post.hintColor.opacity(0.1))
                    .cornerRadius(8)
                }

                VStack(alignment: .leading, spacing: 6) {
                    Text(post.title)
                        .font(.headline)
                        .lineLimit(2)

                    Text(post.body)
                        .font(.subheadline)
                        .foregroundColor(.secondary)
                        .lineLimit(3)

                    if post.displayHintValue == .event {
                        if let locality = post.locality, !locality.isEmpty {
                            Label(locality, systemImage: "location")
                                .font(.caption)
                                .foregroundColor(post.hintColor)
                        }
                        if let extURL = post.externalURL, !extURL.isEmpty,
                           let url = URL(string: extURL) {
                            Button { openURL(url) } label: {
                                Label("Get Tickets", systemImage: "arrow.up.right.square")
                                    .font(.caption.weight(.medium))
                                    .foregroundColor(post.hintColor)
                            }
                            .buttonStyle(.plain)
                        }
                    }
                }
            }

            CardFooter(post: post)
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 12)
    }

    /// Formats a Date into (MMM, D) parts for the badge.
    private static func formatBadge(for date: Date) -> (month: String, day: String) {
        let cal = Calendar.current
        let monthF = DateFormatter()
        monthF.dateFormat = "MMM"
        return (monthF.string(from: date), "\(cal.component(.day, from: date))")
    }

    /// Try to find a date like "April 16" or "Jan 3" in text.
    private static func extractDate(from text: String) -> Date? {
        let detector = try? NSDataDetector(types: NSTextCheckingResult.CheckingType.date.rawValue)
        let range = NSRange(text.startIndex..., in: text)
        if let match = detector?.firstMatch(in: text, range: range), let date = match.date {
            return date
        }
        return nil
    }
}

// MARK: - Deal Card

private struct DealCard: View {
    let post: Post

    /// Try to extract a discount percentage from the post text or external data
    private var discountPercent: Int? {
        // Check steam deal data in externalURL JSON
        if let json = post.externalURL,
           let data = json.data(using: .utf8),
           let dict = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
           let discount = dict["steamDiscount"] as? Int, discount > 0 {
            return discount
        }
        // Regex fallback: look for "XX%" in title or body
        let text = post.title + " " + post.body
        let pattern = #"(\d+)\s*%"#
        if let match = text.range(of: pattern, options: .regularExpression) {
            let matched = String(text[match])
            let numStr = matched.filter(\.isNumber)
            if let num = Int(numStr), num > 0, num <= 100 { return num }
        }
        return nil
    }

    /// Try to extract prices from external data
    private var prices: (current: String, original: String)? {
        guard let json = post.externalURL,
              let data = json.data(using: .utf8),
              let dict = try? JSONSerialization.jsonObject(with: data) as? [String: Any] else { return nil }
        if let price = dict["steamPrice"] as? String,
           let original = dict["steamOriginalPrice"] as? String {
            return (price, original)
        }
        return nil
    }

    /// Try to extract "ends in" from external data
    private var endsIn: String? {
        guard let json = post.externalURL,
              let data = json.data(using: .utf8),
              let dict = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
              let ends = dict["ends_in"] as? String else { return nil }
        return ends
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            CardHeader(post: post)

            // Deal highlight with gradient background
            VStack(alignment: .leading, spacing: 10) {
                if let percent = discountPercent {
                    HStack(alignment: .firstTextBaseline, spacing: 10) {
                        Text("−\(percent)%")
                            .font(.system(size: 34, weight: .heavy, design: .serif))
                            .foregroundColor(post.hintColor)
                        VStack(alignment: .leading, spacing: 2) {
                            Text(post.title)
                                .font(.system(size: 15, weight: .semibold))
                                .lineLimit(2)
                        }
                    }

                    // Price line + ENDS badge
                    if let prices = prices {
                        HStack(spacing: 8) {
                            Text(prices.current)
                                .font(.system(size: 14, weight: .bold, design: .monospaced))
                            Text(prices.original)
                                .font(.system(size: 12, design: .monospaced))
                                .strikethrough()
                                .foregroundColor(.secondary)
                            if let ends = endsIn {
                                Text("ENDS \(ends.uppercased())")
                                    .font(.system(size: 9, weight: .heavy))
                                    .tracking(0.5)
                                    .foregroundColor(.orange)
                                    .padding(.horizontal, 6)
                                    .padding(.vertical, 3)
                                    .background(Color.orange.opacity(0.12))
                                    .clipShape(Capsule())
                            }
                        }
                    }
                } else {
                    // Fallback: banner style with updated typography
                    HStack(spacing: 6) {
                        Image(systemName: "tag.fill")
                            .font(.caption)
                        Text("DEAL")
                            .font(.system(size: 10, weight: .black))
                            .tracking(0.8)
                    }
                    .foregroundColor(.white)
                    .padding(.horizontal, 12)
                    .padding(.vertical, 6)
                    .background(
                        LinearGradient(
                            colors: [.pink, .orange],
                            startPoint: .leading,
                            endPoint: .trailing
                        )
                    )
                    .cornerRadius(8)

                    Text(post.title)
                        .font(.system(size: 15, weight: .semibold))
                        .lineLimit(2)
                }

                Text(post.body)
                    .font(.system(size: 13))
                    .foregroundColor(.secondary)
                    .lineLimit(3)
            }
            .padding(14)
            .background(
                RoundedRectangle(cornerRadius: 12)
                    .fill(
                        LinearGradient(
                            colors: [post.hintColor.opacity(0.18), post.hintColor.opacity(0.04)],
                            startPoint: .topLeading,
                            endPoint: .bottomTrailing
                        )
                    )
            )
            .overlay(
                RoundedRectangle(cornerRadius: 12)
                    .stroke(post.hintColor.opacity(0.20), lineWidth: 1)
            )

            if let imageURL = post.imageURL, !imageURL.isEmpty, let url = URL(string: imageURL) {
                AsyncImage(url: url) { phase in
                    switch phase {
                    case .success(let image):
                        image
                            .resizable()
                            .aspectRatio(contentMode: .fill)
                            .frame(maxHeight: 160)
                            .clipped()
                            .cornerRadius(10)
                    case .failure:
                        EmptyView()
                    default:
                        ProgressView()
                            .frame(height: 100)
                            .frame(maxWidth: .infinity)
                    }
                }
            }

            CardFooter(post: post)
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 12)
    }
}

// MARK: - Place Card

private struct PlaceCard: View {
    let post: Post
    // See DateCard for why we use openURL + Button instead of SwiftUI Link.
    @Environment(\.openURL) private var openURL

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            CardHeader(post: post)

            // Map if coordinates available
            if let lat = post.latitude, let lon = post.longitude {
                Map(initialPosition: .region(MKCoordinateRegion(
                    center: CLLocationCoordinate2D(latitude: lat, longitude: lon),
                    span: MKCoordinateSpan(latitudeDelta: 0.005, longitudeDelta: 0.005)
                ))) {
                    Marker(post.markerLabel, systemImage: "mappin", coordinate: CLLocationCoordinate2D(latitude: lat, longitude: lon))
                        .tint(.green)
                }
                .frame(height: 140)
                .cornerRadius(10)
                .allowsHitTesting(false)
            } else if let imageURL = post.imageURL, !imageURL.isEmpty, let url = URL(string: imageURL) {
                AsyncImage(url: url) { phase in
                    switch phase {
                    case .success(let image):
                        image
                            .resizable()
                            .aspectRatio(contentMode: .fill)
                            .frame(maxHeight: 140)
                            .clipped()
                            .cornerRadius(10)
                    case .failure:
                        EmptyView()
                    default:
                        ProgressView()
                            .frame(height: 100)
                            .frame(maxWidth: .infinity)
                    }
                }
            }

            Text(post.title)
                .font(.headline)
                .lineLimit(2)

            Text(post.body)
                .font(.subheadline)
                .foregroundColor(.secondary)
                .lineLimit(3)

            // CTA for place posts that carry a booking/info link. Without this,
            // skills that set external_url on a `place` post had no clickable
            // surface — the link was silently dropped.
            if let extURL = post.externalURL, !extURL.isEmpty,
               let url = URL(string: extURL) {
                Button { openURL(url) } label: {
                    Label(placeCTALabel(for: extURL), systemImage: "arrow.up.right.square")
                        .font(.caption.weight(.semibold))
                        .foregroundColor(post.hintColor)
                        .padding(.horizontal, 10)
                        .padding(.vertical, 6)
                        .background(post.hintColor.opacity(0.12))
                        .cornerRadius(6)
                }
                .buttonStyle(.plain)
            }

            CardFooter(post: post)
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 12)
    }

    /// Pick a reasonable CTA label based on URL host. We keep this intentionally
    /// simple — "Visit site" is always a safe fallback; booking/reservation hosts
    /// get a stronger verb.
    private func placeCTALabel(for urlString: String) -> String {
        let lower = urlString.lowercased()
        if lower.contains("book") || lower.contains("reserv") || lower.contains("ticket") {
            return "Book"
        }
        if lower.contains("menu") {
            return "View menu"
        }
        if lower.contains("maps.") || lower.contains("/maps") {
            return "Directions"
        }
        return "Visit site"
    }
}

// MARK: - Outfit Card

private struct OutfitCard: View {
    let post: Post
    private let outfitMauve = Color(red: 0.878, green: 0.251, blue: 0.984)
    private let darkBg = Color(red: 0.1, green: 0.086, blue: 0.071) // #1a1612

    private var heroURL: URL? {
        if let hero = post.heroImage, let url = URL(string: hero.url) { return url }
        if let imageURL = post.imageURL, !imageURL.isEmpty { return URL(string: imageURL) }
        return nil
    }

    private var content: OutfitContent { post.outfitContent }

    var body: some View {
        VStack(spacing: 0) {
            // Hero image with gradient overlays
            ZStack(alignment: .top) {
                // Hero image
                if let url = heroURL {
                    AsyncImage(url: url) { phase in
                        switch phase {
                        case .success(let image):
                            image
                                .resizable()
                                .aspectRatio(contentMode: .fill)
                                .frame(height: 320)
                                .clipped()
                        case .failure:
                            Rectangle()
                                .fill(darkBg)
                                .frame(height: 320)
                        default:
                            Rectangle()
                                .fill(darkBg)
                                .frame(height: 320)
                                .overlay(ProgressView().tint(.white))
                        }
                    }
                } else {
                    Rectangle()
                        .fill(darkBg)
                        .frame(height: 320)
                }

                // Top gradient with header info
                VStack {
                    HStack(spacing: 6) {
                        Circle()
                            .fill(outfitMauve)
                            .frame(width: 8, height: 8)
                        Text(post.agentName)
                            .font(.subheadline.weight(.medium))
                            .foregroundColor(.white)
                        Text("Outfit")
                            .font(.caption2.weight(.semibold))
                            .foregroundColor(.white)
                            .padding(.horizontal, 7)
                            .padding(.vertical, 3)
                            .background(.white.opacity(0.2))
                            .cornerRadius(4)
                        Spacer()
                        Text(post.relativeTime)
                            .font(.subheadline)
                            .foregroundColor(.white.opacity(0.6))
                    }
                    .padding(.horizontal, 16)
                    .padding(.top, 14)
                    .padding(.bottom, 32)
                    .background(
                        LinearGradient(
                            colors: [.black.opacity(0.3), .clear],
                            startPoint: .top,
                            endPoint: .bottom
                        )
                    )

                    Spacer()

                    // Bottom gradient with trend + title
                    VStack(alignment: .leading, spacing: 6) {
                        if let trend = content.trend {
                            Text(trend.uppercased())
                                .font(.system(size: 9, weight: .semibold))
                                .tracking(2.5)
                                .foregroundColor(.white.opacity(0.5))
                        }
                        Text(post.title)
                            .font(.system(size: 20, weight: .bold, design: .serif))
                            .foregroundColor(Color(red: 0.96, green: 0.94, blue: 0.92))
                            .lineLimit(3)
                    }
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .padding(.horizontal, 16)
                    .padding(.bottom, 16)
                    .padding(.top, 48)
                    .background(
                        LinearGradient(
                            colors: [.clear, Color(red: 0.078, green: 0.063, blue: 0.047).opacity(0.9)],
                            startPoint: .top,
                            endPoint: .bottom
                        )
                    )
                }
            }
            .frame(height: 320)
            .clipped()

            // "For you" strip
            if let forYou = content.forYou, !forYou.isEmpty {
                VStack(alignment: .leading, spacing: 6) {
                    Text("FOR YOU")
                        .font(.system(size: 9, weight: .heavy))
                        .tracking(1)
                        .foregroundColor(outfitMauve)
                    Text(forYou)
                        .font(.caption)
                        .foregroundColor(.white.opacity(0.6))
                        .lineLimit(2)
                }
                .frame(maxWidth: .infinity, alignment: .leading)
                .padding(.horizontal, 16)
                .padding(.vertical, 12)
                .background(darkBg)
            }

            // Product scroll row
            if !content.products.isEmpty {
                ScrollView(.horizontal, showsIndicators: false) {
                    HStack(spacing: 12) {
                        ForEach(Array(content.products.enumerated()), id: \.offset) { index, product in
                            let productImages = post.imagesByRole("product")
                            VStack(spacing: 4) {
                                if index < productImages.count, let url = URL(string: productImages[index].url) {
                                    AsyncImage(url: url) { phase in
                                        switch phase {
                                        case .success(let image):
                                            image.resizable().aspectRatio(contentMode: .fill)
                                        default:
                                            RoundedRectangle(cornerRadius: 6)
                                                .fill(outfitMauve.opacity(0.15))
                                        }
                                    }
                                    .frame(width: 60, height: 60)
                                    .clipShape(RoundedRectangle(cornerRadius: 6))
                                } else {
                                    RoundedRectangle(cornerRadius: 6)
                                        .fill(outfitMauve.opacity(0.15))
                                        .frame(width: 60, height: 60)
                                        .overlay(
                                            Image(systemName: "tshirt")
                                                .font(.caption)
                                                .foregroundColor(outfitMauve.opacity(0.4))
                                        )
                                }
                                Text(product.name.components(separatedBy: " ").first ?? product.name)
                                    .font(.system(size: 9))
                                    .foregroundColor(.white.opacity(0.4))
                                    .lineLimit(1)
                                Text(product.price)
                                    .font(.system(size: 11, weight: .semibold))
                                    .foregroundColor(Color(red: 0.96, green: 0.94, blue: 0.92))
                            }
                            .frame(width: 60)
                        }
                    }
                    .padding(.horizontal, 16)
                    .padding(.vertical, 10)
                }
                .background(darkBg)
            }

            // Footer
            OutfitFooter(post: post, outfitMauve: outfitMauve)
                .background(darkBg)
        }
    }
}

private struct OutfitFooter: View {
    let post: Post
    let outfitMauve: Color
    @State private var activeReaction: String?

    init(post: Post, outfitMauve: Color) {
        self.post = post
        self.outfitMauve = outfitMauve
        self._activeReaction = State(initialValue: post.myReaction)
    }

    var body: some View {
        HStack(spacing: 6) {
            if let locality = post.locality, !locality.isEmpty {
                Label(locality, systemImage: post.isSourceAttribution ? "link" : "location")
                    .font(.caption2)
                    .foregroundColor(.white.opacity(0.4))
                    .lineLimit(1)
            }
            Spacer()
            ReactionPicker(
                activeReaction: $activeReaction,
                postID: post.id,
                style: .feedDark
            )
            OutfitBookmarkButton(post: post, tintColor: outfitMauve)
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 10)
    }
}

private struct OutfitBookmarkButton: View {
    let post: Post
    let tintColor: Color
    @State var isBookmarked: Bool
    @EnvironmentObject private var eventTracker: EventTracker
    @EnvironmentObject private var apiService: APIService

    init(post: Post, tintColor: Color) {
        self.post = post
        self.tintColor = tintColor
        self._isBookmarked = State(initialValue: post.saved ?? false)
    }

    var body: some View {
        Button {
            UIImpactFeedbackGenerator(style: .light).impactOccurred()
            let wasSaved = isBookmarked
            isBookmarked.toggle()
            Task {
                await apiService.trackEvent(
                    postID: post.id,
                    eventType: wasSaved ? "unsave" : "save"
                )
            }
        } label: {
            Image(systemName: isBookmarked ? "bookmark.fill" : "bookmark")
                .font(.caption)
                .foregroundColor(isBookmarked ? tintColor : .white.opacity(0.4))
                .contentTransition(.symbolEffect(.replace))
        }
        .buttonStyle(.plain)
    }
}

// MARK: - Weather Icon Detection

private struct WeatherInfo {
    let icon: String
    let primaryColor: Color
    let secondaryColor: Color

    static func detect(from text: String) -> WeatherInfo {
        let lower = text.lowercased()

        // Most specific patterns first
        if lower.contains("snow") || lower.contains("blizzard") {
            return WeatherInfo(icon: "cloud.snow.fill", primaryColor: .gray, secondaryColor: .white)
        }
        if lower.contains("thunder") || lower.contains("lightning") || lower.contains("storm") {
            return WeatherInfo(icon: "cloud.bolt.rain.fill", primaryColor: .gray, secondaryColor: .yellow)
        }
        if lower.contains("heavy rain") || lower.contains("downpour") || lower.contains("torrential") {
            return WeatherInfo(icon: "cloud.heavyrain.fill", primaryColor: .gray, secondaryColor: .blue)
        }
        if lower.contains("rain") || lower.contains("drizzle") || lower.contains("shower") {
            return WeatherInfo(icon: "cloud.rain.fill", primaryColor: .gray, secondaryColor: .cyan)
        }
        // "partly cloudy" before generic "cloudy"
        if lower.contains("partly cloudy") || lower.contains("partly sunny") || lower.contains("mix of sun") {
            return WeatherInfo(icon: "cloud.sun.fill", primaryColor: .cyan, secondaryColor: .yellow)
        }
        if lower.contains("overcast") || lower.contains("cloudy") {
            return WeatherInfo(icon: "cloud.fill", primaryColor: .gray, secondaryColor: .gray)
        }
        if lower.contains("fog") || lower.contains("mist") || lower.contains("haze") {
            return WeatherInfo(icon: "cloud.fog.fill", primaryColor: .gray, secondaryColor: .secondary)
        }
        if lower.contains("clear") || lower.contains("sunny") {
            return WeatherInfo(icon: "sun.max.fill", primaryColor: .yellow, secondaryColor: .orange)
        }
        if lower.contains("wind") || lower.contains("gusty") || lower.contains("breezy") {
            return WeatherInfo(icon: "wind", primaryColor: .cyan, secondaryColor: .gray)
        }
        // Default
        return WeatherInfo(icon: "cloud.sun.fill", primaryColor: .cyan, secondaryColor: .yellow)
    }
}
