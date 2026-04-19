import SwiftUI

// MARK: - FeedbackCard (feed router)

/// Routes to the correct card sub-type based on feedbackData.feedbackType.
struct FeedbackCard: View {
    let post: Post

    init?(post: Post) {
        guard post.feedbackData != nil else { return nil }
        self.post = post
    }

    var body: some View {
        if let fd = post.feedbackData {
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
    }
}

// MARK: - Shared appearance

private let feedbackBlue = Color(red: 0.365, green: 0.376, blue: 0.996)
private let feedbackBg = Color(red: 0.055, green: 0.063, blue: 0.18)

private struct FeedbackCardHeader: View {
    let post: Post
    let icon: String

    var body: some View {
        HStack(spacing: 6) {
            Circle()
                .fill(feedbackBlue)
                .frame(width: 8, height: 8)
            Text(post.agentName)
                .font(.subheadline.weight(.medium))
                .foregroundColor(.white)
            HStack(spacing: 4) {
                Image(systemName: icon)
                    .font(.caption2)
                Text("Quick Question")
                    .font(.caption2.weight(.semibold))
            }
            .foregroundColor(.white)
            .padding(.horizontal, 7)
            .padding(.vertical, 3)
            .background(feedbackBlue.opacity(0.25))
            .cornerRadius(4)
            Spacer()
            Text(post.relativeTime)
                .font(.subheadline)
                .foregroundColor(.white.opacity(0.5))
        }
    }
}

// MARK: - PollCardView

struct PollCardView: View {
    let post: Post
    let feedbackData: FeedbackData

    @State private var selectedKey: String? = nil
    @State private var submitted = false
    @State private var tally: [String: Int] = [:]
    @State private var totalVotes = 0
    @EnvironmentObject private var apiService: APIService

    var body: some View {
        VStack(spacing: 0) {
            VStack(alignment: .leading, spacing: 14) {
                FeedbackCardHeader(post: post, icon: "checklist")

                // Question
                Text(feedbackData.question)
                    .font(.headline)
                    .foregroundColor(.white)
                    .lineLimit(3)

                // Optional reason/context
                if let reason = feedbackData.reason, !reason.isEmpty {
                    HStack(spacing: 4) {
                        Image(systemName: "info.circle")
                            .font(.caption2)
                        Text(reason)
                            .font(.caption)
                    }
                    .foregroundColor(feedbackBlue.opacity(0.8))
                }

                // Options
                if let options = feedbackData.options {
                    VStack(spacing: 8) {
                        ForEach(options) { option in
                            PollOptionRow(
                                option: option,
                                selectedKey: $selectedKey,
                                submitted: submitted,
                                votes: tally[option.key] ?? 0,
                                totalVotes: totalVotes,
                                accentColor: feedbackBlue
                            )
                        }
                    }
                }

                // Submit / results footer
                if !submitted {
                    Button {
                        guard let key = selectedKey else { return }
                        Task { await submitPoll(key: key) }
                    } label: {
                        Text("Submit")
                            .font(.subheadline.weight(.semibold))
                            .foregroundColor(selectedKey != nil ? .white : .white.opacity(0.3))
                            .frame(maxWidth: .infinity)
                            .padding(.vertical, 10)
                            .background(
                                RoundedRectangle(cornerRadius: 10)
                                    .fill(selectedKey != nil ? feedbackBlue : feedbackBlue.opacity(0.2))
                            )
                    }
                    .disabled(selectedKey == nil)
                    .buttonStyle(.plain)
                } else {
                    HStack(spacing: 6) {
                        Image(systemName: "checkmark.circle.fill")
                            .font(.caption)
                        Text("Thanks for your input!")
                            .font(.caption.weight(.semibold))
                    }
                    .foregroundColor(feedbackBlue)
                }
            }
            .padding(.horizontal, 16)
            .padding(.vertical, 14)
        }
        .background(feedbackBg)
    }

