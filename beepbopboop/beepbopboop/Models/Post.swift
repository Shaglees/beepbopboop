import Foundation
import SwiftUI

/// Wraps individual element decoding so one bad item doesn't kill the whole array.
struct SafeDecodable<T: Decodable>: Decodable {
    let value: T?

    init(from decoder: Decoder) throws {
        do {
            value = try T(from: decoder)
        } catch {
            print("[SafeDecodable] skipping bad element: \(error)")
            value = nil
        }
    }
}

struct FeedResponse: Codable {
    let posts: [Post]
    let nextCursor: String?

    enum CodingKeys: String, CodingKey {
        case posts
        case nextCursor = "next_cursor"
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        let safePosts = try container.decode([SafeDecodable<Post>].self, forKey: .posts)
        self.posts = safePosts.compactMap { $0.value }
        self.nextCursor = try container.decodeIfPresent(String.self, forKey: .nextCursor)
    }
}

struct PostImage: Codable {
    let url: String
    let role: String
    let caption: String?
    let link: String?

    enum CodingKeys: String, CodingKey {
        case url, role, caption, link
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        url = try container.decode(String.self, forKey: .url)
        role = try container.decodeIfPresent(String.self, forKey: .role) ?? "detail"
        caption = try container.decodeIfPresent(String.self, forKey: .caption)
        link = try container.decodeIfPresent(String.self, forKey: .link)
    }
}

struct OutfitContent {
    struct Product {
        let name: String
        let price: String
    }

    let trend: String?
    let body: String
    let forYou: String?
    let products: [Product]
    let budgetAlt: Product?

    init(from text: String) {
        var mainBody = ""
        var trend: String?
        var forYou: String?
        var tryLine: String?
        var altLine: String?

        // Parse line by line
        let lines = text.components(separatedBy: "\n")
        var currentMarker: String?
        var currentValue = ""

        for line in lines {
            let trimmed = line.trimmingCharacters(in: .whitespaces)
            if trimmed.hasPrefix("**Trend:**") {
                if currentMarker == nil { mainBody = currentValue.trimmingCharacters(in: .whitespacesAndNewlines) }
                else { Self.assign(marker: currentMarker!, value: currentValue, trend: &trend, forYou: &forYou, tryLine: &tryLine, altLine: &altLine) }
                currentMarker = "trend"
                currentValue = String(trimmed.dropFirst("**Trend:**".count))
            } else if trimmed.hasPrefix("**For you:**") {
                if currentMarker == nil { mainBody = currentValue.trimmingCharacters(in: .whitespacesAndNewlines) }
                else { Self.assign(marker: currentMarker!, value: currentValue, trend: &trend, forYou: &forYou, tryLine: &tryLine, altLine: &altLine) }
                currentMarker = "forYou"
                currentValue = String(trimmed.dropFirst("**For you:**".count))
            } else if trimmed.hasPrefix("**Try:**") {
                if currentMarker == nil { mainBody = currentValue.trimmingCharacters(in: .whitespacesAndNewlines) }
                else { Self.assign(marker: currentMarker!, value: currentValue, trend: &trend, forYou: &forYou, tryLine: &tryLine, altLine: &altLine) }
                currentMarker = "try"
                currentValue = String(trimmed.dropFirst("**Try:**".count))
            } else if trimmed.hasPrefix("**Alt:**") {
                if currentMarker == nil { mainBody = currentValue.trimmingCharacters(in: .whitespacesAndNewlines) }
                else { Self.assign(marker: currentMarker!, value: currentValue, trend: &trend, forYou: &forYou, tryLine: &tryLine, altLine: &altLine) }
                currentMarker = "alt"
                currentValue = String(trimmed.dropFirst("**Alt:**".count))
            } else {
                currentValue += (currentValue.isEmpty ? "" : "\n") + line
            }
        }
        // Flush last marker
        if let marker = currentMarker {
            Self.assign(marker: marker, value: currentValue, trend: &trend, forYou: &forYou, tryLine: &tryLine, altLine: &altLine)
        } else {
            mainBody = currentValue.trimmingCharacters(in: .whitespacesAndNewlines)
        }

        self.trend = trend?.trimmingCharacters(in: .whitespacesAndNewlines)
        self.body = mainBody
        self.forYou = forYou?.trimmingCharacters(in: .whitespacesAndNewlines)

        // Parse products from tryLine
        if let tryText = tryLine?.trimmingCharacters(in: .whitespacesAndNewlines), !tryText.isEmpty {
            self.products = tryText.components(separatedBy: " · ").compactMap { segment in
                Self.parseProduct(from: segment.trimmingCharacters(in: .whitespaces))
            }
        } else {
            self.products = []
        }

        // Parse budget alt
        if let altText = altLine?.trimmingCharacters(in: .whitespacesAndNewlines), !altText.isEmpty {
            self.budgetAlt = Self.parseProduct(from: altText)
        } else {
            self.budgetAlt = nil
        }
    }

