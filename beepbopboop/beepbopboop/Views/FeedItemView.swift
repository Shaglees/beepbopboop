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
        if [.outfit, .weather, .scoreboard, .matchup, .standings, .playerSpotlight, .entertainment].contains(post.displayHintValue) {
            cardContent
                .clipShape(RoundedRectangle(cornerRadius: 16))
                .shadow(color: .black.opacity(0.12), radius: 12, x: 0, y: 4)
        } else {
            cardContent
                .background(
                    RoundedRectangle(cornerRadius: 16)
                        .fill(Color(.secondarySystemGroupedBackground))
                )
                .overlay(
                    RoundedRectangle(cornerRadius: 16)
                        .stroke(Color(.separator).opacity(0.2), lineWidth: 0.5)
                )
                .clipShape(RoundedRectangle(cornerRadius: 16))
                .shadow(color: .black.opacity(0.04), radius: 6, x: 0, y: 2)
        }
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
        case .playerSpotlight:
            if let card = PlayerSpotlightCard(post: post) {
                card
            } else {
                StandardCard(post: post)
            }
        case .entertainment:
            EntertainmentCard(post: post)
        default:
            StandardCard(post: post)
        }
    }
}

// MARK: - Shared Components

private struct CardHeader: View {
    let post: Post

    var body: some View {
        HStack(spacing: 6) {
            Circle()
                .fill(post.hintColor)
                .frame(width: 8, height: 8)
            Text(post.agentName)
                .font(.subheadline.weight(.medium))
            Text(post.hintLabel)
                .font(.caption2.weight(.semibold))
                .foregroundColor(post.hintColor)
                .lineLimit(1)
                .fixedSize()
                .padding(.horizontal, 7)
                .padding(.vertical, 3)
                .background(post.hintColor.opacity(0.12))
                .cornerRadius(4)
            Spacer()
            Text(post.relativeTime)
                .font(.subheadline)
                .foregroundStyle(.tertiary)
        }
    }
}

private struct CardFooter: View {
    let post: Post
    @AppStorage var isBookmarked: Bool
    @State private var activeReaction: String?
    @EnvironmentObject private var apiService: APIService
    @EnvironmentObject private var eventTracker: EventTracker

    init(post: Post) {
        self.post = post
        self._isBookmarked = AppStorage(wrappedValue: false, "bookmark_\(post.id)")
        self._activeReaction = State(initialValue: post.myReaction)
    }

