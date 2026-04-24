import SwiftUI

enum BBBDesign {
    static let background = Color(red: 0.965, green: 0.957, blue: 0.937) // #f6f4ef
    static let surface = Color(red: 1.0, green: 0.996, blue: 0.984)
    static let sunken = Color(red: 0.937, green: 0.925, blue: 0.902) // #efece6
    static let ink = Color(red: 0.102, green: 0.094, blue: 0.082) // #1a1815
    static let ink2 = Color(red: 0.318, green: 0.302, blue: 0.275) // #514d46
    static let ink3 = Color(red: 0.541, green: 0.522, blue: 0.486) // #8a857c
    static let line = Color(red: 0.102, green: 0.094, blue: 0.082).opacity(0.08)
    static let lineStrong = Color(red: 0.102, green: 0.094, blue: 0.082).opacity(0.14)
    static let clay = Color(red: 0.757, green: 0.361, blue: 0.204)

    static let cardRadius: CGFloat = 18
    static let innerRadius: CGFloat = 12

    static let cardShadow = Color(red: 0.102, green: 0.094, blue: 0.082).opacity(0.055)
    static let cardShadowDeep = Color(red: 0.102, green: 0.094, blue: 0.082).opacity(0.085)

    static let reactionMore = Color(red: 0.176, green: 0.604, blue: 0.357)
    static let reactionLess = Color(red: 0.737, green: 0.493, blue: 0.149)
    static let reactionStale = Color(red: 0.682, green: 0.577, blue: 0.151)
    static let reactionNotForMe = Color(red: 0.741, green: 0.286, blue: 0.169)

    static func roleColor(for hint: Post.DisplayHintValue, fallback: Color) -> Color {
        switch hint {
        case .weather:
            return Color(red: 0.188, green: 0.561, blue: 0.651)
        case .calendar, .event, .destination:
            return Color(red: 0.169, green: 0.404, blue: 0.682)
        case .deal:
            return Color(red: 0.765, green: 0.314, blue: 0.184)
        case .digest, .brief, .standings:
            return Color(red: 0.42, green: 0.39, blue: 0.33)
        case .comparison, .fitness:
            return Color(red: 0.176, green: 0.604, blue: 0.357)
        case .outfit:
            return Color(red: 0.694, green: 0.204, blue: 0.471)
        case .scoreboard, .matchup, .boxScore, .playerSpotlight:
            return Color(red: 0.706, green: 0.251, blue: 0.176)
        case .movie, .show, .entertainment, .album, .concert:
            return Color(red: 0.576, green: 0.251, blue: 0.667)
        case .restaurant:
            return Color(red: 0.725, green: 0.451, blue: 0.118)
        case .science:
            return Color(red: 0.255, green: 0.447, blue: 0.675)
        case .petSpotlight:
            return Color(red: 0.765, green: 0.518, blue: 0.145)
        case .feedback:
            return Color(red: 0.361, green: 0.310, blue: 0.753)
        case .creatorSpotlight:
            return Color(red: 0.588, green: 0.239, blue: 0.714)
        case .videoEmbed:
            return Color(red: 0.725, green: 0.102, blue: 0.118)
        default:
            return fallback
        }
    }
}

struct BBBCardChassis: ViewModifier {
    func body(content: Content) -> some View {
        content
            .frame(maxWidth: .infinity, alignment: .leading)
            .background(BBBDesign.surface)
            .clipShape(RoundedRectangle(cornerRadius: BBBDesign.cardRadius, style: .continuous))
            .overlay {
                RoundedRectangle(cornerRadius: BBBDesign.cardRadius, style: .continuous)
                    .stroke(BBBDesign.line, lineWidth: 1)
            }
            .shadow(color: BBBDesign.cardShadow, radius: 20, x: 0, y: 5)
            .shadow(color: BBBDesign.cardShadowDeep.opacity(0.35), radius: 1, x: 0, y: 1)
    }
}

extension View {
    func bbbCardChassis() -> some View {
        modifier(BBBCardChassis())
    }
}
