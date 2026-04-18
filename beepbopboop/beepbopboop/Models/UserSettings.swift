import Foundation

struct UserSettings: Codable {
    var locationName: String?
    var latitude: Double?
    var longitude: Double?
    var radiusKm: Double
    var followedTeams: [String]?
    var notificationsEnabled: Bool
    var digestHour: Int

    var hasLocation: Bool {
        latitude != nil && longitude != nil
    }

    init(
        locationName: String? = nil,
        latitude: Double? = nil,
        longitude: Double? = nil,
        radiusKm: Double = 25,
        followedTeams: [String]? = nil,
        notificationsEnabled: Bool = true,
        digestHour: Int = 8
    ) {
        self.locationName = locationName
        self.latitude = latitude
        self.longitude = longitude
        self.radiusKm = radiusKm
        self.followedTeams = followedTeams
        self.notificationsEnabled = notificationsEnabled
        self.digestHour = digestHour
    }

    enum CodingKeys: String, CodingKey {
        case locationName = "location_name"
        case latitude
        case longitude
        case radiusKm = "radius_km"
        case followedTeams = "followed_teams"
        case notificationsEnabled = "notifications_enabled"
        case digestHour = "digest_hour"
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        locationName = try c.decodeIfPresent(String.self, forKey: .locationName)
        latitude = try c.decodeIfPresent(Double.self, forKey: .latitude)
        longitude = try c.decodeIfPresent(Double.self, forKey: .longitude)
        radiusKm = (try? c.decode(Double.self, forKey: .radiusKm)) ?? 25
        followedTeams = try c.decodeIfPresent([String].self, forKey: .followedTeams)
        notificationsEnabled = (try? c.decode(Bool.self, forKey: .notificationsEnabled)) ?? true
        digestHour = (try? c.decode(Int.self, forKey: .digestHour)) ?? 8
    }
}
