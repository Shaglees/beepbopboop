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

private struct FollowUp {
    let parentTagId: String
    let label: String
    let options: [(id: String, label: String, category: String, value: String)]
}

private let followUps: [FollowUp] = [
    FollowUp(parentTagId: "fam-parent", label: "Kids' ages?", options: [
        (id: "age-baby", label: "Baby (0\u{2013}1)", category: "family", value: "parent_baby"),
        (id: "age-toddler", label: "Toddler (2\u{2013}4)", category: "family", value: "parent_toddler"),
        (id: "age-kid", label: "Kid (5\u{2013}12)", category: "family", value: "parent_kid"),
        (id: "age-teen", label: "Teen (13+)", category: "family", value: "parent_teen"),
    ]),
    FollowUp(parentTagId: "pet-dog", label: "Dog size?", options: [
        (id: "dog-small", label: "Small", category: "pets", value: "dog_small"),
        (id: "dog-medium", label: "Medium", category: "pets", value: "dog_medium"),
        (id: "dog-large", label: "Large", category: "pets", value: "dog_large"),
    ]),
    FollowUp(parentTagId: "pet-cat", label: "Indoor or outdoor?", options: [
        (id: "cat-indoor", label: "Indoor", category: "pets", value: "cat_indoor"),
        (id: "cat-outdoor", label: "Outdoor", category: "pets", value: "cat_outdoor"),
    ]),
]

struct OnboardingLifestyleView: View {
    @Binding var lifestyle: [LifestyleTag]
    let onNext: () -> Void
    @State private var selected: Set<String> = []
    @State private var selectedFollowUps: Set<String> = []

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
                                        for fu in followUps where fu.parentTagId == opt.id {
                                            for child in fu.options { selectedFollowUps.remove(child.id) }
                                        }
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

                        ForEach(followUps.filter { fu in selected.contains(fu.parentTagId) && tagOptions.first(where: { t in t.id == fu.parentTagId })?.category == category }, id: \.parentTagId) { fu in
                            VStack(alignment: .leading, spacing: 6) {
                                Text(fu.label)
                                    .font(.system(size: 13, weight: .medium))
                                    .foregroundStyle(.secondary)
                                FlowLayout(spacing: 8) {
                                    ForEach(fu.options, id: \.id) { opt in
                                        Button {
                                            if selectedFollowUps.contains(opt.id) {
                                                selectedFollowUps.remove(opt.id)
                                            } else {
                                                selectedFollowUps.insert(opt.id)
                                            }
                                        } label: {
                                            Text(opt.label)
                                                .font(.system(size: 14))
                                                .padding(.horizontal, 14)
                                                .padding(.vertical, 8)
                                                .background(selectedFollowUps.contains(opt.id) ? Color.accentColor.opacity(0.2) : Color(.systemGray6))
                                                .clipShape(Capsule())
                                        }
                                        .buttonStyle(.plain)
                                    }
                                }
                            }
                            .padding(.leading, 8)
                            .transition(.opacity.combined(with: .move(edge: .top)))
                        }
                    }
                    .padding(.horizontal)
                }

                Button("Continue") {
                    lifestyle = tagOptions
                        .filter { selected.contains($0.id) }
                        .map { LifestyleTag(category: $0.category, value: $0.value) }

                    for fu in followUps {
                        for opt in fu.options where selectedFollowUps.contains(opt.id) {
                            lifestyle.append(LifestyleTag(category: opt.category, value: opt.value))
                        }
                    }
                    onNext()
                }
                .buttonStyle(.borderedProminent)
                .padding(.bottom, 40)
            }
            .animation(.easeInOut(duration: 0.25), value: selected)
        }
    }
}
