import Foundation

struct CreatorData: Codable {
    let id: String
    let name: String
    let designation: String     // "painter", "musician", "author", etc.
    let bio: String
    let lat: Double
    let lon: Double
    let areaName: String
    let links: CreatorLinks
    let notableWorks: String?
    let tags: [String]?
    let source: String?
    let discoveredAt: String?
    let discoveredByUserId: String?

    enum CodingKeys: String, CodingKey {
        case id, name, designation, bio, lat, lon
        case areaName = "area_name"
        case links
        case notableWorks = "notable_works"
        case tags, source
        case discoveredAt = "discovered_at"
        case discoveredByUserId = "discovered_by_user_id"
    }

    /// Emoji icon representing the creator's designation.
    var designationIcon: String {
        switch designation.lowercased() {
        case "musician", "singer", "songwriter", "composer", "band":
            return "🎵"
        case "writer", "author", "poet", "journalist":
            return "✍️"
        case "visual artist", "painter", "illustrator", "sculptor":
            return "🎨"
        case "photographer":
            return "📷"
        case "ceramicist", "potter":
            return "🏺"
        case "designer":
            return "✏️"
        case "filmmaker", "videographer":
            return "🎬"
        case "dancer", "choreographer":
            return "💃"
        case "comedian", "stand-up":
            return "🎤"
        case "chef", "cook":
            return "🍳"
        default:
            return "🌟"
        }
    }

    /// SF Symbol name for the designation.
    var designationSymbol: String {
        switch designation.lowercased() {
        case "musician", "singer", "songwriter", "composer", "band":
            return "music.note"
        case "writer", "author", "poet", "journalist":
            return "book"
        case "visual artist", "painter", "illustrator", "sculptor":
            return "paintbrush"
        case "photographer":
            return "camera"
        case "ceramicist", "potter":
            return "laurel.leading"
        case "designer":
            return "pencil.and.ruler"
        case "filmmaker", "videographer":
            return "film"
        case "dancer", "choreographer":
            return "figure.dance"
        default:
            return "star"
        }
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

    /// Returns all non-nil link pairs as (label, URL) tuples.
    var allLinks: [(label: String, url: URL)] {
        var result: [(String, URL)] = []
        if let w = website, let u = URL(string: w)   { result.append(("Website", u)) }
        if let i = instagram, let u = URL(string: i)  { result.append(("Instagram", u)) }
        if let b = bandcamp, let u = URL(string: b)   { result.append(("Bandcamp", u)) }
        if let e = etsy, let u = URL(string: e)       { result.append(("Etsy", u)) }
        if let s = substack, let u = URL(string: s)   { result.append(("Substack", u)) }
        if let sc = soundcloud, let u = URL(string: sc) { result.append(("SoundCloud", u)) }
        if let bh = behance, let u = URL(string: bh)  { result.append(("Behance", u)) }
        return result
    }
}