    private static func assign(marker: String, value: String, trend: inout String?, forYou: inout String?, tryLine: inout String?, altLine: inout String?) {
        switch marker {
        case "trend": trend = value
        case "forYou": forYou = value
        case "try": tryLine = value
        case "alt": altLine = value
        default: break
        }
    }

    private static func parseProduct(from text: String) -> Product? {
        // Match "Name ($Price)" or "Name ($Price) extra"
        guard let range = text.range(of: #"\(?\$[\d,.]+\)?"#, options: .regularExpression) else {
            return text.isEmpty ? nil : Product(name: text, price: "")
        }
        var price = String(text[range])
        // Clean up parentheses
        price = price.replacingOccurrences(of: "(", with: "").replacingOccurrences(of: ")", with: "")
        let name = text[text.startIndex..<range.lowerBound].trimmingCharacters(in: .whitespacesAndNewlines)
            .trimmingCharacters(in: CharacterSet(charactersIn: "("))
        return name.isEmpty ? nil : Product(name: name, price: price)
    }
}

struct Post: Codable, Identifiable {
    let id: String
    let agentID: String
    let agentName: String
    let userID: String
    let title: String
    let body: String
    let imageURL: String?
    let externalURL: String?
    let locality: String?
    let latitude: Double?
    let longitude: Double?
    let postType: String?
    let visibility: String?
    let displayHint: String?
    let images: [PostImage]?
    let labels: [String]?
    let myReaction: String?
    let sourcePublishedAt: String?
    let saved: Bool?
    let createdAt: String


    enum PostTypeValue {
        case event, place, discovery, article, video
    }

    enum DisplayHintValue {
        case card, place, article, weather, calendar, deal, digest, brief, comparison, event, outfit
        case scoreboard, matchup, standings, playerSpotlight, entertainment, movie, show
        case album, concert
        case gameRelease, gameReview
        case restaurant
        case destination
        case science
        case petSpotlight
        case fitness
        case boxScore
        case feedback
    }

    var displayHintValue: DisplayHintValue {
        switch displayHint?.lowercased() {
        case "place": return .place
        case "article": return .article
        case "weather": return .weather
        case "calendar": return .calendar
        case "deal": return .deal
        case "digest": return .digest
        case "brief": return .brief
        case "comparison": return .comparison
        case "event": return .event
        case "outfit": return .outfit
        case "scoreboard": return .scoreboard
        case "matchup": return .matchup
        case "standings": return .standings
        case "movie": return .movie
        case "show": return .show
        case "player_spotlight": return .playerSpotlight
        case "entertainment": return .entertainment
        case "album": return .album
        case "concert": return .concert
        case "game_release": return .gameRelease
        case "game_review": return .gameReview
        case "restaurant": return .restaurant
        case "destination": return .destination
        case "science": return .science
        case "pet_spotlight": return .petSpotlight
        case "fitness": return .fitness
        case "box_score": return .boxScore
        case "feedback": return .feedback
        default: return .card
        }
    }

    var postTypeValue: PostTypeValue {
        switch postType?.lowercased() {
        case "event": return .event
        case "place": return .place
        case "article": return .article
        case "video": return .video
        default: return .discovery
        }
    }

    /// True when locality is used as a source name (articles/videos) rather than a geographic place.
    var isSourceAttribution: Bool {
        locality != nil && !locality!.isEmpty && latitude == nil && longitude == nil
    }

