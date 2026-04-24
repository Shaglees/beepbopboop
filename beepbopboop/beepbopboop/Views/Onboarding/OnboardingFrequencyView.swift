import SwiftUI

struct OnboardingFrequencyView: View {
    @Binding var targetFrequency: Int?
    let onNext: () -> Void
    @State private var sliderValue: Double = 10

    var body: some View {
        VStack(spacing: 24) {
            Spacer()
            Text("How much content?")
                .font(.system(size: 28, weight: .bold, design: .serif))
            Text("You can change this anytime in settings.")
                .font(.system(size: 15))
                .foregroundStyle(.secondary)
            VStack(spacing: 8) {
                Text("\(Int(sliderValue)) posts per day")
                    .font(.system(size: 20, weight: .semibold, design: .monospaced))
                Slider(value: $sliderValue, in: 3...25, step: 1)
                    .padding(.horizontal, 40)
            }
            Spacer()
            Button("Continue") {
                targetFrequency = Int(sliderValue)
                onNext()
            }
            .buttonStyle(.borderedProminent)
            .padding(.bottom, 40)
        }
    }
}
