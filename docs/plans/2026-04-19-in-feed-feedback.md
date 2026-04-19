# In-Feed Feedback Posts Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add poll, survey, freeform, and rating feedback posts that users can respond to inline in the feed, storing responses in a `user_feedback` table and exposing them via two new API endpoints.

**Architecture:** A new `feedback` display hint stores structured question data as JSON in `external_url`. A `FeedbackRepo` handles upsert/aggregation. iOS renders four card types (poll, rating, freeform, survey) and a unified detail view, all calling two new endpoints. The reference implementation lives in worktree `strange-tesla-fd02c4` — copy from there rather than writing from scratch.

**Tech Stack:** Go (chi router, database/sql, PostgreSQL JSONB), SwiftUI (EnvironmentObject, async/await, optimistic UI), Firebase Auth middleware.

---

### Task 1: Backend — Add "feedback" display hint to post.go

**Files:**
- Modify: `backend/internal/handler/post.go`

**Step 1: Add to ValidDisplayHints map (line 60, after "box_score")**

```go
"box_score":        true,
"feedback":         true,
```

**Step 2: Add to structuredHint boolean (line 192, after "box_score")**

```go
req.DisplayHint == "player_spotlight" || req.DisplayHint == "box_score" ||
    req.DisplayHint == "feedback"
```

**Step 3: Add feedback case to the structured hint validation switch (after the "fitness" case, around line 270)**

```go
case "feedback":
    validateFeedbackData(req.ExternalURL, &errs, &warns)
```

**Step 4: Add feedback to the "external_url required" guard (after "pet_spotlight", around line 277)**

```go
req.DisplayHint == "pet_spotlight" || req.DisplayHint == "feedback" {
```

**Step 5: Add validateFeedbackData function (add before the `// --- Handlers ---` comment)**

```go
// --- Feedback data validation ---

type feedbackDataValidation struct {
	FeedbackType *string `json:"feedback_type"`
	Question     *string `json:"question"`
}

func validateFeedbackData(externalURL string, errs *[]validationIssue, warns *[]validationIssue) {
	var f feedbackDataValidation
	if err := json.Unmarshal([]byte(externalURL), &f); err != nil {
		*errs = append(*errs, validationIssue{Field: "external_url", Code: "invalid_json", Message: "feedback external_url must be valid JSON"})
		return
	}
	if f.FeedbackType == nil || *f.FeedbackType == "" {
		*errs = append(*errs, validationIssue{Field: "external_url.feedback_type", Code: "required", Message: "feedback_type is required (poll, survey, freeform, rating)"})
	} else {
		switch *f.FeedbackType {
		case "poll", "survey", "freeform", "rating":
			// valid
		default:
			*errs = append(*errs, validationIssue{Field: "external_url.feedback_type", Code: "invalid", Message: fmt.Sprintf("unknown feedback_type %q; must be poll, survey, freeform, or rating", *f.FeedbackType)})
		}
	}
	if f.Question == nil || *f.Question == "" {
		*errs = append(*errs, validationIssue{Field: "external_url.question", Code: "required", Message: "feedback question is required"})
	}
}
```

**Step 6: Verify Go compiles**
```bash
cd /Users/shanegleeson/Repos/beepbopboop/backend && go build ./...
```
Expected: no output (clean build)

**Step 7: Commit**
```bash
git add backend/internal/handler/post.go
git commit -m "feat: add feedback display hint to ValidDisplayHints + validation"
```

---

### Task 2: Backend — Add feedback model types to model.go

**Files:**
- Modify: `backend/internal/model/model.go`

**Step 1: Append these types to the end of model.go**

