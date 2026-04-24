import SwiftUI

struct OnboardingNameView: View {
    @Binding var profile: UserProfileIdentity
    let onNext: () -> Void

    var body: some View {
        VStack(spacing: 24) {
            Spacer()
            Text("What should we call you?")
                .font(.system(size: 28, weight: .bold, design: .serif))
            TextField("Your name", text: $profile.displayName)
                .textFieldStyle(.roundedBorder)
                .padding(.horizontal, 40)
            Spacer()
            Button("Continue") { onNext() }
                .disabled(profile.displayName.isEmpty)
                .buttonStyle(.borderedProminent)
                .padding(.bottom, 40)
        }
    }
}