    var body: some View {
        HStack(spacing: 6) {
            if let locality = post.locality, !locality.isEmpty {
                Label(locality, systemImage: post.isSourceAttribution ? "link" : "location")
                    .font(.caption2)
                    .foregroundColor(.secondary)
                    .lineLimit(1)
            }

            Spacer()

            ReactionPicker(
                activeReaction: $activeReaction,
                postID: post.id,
                style: .feedCompact
            )

            ShareLink(
                item: post.shareURL,
                subject: Text(post.title),
                message: Text(post.body.prefix(100))
            ) {
                Image(systemName: "square.and.arrow.up")
                    .font(.caption)
                    .foregroundColor(.secondary)
            }
            .buttonStyle(.plain)
            .simultaneousGesture(TapGesture().onEnded {
                Task { await apiService.trackEvent(postID: post.id, type: "share") }
            })

            Button {
                UIImpactFeedbackGenerator(style: .light).impactOccurred()
                isBookmarked.toggle()
                let eventType = isBookmarked ? "save" : "unsave"
                Task { await apiService.trackEvent(postID: post.id, eventType: eventType) }
            } label: {
                Image(systemName: isBookmarked ? "bookmark.fill" : "bookmark")
                    .font(.caption)
                    .foregroundColor(isBookmarked ? post.hintColor : .secondary)
                    .contentTransition(.symbolEffect(.replace))
            }
            .buttonStyle(.plain)
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
    @State private var isExpanded = false
    @EnvironmentObject private var apiService: APIService

    private struct ReactionDef: Identifiable {
        let key: String
        let icon: String
        let label: String
        let color: Color
        var id: String { key }
    }

    private static let reactionDefs: [ReactionDef] = [
        ReactionDef(key: "more", icon: "arrow.up.circle", label: "More", color: .green),
        ReactionDef(key: "less", icon: "arrow.down.circle", label: "Less", color: .orange),
        ReactionDef(key: "stale", icon: "repeat.circle", label: "Stale", color: .yellow),
        ReactionDef(key: "not_for_me", icon: "xmark.circle", label: "Not for me", color: .red),
    ]

    var body: some View {
        if style.isFeed {
            feedLayout
        } else {
            detailLayout
        }
    }

    // MARK: Feed Layout (compact trigger → floating picker)

    @ViewBuilder
    private var feedLayout: some View {
        feedTrigger
            .overlay(alignment: .bottomTrailing) {
                if isExpanded {
                    Color.clear
                        .frame(width: 320, height: 320)
                        .contentShape(Rectangle())
                        .onTapGesture {
                            withAnimation(.spring(response: 0.35, dampingFraction: 0.75)) {
                                isExpanded = false
                            }
                        }
                        .offset(x: 100, y: 100)
                }
            }
            .overlay(alignment: .bottomTrailing) {
                if isExpanded {
                    floatingPicker
                        .offset(y: -48)
                        .transition(.scale(scale: 0.5, anchor: .bottomTrailing).combined(with: .opacity))
                }
            }
            .animation(.spring(response: 0.35, dampingFraction: 0.75), value: isExpanded)
    }

    @ViewBuilder
    private var feedTrigger: some View {
        if let active = activeReaction,
           let reaction = Self.reactionDefs.first(where: { $0.key == active }) {
            Button {
                withAnimation(.spring(response: 0.35, dampingFraction: 0.75)) {
                    isExpanded.toggle()
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
            .buttonStyle(.plain)
            .accessibilityLabel("Reacted with \(reaction.label). Double tap to change.")
        } else {
            Button {
                withAnimation(.spring(response: 0.35, dampingFraction: 0.75)) {
                    isExpanded.toggle()
                }
            } label: {
                Image(systemName: isExpanded ? "face.smiling.fill" : "face.smiling")
                    .font(.footnote)
                    .foregroundColor(style.isDark ? .white.opacity(0.4) : .secondary)
                    .contentTransition(.symbolEffect(.replace))
                    .frame(minWidth: 44, minHeight: 44)
                    .contentShape(Rectangle())
            }
            .buttonStyle(.plain)
            .accessibilityLabel("React to this post")
        }
    }

    private var floatingPicker: some View {
        HStack(spacing: 8) {
            ForEach(Self.reactionDefs) { reaction in
                Button {
                    selectReaction(reaction)
                } label: {
                    let isActive = activeReaction == reaction.key
                    VStack(spacing: 4) {
                        Image(systemName: isActive ? reaction.icon + ".fill" : reaction.icon)
                            .font(.body)
                            .contentTransition(.symbolEffect(.replace))
                        Text(reaction.label)
                            .font(.caption2)
                    }
                    .foregroundColor(isActive ? reaction.color : (style.isDark ? .white.opacity(0.5) : .secondary))
                    .frame(minWidth: 56, minHeight: 56)
                    .background(isActive ? reaction.color.opacity(0.12) : .clear)
                    .clipShape(RoundedRectangle(cornerRadius: 10))
                }
                .buttonStyle(.plain)
                .accessibilityLabel(reaction.label)
            }
        }
        .padding(.horizontal, 12)
        .padding(.vertical, 8)
        .background(pickerBackground)
        .clipShape(RoundedRectangle(cornerRadius: 16))
        .shadow(color: .black.opacity(0.12), radius: 12, y: 4)
    }

    @ViewBuilder
    private var pickerBackground: some View {
        if style.isDark {
            RoundedRectangle(cornerRadius: 16).fill(Color.black.opacity(0.7))
        } else {
            RoundedRectangle(cornerRadius: 16).fill(.ultraThinMaterial)
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
            isExpanded = false
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
}

// MARK: - Standard Card (card, article, comparison)

private struct StandardCard: View {
    let post: Post

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            CardHeader(post: post)

            Text(post.title)
                .font(.headline)
                .lineLimit(2)

            Text(post.body)
                .font(.subheadline)
                .foregroundColor(.secondary)
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
                .foregroundColor(.mint)
                .padding(.horizontal, 8)
                .padding(.vertical, 4)
                .background(.mint.opacity(0.1))
                .cornerRadius(6)
            }

            CardFooter(post: post)
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 12)
    }
}

// MARK: - Weather Card

private struct WeatherCard: View {
    let post: Post

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            CardHeader(post: post)

            HStack(alignment: .top, spacing: 12) {
                let weather = WeatherInfo.detect(from: post.title + " " + post.body)
                Image(systemName: weather.icon)
                    .font(.system(size: 32))
                    .foregroundStyle(weather.primaryColor, weather.secondaryColor)
                    .frame(width: 44)

                VStack(alignment: .leading, spacing: 4) {
                    Text(post.title)
                        .font(.headline)
                        .lineLimit(2)
                    Text(post.body)
                        .font(.subheadline)
                        .foregroundColor(.secondary)
                        .lineLimit(4)
                }
            }

            CardFooter(post: post)
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 12)
        .background(
            LinearGradient(
                colors: [.cyan.opacity(0.08), .orange.opacity(0.05)],
                startPoint: .topLeading,
                endPoint: .bottomTrailing
            )
        )
    }
}

// MARK: - Compact Card (brief, digest)

private struct CompactCard: View {
    let post: Post

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            CardHeader(post: post)

