import SwiftUI

struct OnboardingInterestCategory: Identifiable {
    let id: String
    let name: String
    let icon: String
    let topics: [String]
    let previewHints: [String]
}

private let interestCategories: [OnboardingInterestCategory] = [
    OnboardingInterestCategory(id: "sports", name: "Sports", icon: "sportscourt", topics: ["NBA", "NFL", "MLB", "Premier League", "MLS"], previewHints: ["scoreboard", "matchup", "player_spotlight"]),
    OnboardingInterestCategory(id: "food", name: "Food", icon: "fork.knife", topics: ["Ramen", "Italian", "Vegan", "Coffee", "Bakeries"], previewHints: ["restaurant", "deal"]),
    OnboardingInterestCategory(id: "music", name: "Music", icon: "music.note", topics: ["Indie Rock", "Hip Hop", "Electronic", "Jazz", "Classical"], previewHints: ["album", "concert"]),
    OnboardingInterestCategory(id: "science", name: "Science", icon: "atom", topics: ["Space", "Biology", "Climate", "AI", "Physics"], previewHints: ["science"]),
    OnboardingInterestCategory(id: "travel", name: "Travel", icon: "airplane", topics: ["Europe", "Asia", "Budget", "Road Trips", "Adventure"], previewHints: ["destination"]),
    OnboardingInterestCategory(id: "fitness", name: "Fitness", icon: "figure.run", topics: ["Running", "Cycling", "Yoga", "Gym", "Swimming"], previewHints: ["fitness"]),
    OnboardingInterestCategory(id: "pets", name: "Pets", icon: "pawprint", topics: ["Dogs", "Cats", "Adoption", "Training"], previewHints: ["pet_spotlight"]),
    OnboardingInterestCategory(id: "fashion", name: "Fashion", icon: "tshirt", topics: ["Streetwear", "Minimalist", "Vintage", "Sustainable"], previewHints: ["outfit"]),
    OnboardingInterestCategory(id: "entertainment", name: "Entertainment", icon: "film", topics: ["Movies", "TV Shows", "Podcasts", "Gaming"], previewHints: ["movie", "show"]),
    OnboardingInterestCategory(id: "tech", name: "Tech", icon: "desktopcomputer", topics: ["AI", "Startups", "Open Source", "Gadgets"], previewHints: ["article"]),
]

struct OnboardingInterestsView: View {
    @Binding var interests: [UserInterest]
    let onNext: () -> Void
    @State private var selectedCategories: Set<String> = []
    @State private var expandedCategory: String? = nil

