import Foundation

struct UserProfileIdentity: Codable {
    var displayName: String
    var avatarUrl: String
    var timezone: String
    var homeLocation: String
    var homeLat: Double?
    var homeLon: Double?

    enum CodingKeys: String, CodingKey {
        case displayName = "display_name"
        case avatarUrl = "avatar_url"
        case timezone
        case homeLocation = "home_location"
        case homeLat = "home_lat"
        case homeLon = "home_lon"
    }
}

struct UserInterest: Codable, Identifiable {
    let id: String
    var category: String
    var topic: String
    var source: String
    var confidence: Double
    var pausedUntil: String?

    enum CodingKeys: String, CodingKey {
        case id, category, topic, source, confidence
        case pausedUntil = "paused_until"
    }
}

struct LifestyleTag: Codable {
    var category: String
    var value: String
}

struct ContentPref: Codable {
    var category: String?
    var depth: String
    var tone: String
    var maxPerDay: Int?

    enum CodingKeys: String, CodingKey {
        case category, depth, tone
        case maxPerDay = "max_per_day"
    }
}

struct UserProfile: Codable {
    var identity: UserProfileIdentity
    var interests: [UserInterest]
    var lifestyle: [LifestyleTag]
    var contentPrefs: [ContentPref]
    var profileInitialized: Bool

    enum CodingKeys: String, CodingKey {
        case identity, interests, lifestyle
        case contentPrefs = "content_prefs"
        case profileInitialized = "profile_initialized"
    }
}