    private func submitPoll(key: String) async {
        var response = FeedbackResponse(type: "poll")
        response.selected = [key]
        do {
            try await apiService.submitFeedback(postID: post.id, response: response)
            let summary = try await apiService.getFeedbackSummary(postID: post.id)
            tally = summary.tally ?? [:]
            totalVotes = summary.totalResponses
            withAnimation(.spring(response: 0.4, dampingFraction: 0.8)) { submitted = true }
        } catch {
            // Silent failure — don't block UI
        }
    }
}

private struct PollOptionRow: View {
    let option: FeedbackOption
    @Binding var selectedKey: String?
    let submitted: Bool
    let votes: Int
    let totalVotes: Int
    let accentColor: Color

    private var pct: Double {
        guard totalVotes > 0 else { return 0 }
        return Double(votes) / Double(totalVotes)
    }

    var body: some View {
        let isSelected = selectedKey == option.key
        Button {
            if !submitted {
                withAnimation(.spring(response: 0.3, dampingFraction: 0.75)) {
                    selectedKey = isSelected ? nil : option.key
                }
                UIImpactFeedbackGenerator(style: .light).impactOccurred()
            }
        } label: {
            ZStack(alignment: .leading) {
                // Bar fill (after submit)
                if submitted {
                    GeometryReader { geo in
                        RoundedRectangle(cornerRadius: 8)
                            .fill(isSelected ? accentColor.opacity(0.35) : Color.white.opacity(0.05))
                            .frame(width: geo.size.width * pct)
                    }
                }

                HStack {
                    Image(systemName: isSelected ? "checkmark.circle.fill" : "circle")
                        .font(.body)
                        .foregroundColor(isSelected ? accentColor : .white.opacity(0.4))

                    Text(option.label)
                        .font(.subheadline)
                        .foregroundColor(isSelected ? .white : .white.opacity(0.7))

                    Spacer()

                    if submitted {
                        Text("\(Int(pct * 100))%")
                            .font(.caption.weight(.semibold))
                            .foregroundColor(isSelected ? accentColor : .white.opacity(0.4))
                    }
                }
                .padding(.horizontal, 12)
                .padding(.vertical, 10)
            }
            .frame(height: 42)
            .background(
                RoundedRectangle(cornerRadius: 8)
                    .fill(isSelected ? accentColor.opacity(0.12) : Color.white.opacity(0.05))
                    .overlay(
                        RoundedRectangle(cornerRadius: 8)
                            .stroke(isSelected ? accentColor.opacity(0.5) : Color.white.opacity(0.08), lineWidth: 1)
                    )
            )
        }
        .buttonStyle(.plain)
        .animation(.spring(response: 0.3, dampingFraction: 0.75), value: isSelected)
    }
}

// MARK: - RatingCardView

struct RatingCardView: View {
    let post: Post
    let feedbackData: FeedbackData

    @State private var selectedValue: Int? = nil
    @State private var submitted = false
    @State private var avgRating: Double? = nil
    @EnvironmentObject private var apiService: APIService

    private var maxValue: Int { Int(feedbackData.maxValue ?? 5) }

