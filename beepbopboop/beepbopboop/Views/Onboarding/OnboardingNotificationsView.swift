import SwiftUI
import UserNotifications

struct OnboardingNotificationsView: View {
    let onNext: () -> Void
    @State private var digestHour = 8
    @State private var granted = false

    var body: some View {
        VStack(spacing: 24) {
            Spacer()
            Text("Stay in the loop")
                .font(.system(size: 28, weight: .bold, design: .serif))
            Text("Get your daily digest and live score alerts.")
                .font(.system(size: 15))
                .foregroundStyle(.secondary)
                .multilineTextAlignment(.center)
                .padding(.horizontal, 40)
            Button(granted ? "Notifications enabled" : "Enable notifications") {
                UNUserNotificationCenter.current().requestAuthorization(options: [.alert, .badge, .sound]) { ok, _ in
                    DispatchQueue.main.async { granted = ok }
                }
            }
            .buttonStyle(.bordered)
            .disabled(granted)
            HStack {
                Text("Daily digest at")
                Picker("Hour", selection: $digestHour) {
                    ForEach(5..<23) { h in
                        Text("\(h % 12 == 0 ? 12 : h % 12) \(h < 12 ? "AM" : "PM")").tag(h)
                    }
                }
                .pickerStyle(.menu)
            }
            .padding(.horizontal, 40)
            Spacer()
            Button("Continue") { onNext() }
                .buttonStyle(.borderedProminent)
                .padding(.bottom, 40)
        }
    }
}
