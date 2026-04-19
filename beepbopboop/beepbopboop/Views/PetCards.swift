import SwiftUI

// MARK: - Palette

private let petCoral = Color(red: 0.976, green: 0.451, blue: 0.086)  // #F97316
private let petTeal  = Color(red: 0.082, green: 0.718, blue: 0.647)  // #14B8A6
private let petCream = Color(red: 1.0, green: 0.984, blue: 0.961)    // #FFFBF5

// MARK: - Pet Spotlight Card

struct PetSpotlightCard: View {
    let post: Post
    let pet: PetData

    init?(post: Post) {
        guard post.displayHintValue == .petSpotlight, let pd = post.petData else { return nil }
        self.post = post
        self.pet = pd
    }

    var body: some View {
        if pet.type == "tip" {
            TipLayout(post: post, pet: pet)
        } else {
            AdoptionLayout(post: post, pet: pet)
        }
    }
}

// MARK: - Adoption Layout

private struct AdoptionLayout: View {
    let post: Post
    let pet: PetData

    var body: some View {
        VStack(spacing: 0) {
            photoSection
            infoSection
        }
        .background(petCream)
    }

    // MARK: Photo

    private var photoSection: some View {
        ZStack(alignment: .topLeading) {
            petPhoto
            ageBadge
            speciesBadge
        }
        .frame(height: 220)
        .clipped()
    }

    @ViewBuilder
    private var petPhoto: some View {
        if let urlStr = pet.photoUrl, let url = URL(string: urlStr) {
            AsyncImage(url: url) { phase in
                switch phase {
                case .success(let image):
                    image
                        .resizable()
                        .aspectRatio(contentMode: .fill)
                        .frame(maxWidth: .infinity, maxHeight: 220)
                        .clipped()
                case .failure:
                    speciesPlaceholder
                default:
                    Color(red: 0.95, green: 0.93, blue: 0.90)
                        .overlay(ProgressView().tint(petCoral))
                }
            }
        } else {
            speciesPlaceholder
        }
    }

    private var speciesPlaceholder: some View {
        Color(red: 0.95, green: 0.93, blue: 0.90)
            .overlay(Text(speciesEmoji).font(.system(size: 64)))
    }

    private var ageBadge: some View {
        Group {
            if let age = pet.age, let gender = pet.gender {
                Text("\(age) · \(gender)")
                    .font(.caption2.weight(.semibold))
                    .foregroundColor(.primary)
                    .padding(.horizontal, 10)
                    .padding(.vertical, 5)
                    .background(petCream.opacity(0.92))
                    .clipShape(Capsule())
                    .padding(10)
            }
        }
    }

    private var speciesBadge: some View {
        HStack {
            Spacer()
            Text(speciesEmoji)
                .font(.title2)
                .padding(10)
        }
    }

    // MARK: Info

    private var infoSection: some View {
        VStack(alignment: .leading, spacing: 10) {
            adoptionCardHeader(post: post)
            if let name = pet.name {
                Text(name)
                    .font(.system(size: 22, weight: .semibold))
            }
            if let breed = pet.breed {
                let sizeLabel = pet.size.map { " · \($0)" } ?? ""
                Text(breed + sizeLabel)
                    .font(.subheadline)
                    .foregroundColor(.secondary)
            }
            if let attrs = pet.attributes {
                PetAttributeGrid(attributes: attrs)
            }
            shelterStrip
            adoptButton
            PetCardFooter(post: post, accentColor: petCoral)
        }
        .padding(16)
    }

    private var shelterStrip: some View {
        Group {
            if let shelter = pet.shelterName, let city = pet.shelterCity {
                HStack(spacing: 4) {
                    Image(systemName: "location.fill")
                        .font(.caption2)
                        .foregroundColor(petCoral)
                    Text("\(shelter) · \(city)")
                        .font(.caption)
                        .foregroundColor(.secondary)
                    if let phone = pet.shelterPhone,
                       let tel = URL(string: "tel:\(phone.filter { $0.isNumber || $0 == "+" })") {
                        Spacer()
                        Link(destination: tel) {
                            Image(systemName: "phone.fill")
                                .font(.caption)
                                .foregroundColor(petTeal)
                        }
                    }
                }
            }
        }
    }

