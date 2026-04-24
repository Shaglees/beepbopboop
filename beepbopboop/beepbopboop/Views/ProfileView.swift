import SwiftUI

struct ProfileView: View {
    @EnvironmentObject var apiService: APIService
    @State private var profile: UserProfile?
    @State private var isLoading = true

    var body: some View {
        NavigationStack {
            Group {
                if isLoading {
                    ProgressView()
                } else if let profile {
                    ScrollView {
                        VStack(alignment: .leading, spacing: 24) {
                            // Identity section
                            VStack(alignment: .leading, spacing: 8) {
                                Text("PROFILE")
                                    .font(.system(size: 11, weight: .medium, design: .monospaced))
                                    .foregroundStyle(.secondary)
                                HStack(spacing: 12) {
                                    Circle()
                                        .fill(Color(.systemGray4))
                                        .frame(width: 48, height: 48)
                                        .overlay(
                                            Text(String(profile.identity.displayName.prefix(1)).uppercased())
                                                .font(.system(size: 20, weight: .bold, design: .serif))
                                        )
                                    VStack(alignment: .leading, spacing: 2) {
                                        Text(profile.identity.displayName)
                                            .font(.system(size: 17, weight: .semibold))
                                        Text("\(profile.identity.homeLocation) · \(profile.identity.timezone)")
                                            .font(.system(size: 13, design: .monospaced))
                                            .foregroundStyle(.secondary)
                                    }
                                }
                            }
                            .padding(.horizontal)

                            // Interests section
                            if !profile.interests.isEmpty {
                                VStack(alignment: .leading, spacing: 8) {
                                    Text("INTERESTS")
                                        .font(.system(size: 11, weight: .medium, design: .monospaced))
                                        .foregroundStyle(.secondary)
                                    FlowLayout(spacing: 8) {
                                        ForEach(profile.interests) { interest in
                                            HStack(spacing: 4) {
                                                Text(interest.topic)
                                                    .font(.system(size: 13))
                                                if interest.source == "inferred" {
                                                    Image(systemName: "sparkles")
                                                        .font(.system(size: 9))
                                                        .foregroundStyle(.secondary)
                                                }
                                                if interest.pausedUntil != nil {
                                                    Image(systemName: "pause.circle")
                                                        .font(.system(size: 9))
                                                        .foregroundStyle(.orange)
                                                }
                                            }
                                            .padding(.horizontal, 12)
                                            .padding(.vertical, 6)
                                            .background(Color(.systemGray6))
                                            .clipShape(Capsule())
                                        }
                                    }
                                }
                                .padding(.horizontal)
                            }

                            // Lifestyle section
                            if !profile.lifestyle.isEmpty {
                                VStack(alignment: .leading, spacing: 8) {
                                    Text("LIFESTYLE")
                                        .font(.system(size: 11, weight: .medium, design: .monospaced))
                                        .foregroundStyle(.secondary)
                                    FlowLayout(spacing: 8) {
                                        ForEach(profile.lifestyle, id: \.value) { tag in
                                            Text(tag.value.replacingOccurrences(of: "_", with: " ").capitalized)
                                                .font(.system(size: 13))
                                                .padding(.horizontal, 12)
                                                .padding(.vertical, 6)
                                                .background(Color(.systemGray6))
                                                .clipShape(Capsule())
                                        }
                                    }
                                }
                                .padding(.horizontal)
                            }

                            // Content prefs section
                            if !profile.contentPrefs.isEmpty {
                                VStack(alignment: .leading, spacing: 8) {
                                    Text("CONTENT PREFERENCES")
                                        .font(.system(size: 11, weight: .medium, design: .monospaced))
                                        .foregroundStyle(.secondary)
                                    ForEach(Array(profile.contentPrefs.enumerated()), id: \.offset) { _, pref in
                                        HStack {
                                            Text(pref.category ?? "Global")
                                                .font(.system(size: 14, weight: .medium))
                                            Spacer()
                                            Text("\(pref.depth) · \(pref.tone)")
                                                .font(.system(size: 13, design: .monospaced))
                                                .foregroundStyle(.secondary)
                                            if let max = pref.maxPerDay {
                                                Text("≤\(max)/day")
                                                    .font(.system(size: 13, design: .monospaced))
                                                    .foregroundStyle(.secondary)
                                            }
                                        }
                                    }
                                }
                                .padding(.horizontal)
                            }
                        }
                        .padding(.vertical)
                    }
                } else {
                    Text("Failed to load profile")
                        .foregroundStyle(.secondary)
                }
            }
            .navigationTitle("Profile")
            .task { await loadProfile() }
        }
    }

    private func loadProfile() async {
        isLoading = true
        defer { isLoading = false }
        profile = try? await apiService.getProfile()
    }
}
