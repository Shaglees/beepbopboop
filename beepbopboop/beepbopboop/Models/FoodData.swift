import Foundation

struct FoodData: Codable {
    let yelpId: String?
    let name: String
    let imageUrl: String?
    let rating: Double
    let reviewCount: Int
    let cuisine: [String]
    let priceRange: String?
    let address: String
    let neighbourhood: String?
    let distanceM: Double?
    let isOpenNow: Bool?
    let phone: String?
    let yelpUrl: String?
    let latitude: Double
    let longitude: Double
    let mustTry: [String]
    let pricePerHead: String?
    let newOpening: Bool

    enum CodingKeys: String, CodingKey {
        case yelpId, name, imageUrl, rating, reviewCount, cuisine
        case priceRange, address, neighbourhood, distanceM, isOpenNow
        case phone, yelpUrl, latitude, longitude, mustTry, pricePerHead, newOpening
    }
}
