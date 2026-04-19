import SwiftUI

struct FeedbackDetailView: View {
    let post: Post
    @Environment(\.dismiss) private var dismiss
    @EnvironmentObject private var apiService: APIService
    @EnvironmentObject private var eventTracker: EventTracker

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 0) {
                // Header band
                VStack(alignment: .leading, spacing: 10) {
                    HStack(spacing: 6) {
                        Image(systemName: "checklist")
                            .font(.title2)
                        Text("Quick Question")
                            .font(.title3.weight(.bold))
                        Spacer()
                    }
                    .foregroundColor(.white)

                    HStack(spacing: 6) {
                        Circle()
                            .fill(feedbackBlue)
                            .frame(width: 8, height: 8)
                        Text(post.agentName)
                            .font(.subheadline.weight(.medium))
                            .foregroundColor(.white.opacity(0.8))
                        Text("·")
                            .foregroundColor(.white.opacity(0.3))
                        Text(post.relativeTime)
                            .font(.subheadline)
                            .foregroundColor(.white.opacity(0.5))
                    }
                }
                .padding(20)
                .background(
                    LinearGradient(
                        colors: [feedbackBlue.opacity(0.5), feedbackBlue.opacity(0.2)],
                        startPoint: .topLeading,
                        endPoint: .bottomTrailing
                    )
                )

                VStack(alignment: .leading, spacing: 20) {
                    // Title
                    Text(post.title)
                        .font(.title3.weight(.bold))

                    // Body
                    if !post.body.isEmpty {
                        Text(post.body)
                            .font(.body)
                            .foregroundColor(.secondary)
                    }

                    // Inline card (interactive)
                    if let fd = post.feedbackData {
                        Group {
                            switch fd.feedbackType {
                            case "poll":
                                PollCardView(post: post, feedbackData: fd)
                            case "rating":
                                RatingCardView(post: post, feedbackData: fd)
                            case "freeform":
                                FreeformCardView(post: post, feedbackData: fd)
                            case "survey":
                                SurveyCardView(post: post, feedbackData: fd)
                            default:
                                PollCardView(post: post, feedbackData: fd)
                            }
                        }
                        .clipShape(RoundedRectangle(cornerRadius: 14))
                    }

                    Divider()
                }
                .padding()
            }
        }
        .navigationTitle("Quick Question")
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button { dismiss() } label: {
                    Image(systemName: "xmark.circle.fill")
                        .foregroundStyle(.secondary)
                }
            }
        }
        .onAppear { eventTracker.fireEvent(postID: post.id, type: "expand") }
    }

    private let feedbackBlue = Color(red: 0.365, green: 0.376, blue: 0.996)
}
