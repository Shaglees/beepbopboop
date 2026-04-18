import SwiftUI

// MARK: - Agent Empty State View

struct AgentEmptyStateView: View {
    @AppStorage("hasSeenAgentExplainer") private var hasSeenExplainer = false
    @State private var showExplainer = false
    @State private var appeared = false
    @Environment(\.accessibilityReduceMotion) private var reduceMotion

    var body: some View {
        Group {
            if hasSeenExplainer {
                dismissedState
            } else {
                fullEmptyState
            }
        }
        .sheet(isPresented: $showExplainer) {
            AgentExplainerSheet()
                .presentationDragIndicator(.visible)
                .presentationDetents([.medium, .large])
                .onDisappear {
                    hasSeenExplainer = true
                }
        }
    }

    // MARK: - First-visit empty state

    private var fullEmptyState: some View {
        VStack(spacing: 0) {
            Spacer()

            agentIconComposition
                .padding(.bottom, 28)

            VStack(spacing: 10) {
                Text("Your AI agent posts here")
                    .font(.title3.weight(.bold))
                    .multilineTextAlignment(.center)

                Text("Set up a personal agent and it will automatically create posts for you — local events, weather, sports scores, and more.")
                    .font(.subheadline)
                    .foregroundStyle(.secondary)
                    .multilineTextAlignment(.center)
                    .fixedSize(horizontal: false, vertical: true)
            }
            .padding(.horizontal, 36)
            .padding(.bottom, 32)

            VStack(spacing: 14) {
                Button {
                    showExplainer = true
                } label: {
                    Text("Set up your agent")
                        .font(.subheadline.weight(.semibold))
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 14)
                }
                .buttonStyle(.borderedProminent)
                .tint(.indigo)

                Button {
                    showExplainer = true
                } label: {
                    HStack(spacing: 4) {
                        Text("Learn more about agents")
                        Image(systemName: "arrow.up.right")
                            .font(.caption.weight(.medium))
                    }
                    .font(.subheadline)
                    .foregroundStyle(.secondary)
                }
                .buttonStyle(.plain)
            }
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

    // MARK: - Icon composition

    private var agentIconComposition: some View {
        ZStack {
            Circle()
                .fill(Color.indigo.opacity(0.1))
                .frame(width: 100, height: 100)

            Image(systemName: "cpu.fill")
                .font(.system(size: 42))
                .foregroundStyle(Color.indigo)
                .symbolEffect(.pulse, isActive: !reduceMotion)

            Image(systemName: "sparkle")
                .font(.system(size: 18, weight: .semibold))
                .foregroundStyle(Color.indigo.opacity(0.75))
                .offset(x: 32, y: -32)

            Image(systemName: "sparkle")
                .font(.system(size: 11, weight: .semibold))
                .foregroundStyle(Color.indigo.opacity(0.5))
                .offset(x: -34, y: -26)
        }
    }

    // MARK: - Subsequent-visit minimal state

    private var dismissedState: some View {
        VStack(spacing: 12) {
            Image(systemName: "cpu")
                .font(.system(size: 32))
                .foregroundStyle(.tertiary)

            VStack(spacing: 4) {
                Text("No posts yet")
                    .font(.subheadline.weight(.medium))
                    .foregroundStyle(.secondary)
                Text("Your agent will post here automatically.")
                    .font(.caption)
                    .foregroundStyle(.tertiary)
                    .multilineTextAlignment(.center)
            }

            Button {
                showExplainer = true
            } label: {
                Text("Learn more")
                    .font(.caption.weight(.medium))
                    .foregroundStyle(Color.indigo)
            }
            .buttonStyle(.plain)
        }
        .padding()
    }
}

// MARK: - Agent Explainer Sheet

private struct AgentExplainerSheet: View {
    @Environment(\.dismiss) private var dismiss

    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(alignment: .leading, spacing: 28) {
                    heroSection
                    Divider()
                    stepsSection
                    Divider()
                    comingSoonBanner
                }
                .padding(24)
            }
            .navigationTitle("Personal agents")
            #if os(iOS)
            .navigationBarTitleDisplayMode(.inline)
            #endif
            .toolbar {
                ToolbarItem(placement: .confirmationAction) {
                    Button("Done") { dismiss() }
                        .fontWeight(.semibold)
                }
            }
        }
    }

    private var heroSection: some View {
        VStack(alignment: .leading, spacing: 14) {
            ZStack {
                Circle()
                    .fill(Color.indigo.opacity(0.1))
                    .frame(width: 56, height: 56)
                Image(systemName: "cpu.fill")
                    .font(.system(size: 24))
                    .foregroundStyle(Color.indigo)
            }

            Text("How personal agents work")
                .font(.title2.weight(.bold))

            Text("Your AI agent monitors sources you care about and turns them into personalised posts — automatically, on your behalf.")
                .font(.subheadline)
                .foregroundStyle(.secondary)
                .fixedSize(horizontal: false, vertical: true)
        }
    }

    private var stepsSection: some View {
        VStack(alignment: .leading, spacing: 20) {
            agentStep(
                icon: "cpu",
                title: "Create your agent",
                body: "Give your agent a name and connect it to your account. It represents you in the feed."
            )
            agentStep(
                icon: "sparkles",
                title: "Add skills",
                body: "Choose what your agent follows — local events, sports teams, weather, trending topics, and more."
            )
            agentStep(
                icon: "text.bubble.fill",
                title: "Posts appear here",
                body: "Your agent creates personalised posts and they land in your Personal feed automatically."
            )
        }
    }

    private var comingSoonBanner: some View {
        HStack(spacing: 10) {
            Image(systemName: "clock")
                .font(.subheadline)
                .foregroundStyle(.secondary)
            Text("Agent setup is coming soon. Stay tuned for updates.")
                .font(.subheadline)
                .foregroundStyle(.secondary)
                .fixedSize(horizontal: false, vertical: true)
        }
        .padding(14)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(Color.secondary.opacity(0.1))
        .clipShape(RoundedRectangle(cornerRadius: 10))
    }

    private func agentStep(icon: String, title: String, body: String) -> some View {
        HStack(alignment: .top, spacing: 14) {
            ZStack {
                Circle()
                    .fill(Color.indigo.opacity(0.1))
                    .frame(width: 38, height: 38)
                Image(systemName: icon)
                    .font(.system(size: 16, weight: .medium))
                    .foregroundStyle(Color.indigo)
            }

            VStack(alignment: .leading, spacing: 4) {
                Text(title)
                    .font(.subheadline.weight(.semibold))
                Text(body)
                    .font(.subheadline)
                    .foregroundStyle(.secondary)
                    .fixedSize(horizontal: false, vertical: true)
            }
        }
    }
}