    var body: some View {
        VStack(alignment: .leading, spacing: 14) {
            FeedbackCardHeader(post: post, icon: "star.fill")

            Text(feedbackData.question)
                .font(.headline)
                .foregroundColor(.white)
                .lineLimit(3)

            if let reason = feedbackData.reason, !reason.isEmpty {
                HStack(spacing: 4) {
                    Image(systemName: "info.circle")
                        .font(.caption2)
                    Text(reason)
                        .font(.caption)
                }
                .foregroundColor(feedbackBlue.opacity(0.8))
            }

            // Star row
            HStack(spacing: 10) {
                ForEach(1...maxValue, id: \.self) { i in
                    Button {
                        if !submitted {
                            withAnimation(.spring(response: 0.25, dampingFraction: 0.6)) {
                                selectedValue = i
                            }
                            UIImpactFeedbackGenerator(style: .light).impactOccurred()
                        }
                    } label: {
                        Image(systemName: (selectedValue ?? 0) >= i ? "star.fill" : "star")
                            .font(.title2)
                            .foregroundColor((selectedValue ?? 0) >= i ? .yellow : .white.opacity(0.3))
                            .scaleEffect((selectedValue ?? 0) == i ? 1.2 : 1.0)
                    }
                    .buttonStyle(.plain)
                }
                Spacer()
            }

            if !submitted {
                Button {
                    guard let val = selectedValue else { return }
                    Task { await submitRating(value: Double(val)) }
                } label: {
                    Text("Submit")
                        .font(.subheadline.weight(.semibold))
                        .foregroundColor(selectedValue != nil ? .white : .white.opacity(0.3))
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 10)
                        .background(
                            RoundedRectangle(cornerRadius: 10)
                                .fill(selectedValue != nil ? feedbackBlue : feedbackBlue.opacity(0.2))
                        )
                }
                .disabled(selectedValue == nil)
                .buttonStyle(.plain)
            } else {
                HStack(spacing: 6) {
                    Image(systemName: "checkmark.circle.fill")
                        .font(.caption)
                    if let avg = avgRating {
                        Text(String(format: "Thanks! Average rating: %.1f / \(maxValue)", avg))
                            .font(.caption.weight(.semibold))
                    } else {
                        Text("Thanks for your rating!")
                            .font(.caption.weight(.semibold))
                    }
                }
                .foregroundColor(feedbackBlue)
            }
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 14)
        .background(feedbackBg)
    }

    private func submitRating(value: Double) async {
        var response = FeedbackResponse(type: "rating")
        response.value = value
        do {
            try await apiService.submitFeedback(postID: post.id, response: response)
            let summary = try await apiService.getFeedbackSummary(postID: post.id)
            avgRating = summary.avgRating
            withAnimation(.spring(response: 0.4, dampingFraction: 0.8)) { submitted = true }
        } catch {}
    }
}

// MARK: - FreeformCardView

struct FreeformCardView: View {
    let post: Post
    let feedbackData: FeedbackData

    @State private var text = ""
    @State private var submitted = false
    @FocusState private var isFocused: Bool
    @EnvironmentObject private var apiService: APIService

    private var isSubmittable: Bool {
        !text.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 14) {
            FeedbackCardHeader(post: post, icon: "text.cursor")

            Text(feedbackData.question)
                .font(.headline)
                .foregroundColor(.white)
                .lineLimit(3)

            if let reason = feedbackData.reason, !reason.isEmpty {
                HStack(spacing: 4) {
                    Image(systemName: "info.circle")
                        .font(.caption2)
                    Text(reason)
                        .font(.caption)
                }
                .foregroundColor(feedbackBlue.opacity(0.8))
            }

            if !submitted {
                // Text editor
                ZStack(alignment: .topLeading) {
                    RoundedRectangle(cornerRadius: 10)
                        .fill(Color.white.opacity(0.06))
                        .overlay(
                            RoundedRectangle(cornerRadius: 10)
                                .stroke(isFocused ? feedbackBlue.opacity(0.6) : Color.white.opacity(0.1), lineWidth: 1)
                        )
                        .frame(minHeight: 80)

                    if text.isEmpty {
                        Text("Type your answer…")
                            .font(.subheadline)
                            .foregroundColor(.white.opacity(0.3))
                            .padding(10)
                            .allowsHitTesting(false)
                    }

                    TextEditor(text: $text)
                        .font(.subheadline)
                        .foregroundColor(.white)
                        .scrollContentBackground(.hidden)
                        .background(Color.clear)
                        .padding(6)
                        .focused($isFocused)
                        .frame(minHeight: 80)
                }

                Button {
                    Task { await submitFreeform() }
                } label: {
                    Text("Submit")
                        .font(.subheadline.weight(.semibold))
                        .foregroundColor(isSubmittable ? .white : .white.opacity(0.3))
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 10)
                        .background(
                            RoundedRectangle(cornerRadius: 10)
                                .fill(isSubmittable ? feedbackBlue : feedbackBlue.opacity(0.2))
                        )
                }
                .disabled(!isSubmittable)
                .buttonStyle(.plain)
            } else {
                HStack(spacing: 6) {
                    Image(systemName: "checkmark.circle.fill")
                        .font(.caption)
                    Text("Got it, thanks!")
                        .font(.caption.weight(.semibold))
                }
                .foregroundColor(feedbackBlue)
                .padding(.vertical, 8)
            }
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 14)
        .background(feedbackBg)
    }

    private func submitFreeform() async {
        var response = FeedbackResponse(type: "freeform")
        response.text = text.trimmingCharacters(in: .whitespacesAndNewlines)
        do {
            try await apiService.submitFeedback(postID: post.id, response: response)
            withAnimation(.spring(response: 0.4, dampingFraction: 0.8)) { submitted = true }
        } catch {}
    }
}