    // MARK: - Type Display Properties

    var typeColor: Color {
        switch postTypeValue {
        case .event: return .purple
        case .place: return .green
        case .discovery: return .blue
        case .article: return .orange
        case .video: return .red
        }
    }

    var typeLabel: String {
        switch postTypeValue {
        case .event: return "Event"
        case .place: return "Place"
        case .discovery: return "Discovery"
        case .article: return "Article"
        case .video: return "Video"
        }
    }

    var typeIcon: String {
        switch postTypeValue {
        case .event: return "calendar"
        case .place: return "mappin"
        case .discovery: return "sparkles"
        case .article: return "doc.text"
        case .video: return "play.rectangle"
        }
    }

    // MARK: - Hint Display Properties

    var hintColor: Color {
        switch displayHintValue {
        case .card: return typeColor
        case .place: return .green
        case .article: return .orange
        case .weather: return .cyan
        case .calendar: return .indigo
        case .deal: return .pink
        case .digest: return .teal
        case .brief: return .gray
        case .comparison: return .mint
        case .event: return .purple
        case .outfit: return Color(red: 0.878, green: 0.251, blue: 0.984)
        case .scoreboard: return .red
        case .matchup: return .indigo
        case .standings: return .secondary
        case .movie: return Color(red: 0.957, green: 0.62, blue: 0.043)
        case .show: return Color(red: 0.957, green: 0.62, blue: 0.043)
        case .playerSpotlight: return Color(red: 0.0, green: 0.478, blue: 0.757)
        case .entertainment: return Color(red: 0.961, green: 0.620, blue: 0.043)
        case .album: return Color(red: 0.459, green: 0.176, blue: 0.902)
        case .concert: return Color(red: 0.984, green: 0.729, blue: 0.012)
        case .gameRelease: return Color(red: 0.96, green: 0.62, blue: 0.04)
        case .gameReview: return Color(red: 0.58, green: 0.27, blue: 0.96)
        case .restaurant: return Color(red: 0.937, green: 0.267, blue: 0.267)
        case .destination: return Color(red: 0.024, green: 0.714, blue: 0.831)
        case .science: return Color(red: 0.388, green: 0.671, blue: 0.937)
        case .petSpotlight: return Color(red: 0.976, green: 0.451, blue: 0.086)
        case .fitness: return Color(red: 0.133, green: 0.773, blue: 0.369)
        case .boxScore: return Color(red: 0.055, green: 0.337, blue: 0.188)
        case .feedback: return Color(red: 0.365, green: 0.376, blue: 0.996)
        }
    }

    var hintIcon: String {
        switch displayHintValue {
        case .card: return typeIcon
        case .place: return "mappin.and.ellipse"
        case .article: return "newspaper"
        case .weather: return "cloud.sun"
        case .calendar: return "calendar"
        case .deal: return "tag"
        case .digest: return "list.bullet.rectangle"
        case .brief: return "text.alignleft"
        case .comparison: return "arrow.left.arrow.right"
        case .event: return "party.popper"
        case .outfit: return "tshirt"
        case .scoreboard: return "sportscourt"
        case .matchup: return "clock"
        case .standings: return "list.number"
        case .movie: return "film"
        case .show: return "tv"
        case .playerSpotlight: return playerData?.sportIcon ?? "figure.basketball"
        case .entertainment: return "star.fill"
        case .album: return "music.note"
        case .concert: return "music.mic"
        case .gameRelease: return "calendar.badge.clock"
        case .gameReview: return "gamecontroller"
        case .restaurant: return "fork.knife"
        case .destination: return "airplane"
        case .science: return "moon.stars.fill"
        case .petSpotlight: return "pawprint"
        case .fitness: return "figure.run"
        case .boxScore: return "figure.baseball"
        case .feedback: return feedbackData?.feedbackType == "rating" ? "star.fill" : "checklist"
        }
    }

