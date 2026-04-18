import Foundation
import SwiftUI

struct EntertainmentData: Codable {
    let subject: String
    let subjectImageUrl: String?
    let headline: String
    let source: String
    let sourceUrl: String?
    let publishedAt: String?
    let category: String?       // "award" | "appearance" | "project" | "social" | "news"
    let quote: String?
    let relatedProject: String?
    let tags: [String]?

    var categoryBadgeColor: Color {
        switch category {
        case "award":      return Color(hexString: "#F59E0B")
        case "project":    return Color(hexString: "#8B5CF6")
        case "appearance": return Color(hexString: "#EC4899")
        default:           return Color(hexString: "#6B7280")
        }
    }

    var categoryLabel: String {
        switch category {
        case "award":      return "🏆 AWARD"
        case "project":    return "🎬 PROJECT"
        case "appearance": return "👠 APPEARANCE"
        case "social":     return "📱 SOCIAL"
        default:           return "📰 NEWS"
        }
    }
}
