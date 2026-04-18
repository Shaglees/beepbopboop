import Foundation

struct TravelData: Codable {
    let city: String
    let country: String
    let latitude: Double
    let longitude: Double
    let heroImageUrl: String?
    let currentTempC: Double?
    let currentCondition: String?
    let currentConditionCode: Int?
    let weekendForecast: String?
    let bestTimeToVisit: String?
    let knownFor: [String]
    let flightPriceFrom: String?
    let flightPriceNote: String?
    let currency: String?
    let timeZone: String?
    let visaRequired: Bool?
    let wikiUrl: String?
}
