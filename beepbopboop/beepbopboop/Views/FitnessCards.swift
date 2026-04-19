import SwiftUI
import MapKit

// MARK: - FitnessCard

struct FitnessCard: View {
    let post: Post
    let fitness: FitnessData

    init?(post: Post) {
        guard let fd = post.fitnessData else { return nil }
        self.post = post
        self.fitness = fd
    }

    var body: some View {
        if fitness.isEvent {
            FitnessEventCard(post: post, fitness: fitness)
        } else {
            FitnessWorkoutCard(post: post, fitness: fitness)
        }
    }
}

// MARK: - Workout Card

private struct FitnessWorkoutCard: View {
    let post: Post
    let fitness: FitnessData

    private let fitnessGreen = Color(red: 0.133, green: 0.773, blue: 0.369)
    private let darkBg = Color(red: 0.067, green: 0.094, blue: 0.153) // #111827

    var body: some View {
        VStack(spacing: 0) {
            // Header band
            ZStack {
                darkBg
                DiagonalHatchPattern()
                    .opacity(0.06)

                VStack(spacing: 10) {
                    // Agent header
                    HStack(spacing: 6) {
                        Circle()
                            .fill(fitnessGreen)
                            .frame(width: 8, height: 8)
                        Text(post.agentName)
                            .font(.subheadline.weight(.medium))
                            .foregroundColor(.white)
                        Text("Fitness")
                            .font(.caption2.weight(.semibold))
                            .foregroundColor(fitnessGreen)
                            .padding(.horizontal, 7)
                            .padding(.vertical, 3)
                            .background(fitnessGreen.opacity(0.18))
                            .cornerRadius(4)
                        Spacer()
                        if let level = fitness.level {
                            LevelBadge(level: level)
                        }
                        Text(post.relativeTime)
                            .font(.subheadline)
                            .foregroundColor(.white.opacity(0.4))
                    }

                    // Icon + title row
                    HStack(alignment: .center, spacing: 14) {
                        Image(systemName: fitness.activityIcon)
                            .font(.system(size: 36, weight: .bold))
                            .foregroundColor(fitnessGreen)
                            .frame(width: 56, height: 56)
                            .background(fitnessGreen.opacity(0.15))
                            .cornerRadius(12)

                        VStack(alignment: .leading, spacing: 4) {
                            Text(post.title)
                                .font(.headline)
                                .foregroundColor(.white)
                                .lineLimit(2)

                            // Duration + calories
                            HStack(spacing: 12) {
                                if let dur = fitness.durationMin {
                                    Label("\(dur) min", systemImage: "timer")
                                        .font(.caption.weight(.medium))
                                        .foregroundColor(.white.opacity(0.7))
                                }
                                if let cal = fitness.caloriesBurn {
                                    Label(cal, systemImage: "flame")
                                        .font(.caption.weight(.medium))
                                        .foregroundColor(.white.opacity(0.7))
                                }
                            }
                        }
                    }
                    .frame(maxWidth: .infinity, alignment: .leading)

                    // Muscle group chips
                    if !fitness.muscleGroups.isEmpty {
                        ScrollView(.horizontal, showsIndicators: false) {
                            HStack(spacing: 6) {
                                ForEach(fitness.muscleGroups, id: \.self) { group in
                                    Text(group.capitalized)
                                        .font(.caption2.weight(.semibold))
                                        .foregroundColor(fitnessGreen)
                                        .padding(.horizontal, 8)
                                        .padding(.vertical, 4)
                                        .background(fitnessGreen.opacity(0.15))
                                        .cornerRadius(6)
                                }
                            }
                        }
                    }
                }
                .padding(.horizontal, 16)
                .padding(.vertical, 14)
            }
            .frame(maxWidth: .infinity)

            // Exercise list
            if let exercises = fitness.exercises, !exercises.isEmpty {
                VStack(spacing: 0) {
                    let displayed = Array(exercises.prefix(3))
                    let overflow = exercises.count - displayed.count

                    ForEach(Array(displayed.enumerated()), id: \.offset) { idx, ex in
                        ExerciseRow(exercise: ex, accent: fitnessGreen)
                        if idx < displayed.count - 1 || overflow > 0 {
                            Divider().padding(.leading, 16)
                        }
                    }

                    if overflow > 0 {
                        HStack {
                            Text("+\(overflow) more exercises")
                                .font(.caption)
                                .foregroundColor(.secondary)
                            Spacer()
                        }
                        .padding(.horizontal, 16)
                        .padding(.vertical, 8)
                    }
                }
                .background(Color(.systemBackground))
            }

            // Equipment row + footer
            VStack(spacing: 8) {
                if !fitness.equipmentNeeded.isEmpty {
                    HStack(spacing: 6) {
                        Image(systemName: "dumbbell")
                            .font(.caption2)
                            .foregroundColor(.secondary)
                        Text(fitness.equipmentNeeded.joined(separator: " · "))
                            .font(.caption)
                            .foregroundColor(.secondary)
                            .lineLimit(1)
                        Spacer()
                    }
                }

                if let sourceUrl = fitness.sourceUrl, let url = URL(string: sourceUrl) {
                    Link(destination: url) {
                        HStack(spacing: 4) {
                            Text("View full workout")
                                .font(.caption.weight(.semibold))
                                .foregroundColor(fitnessGreen)
                            Image(systemName: "arrow.up.right")
                                .font(.caption2)
                                .foregroundColor(fitnessGreen)
                            Spacer()
                        }
                    }
                }
            }
            .padding(.horizontal, 16)
            .padding(.vertical, 10)
            .background(Color(.systemBackground))

            Divider()

            FitnessCardFooter(post: post)
                .padding(.horizontal, 16)
                .padding(.vertical, 10)
        }
        .clipShape(RoundedRectangle(cornerRadius: 0))
    }
}

