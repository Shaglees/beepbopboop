import Foundation

struct PetData: Codable {
    let type: String                    // "adoption" | "tip" | "breed"

    // Adoption fields
    let petfinderId: String?
    let name: String?
    let species: String?                // "dog" | "cat" | "rabbit" | "bird"
    let breed: String?
    let age: String?                    // "Baby" | "Young" | "Adult" | "Senior"
    let gender: String?
    let size: String?
    let color: String?
    let photoUrl: String?
    let description: String?
    let attributes: PetAttributes?
    let shelterName: String?
    let shelterCity: String?
    let shelterPhone: String?
    let shelterEmail: String?
    let petfinderUrl: String?
    let latitude: Double?
    let longitude: Double?

    // Tip fields
    let speciesList: [String]?
    let topic: String?
    let tipTitle: String?
    let sourceOrg: String?
    let sourceUrl: String?
    let tags: [String]?
}

struct PetAttributes: Codable {
    let spayedNeutered: Bool?
    let shotsCurrent: Bool?
    let houseTrained: Bool?
    let goodWithChildren: Bool?
    let goodWithDogs: Bool?
    let goodWithCats: Bool?
}