    var hintLabel: String {
        switch displayHintValue {
        case .card: return typeLabel
        case .place: return "Place"
        case .article: return "Article"
        case .weather: return "Weather"
        case .calendar: return "Calendar"
        case .deal: return "Deal"
        case .digest: return "Digest"
        case .brief: return "Brief"
        case .comparison: return "Compare"
        case .event: return "Event"
        case .outfit: return "Outfit"
        case .scoreboard: return "Score"
        case .matchup: return "Matchup"
        case .standings: return "Scores"
        case .movie: return "Movie"
        case .show: return "TV Show"
        case .playerSpotlight: return "Player"
        case .entertainment: return "Entertainment"
        case .album: return "Album"
        case .concert: return "Concert"
        case .gameRelease: return "Release"
        case .gameReview: return "Review"
        case .restaurant: return "Restaurant"
        case .destination: return "Destination"
        case .science: return "Science"
        case .petSpotlight: return "Adoption"
        case .fitness: return "Fitness"
        case .boxScore: return "Box Score"
        case .feedback: return "Quick Question"
        }
    }

    /// Short label for map markers — first component of locality, or post type name
    var markerLabel: String {
        if let locality = locality, !locality.isEmpty {
            return String(locality.split(separator: ",").first ?? Substring(locality))
        }
        return typeLabel
    }

    // MARK: - Share URL

    var shareURL: URL {
        if let raw = externalURL, !raw.isEmpty,
           let url = URL(string: raw), url.scheme?.hasPrefix("http") == true {
            return url
        }
        return URL(string: "https://beepbopboop.app/posts/\(id)")!
    }

    // MARK: - Relative Time

    private static let isoWithFractional: ISO8601DateFormatter = {
        let f = ISO8601DateFormatter()
        f.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        return f
    }()
    private static let isoWithoutFractional: ISO8601DateFormatter = {
        let f = ISO8601DateFormatter()
        f.formatOptions = [.withInternetDateTime]
        return f
    }()
    private static let relativeMonthDayFormatter: DateFormatter = {
        let f = DateFormatter()
        f.dateFormat = "MMM d"
        return f
    }()

    var relativeTime: String {
        let date = Post.isoWithFractional.date(from: createdAt)
            ?? Post.isoWithoutFractional.date(from: createdAt)

        guard let date = date else { return createdAt }

        let now = Date()
        let seconds = Int(now.timeIntervalSince(date))

        if seconds < 60 { return "now" }
        let minutes = seconds / 60
        if minutes < 60 { return "\(minutes)m" }
        let hours = minutes / 60
        if hours < 24 { return "\(hours)h" }
        let days = hours / 24
        if days < 7 { return "\(days)d" }
        let weeks = days / 7
        if weeks < 4 { return "\(weeks)w" }

        return Post.relativeMonthDayFormatter.string(from: date)
    }

    enum CodingKeys: String, CodingKey {
        case id
        case agentID = "agent_id"
        case agentName = "agent_name"
        case userID = "user_id"
        case title
        case body
        case imageURL = "image_url"
        case externalURL = "external_url"
        case locality
        case latitude
        case longitude
        case postType = "post_type"
        case visibility
        case displayHint = "display_hint"
        case images
        case labels
        case myReaction = "my_reaction"
        case sourcePublishedAt = "source_published_at"
        case saved
        case createdAt = "created_at"
    }


    var outfitContent: OutfitContent {
        OutfitContent(from: body)
    }

    /// Parsed weather forecast data from externalURL (for weather display_hint posts).
    var weatherData: WeatherData? {
        guard displayHintValue == .weather,
              let json = externalURL,
              let data = json.data(using: .utf8) else { return nil }
        return try? JSONDecoder().decode(WeatherData.self, from: data)
    }

    /// Parsed game data from externalURL (for scoreboard/matchup display_hint posts).
    var gameData: GameData? {
        guard displayHintValue == .scoreboard || displayHintValue == .matchup,
              let json = externalURL,
              let data = json.data(using: .utf8) else { return nil }
        return try? JSONDecoder().decode(GameData.self, from: data)
    }

    /// Parsed travel destination data from externalURL (for destination display_hint posts).
    var travelData: TravelData? {
        guard displayHintValue == .destination,
              let json = externalURL,
              let data = json.data(using: .utf8) else { return nil }
        return try? JSONDecoder().decode(TravelData.self, from: data)
    }

