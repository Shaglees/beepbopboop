import Foundation

struct UserSettings: Codable {
    var locationName: String?
    var latitude: Double?
    var longitude: Double?
    var radiusKm: Double
    var followedTeams: [String]?

    var hasLocation: Bool {
        latitude != nil && longitude != nil
    }

    enum CodingKeys: String, CodingKey {
        case locationName = "location_name"
        case latitude
        case longitude
        case radiusKm = "radius_km"
        case followedTeams = "followed_teams"
    }
}
