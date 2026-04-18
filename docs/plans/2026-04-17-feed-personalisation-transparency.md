# Feed Personalisation Transparency Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Surface the user's computed feed weights in Settings as a human-readable summary ("You engage most with: Sports, Local Events, Weather") with a brief explainer encouraging reactions.

**Architecture:** New Firebase-authenticated `GET /user/weights/summary` endpoint in the backend returns `top_labels`, `data_points`, and `last_updated`. The iOS Settings view fetches this and renders a read-only "Your Feed" section above the existing location section.

**Tech Stack:** Go (chi router, Firebase auth middleware), Swift/SwiftUI (Form + Section pattern), PostgreSQL (post_events, user_weights tables)

---

### Task 1: Backend — WeightsSummaryHandler

**Files:**
- Create: `backend/internal/handler/weights_summary.go`

**Step 1: Create the handler file**

```go
package handler

import (
	"net/http"
	"sort"
	"strings"

	"github.com/shanegleeson/beepbopboop/backend/internal/middleware"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

// labelDisplayNames maps raw label keys to user-friendly display names.
var labelDisplayNames = map[string]string{
	"sports":        "Sports",
	"local_events":  "Local Events",
	"local-events":  "Local Events",
	"weather":       "Weather",
	"fashion":       "Fashion",
	"trending":      "Trending",
	"hacker-news":   "Tech News",
	"hacker_news":   "Tech News",
	"nhl":           "NHL Hockey",
	"nba":           "NBA Basketball",
	"nfl":           "NFL Football",
	"music":         "Music",
	"food":          "Food & Drink",
	"travel":        "Travel",
	"technology":    "Technology",
	"arts":          "Arts & Culture",
	"community":     "Community",
	"health":        "Health",
	"business":      "Business",
}

var summaryDefaultWeights = &repository.FeedWeights{
	FreshnessBias: 0.8,
	GeoBias:       0.3,
	LabelWeights:  map[string]float64{"fashion": 0.4, "sports": 0.4, "trending": 0.3},
	TypeWeights:   map[string]float64{"event": 0.3, "discovery": 0.2, "article": 0.1, "video": 0.2},
}

type weightsSummaryResponse struct {
	TopLabels  []string `json:"top_labels"`
	DataPoints int      `json:"data_points"`
}

type WeightsSummaryHandler struct {
	userRepo    *repository.UserRepo
	weightsRepo *repository.WeightsRepo
	eventRepo   *repository.EventRepo
}

func NewWeightsSummaryHandler(
	userRepo *repository.UserRepo,
	weightsRepo *repository.WeightsRepo,
	eventRepo *repository.EventRepo,
) *WeightsSummaryHandler {
	return &WeightsSummaryHandler{
		userRepo:    userRepo,
		weightsRepo: weightsRepo,
		eventRepo:   eventRepo,
	}
}

// GetSummary returns a human-readable summary of the user's feed personalisation (Firebase auth).
func (h *WeightsSummaryHandler) GetSummary(w http.ResponseWriter, r *http.Request) {
	uid := middleware.FirebaseUIDFromContext(r.Context())
	user, err := h.userRepo.FindOrCreateByFirebaseUID(uid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to resolve user"})
		return
	}

	summary, err := h.eventRepo.Summary(user.ID, 14)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load engagement"})
		return
	}

	weights, err := h.weightsRepo.GetOrCompute(user.ID, h.eventRepo, summaryDefaultWeights)
	if err != nil || weights == nil {
		weights = summaryDefaultWeights
	}

	type kv struct {
		key string
		val float64
	}
	sorted := make([]kv, 0, len(weights.LabelWeights))
	for k, v := range weights.LabelWeights {
		sorted = append(sorted, kv{k, v})
	}
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].val > sorted[j].val })

	topLabels := make([]string, 0, 5)
	for _, item := range sorted {
		if len(topLabels) >= 5 {
			break
		}
		if name, ok := labelDisplayNames[item.key]; ok {
			topLabels = append(topLabels, name)
		} else {
			topLabels = append(topLabels, toDisplayName(item.key))
		}
	}

	writeJSON(w, http.StatusOK, weightsSummaryResponse{
		TopLabels:  topLabels,
		DataPoints: summary.TotalEvents,
	})
}

// toDisplayName converts a raw label key like "local-events" to "Local Events".
func toDisplayName(key string) string {
	s := strings.ReplaceAll(key, "-", " ")
	s = strings.ReplaceAll(s, "_", " ")
	words := strings.Fields(s)
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}
```