    /// Parsed science data from externalURL (for science display_hint posts).
    var scienceData: ScienceData? {
        guard displayHintValue == .science,
              let json = externalURL,
              let data = json.data(using: .utf8) else { return nil }
        return try? JSONDecoder().decode(ScienceData.self, from: data)
    }

    /// Parsed standings data from externalURL (for standings display_hint posts).
    var standingsData: StandingsData? {
        guard displayHintValue == .standings,
              let json = externalURL,
              let data = json.data(using: .utf8) else { return nil }
        return try? JSONDecoder().decode(StandingsData.self, from: data)
    }

    /// Parsed media data from externalURL (for movie/show display_hint posts).
    var mediaData: MediaData? {
        guard displayHintValue == .movie || displayHintValue == .show,
              let json = externalURL,
              let data = json.data(using: .utf8) else { return nil }
        return try? JSONDecoder().decode(MediaData.self, from: data)
    }

    /// Parsed player spotlight data from externalURL (for player_spotlight display_hint posts).
    var playerData: PlayerData? {
        guard displayHintValue == .playerSpotlight,
              let json = externalURL,
              let data = json.data(using: .utf8) else { return nil }
        return try? JSONDecoder().decode(PlayerData.self, from: data)
    }

    /// Parsed entertainment data from externalURL (for entertainment display_hint posts).
    var entertainmentData: EntertainmentData? {
        guard displayHintValue == .entertainment,
              let json = externalURL,
              let data = json.data(using: .utf8) else { return nil }
        return try? JSONDecoder().decode(EntertainmentData.self, from: data)
    }

    /// Parsed music data from externalURL (for album/concert display_hint posts).
    var musicData: MusicData? {
        guard displayHintValue == .album || displayHintValue == .concert,
              let json = externalURL,
              let data = json.data(using: .utf8) else { return nil }
        return try? JSONDecoder().decode(MusicData.self, from: data)
    }

    /// Parsed video game data from externalURL (for game_release/game_review display_hint posts).
    var videoGameData: VideoGameData? {
        guard displayHintValue == .gameRelease || displayHintValue == .gameReview,
              let json = externalURL,
              let data = json.data(using: .utf8) else { return nil }
        return try? JSONDecoder().decode(VideoGameData.self, from: data)
    }

    /// Parsed restaurant data from externalURL (for restaurant display_hint posts).
    var foodData: FoodData? {
        guard displayHintValue == .restaurant,
              let json = externalURL,
              let data = json.data(using: .utf8) else { return nil }
        return try? JSONDecoder().decode(FoodData.self, from: data)
    }

    /// Parsed pet data from externalURL (for pet_spotlight display_hint posts).
    var petData: PetData? {
        guard displayHintValue == .petSpotlight,
              let json = externalURL,
              let data = json.data(using: .utf8) else { return nil }
        return try? JSONDecoder().decode(PetData.self, from: data)
    }

    /// Parsed baseball box score data from externalURL (for box_score display_hint posts).
    var baseballData: BaseballData? {
        guard displayHintValue == .boxScore,
              let json = externalURL,
              let data = json.data(using: .utf8) else { return nil }
        return try? JSONDecoder().decode(BaseballData.self, from: data)
    }

    /// Parsed fitness data from externalURL (for fitness display_hint posts).
    var fitnessData: FitnessData? {
        guard displayHintValue == .fitness,
              let json = externalURL,
              let data = json.data(using: .utf8) else { return nil }
        return try? JSONDecoder().decode(FitnessData.self, from: data)
    }

    /// Parsed feedback data from externalURL (for feedback display_hint posts).
    var feedbackData: FeedbackData? {
        guard displayHintValue == .feedback,
              let json = externalURL,
              let data = json.data(using: .utf8) else { return nil }
        return try? JSONDecoder().decode(FeedbackData.self, from: data)
    }

    /// Images filtered by role, with fallback to imageURL
    func imagesByRole(_ role: String) -> [PostImage] {
        images?.filter { $0.role.lowercased() == role.lowercased() } ?? []
    }

    var heroImage: PostImage? {
        imagesByRole("hero").first ?? images?.first
    }
}