    private var adoptButton: some View {
        Group {
            if let urlStr = pet.petfinderUrl, let url = URL(string: urlStr), let name = pet.name {
                Link(destination: url) {
                    HStack {
                        Text("Meet \(name) on Petfinder")
                            .font(.subheadline.weight(.semibold))
                        Image(systemName: "arrow.up.right")
                            .font(.caption.weight(.bold))
                    }
                    .foregroundColor(.white)
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 12)
                    .background(petCoral)
                    .cornerRadius(12)
                }
                .buttonStyle(.plain)
            }
        }
    }

    private var speciesEmoji: String {
        switch pet.species?.lowercased() {
        case "cat":    return "🐱"
        case "rabbit": return "🐰"
        case "bird":   return "🐦"
        default:       return "🐾"
        }
    }
}

// MARK: - Attribute Grid

private struct PetAttributeGrid: View {
    let attributes: PetAttributes

    private struct Item: Identifiable {
        let id: String
        let label: String
        let icon: String
        let value: Bool?
    }

    private var items: [Item] {
        [
            Item(id: "shots",  label: "Vaccinated",    icon: "syringe",                          value: attributes.shotsCurrent),
            Item(id: "house",  label: "House trained",  icon: "house",                            value: attributes.houseTrained),
            Item(id: "kids",   label: "Good w/ kids",   icon: "figure.2.and.child.holdinghands",  value: attributes.goodWithChildren),
            Item(id: "dogs",   label: "Good w/ dogs",   icon: "pawprint",                         value: attributes.goodWithDogs),
            Item(id: "cats",   label: "Good w/ cats",   icon: "cat",                              value: attributes.goodWithCats),
        ]
    }

    var body: some View {
        HStack(spacing: 8) {
            ForEach(items) { item in
                if let val = item.value {
                    VStack(spacing: 3) {
                        ZStack {
                            Circle()
                                .fill(val ? petTeal.opacity(0.12) : Color.red.opacity(0.10))
                                .frame(width: 32, height: 32)
                            Image(systemName: val ? item.icon : "xmark")
                                .font(.system(size: 12, weight: .medium))
                                .foregroundColor(val ? petTeal : .red)
                        }
                        Text(item.label)
                            .font(.system(size: 8, weight: .medium))
                            .foregroundColor(.secondary)
                            .multilineTextAlignment(.center)
                            .fixedSize(horizontal: false, vertical: true)
                    }
                    .frame(maxWidth: .infinity)
                }
            }
        }
        .padding(.vertical, 4)
    }
}

// MARK: - Tip Layout

private struct TipLayout: View {
    let post: Post
    let pet: PetData

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            tipCardHeader(post: post)
            iconHeader
            Text(post.title)
                .font(.system(size: 18, weight: .semibold))
            Text(post.body)
                .font(.subheadline)
                .foregroundColor(.secondary)
                .lineLimit(4)
            if let org = pet.sourceOrg, let urlStr = pet.sourceUrl, let url = URL(string: urlStr) {
                Link("Source: \(org)", destination: url)
                    .font(.caption)
                    .foregroundColor(petTeal)
                    .buttonStyle(.plain)
            }
            if let tags = pet.tags, !tags.isEmpty {
                tagRow(tags: tags)
            }
            PetCardFooter(post: post, accentColor: petTeal)
        }
        .padding(16)
        .background(petCream)
    }

    private var iconHeader: some View {
        HStack(spacing: 10) {
            ZStack {
                RoundedRectangle(cornerRadius: 12)
                    .fill(petTeal.opacity(0.15))
                    .frame(width: 52, height: 52)
                Text(speciesEmoji)
                    .font(.system(size: 28))
            }
            if let topic = pet.topic {
                Text(topic.capitalized)
                    .font(.caption.weight(.semibold))
                    .foregroundColor(petTeal)
                    .padding(.horizontal, 10)
                    .padding(.vertical, 5)
                    .background(petTeal.opacity(0.12))
                    .cornerRadius(8)
            }
        }
    }

    private func tagRow(tags: [String]) -> some View {
        HStack(spacing: 6) {
            ForEach(tags.prefix(3), id: \.self) { tag in
                Text(tag)
                    .font(.caption2.weight(.medium))
                    .foregroundColor(.secondary)
                    .padding(.horizontal, 8)
                    .padding(.vertical, 4)
                    .background(Color(.systemGray6))
                    .cornerRadius(6)
            }
        }
    }

    private var speciesEmoji: String {
        let list = pet.speciesList ?? []
        if list.contains("cat") { return "🐱" }
        if list.contains("rabbit") { return "🐰" }
        if list.contains("bird") { return "🐦" }
        return "🐾"
    }
}