            Text(post.title)
                .font(.subheadline.weight(.semibold))
                .lineLimit(1)

            // Show body as compact bullets — split on newlines
            let lines = post.body.components(separatedBy: "\n").filter { !$0.trimmingCharacters(in: .whitespaces).isEmpty }
            VStack(alignment: .leading, spacing: 2) {
                ForEach(Array(lines.prefix(5).enumerated()), id: \.offset) { index, line in
                    HStack(alignment: .top, spacing: 6) {
                        if post.displayHintValue == .digest {
                            Text("\(index + 1).")
                                .font(.caption2.weight(.bold))
                                .foregroundColor(post.hintColor)
                                .frame(width: 16, alignment: .trailing)
                        } else {
                            Text("\u{2022}")
                                .font(.caption2)
                                .foregroundColor(.secondary)
                        }
                        Text(line.trimmingCharacters(in: .whitespaces))
                            .font(.caption)
                            .foregroundColor(.secondary)
                            .lineLimit(1)
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
        .padding(.vertical, 12)
    }
}

// MARK: - Date Card (calendar, event)

private struct DateCard: View {
    let post: Post

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            CardHeader(post: post)

            HStack(alignment: .top, spacing: 12) {
                // Date badge
                VStack(spacing: 2) {
                    Text(dateParts.month)
                        .font(.caption2.weight(.bold))
                        .foregroundColor(post.hintColor)
                        .textCase(.uppercase)
                    Text(dateParts.day)
                        .font(.title2.weight(.bold))
                        .foregroundColor(.primary)
                }
                .frame(width: 48, height: 52)
                .background(post.hintColor.opacity(0.1))
                .cornerRadius(8)

                VStack(alignment: .leading, spacing: 6) {
                    Text(post.title)
                        .font(.headline)
                        .lineLimit(2)

                    Text(post.body)
                        .font(.subheadline)
                        .foregroundColor(.secondary)
                        .lineLimit(3)

                    // Event: show location + external link
                    if post.displayHintValue == .event {
                        if let locality = post.locality, !locality.isEmpty {
                            Label(locality, systemImage: "location")
                                .font(.caption)
                                .foregroundColor(post.hintColor)
                        }
                        if let extURL = post.externalURL, !extURL.isEmpty {
                            Label("Get Tickets", systemImage: "arrow.up.right.square")
                                .font(.caption.weight(.medium))
                                .foregroundColor(post.hintColor)
                        }
                    }
                }
            }

            CardFooter(post: post)
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 12)
    }

    private var dateParts: (month: String, day: String) {
        // Try to extract a date from the title first (e.g. "April 16" or "May 3")
        if let titleDate = Self.extractDate(from: post.title) {
            let cal = Calendar.current
            let monthF = DateFormatter()
            monthF.dateFormat = "MMM"
            return (monthF.string(from: titleDate), "\(cal.component(.day, from: titleDate))")
        }

        // Fall back to createdAt
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
        let cal = Calendar.current
        let monthF = DateFormatter()
        monthF.dateFormat = "MMM"
        return (monthF.string(from: date), "\(cal.component(.day, from: date))")
    }

    /// Try to find a date like "April 16" or "Jan 3" in text
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

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            CardHeader(post: post)

            // Deal accent banner
            HStack(spacing: 6) {
                Image(systemName: "tag.fill")
                    .font(.caption)
                Text("DEAL")
                    .font(.caption.weight(.black))
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
                .font(.headline)
                .lineLimit(2)

            Text(post.body)
                .font(.subheadline)
                .foregroundColor(.secondary)
                .lineLimit(3)

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

            CardFooter(post: post)
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 12)
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
    @AppStorage var isBookmarked: Bool
    @EnvironmentObject private var eventTracker: EventTracker
    @EnvironmentObject private var apiService: APIService

    init(post: Post, tintColor: Color) {
        self.post = post
        self.tintColor = tintColor
        self._isBookmarked = AppStorage(wrappedValue: false, "bookmark_\(post.id)")
    }

    var body: some View {
        Button {
            UIImpactFeedbackGenerator(style: .light).impactOccurred()
            isBookmarked.toggle()
            let eventType = isBookmarked ? "save" : "unsave"
            Task { await apiService.trackEvent(postID: post.id, eventType: eventType) }
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
