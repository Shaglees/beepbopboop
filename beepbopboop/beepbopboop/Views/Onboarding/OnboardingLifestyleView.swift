import SwiftUI

private struct TagOption: Identifiable {
    let id: String
    let category: String
    let value: String
    let label: String
}

private let tagOptions: [TagOption] = [
    TagOption(id: "diet-veg", category: "diet", value: "vegetarian", label: "Vegetarian"),
    TagOption(id: "diet-vegan", category: "diet", value: "vegan", label: "Vegan"),
    TagOption(id: "diet-gf", category: "diet", value: "gluten_free", label: "Gluten-free"),
    TagOption(id: "diet-halal", category: "diet", value: "halal", label: "Halal"),
    TagOption(id: "diet-kosher", category: "diet", value: "kosher", label: "Kosher"),
    TagOption(id: "fit-run", category: "fitness", value: "runner", label: "Runner"),
    TagOption(id: "fit-cycle", category: "fitness", value: "cyclist", label: "Cyclist"),
    TagOption(id: "fit-gym", category: "fitness", value: "gym", label: "Gym"),
    TagOption(id: "fit-yoga", category: "fitness", value: "yoga", label: "Yoga"),
    TagOption(id: "fit-swim", category: "fitness", value: "swimmer", label: "Swimmer"),
    TagOption(id: "pet-dog", category: "pets", value: "dog_owner", label: "Dog owner"),
    TagOption(id: "pet-cat", category: "pets", value: "cat_owner", label: "Cat owner"),
    TagOption(id: "fam-parent", category: "family", value: "parent", label: "Parent"),
    TagOption(id: "fam-couple", category: "family", value: "couple", label: "Couple"),
]

struct OnboardingLifestyleView: View {
    @Binding var lifestyle: [LifestyleTag]
    let onNext: () -> Void
    @State private var selected: Set<String> = []

    var body: some View {
        ScrollView {
            VStack(spacing: 24) {
                Text("Tell us about you")
                    .font(.system(size: 28, weight: .bold, design: .serif))
                    .padding(.top, 20)
                Text("This helps personalize your feed. Skip if you prefer.")
                    .font(.system(size: 15))
                    .foregroundStyle(.secondary)

                ForEach(["diet", "fitness", "pets", "family"], id: \.self) { category in
                    VStack(alignment: .leading, spacing: 8) {
                        Text(category.capitalized)
                            .font(.system(size: 11, weight: .medium, design: .monospaced))
                            .foregroundStyle(.secondary)
                        FlowLayout(spacing: 8) {
                            ForEach(tagOptions.filter { $0.category == category }) { opt in
                                Button {
                                    if selected.contains(opt.id) {
                                        selected.remove(opt.id)
                                    } else {
                                        selected.insert(opt.id)
                                    }
                                } label: {
                                    Text(opt.label)
                                        .font(.system(size: 14))
                                        .padding(.horizontal, 14)
                                        .padding(.vertical, 8)
                                        .background(selected.contains(opt.id) ? Color.accentColor.opacity(0.2) : Color(.systemGray6))
                                        .clipShape(Capsule())
                                }
                                .buttonStyle(.plain)
                            }
                        }
                    }
                    .padding(.horizontal)
                }

                Button("Continue") {
                    lifestyle = tagOptions
                        .filter { selected.contains($0.id) }
                        .map { LifestyleTag(category: $0.category, value: $0.value) }
                    onNext()
                }
                .buttonStyle(.borderedProminent)
                .padding(.bottom, 40)
            }
        }
    }
}