// MARK: - Event Card

private struct FitnessEventCard: View {
    let post: Post
    let fitness: FitnessData

    private let fitnessGreen = Color(red: 0.133, green: 0.773, blue: 0.369)

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            // Agent header
            HStack(spacing: 6) {
                Circle()
                    .fill(fitnessGreen)
                    .frame(width: 8, height: 8)
                Text(post.agentName)
                    .font(.subheadline.weight(.medium))
                Text("Fitness")
                    .font(.caption2.weight(.semibold))
                    .foregroundColor(fitnessGreen)
                    .padding(.horizontal, 7)
                    .padding(.vertical, 3)
                    .background(fitnessGreen.opacity(0.12))
                    .cornerRadius(4)
                Spacer()
                Text(post.relativeTime)
                    .font(.subheadline)
                    .foregroundStyle(.tertiary)
            }

            HStack(alignment: .top, spacing: 12) {
                // Activity icon badge (replaces date badge for events)
                VStack(spacing: 4) {
                    Image(systemName: fitness.activityIcon)
                        .font(.system(size: 20, weight: .semibold))
                        .foregroundColor(fitnessGreen)
                    if let date = fitness.date, let dayStr = Self.dayString(from: date) {
                        Text(dayStr.month)
                            .font(.system(size: 9, weight: .bold))
                            .foregroundColor(fitnessGreen)
                            .textCase(.uppercase)
                        Text(dayStr.day)
                            .font(.title3.weight(.bold))
                            .foregroundColor(.primary)
                    }
                }
                .frame(width: 48, height: 60)
                .background(fitnessGreen.opacity(0.1))
                .cornerRadius(10)

                VStack(alignment: .leading, spacing: 5) {
                    Text(post.title)
                        .font(.headline)
                        .lineLimit(2)

                    // Time + recurrence
                    HStack(spacing: 8) {
                        if let time = fitness.startTime {
                            Label(time, systemImage: "clock")
                                .font(.caption)
                                .foregroundColor(.secondary)
                        }
                        if let rule = fitness.recurrenceRule {
                            Text(rule)
                                .font(.caption.weight(.semibold))
                                .foregroundColor(fitnessGreen)
                                .padding(.horizontal, 6)
                                .padding(.vertical, 2)
                                .background(fitnessGreen.opacity(0.1))
                                .cornerRadius(4)
                        }
                    }

                    // Location
                    if let loc = fitness.location {
                        Label(loc, systemImage: "location")
                            .font(.caption)
                            .foregroundColor(fitnessGreen)
                            .lineLimit(1)
                    }

                    // Price + registration
                    HStack(spacing: 8) {
                        if let price = fitness.price {
                            let isFree = price.lowercased() == "free"
                            Text(isFree ? "FREE" : price)
                                .font(.caption.weight(.bold))
                                .foregroundColor(isFree ? fitnessGreen : .primary)
                                .padding(.horizontal, 6)
                                .padding(.vertical, 2)
                                .background(isFree ? fitnessGreen.opacity(0.12) : Color(.systemGray5))
                                .cornerRadius(4)
                        }
                        if let regUrl = fitness.registrationUrl, let url = URL(string: regUrl) {
                            Link(destination: url) {
                                Label("Register", systemImage: "arrow.up.right.square")
                                    .font(.caption.weight(.medium))
                                    .foregroundColor(fitnessGreen)
                            }
                        }
                    }
                }
            }

