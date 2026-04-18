import Foundation

struct FitnessData: Codable {
    let type: String
    let activity: String?
    let level: String?
    let durationMin: Int?
    let muscleGroups: [String]
    let exercises: [FitnessExercise]?
    let caloriesBurn: String?
    let equipmentNeeded: [String]
    let sourceUrl: String?
    // Event fields
    let eventName: String?
    let date: String?
    let startTime: String?
    let location: String?
    let price: String?
    let registrationUrl: String?
    let latitude: Double?
    let longitude: Double?
    let recurring: Bool?
    let recurrenceRule: String?

    enum CodingKeys: String, CodingKey {
        case type, activity, level, exercises, location, price, date, recurring
        case durationMin, muscleGroups, caloriesBurn, equipmentNeeded, sourceUrl
        case eventName, startTime, registrationUrl, latitude, longitude, recurrenceRule
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        type = try c.decodeIfPresent(String.self, forKey: .type) ?? "workout"
        activity = try c.decodeIfPresent(String.self, forKey: .activity)
        level = try c.decodeIfPresent(String.self, forKey: .level)
        durationMin = try c.decodeIfPresent(Int.self, forKey: .durationMin)
        muscleGroups = try c.decodeIfPresent([String].self, forKey: .muscleGroups) ?? []
        exercises = try c.decodeIfPresent([FitnessExercise].self, forKey: .exercises)
        caloriesBurn = try c.decodeIfPresent(String.self, forKey: .caloriesBurn)
        equipmentNeeded = try c.decodeIfPresent([String].self, forKey: .equipmentNeeded) ?? []
        sourceUrl = try c.decodeIfPresent(String.self, forKey: .sourceUrl)
        eventName = try c.decodeIfPresent(String.self, forKey: .eventName)
        date = try c.decodeIfPresent(String.self, forKey: .date)
        startTime = try c.decodeIfPresent(String.self, forKey: .startTime)
        location = try c.decodeIfPresent(String.self, forKey: .location)
        price = try c.decodeIfPresent(String.self, forKey: .price)
        registrationUrl = try c.decodeIfPresent(String.self, forKey: .registrationUrl)
        latitude = try c.decodeIfPresent(Double.self, forKey: .latitude)
        longitude = try c.decodeIfPresent(Double.self, forKey: .longitude)
        recurring = try c.decodeIfPresent(Bool.self, forKey: .recurring)
        recurrenceRule = try c.decodeIfPresent(String.self, forKey: .recurrenceRule)
    }

    var isWorkout: Bool { type == "workout" }
    var isEvent: Bool { type == "event" }
    var isNutrition: Bool { type == "nutrition" }
    var isWellness: Bool { type == "wellness" }

    var activityIcon: String {
        switch activity?.lowercased() {
        case "running", "run": return "figure.run"
        case "yoga": return "figure.yoga"
        case "cycling", "bike": return "figure.outdoor.cycle"
        case "hiit": return "figure.highintensity.intervaltraining"
        case "swimming", "swim": return "figure.pool.swim"
        case "walking", "walk", "hiking": return "figure.walk"
        default: return "figure.strengthtraining.traditional"
        }
    }

    var levelColor: String {
        switch level?.lowercased() {
        case "beginner": return "green"
        case "advanced": return "red"
        default: return "orange"
        }
    }
}

struct FitnessExercise: Codable {
    let name: String
    let sets: Int?
    let reps: String?
    let restSec: Int?
}
