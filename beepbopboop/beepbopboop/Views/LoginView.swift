import SwiftUI

struct LoginView: View {
    @ObservedObject var authService: AuthService
    @State private var identifier: String = "test-user-1"

    var body: some View {
        VStack(spacing: 20) {
            Text("BeepBopBoop")
                .font(.largeTitle)
                .fontWeight(.bold)

            Text("Dev Mode")
                .font(.caption)
                .foregroundColor(.secondary)

            TextField("User Identifier", text: $identifier)
                .textFieldStyle(.roundedBorder)
                .textInputAutocapitalization(.never)

            Button(action: {
                authService.signIn(identifier: identifier)
            }) {
                Text("Sign In")
                    .frame(maxWidth: .infinity)
            }
            .buttonStyle(.borderedProminent)
            .disabled(identifier.isEmpty)
        }
        .padding(32)
    }
}