```go
// UserFeedback stores a raw response to a feedback post.
type UserFeedback struct {
	ID        int64           `json:"id"`
	PostID    string          `json:"post_id"`
	UserID    string          `json:"user_id"`
	Response  json.RawMessage `json:"response"`
	CreatedAt time.Time       `json:"created_at"`
}

// FeedbackResponseBody is the request body for POST /posts/{postID}/response.
type FeedbackResponseBody struct {
	Type     string          `json:"type"`     // "poll", "freeform", "rating", "survey"
	Selected []string        `json:"selected"` // poll: selected option keys
	Text     string          `json:"text"`     // freeform: free text answer
	Value    *float64        `json:"value"`    // rating: numeric value
	Answers  json.RawMessage `json:"answers"`  // survey: array of {question, answer}
}

// FeedbackSummary is the aggregated response summary for GET /posts/{postID}/responses.
type FeedbackSummary struct {
	PostID         string          `json:"post_id"`
	TotalResponses int             `json:"total_responses"`
	MyResponse     json.RawMessage `json:"my_response,omitempty"`
	Tally          map[string]int  `json:"tally,omitempty"` // poll: option key → count
	AvgRating      *float64        `json:"avg_rating,omitempty"`
}

// FeedbackData is parsed from external_url for feedback display hints.
type FeedbackData struct {
	FeedbackType string           `json:"feedback_type"` // "poll", "freeform", "rating", "survey"
	Question     string           `json:"question"`
	Reason       string           `json:"reason,omitempty"`
	Options      []FeedbackOption `json:"options,omitempty"`
	MinValue     *float64         `json:"min_value,omitempty"`
	MaxValue     *float64         `json:"max_value,omitempty"`
	Questions    []SurveyQuestion `json:"questions,omitempty"`
}

// FeedbackOption is one choice in a poll.
type FeedbackOption struct {
	Key   string `json:"key"`
	Label string `json:"label"`
}

// SurveyQuestion is one question in a multi-question survey.
type SurveyQuestion struct {
	Key     string           `json:"key"`
	Text    string           `json:"text"`
	Type    string           `json:"type"` // "poll", "freeform", "rating"
	Options []FeedbackOption `json:"options,omitempty"`
}
```

**Step 2: Verify Go compiles**
```bash
cd /Users/shanegleeson/Repos/beepbopboop/backend && go build ./...
```

**Step 3: Commit**
```bash
git add backend/internal/model/model.go
git commit -m "feat: add feedback model types (UserFeedback, FeedbackData, FeedbackSummary)"
```

---

### Task 3: Backend — Create FeedbackRepo

**Files:**
- Create: `backend/internal/repository/feedback_repo.go`

**Step 1: Copy from reference worktree**
```bash
cp /Users/shanegleeson/Repos/beepbopboop/.claude/worktrees/strange-tesla-fd02c4/backend/internal/repository/feedback_repo.go \
   /Users/shanegleeson/Repos/beepbopboop/backend/internal/repository/feedback_repo.go
```

**Step 2: Verify Go compiles**
```bash
cd /Users/shanegleeson/Repos/beepbopboop/backend && go build ./...
```

**Step 3: Commit**
```bash
git add backend/internal/repository/feedback_repo.go
git commit -m "feat: add FeedbackRepo (upsert + aggregated summary)"
```

---

### Task 4: Backend — Add database schema for user_feedback

**Files:**
- Modify: `backend/internal/database/database.go`

**Step 1: Add user_feedback table and preference_context column**

After the last `db.Exec(...)` before `return db, nil`, add:

```go
// user_feedback stores raw responses to feedback posts
db.Exec(`CREATE TABLE IF NOT EXISTS user_feedback (
    id          BIGSERIAL PRIMARY KEY,
    post_id     TEXT NOT NULL REFERENCES posts(id),
    user_id     TEXT NOT NULL REFERENCES users(id),
    response    JSONB NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
)`)
db.Exec("CREATE INDEX IF NOT EXISTS idx_user_feedback_post_id ON user_feedback(post_id)")
db.Exec("CREATE INDEX IF NOT EXISTS idx_user_feedback_user_id ON user_feedback(user_id)")
db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_user_feedback_post_user ON user_feedback(post_id, user_id)")
// preference_context: agent-writable summary injected into agent prompts
db.Exec("ALTER TABLE user_settings ADD COLUMN IF NOT EXISTS preference_context JSONB")
```

