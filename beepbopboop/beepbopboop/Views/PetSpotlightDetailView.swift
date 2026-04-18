import SwiftUI
import MapKit

struct PetSpotlightDetailView: View {
    let post: Post
    @Environment(\.dismiss) private var dismiss

    private var data: PetData? { post.petData }

    private let petOrange = Color(red: 0.976, green: 0.451, blue: 0.086)

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 0) {
                if let data = data, data.type == "tip" {
                    tipDetailBody(data: data)
                } else if let data = data {
                    adoptionDetailBody(data: data)
                } else {
                    // Fallback
                    fallbackBody
                }
            }
        }
        .ignoresSafeArea(edges: .top)
        .navigationTitle("")
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button { dismiss() } label: {
                    Image(systemName: "xmark.circle.fill")
                        .foregroundStyle(.secondary)
                }
            }
        }
    }

    // MARK: - Adoption Detail

    @ViewBuilder
    private func adoptionDetailBody(data: PetData) -> some View {
        // Hero
        heroSection(data: data)

        VStack(alignment: .leading, spacing: 20) {
            // Agent + time
            HStack(spacing: 6) {
                Circle()
                    .fill(petOrange)
                    .frame(width: 10, height: 10)
                Text(post.agentName)
                    .font(.subheadline.weight(.medium))
                Text("·")
                    .foregroundColor(.secondary)
                Text(post.relativeTime)
                    .font(.subheadline)
                    .foregroundColor(.secondary)
                Spacer()
            }

            // Info badges row
            infoBadgesRow(data: data)

            // Attribute tiles
            if let attrs = data.attributes {
                attributesSection(attrs)
            }

            // About
            let aboutText = data.description ?? post.body
            if !aboutText.isEmpty {
                aboutSection(aboutText)
            }

            // Shelter info
            if data.shelterName != nil || data.shelterCity != nil {
                shelterCard(data: data)
            }

            // Map
            if let lat = data.latitude, let lon = data.longitude {
                mapSection(lat: lat, lon: lon, name: data.name ?? post.title)
            }

            // Adopt Me button
            if let urlStr = data.petfinderUrl, let url = URL(string: urlStr), let name = data.name {
                adoptButton(url: url, name: name)
            }

            Divider()

            PostDetailEngagementBar(post: post)
        }
        .padding()
    }

    // MARK: - Hero Section

    @ViewBuilder
    private func heroSection(data: PetData) -> some View {
        let heroURL: URL? = {
            if let raw = data.photoUrl, let url = URL(string: raw) { return url }
            if let raw = post.heroImage?.url, let url = URL(string: raw) { return url }
            if let raw = post.imageURL, !raw.isEmpty, let url = URL(string: raw) { return url }
            return nil
        }()

        if let url = heroURL {
            GeometryReader { geo in
                AsyncImage(url: url) { phase in
                    switch phase {
                    case .success(let img):
                        img.resizable()
                            .aspectRatio(contentMode: .fill)
                            .frame(width: geo.size.width, height: 300)
                            .clipped()
                            .overlay(alignment: .bottom) {
                                LinearGradient(
                                    colors: [.clear, Color(.systemBackground)],
                                    startPoint: .center,
                                    endPoint: .bottom
                                )
                                .frame(height: 140)
                            }
                            .overlay(alignment: .bottomLeading) {
                                heroNameOverlay(data: data)
                                    .padding(16)
                            }
                    case .failure:
                        heroFallback(data: data, width: geo.size.width)
                    default:
                        Rectangle()
                            .fill(Color(.systemGroupedBackground))
                            .frame(width: geo.size.width, height: 300)
                            .overlay(ProgressView().tint(petOrange))
                    }
                }
            }
            .frame(height: 300)
        } else {
            GeometryReader { geo in
                heroFallback(data: data, width: geo.size.width)
            }
            .frame(height: 300)
        }
    }

    @ViewBuilder
    private func heroFallback(data: PetData, width: CGFloat) -> some View {
        ZStack(alignment: .bottomLeading) {
            LinearGradient(
                colors: [petOrange.opacity(0.8), petOrange.opacity(0.4)],
                startPoint: .topLeading,
                endPoint: .bottomTrailing
            )
            .frame(width: width, height: 300)

            Image(systemName: "pawprint.fill")
                .font(.system(size: 80))
                .foregroundStyle(.white.opacity(0.15))
                .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .center)

            heroNameOverlay(data: data)
                .padding(16)
        }
    }

    @ViewBuilder
    private func heroNameOverlay(data: PetData) -> some View {
        VStack(alignment: .leading, spacing: 4) {
            if let name = data.name {
                Text(name)
                    .font(.system(size: 32, weight: .bold))
                    .foregroundStyle(.white)
                    .shadow(color: .black.opacity(0.4), radius: 4)
            }
            HStack(spacing: 6) {
                if let species = data.species {
                    Text(species.capitalized)
                        .font(.subheadline.weight(.semibold))
                        .foregroundStyle(.white.opacity(0.9))
                }
                if let breed = data.breed, !breed.isEmpty {
                    Text("·")
                        .foregroundStyle(.white.opacity(0.6))
                    Text(breed)
                        .font(.subheadline)
                        .foregroundStyle(.white.opacity(0.85))
                        .lineLimit(1)
                }
            }
        }
    }

    // MARK: - Info Badges

    @ViewBuilder
    private func infoBadgesRow(data: PetData) -> some View {
        let items: [(String, String)] = {
            var list: [(String, String)] = []
            if let age = data.age { list.append(("birthday.cake", age)) }
            if let gender = data.gender { list.append((data.gender == "Male" ? "mustache" : "figure.stand.dress", gender)) }
            if let size = data.size { list.append(("ruler", size)) }
            if let color = data.color, !color.isEmpty { list.append(("paintpalette", color)) }
            return list
        }()

        if !items.isEmpty {
            ScrollView(.horizontal, showsIndicators: false) {
                HStack(spacing: 8) {
                    ForEach(items, id: \.0) { icon, label in
                        Label(label, systemImage: icon)
                            .font(.caption.weight(.semibold))
                            .foregroundStyle(petOrange)
                            .padding(.horizontal, 12)
                            .padding(.vertical, 7)
                            .background(petOrange.opacity(0.1), in: Capsule())
                            .overlay(Capsule().stroke(petOrange.opacity(0.2), lineWidth: 1))
                    }
                }
            }
        }
    }

    // MARK: - Attributes Grid

    @ViewBuilder
    private func attributesSection(_ attrs: PetAttributes) -> some View {
        let attrItems: [(String, String, Bool?)] = [
            ("House Trained",    "house.fill",                        attrs.houseTrained),
            ("Good w/ Kids",     "figure.2.and.child.holdinghands",   attrs.goodWithChildren),
            ("Good w/ Dogs",     "pawprint",                          attrs.goodWithDogs),
            ("Good w/ Cats",     "pawprint.fill",                     attrs.goodWithCats),
            ("Vaccinated",       "syringe",                           attrs.shotsCurrent),
            ("Spayed/Neutered",  "scissors",                          attrs.spayedNeutered),
        ]
        let visible = attrItems.filter { $0.2 != nil }

        if !visible.isEmpty {
            VStack(alignment: .leading, spacing: 10) {
                Text("TRAITS")
                    .font(.system(size: 11, weight: .bold))
                    .tracking(1.5)
                    .foregroundStyle(.secondary)

                let columns = [GridItem(.flexible()), GridItem(.flexible())]
                LazyVGrid(columns: columns, spacing: 8) {
                    ForEach(visible, id: \.0) { label, icon, value in
                        attributeTile(label, icon: icon, value: value)
                    }
                }
            }
        }
    }

    @ViewBuilder
    private func attributeTile(_ label: String, icon: String, value: Bool?) -> some View {
        if let v = value {
            HStack(spacing: 6) {
                Image(systemName: icon)
                    .font(.caption)
                    .foregroundStyle(v ? .green : .secondary)
                    .frame(width: 16)
                Text(label)
                    .font(.caption.weight(.medium))
                    .foregroundStyle(.primary)
                    .lineLimit(1)
                Spacer()
                Image(systemName: v ? "checkmark.circle.fill" : "xmark.circle")
                    .font(.caption)
                    .foregroundStyle(v ? .green : Color(.systemFill))
            }
            .padding(.horizontal, 10)
            .padding(.vertical, 8)
            .background(Color(.secondarySystemGroupedBackground), in: RoundedRectangle(cornerRadius: 8))
        }
    }

    // MARK: - About

    @ViewBuilder
    private func aboutSection(_ text: String) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            Text("ABOUT")
                .font(.system(size: 11, weight: .bold))
                .tracking(1.5)
                .foregroundStyle(.secondary)

            Text(text)
                .font(.body)
                .foregroundStyle(.primary)
                .lineSpacing(4)
        }
    }

    // MARK: - Shelter Card

    @ViewBuilder
    private func shelterCard(data: PetData) -> some View {
        VStack(alignment: .leading, spacing: 12) {
            Text("SHELTER")
                .font(.system(size: 11, weight: .bold))
                .tracking(1.5)
                .foregroundStyle(.secondary)

            VStack(alignment: .leading, spacing: 10) {
                if let name = data.shelterName {
                    Label(name, systemImage: "building.2.fill")
                        .font(.subheadline.weight(.semibold))
                        .foregroundStyle(.primary)
                }
                if let city = data.shelterCity {
                    Label(city, systemImage: "location.fill")
                        .font(.subheadline)
                        .foregroundStyle(.secondary)
                }

                if data.shelterPhone != nil || data.shelterEmail != nil {
                    Divider()

                    HStack(spacing: 12) {
                        if let phone = data.shelterPhone,
                           let tel = URL(string: "tel:\(phone.filter { $0.isNumber || $0 == "+" })") {
                            Link(destination: tel) {
                                Label("Call Shelter", systemImage: "phone.fill")
                                    .font(.subheadline.weight(.semibold))
                                    .foregroundStyle(.white)
                                    .frame(maxWidth: .infinity)
                                    .padding(.vertical, 10)
                                    .background(petOrange, in: RoundedRectangle(cornerRadius: 10))
                            }
                        }

                        if let email = data.shelterEmail,
                           let mailto = URL(string: "mailto:\(email)") {
                            Link(destination: mailto) {
                                Label("Email Shelter", systemImage: "envelope.fill")
                                    .font(.subheadline.weight(.semibold))
                                    .foregroundStyle(petOrange)
                                    .frame(maxWidth: .infinity)
                                    .padding(.vertical, 10)
                                    .background(petOrange.opacity(0.1), in: RoundedRectangle(cornerRadius: 10))
                                    .overlay(
                                        RoundedRectangle(cornerRadius: 10)
                                            .stroke(petOrange.opacity(0.3), lineWidth: 1)
                                    )
                            }
                        }
                    }
                }
            }
            .padding(14)
            .background(Color(.secondarySystemGroupedBackground), in: RoundedRectangle(cornerRadius: 12))
        }
    }

    // MARK: - Map

    @ViewBuilder
    private func mapSection(lat: Double, lon: Double, name: String) -> some View {
        let coord = CLLocationCoordinate2D(latitude: lat, longitude: lon)
        Map(initialPosition: .region(MKCoordinateRegion(
            center: coord,
            span: MKCoordinateSpan(latitudeDelta: 0.012, longitudeDelta: 0.012)
        ))) {
            Marker(name, systemImage: "pawprint.fill", coordinate: coord)
                .tint(petOrange)
        }
        .frame(height: 140)
        .clipShape(RoundedRectangle(cornerRadius: 12))
    }

    // MARK: - Adopt Button

    @ViewBuilder
    private func adoptButton(url: URL, name: String) -> some View {
        Link(destination: url) {
            HStack(spacing: 8) {
                Image(systemName: "heart.fill")
                Text("Meet \(name) on Petfinder")
                    .font(.subheadline.weight(.semibold))
            }
            .foregroundStyle(.white)
            .frame(maxWidth: .infinity)
            .padding(.vertical, 14)
            .background(petOrange, in: RoundedRectangle(cornerRadius: 12))
        }
        .buttonStyle(.plain)
    }

    // MARK: - Tip Detail

    @ViewBuilder
    private func tipDetailBody(data: PetData) -> some View {
        // Tip header banner
        VStack(alignment: .leading, spacing: 0) {
            ZStack(alignment: .bottomLeading) {
                LinearGradient(
                    colors: [Color(red: 0.082, green: 0.718, blue: 0.647).opacity(0.85),
                             Color(red: 0.082, green: 0.718, blue: 0.647).opacity(0.4)],
                    startPoint: .topLeading,
                    endPoint: .bottomTrailing
                )
                .frame(height: 160)

                Image(systemName: "lightbulb.max.fill")
                    .font(.system(size: 60))
                    .foregroundStyle(.white.opacity(0.12))
                    .frame(maxWidth: .infinity, alignment: .trailing)
                    .padding(.trailing, 24)
                    .padding(.bottom, 24)

                VStack(alignment: .leading, spacing: 6) {
                    if let topic = data.topic {
                        Text(topic.uppercased())
                            .font(.system(size: 10, weight: .bold))
                            .tracking(2)
                            .foregroundStyle(.white.opacity(0.75))
                    }
                    Text(data.tipTitle ?? post.title)
                        .font(.title2.weight(.bold))
                        .foregroundStyle(.white)
                        .lineLimit(3)
                }
                .padding(20)
            }

            VStack(alignment: .leading, spacing: 20) {
                // Agent + time
                HStack(spacing: 6) {
                    Circle()
                        .fill(Color(red: 0.082, green: 0.718, blue: 0.647))
                        .frame(width: 10, height: 10)
                    Text(post.agentName)
                        .font(.subheadline.weight(.medium))
                    Text("·")
                        .foregroundColor(.secondary)
                    Text(post.relativeTime)
                        .font(.subheadline)
                        .foregroundColor(.secondary)
                }

                // Body
                if !post.body.isEmpty {
                    Text(post.body)
                        .font(.body)
                        .foregroundStyle(.primary)
                        .lineSpacing(4)
                }

                // Source
                if let org = data.sourceOrg, let urlStr = data.sourceUrl, let url = URL(string: urlStr) {
                    Link(destination: url) {
                        Label("Source: \(org)", systemImage: "link")
                            .font(.caption)
                            .foregroundStyle(Color(red: 0.082, green: 0.718, blue: 0.647))
                    }
                }

                // Tags
                if let tags = data.tags, !tags.isEmpty {
                    ScrollView(.horizontal, showsIndicators: false) {
                        HStack(spacing: 6) {
                            ForEach(tags.prefix(5), id: \.self) { tag in
                                Text(tag)
                                    .font(.caption2.weight(.medium))
                                    .foregroundStyle(.secondary)
                                    .padding(.horizontal, 8)
                                    .padding(.vertical, 4)
                                    .background(Color(.systemGray6), in: RoundedRectangle(cornerRadius: 6))
                            }
                        }
                    }
                }

                Divider()

                PostDetailEngagementBar(post: post)
            }
            .padding()
        }
    }

    // MARK: - Fallback

    private var fallbackBody: some View {
        VStack(alignment: .leading, spacing: 16) {
            Text(post.title)
                .font(.title2.weight(.bold))
            Text(post.body)
                .font(.body)
                .foregroundStyle(.primary)
                .lineSpacing(4)
            Divider()
            PostDetailEngagementBar(post: post)
        }
        .padding()
        .padding(.top, 60)
    }
}
