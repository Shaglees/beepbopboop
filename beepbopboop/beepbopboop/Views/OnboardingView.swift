import SwiftUI

struct OnboardingView: View {
    private let apiService: APIService
    let onComplete: () -> Void

    @State private var currentStep = 0
    @State private var profile = UserProfileIdentity(
        displayName: "",
        avatarUrl: "",
        timezone: "UTC+0",
        homeLocation: "",
        homeLat: nil,
        homeLon: nil
    )
    @State private var interests: [UserInterest] = []
    @State private var lifestyle: [LifestyleTag] = []
    @State private var contentPrefs: [ContentPref] = []
    @State private var targetFrequency: Int? = nil

    private let totalSteps = 7

    init(apiService: APIService, onComplete: @escaping () -> Void) {
        self.apiService = apiService
        self.onComplete = onComplete
    }

    var body: some View {
        VStack(spacing: 0) {
            ProgressView(value: Double(currentStep + 1), total: Double(totalSteps))
                .tint(.accentColor)
                .padding(.horizontal)
                .padding(.top, 8)

            TabView(selection: $currentStep) {
                OnboardingNameView(profile: $profile, onNext: nextStep)
                    .tag(0)
                OnboardingLocationView(profile: $profile, onNext: nextStep)
                    .tag(1)
                OnboardingNotificationsView(onNext: nextStep)
                    .tag(2)
                OnboardingInterestsView(interests: $interests, onNext: nextStep)
                    .tag(3)
                OnboardingFrequencyView(targetFrequency: $targetFrequency, onNext: nextStep)
                    .tag(4)
                OnboardingLifestyleView(lifestyle: $lifestyle, onNext: nextStep)
                    .tag(5)
                OnboardingPrefsView(contentPrefs: $contentPrefs, onComplete: finish)
                    .tag(6)
            }
            .tabViewStyle(.page(indexDisplayMode: .never))
            .animation(.easeInOut, value: currentStep)
        }
    }

    private func nextStep() {
        if currentStep < totalSteps - 1 {
            currentStep += 1
        }
    }

    private func finish() {
        Task {
            try? await apiService.updateProfile(identity: profile)
            try? await apiService.setInterests(interests)
            if !lifestyle.isEmpty {
                try? await apiService.setLifestyle(lifestyle)
            }

            var finalPrefs = contentPrefs
            if let freq = targetFrequency {
                if let idx = finalPrefs.firstIndex(where: { $0.category == nil }) {
                    finalPrefs[idx].maxPerDay = freq
                } else {
                    finalPrefs.append(ContentPref(category: nil, depth: "standard", tone: "casual", maxPerDay: freq))
                }
            }
            if !finalPrefs.isEmpty {
                try? await apiService.setContentPrefs(finalPrefs)
            }
            onComplete()
        }
    }
}