**Step 2: Verify Go compiles**
```bash
cd /Users/shanegleeson/Repos/beepbopboop/backend && go build ./...
```

**Step 3: Commit**
```bash
git add backend/internal/database/database.go
git commit -m "feat: add user_feedback table + preference_context column"
```

---

### Task 5: Backend — Create FeedbackHandler

**Files:**
- Create: `backend/internal/handler/feedback.go`

**Step 1: Copy from reference worktree**
```bash
cp /Users/shanegleeson/Repos/beepbopboop/.claude/worktrees/strange-tesla-fd02c4/backend/internal/handler/feedback.go \
   /Users/shanegleeson/Repos/beepbopboop/backend/internal/handler/feedback.go
```

**Step 2: Verify Go compiles**
```bash
cd /Users/shanegleeson/Repos/beepbopboop/backend && go build ./...
```

**Step 3: Commit**
```bash
git add backend/internal/handler/feedback.go
git commit -m "feat: add FeedbackHandler (SubmitResponse + GetResponses)"
```

---

### Task 6: Backend — Register feedback routes in main.go

**Files:**
- Modify: `backend/cmd/server/main.go`

**Step 1: Add feedbackRepo after existing repos**

After `pushTokenRepo := repository.NewPushTokenRepo(db)`, add:
```go
feedbackRepo := repository.NewFeedbackRepo(db)
```

**Step 2: Add feedbackH after existing handlers**

After `sportsH := handler.NewSportsHandler(sportsSvc)`, add:
```go
feedbackH := handler.NewFeedbackHandler(userRepo, feedbackRepo)
```

**Step 3: Register routes in the Firebase-auth group**

After `r.Get("/sports/scores", sportsH.GetScores)`, add:
```go
r.Post("/posts/{postID}/response", feedbackH.SubmitResponse)
r.Get("/posts/{postID}/responses", feedbackH.GetResponses)
```

**Step 4: Verify Go compiles**
```bash
cd /Users/shanegleeson/Repos/beepbopboop/backend && go build ./...
```

**Step 5: Commit**
```bash
git add backend/cmd/server/main.go
git commit -m "feat: register feedback routes POST/GET /posts/{id}/response(s)"
```

---

### Task 7: iOS — Create FeedbackData.swift

**Files:**
- Create: `beepbopboop/beepbopboop/Models/FeedbackData.swift`

**Step 1: Copy from reference worktree**
```bash
cp /Users/shanegleeson/Repos/beepbopboop/.claude/worktrees/strange-tesla-fd02c4/beepbopboop/beepbopboop/Models/FeedbackData.swift \
   /Users/shanegleeson/Repos/beepbopboop/beepbopboop/beepbopboop/Models/FeedbackData.swift
```

**Step 2: Commit**
```bash
git add beepbopboop/beepbopboop/Models/FeedbackData.swift
git commit -m "feat: add FeedbackData iOS model (FeedbackData, FeedbackResponse, FeedbackSummary)"
```

---

### Task 8: iOS — Extend Post.swift with feedback support

**Files:**
- Modify: `beepbopboop/beepbopboop/Models/Post.swift`

**Step 1: Add `.feedback` to the DisplayHintValue enum (line 192, after `.boxScore`)**

```swift
case boxScore
case feedback
```

**Step 2: Add `"feedback"` case to `displayHintValue` switch (line 223, before `default`)**

```swift
case "feedback": return .feedback
```

**Step 3: Add feedback to `hintColor` switch (line 306, before the closing `}`)**

```swift
case .feedback: return Color(red: 0.365, green: 0.376, blue: 0.996)
```

**Step 4: Add feedback to `hintIcon` switch (line 339, before closing `}`)**

```swift
case .feedback: return feedbackData?.feedbackType == "rating" ? "star.fill" : "checklist"
```

**Step 5: Add feedback to `hintLabel` switch (line 372, before closing `}`)**

```swift
case .feedback: return "Quick Question"
```

**Step 6: Add `feedbackData` computed property**

In the data parsing section (after the last existing `var xxxData: ...` computed property, around line 575), add:

