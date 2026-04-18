import Combine
import Foundation
import UserNotifications

@MainActor
class NotificationService: ObservableObject {
    @Published var authorizationStatus: UNAuthorizationStatus = .notDetermined

    private let center = UNUserNotificationCenter.current()

    func checkStatus() async {
        let settings = await center.notificationSettings()
        authorizationStatus = settings.authorizationStatus
    }

    func requestAuthorization() async -> Bool {
        do {
            let granted = try await center.requestAuthorization(options: [.alert, .sound, .badge])
            let settings = await center.notificationSettings()
            authorizationStatus = settings.authorizationStatus
            return granted
        } catch {
            return false
        }
    }

    // Schedules (or replaces) tomorrow's daily digest notification.
    // Call this on each launch after loading the user's digest posts.
    func scheduleDailyDigest(posts: [DigestPostPreview], digestHour: Int) async {
        guard authorizationStatus == .authorized else { return }

        center.removePendingNotificationRequests(withIdentifiers: ["daily-digest"])

        guard !posts.isEmpty else { return }

        let top = posts[0]
        let content = UNMutableNotificationContent()
        content.title = "Your daily digest"
        content.body = posts.count > 1
            ? "\(top.title) and \(posts.count - 1) more"
            : top.title
        content.sound = .default
        if let postID = posts.first?.id {
            content.userInfo = ["post_id": postID]
        }

        var components = DateComponents()
        components.hour = digestHour
        components.minute = 0
        let trigger = UNCalendarNotificationTrigger(dateMatching: components, repeats: true)

        let request = UNNotificationRequest(
            identifier: "daily-digest",
            content: content,
            trigger: trigger
        )

        try? await center.add(request)
    }

    func cancelDailyDigest() {
        center.removePendingNotificationRequests(withIdentifiers: ["daily-digest"])
    }
}

struct DigestPostPreview {
    let id: String
    let title: String
    let body: String
}