    var body: some View {
        ScrollView {
            VStack(spacing: 24) {
                Text("What interests you?")
                    .font(.system(size: 28, weight: .bold, design: .serif))
                    .padding(.top, 20)
                Text("Pick at least 3. Tap to preview what you'll see.")
                    .font(.system(size: 15))
                    .foregroundStyle(.secondary)

                LazyVGrid(columns: [GridItem(.adaptive(minimum: 100), spacing: 12)], spacing: 12) {
                    ForEach(interestCategories) { cat in
                        Button {
                            if expandedCategory == cat.id {
                                expandedCategory = nil
                            } else {
                                expandedCategory = cat.id
                                selectedCategories.insert(cat.id)
                            }
                        } label: {
                            VStack(spacing: 6) {
                                Image(systemName: cat.icon)
                                    .font(.title2)
                                Text(cat.name)
                                    .font(.system(size: 13, weight: .medium))
                            }
                            .frame(maxWidth: .infinity)
                            .padding(.vertical, 16)
                            .background(selectedCategories.contains(cat.id) ? Color.accentColor.opacity(0.15) : Color(.systemGray6))
                            .clipShape(RoundedRectangle(cornerRadius: 12))
                            .overlay(
                                RoundedRectangle(cornerRadius: 12)
                                    .stroke(selectedCategories.contains(cat.id) ? Color.accentColor : .clear, lineWidth: 2)
                            )
                        }
                        .buttonStyle(.plain)
                    }
                }
                .padding(.horizontal)

                if let catID = expandedCategory, let cat = interestCategories.first(where: { $0.id == catID }) {
                    VStack(alignment: .leading, spacing: 8) {
                        Text("\(cat.name) cards you'll see:")
                            .font(.system(size: 13, weight: .medium, design: .monospaced))
                            .foregroundStyle(.secondary)
                            .padding(.horizontal)

                        ScrollView(.horizontal, showsIndicators: false) {
                            HStack(spacing: 12) {
                                ForEach(cat.previewHints, id: \.self) { hint in
                                    RoundedRectangle(cornerRadius: 16)
                                        .fill(Color(.systemGray5))
                                        .frame(width: 240, height: 160)
                                        .overlay(
                                            Text(hint.replacingOccurrences(of: "_", with: " ").capitalized)
                                                .font(.system(size: 15, weight: .semibold, design: .serif))
                                        )
                                }
                            }
                            .padding(.horizontal)
                        }

                        FlowLayout(spacing: 8) {
                            ForEach(cat.topics, id: \.self) { topic in
                                let isSelected = interests.contains(where: { $0.category == cat.id && $0.topic == topic })
                                Button {
                                    if isSelected {
                                        interests.removeAll { $0.category == cat.id && $0.topic == topic }
                                    } else {
                                        interests.append(UserInterest(id: UUID().uuidString, category: cat.id, topic: topic, source: "user", confidence: 1.0, pausedUntil: nil))
                                    }
                                } label: {
                                    Text(topic)
                                        .font(.system(size: 13))
                                        .padding(.horizontal, 12)
                                        .padding(.vertical, 6)
                                        .background(isSelected ? Color.accentColor.opacity(0.2) : Color(.systemGray6))
                                        .clipShape(Capsule())
                                }
                                .buttonStyle(.plain)
                            }
                        }
                        .padding(.horizontal)
                    }
                }

                Button("Continue") {
                    for catID in selectedCategories {
                        if !interests.contains(where: { $0.category == catID }) {
                            interests.append(UserInterest(id: UUID().uuidString, category: catID, topic: catID, source: "user", confidence: 1.0, pausedUntil: nil))
                        }
                    }
                    onNext()
                }
                .disabled(selectedCategories.count < 3)
                .buttonStyle(.borderedProminent)
                .padding(.bottom, 40)
            }
        }
    }
}

struct FlowLayout: Layout {
    var spacing: CGFloat = 8
    func sizeThatFits(proposal: ProposedViewSize, subviews: Subviews, cache: inout ()) -> CGSize {
        var width: CGFloat = 0
        var height: CGFloat = 0
        var rowHeight: CGFloat = 0
        var rowWidth: CGFloat = 0
        let maxWidth = proposal.width ?? .infinity
        for sub in subviews {
            let size = sub.sizeThatFits(.unspecified)
            if rowWidth + size.width > maxWidth {
                width = max(width, rowWidth - spacing)
                height += rowHeight + spacing
                rowWidth = 0; rowHeight = 0
            }
            rowWidth += size.width + spacing
            rowHeight = max(rowHeight, size.height)
        }
        height += rowHeight
        return CGSize(width: max(width, rowWidth - spacing), height: height)
    }
    func placeSubviews(in bounds: CGRect, proposal: ProposedViewSize, subviews: Subviews, cache: inout ()) {
        var x = bounds.minX; var y = bounds.minY; var rowHeight: CGFloat = 0
        for sub in subviews {
            let size = sub.sizeThatFits(.unspecified)
            if x + size.width > bounds.maxX {
                x = bounds.minX; y += rowHeight + spacing; rowHeight = 0
            }
            sub.place(at: CGPoint(x: x, y: y), proposal: .unspecified)
            x += size.width + spacing
            rowHeight = max(rowHeight, size.height)
        }
    }
}