```swift
/// Parsed feedback data from externalURL (for feedback display_hint posts).
var feedbackData: FeedbackData? {
    guard displayHintValue == .feedback,
          let json = externalURL,
          let data = json.data(using: .utf8) else { return nil }
    return try? JSONDecoder().decode(FeedbackData.self, from: data)
}
```

**Step 7: Commit**
```bash
git add beepbopboop/beepbopboop/Models/Post.swift
git commit -m "feat: extend Post model with .feedback DisplayHintValue + feedbackData parser"
```

---

### Task 9: iOS — Create FeedbackCards.swift

**Files:**
- Create: `beepbopboop/beepbopboop/Views/FeedbackCards.swift`

**Step 1: Copy from reference worktree**
```bash
cp /Users/shanegleeson/Repos/beepbopboop/.claude/worktrees/strange-tesla-fd02c4/beepbopboop/beepbopboop/Views/FeedbackCards.swift \
   /Users/shanegleeson/Repos/beepbopboop/beepbopboop/beepbopboop/Views/FeedbackCards.swift
```

**Step 2: Commit**
```bash
git add beepbopboop/beepbopboop/Views/FeedbackCards.swift
git commit -m "feat: add FeedbackCard, PollCardView, RatingCardView, FreeformCardView, SurveyCardView"
```

---

### Task 10: iOS — Create FeedbackDetailView.swift

**Files:**
- Create: `beepbopboop/beepbopboop/Views/FeedbackDetailView.swift`

**Step 1: Copy from reference worktree**
```bash
cp /Users/shanegleeson/Repos/beepbopboop/.claude/worktrees/strange-tesla-fd02c4/beepbopboop/beepbopboop/Views/FeedbackDetailView.swift \
   /Users/shanegleeson/Repos/beepbopboop/beepbopboop/beepbopboop/Views/FeedbackDetailView.swift
```

**Step 2: Commit**
```bash
git add beepbopboop/beepbopboop/Views/FeedbackDetailView.swift
git commit -m "feat: add FeedbackDetailView"
```

---

### Task 11: iOS — Wire .feedback into FeedItemView.swift

**Files:**
- Modify: `beepbopboop/beepbopboop/Views/FeedItemView.swift`

**Step 1: Add `.feedback` to the specialized-card array (line 16)**

Change:
```swift
if [.outfit, .weather, .scoreboard, .matchup, .standings, .movie, .show, .playerSpotlight, .entertainment, .album, .concert, .gameRelease, .gameReview, .restaurant, .destination, .science, .petSpotlight, .fitness].contains(post.displayHintValue) {
```
To:
```swift
if [.outfit, .weather, .scoreboard, .matchup, .standings, .movie, .show, .playerSpotlight, .entertainment, .album, .concert, .gameRelease, .gameReview, .restaurant, .destination, .science, .petSpotlight, .fitness, .feedback].contains(post.displayHintValue) {
```

**Step 2: Add feedback case in cardContent switch (after the `.fitness` case)**

```swift
case .feedback:
    if let card = FeedbackCard(post: post) {
        card
    } else {
        StandardCard(post: post)
    }
```

**Step 3: Commit**
```bash
git add beepbopboop/beepbopboop/Views/FeedItemView.swift
git commit -m "feat: wire .feedback into FeedItemView"
```

---

### Task 12: iOS — Wire .feedback into PostDetailView.swift

**Files:**
- Modify: `beepbopboop/beepbopboop/Views/PostDetailView.swift`

**Step 1: Add feedback case in the detail body switch (before the `default` case at line 75)**

```swift
case .feedback:
    FeedbackDetailView(post: post)
```

**Step 2: Commit**
```bash
git add beepbopboop/beepbopboop/Views/PostDetailView.swift
git commit -m "feat: wire .feedback into PostDetailView"
```

---

### Task 13: iOS — Add APIService feedback methods

**Files:**
- Modify: `beepbopboop/beepbopboop/Services/APIService.swift`

**Step 1: Find the end of APIService (before the last `}`) and add**