**Step 2: Verify it compiles**

```bash
cd backend && go build ./...
```

Expected: no output (success)

---

### Task 2: Backend — Register route in main.go

**Files:**
- Modify: `backend/cmd/server/main.go`

**Step 1: Instantiate the handler** (after the existing handler instantiations, before route registration)

Add after `weightsH := handler.NewWeightsHandler(agentRepo, weightsRepo)`:
```go
weightsSummaryH := handler.NewWeightsSummaryHandler(userRepo, weightsRepo, eventRepo)
```

**Step 2: Register route** inside the Firebase auth group (after `/user/templates` line):
```go
r.Get("/user/weights/summary", weightsSummaryH.GetSummary)
```

**Step 3: Build check**

```bash
cd backend && go build ./...
```

Expected: no output (success)

---

### Task 3: iOS — WeightsSummary model

**Files:**
- Create: `beepbopboop/beepbopboop/Models/WeightsSummary.swift`

```swift
struct WeightsSummary: Codable {
    let topLabels: [String]
    let dataPoints: Int

    enum CodingKeys: String, CodingKey {
        case topLabels = "top_labels"
        case dataPoints = "data_points"
    }
}
```

---

### Task 4: iOS — APIService.getWeightsSummary()

**Files:**
- Modify: `beepbopboop/beepbopboop/Services/APIService.swift`

Add after `getSettings()`:
```swift
@MainActor
func getWeightsSummary() async throws -> WeightsSummary {
    let token = authService.getToken()
    guard let url = URL(string: "\(baseURL)/user/weights/summary") else {
        throw APIError.invalidURL
    }
    var request = URLRequest(url: url)
    request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
    let (data, response) = try await URLSession.shared.data(for: request)
    guard let httpResponse = response as? HTTPURLResponse,
          httpResponse.statusCode == 200 else {
        throw APIError.httpError((response as? HTTPURLResponse)?.statusCode ?? 0)
    }
    return try JSONDecoder().decode(WeightsSummary.self, from: data)
}
```

---

### Task 5: iOS — SettingsViewModel update

**Files:**
- Modify: `beepbopboop/beepbopboop/Views/SettingsView.swift`

Add `@Published var weightsSummary: WeightsSummary?` with the other `@Published` properties.

In `loadSettings()`, after the settings fetch do-catch block, add:
```swift
weightsSummary = try? await apiService.getWeightsSummary()
```

---

### Task 6: iOS — SettingsView "Your Feed" section

**Files:**
- Modify: `beepbopboop/beepbopboop/Views/SettingsView.swift`

Add a new section at the top of the Form (before `Section("Location")`):

```swift
Section("Your Feed") {
    if let summary = viewModel.weightsSummary {
        if summary.dataPoints < 10 {
            Label("Still learning — keep scrolling and react to posts", systemImage: "brain")
                .foregroundStyle(.secondary)
                .font(.subheadline)
        } else {
            VStack(alignment: .leading, spacing: 6) {
                Text("You engage most with:")
                    .font(.subheadline)
                    .foregroundStyle(.secondary)
                Text(summary.topLabels.prefix(3).joined(separator: " · "))
                    .font(.subheadline)
                    .fontWeight(.medium)
            }
            .padding(.vertical, 2)

            Text("Based on \(summary.dataPoints) interactions")
                .font(.caption)
                .foregroundStyle(.secondary)
        }
    } else {
        Text("Reacting to posts helps your feed improve")
            .font(.subheadline)
            .foregroundStyle(.secondary)
    }
}
```

---

### Task 7: Build

```bash
cd beepbopboop && xcodebuild -scheme beepbopboop -destination 'platform=iOS Simulator,name=iPhone 16' build 2>&1 | tail -5
```

Expected: `** BUILD SUCCEEDED **`

---

### Task 8: Commit and PR

```bash
git add backend/internal/handler/weights_summary.go \
        backend/cmd/server/main.go \
        beepbopboop/beepbopboop/Models/WeightsSummary.swift \
        beepbopboop/beepbopboop/Services/APIService.swift \
        beepbopboop/beepbopboop/Views/SettingsView.swift
git commit -m "feat: personalisation transparency — feed summary in Settings (#32)"
```

Create PR:
```bash
gh pr create --title "feat: \"Your feed is learning\" — personalisation transparency (#32)" \
  --body "..."
```

### Task 9: Update issue

```bash
gh issue comment 32 --body "..."
```