// MARK: - Card Headers (private helpers)

private func adoptionCardHeader(post: Post) -> some View {
    HStack(spacing: 6) {
        Circle().fill(petCoral).frame(width: 8, height: 8)
        Text(post.agentName).font(.subheadline.weight(.medium))
        Text("Adoption")
            .font(.caption2.weight(.semibold))
            .foregroundColor(petCoral)
            .padding(.horizontal, 7).padding(.vertical, 3)
            .background(petCoral.opacity(0.12)).cornerRadius(4)
        Spacer()
        Text(post.relativeTime).font(.subheadline).foregroundStyle(.tertiary)
    }
}

private func tipCardHeader(post: Post) -> some View {
    HStack(spacing: 6) {
        Circle().fill(petTeal).frame(width: 8, height: 8)
        Text(post.agentName).font(.subheadline.weight(.medium))
        Text("Pet Tip")
            .font(.caption2.weight(.semibold))
            .foregroundColor(petTeal)
            .padding(.horizontal, 7).padding(.vertical, 3)
            .background(petTeal.opacity(0.12)).cornerRadius(4)
        Spacer()
        Text(post.relativeTime).font(.subheadline).foregroundStyle(.tertiary)
    }
}

// MARK: - Shared Card Footer

private struct PetCardFooter: View {
    let post: Post
    let accentColor: Color
    @AppStorage var isBookmarked: Bool
    @State private var activeReaction: String?
    @EnvironmentObject private var apiService: APIService

    init(post: Post, accentColor: Color) {
        self.post = post
        self.accentColor = accentColor
        self._isBookmarked = AppStorage(wrappedValue: post.mySaved, "bookmark_\(post.id)")
        self._activeReaction = State(initialValue: post.myReaction)
    }

    var body: some View {
        HStack(spacing: 6) {
            if let locality = post.locality, !locality.isEmpty {
                Label(locality, systemImage: post.isSourceAttribution ? "link" : "location")
                    .font(.caption2)
                    .foregroundColor(.secondary)
                    .lineLimit(1)
            }
            Spacer()
            ReactionPicker(activeReaction: $activeReaction, postID: post.id, style: .feedCompact)
            Button {
                UIImpactFeedbackGenerator(style: .light).impactOccurred()
                let wasSaved = isBookmarked
                withAnimation(.bouncy) { isBookmarked.toggle() }
                Task {
                    do {
                        if wasSaved {
                            try await apiService.unsavePost(postID: post.id)
                        } else {
                            try await apiService.savePost(postID: post.id)
                        }
                    } catch {
                        isBookmarked = wasSaved
                    }
                }
            } label: {
                Image(systemName: isBookmarked ? "bookmark.fill" : "bookmark")
                    .font(.caption)
                    .foregroundColor(isBookmarked ? accentColor : .secondary)
                    .contentTransition(.symbolEffect(.replace))
            }
            .buttonStyle(.plain)
        }
    }
}