```swift
// MARK: - Feedback

/// Submit a response to a feedback post.
@MainActor
func submitFeedback(postID: String, response: FeedbackResponse) async throws {
    let token = authService.getToken()
    guard let url = URL(string: "\(baseURL)/posts/\(postID)/response") else {
        throw APIError.invalidURL
    }
    var request = URLRequest(url: url)
    request.httpMethod = "POST"
    request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
    request.setValue("application/json", forHTTPHeaderField: "Content-Type")
    request.httpBody = try JSONEncoder().encode(response)

    let (_, httpResponse) = try await URLSession.shared.data(for: request)
    guard let http = httpResponse as? HTTPURLResponse,
          (200...299).contains(http.statusCode) else {
        throw APIError.httpError((httpResponse as? HTTPURLResponse)?.statusCode ?? 0)
    }
}

/// Fetch aggregated responses for a feedback post.
@MainActor
func getFeedbackSummary(postID: String) async throws -> FeedbackSummary {
    let token = authService.getToken()
    guard let url = URL(string: "\(baseURL)/posts/\(postID)/responses") else {
        throw APIError.invalidURL
    }
    var request = URLRequest(url: url)
    request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")

    let (data, httpResponse) = try await URLSession.shared.data(for: request)
    guard let http = httpResponse as? HTTPURLResponse, http.statusCode == 200 else {
        throw APIError.httpError((httpResponse as? HTTPURLResponse)?.statusCode ?? 0)
    }
    return try JSONDecoder().decode(FeedbackSummary.self, from: data)
}
```

**Step 2: Commit**
```bash
git add beepbopboop/beepbopboop/Services/APIService.swift
git commit -m "feat: add submitFeedback + getFeedbackSummary to APIService"
```

---

### Task 14: Build verification — Go

**Step 1: Full Go build**
```bash
cd /Users/shanegleeson/Repos/beepbopboop/backend && go build ./...
```
Expected: no output (clean build)

---

### Task 15: Build verification — iOS

**Step 1: iOS Simulator build**
```bash
cd /Users/shanegleeson/Repos/beepbopboop && \
xcodebuild build \
  -scheme beepbopboop \
  -destination 'platform=iOS Simulator,name=iPhone 16 Pro' \
  CODE_SIGNING_ALLOWED=NO \
  -quiet \
  -project beepbopboop/beepbopboop.xcodeproj
```
Expected: `** BUILD SUCCEEDED **`

---

### Task 16: Create PR

**Step 1: Push branch**
```bash
cd /Users/shanegleeson/Repos/beepbopboop && git push -u origin feat/in-feed-feedback
```

**Step 2: Create PR**
```bash
gh pr create \
  --title "feat: in-feed feedback posts (polls, surveys, freeform, ratings)" \
  --body "$(cat <<'EOF'
## Summary
- Adds `feedback` display hint with four sub-types: poll, rating, freeform, survey
- New `user_feedback` table (JSONB responses, one per user per post, upsertable)
- `POST /posts/{id}/response` — Firebase-auth endpoint to submit a response
- `GET /posts/{id}/responses` — returns aggregated tally + avg rating + caller's own answer
- iOS: `FeedbackCard` router + `PollCardView`, `RatingCardView`, `FreeformCardView`, `SurveyCardView` with optimistic submission UI
- iOS: `FeedbackDetailView` for full-screen interaction
- `user_settings.preference_context JSONB` column for future agent prompt injection

## Test plan
- [ ] Create a poll post via agent API with `display_hint: "feedback"` and `feedback_type: "poll"`
- [ ] Verify it renders as PollCardView in the feed
- [ ] Tap an option and submit — confirm "Thanks for your input!" state and bar fills appear
- [ ] Call `GET /posts/{id}/responses` and confirm `tally` reflects the vote
- [ ] Repeat with freeform, rating, survey types
- [ ] Submit twice to the same post — confirm upsert (count stays at 1)
- [ ] Verify `go build ./...` passes
- [ ] Verify iOS simulator build passes

Closes #26

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
)"
```
