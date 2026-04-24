import SwiftUI

struct OnboardingPrefsView: View {
    @Binding var contentPrefs: [ContentPref]
    let onComplete: () -> Void
    @State private var depth = "standard"
    @State private var tone = "casual"

    var body: some View {
        VStack(spacing: 24) {
            Spacer()
            Text("Content style")
                .font(.system(size: 28, weight: .bold, design: .serif))
            Text("How should your feed feel?")
                .font(.system(size: 15))
                .foregroundStyle(.secondary)

            VStack(alignment: .leading, spacing: 12) {
                Text("DEPTH")
                    .font(.system(size: 11, weight: .medium, design: .monospaced))
                    .foregroundStyle(.secondary)
                Picker("Depth", selection: $depth) {
                    Text("Brief").tag("brief")
                    Text("Standard").tag("standard")
                    Text("Detailed").tag("detailed")
                }
                .pickerStyle(.segmented)
            }
            .padding(.horizontal, 40)

            VStack(alignment: .leading, spacing: 12) {
                Text("TONE")
                    .font(.system(size: 11, weight: .medium, design: .monospaced))
                    .foregroundStyle(.secondary)
                Picker("Tone", selection: $tone) {
                    Text("Casual").tag("casual")
                    Text("Informative").tag("informative")
                    Text("Playful").tag("playful")
                }
                .pickerStyle(.segmented)
            }
            .padding(.horizontal, 40)

            Spacer()
            Button("Finish setup") {
                contentPrefs = [ContentPref(category: nil, depth: depth, tone: tone, maxPerDay: nil)]
                onComplete()
            }
            .buttonStyle(.borderedProminent)
            .padding(.bottom, 40)
        }
    }
}
