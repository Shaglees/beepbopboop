import Foundation
import SwiftUI

struct ScienceData: Codable {
    let category: String
    let source: String
    let sourceUrl: String?
    let headline: String
    let heroImageUrl: String?
    let publishedAt: String?
    let institution: String?
    let doi: String?
    let arxivId: String?
    let readMoreUrl: String?
    let tags: [String]

    enum CodingKeys: String, CodingKey {
        case category, source, headline, institution, doi, tags
        case sourceUrl = "sourceUrl"
        case heroImageUrl = "heroImageUrl"
        case publishedAt = "publishedAt"
        case arxivId = "arxivId"
        case readMoreUrl = "readMoreUrl"
    }

    var categoryColor: Color {
        switch category.lowercased() {
        case "space": return Color(red: 0.118, green: 0.227, blue: 0.373)   // deep navy #1E3A5F
        case "nature": return Color(red: 0.078, green: 0.325, blue: 0.173)  // deep green #14532D
        case "technology": return Color(red: 0.231, green: 0.027, blue: 0.392) // deep purple #3B0764
        default: return Color(red: 0.486, green: 0.176, blue: 0.071)        // deep orange #7C2D12
        }
    }

    var categoryAccentColor: Color {
        switch category.lowercased() {
        case "space": return Color(red: 0.388, green: 0.671, blue: 0.937)   // sky blue
        case "nature": return Color(red: 0.365, green: 0.847, blue: 0.490)  // leaf green
        case "technology": return Color(red: 0.816, green: 0.537, blue: 0.996) // soft purple
        default: return Color(red: 0.992, green: 0.620, blue: 0.231)        // warm orange
        }
    }

    var categoryIcon: String {
        switch category.lowercased() {
        case "space": return "moon.stars.fill"
        case "nature": return "leaf.fill"
        case "technology": return "cpu.fill"
        default: return "flask.fill"
        }
    }

    var categoryLabel: String {
        switch category.lowercased() {
        case "space": return "Space"
        case "nature": return "Nature"
        case "technology": return "Technology"
        case "research": return "Research"
        default: return category.capitalized
        }
    }

    var formattedDate: String? {
        guard let raw = publishedAt else { return nil }
        let formats = ["yyyy-MM-dd", "yyyy-MM-dd'T'HH:mm:ssZ", "yyyy-MM-dd'T'HH:mm:ss.SSSZ"]
        let output = DateFormatter()
        output.dateFormat = "MMM d, yyyy"
        for format in formats {
            let parser = DateFormatter()
            parser.dateFormat = format
            if let date = parser.date(from: raw) {
                return output.string(from: date)
            }
        }
        return raw
    }

    var doiUrl: URL? {
        guard let doi = doi else { return nil }
        return URL(string: "https://doi.org/\(doi)")
    }

    var arxivUrl: URL? {
        guard let id = arxivId else { return nil }
        return URL(string: "https://arxiv.org/abs/\(id)")
    }

    var primaryReadUrl: URL? {
        if let urlStr = readMoreUrl, let url = URL(string: urlStr) { return url }
        return doiUrl ?? arxivUrl
    }
}
