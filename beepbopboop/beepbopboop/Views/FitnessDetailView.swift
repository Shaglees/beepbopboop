import SwiftUI
import MapKit

struct FitnessDetailView: View {
    let post: Post
    @Environment(\.dismiss) private var dismiss

    private var data: FitnessData? { post.fitnessData }

    private static let fitnessGreen = Color(red: 0.133, green: 0.773, blue: 0.369)

    private var heroURL: URL? {
        if let h = post.heroImage?.url, !h.isEmpty { return URL(string: h) }
        if let i = post.imageURL, !i.isEmpty { return URL(string: i) }
        return nil
    }

    // MARK: - Level helpers

    private var levelIndex: Int {
        switch data?.level?.lowercased() {
        case "beginner": return 1
        case "advanced": return 3
        default: return 2
        }
    }

    private var levelColor: Color {
        switch data?.levelColor {
        case "green": return .green
        case "red": return .red
        default: return .orange
        }
    }

    private var levelDescription: String {
        switch data?.level?.lowercased() {
        case "beginner": return "Suitable for all levels"
        case "advanced": return "High intensity"
        default: return "Some experience recommended"
        }
    }

    // MARK: - Body

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 0) {
                heroSection
                VStack(alignment: .leading, spacing: 20) {
                    statsStrip
                    if data?.level != nil {
                        difficultyBadge
                    }
                    if let groups = data?.muscleGroups, !groups.isEmpty {
                        muscleGroupsSection(groups)
                    }
                    if let exercises = data?.exercises, !exercises.isEmpty {
                        exerciseListSection(exercises)
                    }
                    equipmentSection
                    if !post.body.isEmpty {
                        Text(post.body)
                            .font(.body)
                            .lineSpacing(4)
                    }
                    if let d = data, d.isEvent {
                        eventSection(d)
                    }
                    if let src = data?.sourceUrl, !src.isEmpty, let url = URL(string: src) {
                        Link(destination: url) {
                            Label("View Workout", systemImage: "arrow.up.right.square")
                                .font(.subheadline.weight(.semibold))
                                .foregroundStyle(Self.fitnessGreen)
                                .frame(maxWidth: .infinity)
                                .padding(.vertical, 12)
                                .background(Self.fitnessGreen.opacity(0.1), in: RoundedRectangle(cornerRadius: 10))
                                .overlay(
                                    RoundedRectangle(cornerRadius: 10)
                                        .stroke(Self.fitnessGreen.opacity(0.3), lineWidth: 1)
                                )
                        }
                    }
                    Divider()
                    PostDetailEngagementBar(post: post)
                }
                .padding(16)
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

    // MARK: - Hero

    @ViewBuilder
    private var heroSection: some View {
        if let url = heroURL {
            GeometryReader { geo in
                AsyncImage(url: url) { phase in
                    switch phase {
                    case .success(let img):
                        img.resizable()
                            .aspectRatio(contentMode: .fill)
                            .frame(width: geo.size.width, height: 240)
                            .clipped()
                            .overlay {
                                LinearGradient(
                                    colors: [.clear, Color(.systemBackground)],
                                    startPoint: .center,
                                    endPoint: .bottom
                                )
                            }
                            .overlay(alignment: .bottomLeading) {
                                heroTextOverlay.padding(16)
                            }
                    case .failure:
                        gradientHero(width: geo.size.width)
                    default:
                        Color.secondary.opacity(0.2)
                            .frame(height: 240)
                            .overlay(ProgressView())
                    }
                }
            }
            .frame(height: 240)
        } else {
            GeometryReader { geo in
                gradientHero(width: geo.size.width)
            }
            .frame(height: 240)
        }
    }

    @ViewBuilder
    private func gradientHero(width: CGFloat) -> some View {
        ZStack(alignment: .bottomLeading) {
            LinearGradient(
                colors: [Self.fitnessGreen.opacity(0.85), Color(red: 0.06, green: 0.12, blue: 0.08)],
                startPoint: .topLeading,
                endPoint: .bottomTrailing
            )
            .frame(width: width, height: 240)

            if let icon = data?.activityIcon {
                Image(systemName: icon)
                    .font(.system(size: 120, weight: .ultraLight))
                    .foregroundStyle(.white.opacity(0.08))
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            }

            heroTextOverlay.padding(16)
        }
    }

    @ViewBuilder
    private var heroTextOverlay: some View {
        VStack(alignment: .leading, spacing: 4) {
            if let activity = data?.activity {
                Text(activity.localizedCapitalized)
                    .font(.title2.weight(.bold))
                    .foregroundStyle(.white)
            } else {
                Text(post.title)
                    .font(.title2.weight(.bold))
                    .foregroundStyle(.white)
                    .lineLimit(2)
            }
            if let mins = data?.durationMin {
                HStack(spacing: 4) {
                    Image(systemName: "clock.fill")
                        .font(.caption)
                    Text("\(mins) min")
                        .font(.subheadline.weight(.medium))
                }
                .foregroundStyle(.white.opacity(0.85))
            }
        }
    }

    // MARK: - Stats Strip

    @ViewBuilder
    private var statsStrip: some View {
        let hasDuration = data?.durationMin != nil
        let hasCalories = data?.caloriesBurn != nil
        let hasActivity = data?.activity != nil

        if hasDuration || hasCalories || hasActivity {
            HStack(spacing: 10) {
                if let mins = data?.durationMin {
                    statTile(icon: "clock.fill", label: "\(mins) min")
                }
                if let cal = data?.caloriesBurn {
                    statTile(icon: "flame.fill", label: cal)
                }
                if let activity = data?.activity {
                    statTile(icon: data?.activityIcon ?? "figure.run", label: activity.localizedCapitalized)
                }
            }
        }
    }

    @ViewBuilder
    private func statTile(icon: String, label: String) -> some View {
        HStack(spacing: 6) {
            Image(systemName: icon)
                .font(.subheadline)
                .foregroundStyle(Self.fitnessGreen)
            Text(label)
                .font(.subheadline.weight(.semibold))
                .foregroundStyle(.primary)
                .lineLimit(1)
        }
        .frame(maxWidth: .infinity)
        .padding(.vertical, 12)
        .background(Self.fitnessGreen.opacity(0.08), in: RoundedRectangle(cornerRadius: 10))
        .overlay(
            RoundedRectangle(cornerRadius: 10)
                .stroke(Self.fitnessGreen.opacity(0.2), lineWidth: 1)
        )
    }

    // MARK: - Difficulty Badge

    @ViewBuilder
    private var difficultyBadge: some View {
        VStack(alignment: .leading, spacing: 6) {
            Text("DIFFICULTY")
                .font(.system(size: 11, weight: .bold))
                .tracking(1.5)
                .foregroundStyle(.secondary)

            HStack(spacing: 12) {
                VStack(alignment: .leading, spacing: 6) {
                    HStack(spacing: 4) {
                        ForEach(0..<3) { i in
                            RoundedRectangle(cornerRadius: 2)
                                .fill(i < levelIndex ? levelColor : Color(.systemFill))
                                .frame(width: 24, height: 8)
                        }
                    }
                    Text(data?.level?.localizedCapitalized ?? "Intermediate")
                        .font(.subheadline.weight(.bold))
                        .foregroundStyle(levelColor)
                }
                Spacer()
                Text(levelDescription)
                    .font(.caption)
                    .foregroundStyle(.secondary)
                    .multilineTextAlignment(.trailing)
            }
            .padding(14)
            .background(levelColor.opacity(0.1), in: RoundedRectangle(cornerRadius: 12))
        }
    }

    // MARK: - Muscle Groups

    @ViewBuilder
    private func muscleGroupsSection(_ groups: [String]) -> some View {
        VStack(alignment: .leading, spacing: 10) {
            Text("MUSCLE GROUPS")
                .font(.system(size: 11, weight: .bold))
                .tracking(1.5)
                .foregroundStyle(.secondary)

            ScrollView(.horizontal, showsIndicators: false) {
                HStack(spacing: 8) {
                    ForEach(groups, id: \.self) { group in
                        HStack(spacing: 5) {
                            Image(systemName: "figure.strengthtraining.traditional")
                                .font(.caption2)
                            Text(group)
                                .font(.subheadline.weight(.medium))
                        }
                        .foregroundStyle(Self.fitnessGreen)
                        .padding(.horizontal, 12)
                        .padding(.vertical, 7)
                        .background(Self.fitnessGreen.opacity(0.1), in: Capsule())
                        .overlay(
                            Capsule()
                                .stroke(Self.fitnessGreen.opacity(0.3), lineWidth: 1)
                        )
                    }
                }
            }
        }
    }

    // MARK: - Exercise List

    @ViewBuilder
    private func exerciseListSection(_ exercises: [FitnessExercise]) -> some View {
        VStack(alignment: .leading, spacing: 10) {
            Text("EXERCISES")
                .font(.system(size: 11, weight: .bold))
                .tracking(1.5)
                .foregroundStyle(.secondary)

            VStack(spacing: 0) {
                ForEach(Array(exercises.enumerated()), id: \.offset) { index, exercise in
                    exerciseRow(exercise, index: index, isAlternate: index % 2 == 1)
                    if index < exercises.count - 1 {
                        Divider()
                            .padding(.leading, 44)
                    }
                }
            }
            .background(Color(.secondarySystemGroupedBackground), in: RoundedRectangle(cornerRadius: 12))
            .overlay(
                RoundedRectangle(cornerRadius: 12)
                    .stroke(Color(.separator).opacity(0.3), lineWidth: 1)
            )
        }
    }

    @ViewBuilder
    private func exerciseRow(_ exercise: FitnessExercise, index: Int, isAlternate: Bool) -> some View {
        HStack(spacing: 12) {
            Text("\(index + 1)")
                .font(.caption.weight(.bold).monospacedDigit())
                .foregroundStyle(Self.fitnessGreen)
                .frame(width: 24, alignment: .center)
                .padding(.vertical, 2)
                .background(Self.fitnessGreen.opacity(0.12), in: Circle())

            VStack(alignment: .leading, spacing: 3) {
                Text(exercise.name)
                    .font(.subheadline.weight(.semibold))
                    .foregroundStyle(.primary)

                HStack(spacing: 6) {
                    if let sets = exercise.sets, let reps = exercise.reps {
                        Text("\(sets) sets × \(reps)")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    } else if let sets = exercise.sets {
                        Text("\(sets) sets")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    } else if let reps = exercise.reps {
                        Text(reps)
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }

                    if let rest = exercise.restSec, rest > 0 {
                        Text("·")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                        Text("\(rest)s rest")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }
                }
            }

            Spacer()
        }
        .padding(.horizontal, 14)
        .padding(.vertical, 10)
        .background(isAlternate ? Color(.tertiarySystemGroupedBackground) : Color(.secondarySystemGroupedBackground))
    }

    // MARK: - Equipment

    @ViewBuilder
    private var equipmentSection: some View {
        let equipment = data?.equipmentNeeded ?? []
        VStack(alignment: .leading, spacing: 10) {
            Text("EQUIPMENT")
                .font(.system(size: 11, weight: .bold))
                .tracking(1.5)
                .foregroundStyle(.secondary)

            if equipment.isEmpty {
                HStack(spacing: 8) {
                    Image(systemName: "checkmark.circle.fill")
                        .foregroundStyle(Self.fitnessGreen)
                    Text("No equipment needed")
                        .font(.subheadline.weight(.medium))
                        .foregroundStyle(Self.fitnessGreen)
                }
                .padding(12)
                .frame(maxWidth: .infinity, alignment: .leading)
                .background(Self.fitnessGreen.opacity(0.08), in: RoundedRectangle(cornerRadius: 10))
            } else {
                VStack(alignment: .leading, spacing: 0) {
                    ForEach(Array(equipment.enumerated()), id: \.offset) { index, item in
                        HStack(spacing: 10) {
                            Image(systemName: "checkmark.circle")
                                .foregroundStyle(Self.fitnessGreen)
                                .font(.subheadline)
                            Text(item)
                                .font(.subheadline)
                                .foregroundStyle(.primary)
                            Spacer()
                        }
                        .padding(.horizontal, 14)
                        .padding(.vertical, 10)
                        if index < equipment.count - 1 {
                            Divider().padding(.leading, 44)
                        }
                    }
                }
                .background(Color(.secondarySystemGroupedBackground), in: RoundedRectangle(cornerRadius: 12))
                .overlay(
                    RoundedRectangle(cornerRadius: 12)
                        .stroke(Color(.separator).opacity(0.3), lineWidth: 1)
                )
            }
        }
    }

    // MARK: - Event Section

    @ViewBuilder
    private func eventSection(_ d: FitnessData) -> some View {
        VStack(alignment: .leading, spacing: 14) {
            Text("EVENT DETAILS")
                .font(.system(size: 11, weight: .bold))
                .tracking(1.5)
                .foregroundStyle(.secondary)

            VStack(alignment: .leading, spacing: 0) {
                if let name = d.eventName, !name.isEmpty {
                    eventDetailRow(icon: "sportscourt.fill", label: name)
                    Divider().padding(.leading, 44)
                }
                if let date = d.date {
                    let display = formattedEventDate(date)
                    eventDetailRow(icon: "calendar", label: d.startTime.map { "\(display) at \($0)" } ?? display)
                    Divider().padding(.leading, 44)
                }
                if let location = d.location, !location.isEmpty {
                    eventDetailRow(icon: "mappin.and.ellipse", label: location)
                    Divider().padding(.leading, 44)
                }
                if let price = d.price, !price.isEmpty {
                    eventDetailRow(icon: "ticket.fill", label: price)
                    if d.recurring == true || d.recurrenceRule != nil {
                        Divider().padding(.leading, 44)
                    }
                }
                if d.recurring == true {
                    eventDetailRow(icon: "repeat", label: d.recurrenceRule ?? "Recurring event")
                }
            }
            .background(Color(.secondarySystemGroupedBackground), in: RoundedRectangle(cornerRadius: 12))
            .overlay(
                RoundedRectangle(cornerRadius: 12)
                    .stroke(Color(.separator).opacity(0.3), lineWidth: 1)
            )

            // Map
            if let lat = d.latitude, let lon = d.longitude {
                let coord = CLLocationCoordinate2D(latitude: lat, longitude: lon)
                Map(initialPosition: .region(MKCoordinateRegion(
                    center: coord,
                    span: MKCoordinateSpan(latitudeDelta: 0.008, longitudeDelta: 0.008)
                ))) {
                    Marker(d.eventName ?? post.title, systemImage: "figure.run", coordinate: coord)
                        .tint(Self.fitnessGreen)
                }
                .frame(height: 180)
                .cornerRadius(12)
            }

            // Registration button
            if let regUrl = d.registrationUrl, !regUrl.isEmpty, let url = URL(string: regUrl) {
                Link(destination: url) {
                    Label("Register Now", systemImage: "arrow.up.right.square")
                        .font(.subheadline.weight(.semibold))
                        .foregroundStyle(.white)
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 13)
                        .background(Self.fitnessGreen, in: RoundedRectangle(cornerRadius: 10))
                }
            }
        }
    }

    @ViewBuilder
    private func eventDetailRow(icon: String, label: String) -> some View {
        HStack(spacing: 10) {
            Image(systemName: icon)
                .foregroundStyle(Self.fitnessGreen)
                .font(.subheadline)
                .frame(width: 24, alignment: .center)
            Text(label)
                .font(.subheadline)
                .foregroundStyle(.primary)
            Spacer()
        }
        .padding(.horizontal, 14)
        .padding(.vertical, 10)
    }

    // MARK: - Helpers

    private func formattedEventDate(_ dateStr: String) -> String {
        let f = DateFormatter()
        f.dateFormat = "yyyy-MM-dd"
        f.timeZone = TimeZone(identifier: "UTC")
        guard let date = f.date(from: dateStr) else { return dateStr }
        f.dateFormat = "EEEE, MMM d"
        f.timeZone = .current
        return f.string(from: date)
    }
}
