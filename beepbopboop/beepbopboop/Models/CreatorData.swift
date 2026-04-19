import Foundation

struct CreatorData: Codable {
    let designation: String
    let links: CreatorLinks?
    let notableWorks: String?
    let tags: [String]?
    let source: String?
    let areaName: String?

    enum CodingKeys: String, CodingKey {
        case designation
        case links
        case notableWorks = "notable_works"
        case tags
        case source
        case areaName = "area_name"
    }
}

struct CreatorLinks: Codable {
    let website: String?
    let instagram: String?
    let bandcamp: String?
    let etsy: String?
    let substack: String?
    let soundcloud: String?
    let behance: String?
}