// MARK: - SurveyCardView

struct SurveyCardView: View {
    let post: Post
    let feedbackData: FeedbackData

    @State private var submitted = false
    @State private var pollAnswers: [String: String] = [:]   // question key → selected option key
    @State private var freeformAnswers: [String: String] = [:] // question key → text
    @State private var ratingAnswers: [String: Int] = [:]    // question key → star value
    @EnvironmentObject private var apiService: APIService

    private var questions: [SurveyQuestion] { feedbackData.questions ?? [] }

    private var allAnswered: Bool {
        for q in questions {
            switch q.type {
            case "poll":
                if pollAnswers[q.key] == nil { return false }
            case "freeform":
                if (freeformAnswers[q.key] ?? "").trimmingCharacters(in: .whitespacesAndNewlines).isEmpty { return false }
            case "rating":
                if ratingAnswers[q.key] == nil { return false }
            default:
                break
            }
        }
        return true
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 14) {
            FeedbackCardHeader(post: post, icon: "list.bullet.clipboard")

            Text(feedbackData.question)
                .font(.headline)
                .foregroundColor(.white)
                .lineLimit(3)

            if let reason = feedbackData.reason, !reason.isEmpty {
                HStack(spacing: 4) {
                    Image(systemName: "info.circle").font(.caption2)
                    Text(reason).font(.caption)
                }
                .foregroundColor(feedbackBlue.opacity(0.8))
            }

            if !submitted {
                VStack(alignment: .leading, spacing: 16) {
                    ForEach(Array(questions.enumerated()), id: \.element.id) { idx, question in
                        VStack(alignment: .leading, spacing: 8) {
                            HStack(spacing: 6) {
                                Text("\(idx + 1).")
                                    .font(.caption2.weight(.bold))
                                    .foregroundColor(feedbackBlue)
                                    .frame(width: 16, alignment: .trailing)
                                Text(question.text)
                                    .font(.subheadline.weight(.medium))
                                    .foregroundColor(.white)
                            }

                            switch question.type {
                            case "poll":
                                if let opts = question.options {
                                    VStack(spacing: 6) {
                                        ForEach(opts) { opt in
                                            let isSelected = pollAnswers[question.key] == opt.key
                                            Button {
                                                withAnimation(.spring(response: 0.25, dampingFraction: 0.7)) {
                                                    pollAnswers[question.key] = isSelected ? nil : opt.key
                                                }
                                                UIImpactFeedbackGenerator(style: .light).impactOccurred()
                                            } label: {
                                                HStack {
                                                    Image(systemName: isSelected ? "checkmark.circle.fill" : "circle")
                                                        .font(.body)
                                                        .foregroundColor(isSelected ? feedbackBlue : .white.opacity(0.4))
                                                    Text(opt.label)
                                                        .font(.subheadline)
                                                        .foregroundColor(isSelected ? .white : .white.opacity(0.7))
                                                    Spacer()
                                                }
                                                .padding(.horizontal, 12)
                                                .padding(.vertical, 8)
                                                .background(
                                                    RoundedRectangle(cornerRadius: 8)
                                                        .fill(isSelected ? feedbackBlue.opacity(0.12) : Color.white.opacity(0.05))
                                                        .overlay(
                                                            RoundedRectangle(cornerRadius: 8)
                                                                .stroke(isSelected ? feedbackBlue.opacity(0.5) : Color.white.opacity(0.08), lineWidth: 1)
                                                        )
                                                )
                                            }
                                            .buttonStyle(.plain)
                                        }
                                    }
                                }
                            case "rating":
                                HStack(spacing: 8) {
                                    ForEach(1...5, id: \.self) { i in
                                        Button {
                                            withAnimation(.spring(response: 0.25, dampingFraction: 0.6)) {
                                                ratingAnswers[question.key] = i
                                            }
                                            UIImpactFeedbackGenerator(style: .light).impactOccurred()
                                        } label: {
                                            Image(systemName: (ratingAnswers[question.key] ?? 0) >= i ? "star.fill" : "star")
                                                .font(.title3)
                                                .foregroundColor((ratingAnswers[question.key] ?? 0) >= i ? .yellow : .white.opacity(0.3))
                                        }
                                        .buttonStyle(.plain)
                                    }
                                    Spacer()
                                }
                            default: // freeform
                                TextField("Your answer…", text: Binding(
                                    get: { freeformAnswers[question.key] ?? "" },
                                    set: { freeformAnswers[question.key] = $0 }
                                ))
                                .textFieldStyle(.plain)
                                .font(.subheadline)
                                .foregroundColor(.white)
                                .padding(10)
                                .background(
                                    RoundedRectangle(cornerRadius: 8)
                                        .fill(Color.white.opacity(0.06))
                                        .overlay(
                                            RoundedRectangle(cornerRadius: 8)
                                                .stroke(Color.white.opacity(0.1), lineWidth: 1)
                                        )
                                )
                            }
                        }
                    }
                }

                Button {
                    Task { await submitSurvey() }
                } label: {
                    Text("Submit Survey")
                        .font(.subheadline.weight(.semibold))
                        .foregroundColor(allAnswered ? .white : .white.opacity(0.3))
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 10)
                        .background(
                            RoundedRectangle(cornerRadius: 10)
                                .fill(allAnswered ? feedbackBlue : feedbackBlue.opacity(0.2))
                        )
                }
                .disabled(!allAnswered)
                .buttonStyle(.plain)
            } else {
                HStack(spacing: 6) {
                    Image(systemName: "checkmark.circle.fill").font(.caption)
                    Text("Survey complete — thanks!")
                        .font(.caption.weight(.semibold))
                }
                .foregroundColor(feedbackBlue)
                .padding(.vertical, 8)
            }
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 14)
        .background(feedbackBg)
    }

    private func submitSurvey() async {
        var answers: [SurveyAnswer] = []
        for q in questions {
            switch q.type {
            case "poll":
                let sel = pollAnswers[q.key].map { [$0] } ?? []
                answers.append(SurveyAnswer(question: q.key, type: "poll", selected: sel))
            case "freeform":
                let t = freeformAnswers[q.key] ?? ""
                answers.append(SurveyAnswer(question: q.key, type: "freeform", text: t))
            case "rating":
                let v = Double(ratingAnswers[q.key] ?? 0)
                answers.append(SurveyAnswer(question: q.key, type: "rating", value: v))
            default:
                break
            }
        }

        var response = FeedbackResponse(type: "survey")
        response.answers = answers

        do {
            try await apiService.submitFeedback(postID: post.id, response: response)
            withAnimation(.spring(response: 0.4, dampingFraction: 0.8)) { submitted = true }
        } catch {}
    }
}