            // Mini map if coordinates available
            if let lat = fitness.latitude, let lon = fitness.longitude {
                EventMiniMap(latitude: lat, longitude: lon)
                    .frame(height: 100)
                    .cornerRadius(8)
                    .clipped()
            }

            FitnessCardFooter(post: post)
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 12)
    }

    private static func dayString(from dateStr: String) -> (month: String, day: String)? {
        let f = DateFormatter()
        f.dateFormat = "yyyy-MM-dd"
        guard let date = f.date(from: dateStr) else { return nil }
        let mf = DateFormatter()
        mf.dateFormat = "MMM"
        return (mf.string(from: date), "\(Calendar.current.component(.day, from: date))")
    }
}

// MARK: - Supporting Views

private struct ExerciseRow: View {
    let exercise: FitnessExercise
    let accent: Color

    private var setsReps: String {
        let parts: [String] = [
            exercise.sets.map { "\($0)" },
            exercise.reps
        ].compactMap { $0 }
        return parts.joined(separator: "×")
    }

    var body: some View {
        HStack {
            Text(exercise.name)
                .font(.subheadline)
                .foregroundColor(.primary)
            Spacer()
            if !setsReps.isEmpty {
                Text(setsReps)
                    .font(.subheadline.weight(.semibold).monospacedDigit())
                    .foregroundColor(accent)
            }
            if let rest = exercise.restSec {
                Text("\(rest)s rest")
                    .font(.caption)
                    .foregroundColor(.secondary)
                    .padding(.leading, 8)
            }
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 10)
    }
}

private struct LevelBadge: View {
    let level: String

    private var color: Color {
        switch level.lowercased() {
        case "beginner": return .green
        case "advanced": return .red
        default: return .orange
        }
    }

    var body: some View {
        Text(level.uppercased())
            .font(.system(size: 9, weight: .black))
            .tracking(0.5)
            .foregroundColor(color)
            .padding(.horizontal, 6)
            .padding(.vertical, 3)
            .background(color.opacity(0.15))
            .cornerRadius(4)
    }
}

private struct DiagonalHatchPattern: View {
    var body: some View {
        Canvas { ctx, size in
            let spacing: CGFloat = 12
            let lineWidth: CGFloat = 1
            ctx.stroke(
                {
                    var path = Path()
                    var x: CGFloat = -size.height
                    while x < size.width + size.height {
                        path.move(to: CGPoint(x: x, y: 0))
                        path.addLine(to: CGPoint(x: x + size.height, y: size.height))
                        x += spacing
                    }
                    return path
                }(),
                with: .color(.white),
                lineWidth: lineWidth
            )
        }
    }
}

private struct EventMiniMap: View {
    let latitude: Double
    let longitude: Double

    var body: some View {
        Map(coordinateRegion: .constant(MKCoordinateRegion(
            center: CLLocationCoordinate2D(latitude: latitude, longitude: longitude),
            span: MKCoordinateSpan(latitudeDelta: 0.01, longitudeDelta: 0.01)
        )), annotationItems: [MapPin(coordinate: CLLocationCoordinate2D(latitude: latitude, longitude: longitude))]) { pin in
            MapMarker(coordinate: pin.coordinate, tint: Color(red: 0.133, green: 0.773, blue: 0.369))
        }
        .disabled(true)
    }
}

private struct MapPin: Identifiable {
    let id = UUID()
    let coordinate: CLLocationCoordinate2D
}

// MARK: - Fitness Card Footer

private struct FitnessCardFooter: View {
    let post: Post
    @AppStorage var isBookmarked: Bool
    @State private var activeReaction: String?
    @EnvironmentObject private var apiService: APIService

    init(post: Post) {
        self.post = post
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
            ReactionPicker(
                activeReaction: $activeReaction,
                postID: post.id,
                style: .feedCompact
            )
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
                    .foregroundColor(isBookmarked ? Color(red: 0.133, green: 0.773, blue: 0.369) : .secondary)
                    .contentTransition(.symbolEffect(.replace))
            }
            .buttonStyle(.plain)
        }
    }
}
