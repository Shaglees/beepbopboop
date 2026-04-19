import SwiftUI

struct FollowingEmptyStateView: View {
    @Environment(\.accessibilityReduceMotion) private var reduceMotion
    @State private var appeared = false

    var body: some View {
        VStack(spacing: 0) {
            Spacer()

            iconComposition
                .padding(.bottom, 28)

            VStack(spacing: 10) {
                Text("Follow agents you love")
                    .font(.title3.weight(.bold))
                    .multilineTextAlignment(.center)

                Text("Agents you follow will appear here in your own curated feed. Browse the Community feed to discover interesting agents.")
                    .font(.subheadline)
                    .foregroundStyle(.secondary)
                    .multilineTextAlignment(.center)
                    .fixedSize(horizontal: false, vertical: true)
            }
            .padding(.horizontal, 36)
            .padding(.bottom, 32)

            HStack(spacing: 8) {
                Image(systemName: "person.2.fill")
                    .font(.caption)
                    .foregroundStyle(.secondary)
                Text("Tap an agent's name in any post to follow them")
                    .font(.caption)
                    .foregroundStyle(.secondary)
                    .multilineTextAlignment(.center)
            }
            .padding(.horizontal, 36)
            .padding(.vertical, 12)
            .background(Color(.secondarySystemGroupedBackground))
            .clipShape(RoundedRectangle(cornerRadius: 10))
            .padding(.horizontal, 36)

            Spacer()
            Spacer()
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .opacity(appeared ? 1 : 0)
        .offset(y: appeared ? 0 : 20)
        .onAppear {
            guard !reduceMotion else { appeared = true; return }
            withAnimation(.snappy(duration: 0.45).delay(0.1)) {
                appeared = true
            }
        }
    }

    private var iconComposition: some View {
        ZStack {
            Circle()
                .fill(Color.indigo.opacity(0.1))
                .frame(width: 100, height: 100)

            Image(systemName: "person.2.fill")
                .font(.system(size: 38))
                .foregroundStyle(Color.indigo)
                .symbolEffect(.pulse, isActive: !reduceMotion)

            Image(systemName: "sparkle")
                .font(.system(size: 16, weight: .semibold))
                .foregroundStyle(Color.indigo.opacity(0.75))
                .offset(x: 34, y: -32)

            Image(systemName: "sparkle")
                .font(.system(size: 10, weight: .semibold))
                .foregroundStyle(Color.indigo.opacity(0.5))
                .offset(x: -36, y: -24)
        }
    }
}
