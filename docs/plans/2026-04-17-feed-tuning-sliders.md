# Feed Tuning Sliders Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add two feed-tuning sliders (geoBias, freshnessBias) to iOS Settings that read/write `GET/PUT /user/weights` via new Firebase-auth backend endpoints.

**Architecture:** Backend gets two new Firebase-auth handler methods mirroring the existing agent-auth handlers. iOS gets a `FeedWeights` model, two `APIService` methods, and a new "Tune your feed" section in SettingsView backed by SettingsViewModel additions. Sliders auto-save on release (debounced 500ms).

**Tech Stack:** Go (chi router, net/http, database/sql), Swift/SwiftUI (URLSession, async/await, @MainActor ObservableObject, Combine)

---

### Task 1: Backend — Firebase-auth weights handler methods

**Files:**
- Modify: `backend/internal/handler/weights.go`

**Step 1: Read current `weights.go`** (already done — see design doc)

**Step 2: Add `userRepo` to `WeightsHandler` and new Firebase methods**

Replace the struct and constructor, then add the two new methods at the bottom of the file. Full replacement:

```go
package handler

import (
	"encoding/json"
	"net/http"

	"github.com/shanegleeson/beepbopboop/backend/internal/middleware"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

type WeightsHandler struct {
	agentRepo   *repository.AgentRepo
	userRepo    *repository.UserRepo
	weightsRepo *repository.WeightsRepo
}

func NewWeightsHandler(agentRepo *repository.AgentRepo, userRepo *repository.UserRepo, weightsRepo *repository.WeightsRepo) *WeightsHandler {
	return &WeightsHandler{
		agentRepo:   agentRepo,
		userRepo:    userRepo,
		weightsRepo: weightsRepo,
	}
}

// GetWeights returns the current user weights (agent-auth).
func (h *WeightsHandler) GetWeights(w http.ResponseWriter, r *http.Request) {
	agentID := middleware.AgentIDFromContext(r.Context())
	agent, err := h.agentRepo.GetByID(agentID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to resolve agent"})
		return
	}

	weights, err := h.weightsRepo.Get(agent.UserID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load weights"})
		return
	}

	if weights == nil {
		writeJSON(w, http.StatusOK, map[string]any{"user_id": agent.UserID, "weights": nil})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(weights)
}

type updateWeightsRequest struct {
	Weights json.RawMessage `json:"weights"`
}

// UpdateWeights sets user preference weights (agent-auth, pushed by Lobs).
func (h *WeightsHandler) UpdateWeights(w http.ResponseWriter, r *http.Request) {
	agentID := middleware.AgentIDFromContext(r.Context())
	agent, err := h.agentRepo.GetByID(agentID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to resolve agent"})
		return
	}

	var req updateWeightsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if len(req.Weights) == 0 || !json.Valid(req.Weights) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "weights must be valid JSON"})
		return
	}

	weights, err := h.weightsRepo.Upsert(agent.UserID, req.Weights)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save weights"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(weights)
}

// GetWeightsFirebase returns the current user weights (Firebase-auth, mobile client).
func (h *WeightsHandler) GetWeightsFirebase(w http.ResponseWriter, r *http.Request) {
	uid := middleware.FirebaseUIDFromContext(r.Context())
	user, err := h.userRepo.FindOrCreateByFirebaseUID(uid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to resolve user"})
		return
	}

	weights, err := h.weightsRepo.Get(user.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load weights"})
		return
	}

	if weights == nil {
		writeJSON(w, http.StatusOK, map[string]any{"user_id": user.ID, "weights": nil})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(weights)
}

// UpdateWeightsFirebase sets user preference weights (Firebase-auth, mobile client).
// Accepts flat FeedWeights JSON: {"freshness_bias": 0.8, "geo_bias": 0.5, ...}
func (h *WeightsHandler) UpdateWeightsFirebase(w http.ResponseWriter, r *http.Request) {
	uid := middleware.FirebaseUIDFromContext(r.Context())
	user, err := h.userRepo.FindOrCreateByFirebaseUID(uid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to resolve user"})
		return
	}

	var raw json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if len(raw) == 0 || !json.Valid(raw) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "weights must be valid JSON"})
		return
	}

	weights, err := h.weightsRepo.Upsert(user.ID, raw)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save weights"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(weights)
}
```

**Step 3: Fix the constructor call in `main.go`**

In `backend/cmd/server/main.go`, find:
```go
weightsH := handler.NewWeightsHandler(agentRepo, weightsRepo)
```
Change to:
```go
weightsH := handler.NewWeightsHandler(agentRepo, userRepo, weightsRepo)
```

**Step 4: Register Firebase-auth routes in `main.go`**

In the Firebase-authenticated routes group (after `r.Put("/user/settings", settingsH.UpdateSettings)`), add:
```go
r.Get("/user/weights", weightsH.GetWeightsFirebase)
r.Put("/user/weights", weightsH.UpdateWeightsFirebase)
```

**Step 5: Verify backend compiles**

```bash
cd backend && go build ./...
```
Expected: no output (success).

**Step 6: Run existing backend tests**

```bash
cd backend && go test ./internal/handler/... -v -count=1
```
Expected: all pass (the new handler methods don't break existing tests).

**Step 7: Commit**

```bash
git add backend/internal/handler/weights.go backend/cmd/server/main.go
git commit -m "feat: add Firebase-auth GET/PUT /user/weights endpoints for iOS client"
```

---

### Task 2: Backend — Tests for Firebase-auth weights handlers

**Files:**
- Create: `backend/internal/handler/weights_test.go`

**Step 1: Write the test file**

```go
package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/shanegleeson/beepbopboop/backend/internal/database"
	"github.com/shanegleeson/beepbopboop/backend/internal/handler"
	"github.com/shanegleeson/beepbopboop/backend/internal/middleware"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

func TestGetWeightsFirebase_NoWeights(t *testing.T) {
	db := database.OpenTestDB(t)
	userRepo := repository.NewUserRepo(db)
	agentRepo := repository.NewAgentRepo(db)
	weightsRepo := repository.NewWeightsRepo(db)
	h := handler.NewWeightsHandler(agentRepo, userRepo, weightsRepo)

	req := httptest.NewRequest("GET", "/user/weights", nil)
	req = req.WithContext(middleware.WithFirebaseUID(req.Context(), "firebase-new-user"))
	rec := httptest.NewRecorder()

	h.GetWeightsFirebase(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["weights"] != nil {
		t.Errorf("expected nil weights for new user, got %v", resp["weights"])
	}
}

func TestUpdateWeightsFirebase_ThenGet(t *testing.T) {
	db := database.OpenTestDB(t)
	userRepo := repository.NewUserRepo(db)
	agentRepo := repository.NewAgentRepo(db)
	weightsRepo := repository.NewWeightsRepo(db)
	h := handler.NewWeightsHandler(agentRepo, userRepo, weightsRepo)

	body := `{"freshness_bias":0.65,"geo_bias":0.8}`

	putReq := httptest.NewRequest("PUT", "/user/weights", bytes.NewBufferString(body))
	putReq = putReq.WithContext(middleware.WithFirebaseUID(putReq.Context(), "firebase-tuner"))
	putRec := httptest.NewRecorder()

	h.UpdateWeightsFirebase(putRec, putReq)

	if putRec.Code != http.StatusOK {
		t.Fatalf("PUT expected 200, got %d: %s", putRec.Code, putRec.Body.String())
	}

	// Now GET and verify the weights persisted
	getReq := httptest.NewRequest("GET", "/user/weights", nil)
	getReq = getReq.WithContext(middleware.WithFirebaseUID(getReq.Context(), "firebase-tuner"))
	getRec := httptest.NewRecorder()

	h.GetWeightsFirebase(getRec, getReq)

	if getRec.Code != http.StatusOK {
		t.Fatalf("GET expected 200, got %d: %s", getRec.Code, getRec.Body.String())
	}

	var resp map[string]any
	json.NewDecoder(getRec.Body).Decode(&resp)
	if resp["weights"] == nil {
		t.Error("expected weights to be set after PUT")
	}

	weightsMap, ok := resp["weights"].(map[string]any)
	if !ok {
		t.Fatalf("expected weights to be an object, got %T", resp["weights"])
	}
	if weightsMap["geo_bias"] != 0.8 {
		t.Errorf("expected geo_bias 0.8, got %v", weightsMap["geo_bias"])
	}
	if weightsMap["freshness_bias"] != 0.65 {
		t.Errorf("expected freshness_bias 0.65, got %v", weightsMap["freshness_bias"])
	}
}

func TestUpdateWeightsFirebase_InvalidJSON(t *testing.T) {
	db := database.OpenTestDB(t)
	userRepo := repository.NewUserRepo(db)
	agentRepo := repository.NewAgentRepo(db)
	weightsRepo := repository.NewWeightsRepo(db)
	h := handler.NewWeightsHandler(agentRepo, userRepo, weightsRepo)

	putReq := httptest.NewRequest("PUT", "/user/weights", bytes.NewBufferString("not-json"))
	putReq = putReq.WithContext(middleware.WithFirebaseUID(putReq.Context(), "firebase-bad"))
	putRec := httptest.NewRecorder()

	h.UpdateWeightsFirebase(putRec, putReq)

	if putRec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", putRec.Code)
	}
}
```

**Step 2: Run the tests**

```bash
cd backend && go test ./internal/handler/... -v -run TestGetWeightsFirebase -run TestUpdateWeightsFirebase -count=1
```
Expected: all three tests PASS.

**Step 3: Run full test suite to catch regressions**

```bash
cd backend && go test ./... -count=1
```
Expected: all pass.

**Step 4: Commit**

```bash
git add backend/internal/handler/weights_test.go
git commit -m "test: Firebase-auth weights handler coverage"
```

---

### Task 3: iOS — FeedWeights model

**Files:**
- Create: `beepbopboop/beepbopboop/Models/FeedWeights.swift`

**Step 1: Create the model file**

```swift
struct FeedWeights: Codable {
    var labelWeights: [String: Double]?
    var typeWeights: [String: Double]?
    var freshnessBias: Double
    var geoBias: Double

    static let defaults = FeedWeights(freshnessBias: 0.8, geoBias: 0.5)

    enum CodingKeys: String, CodingKey {
        case labelWeights = "label_weights"
        case typeWeights = "type_weights"
        case freshnessBias = "freshness_bias"
        case geoBias = "geo_bias"
    }
}

// Wrapper matching GET /user/weights response envelope
struct UserWeightsResponse: Codable {
    let userId: String?
    let weights: FeedWeights?

    enum CodingKeys: String, CodingKey {
        case userId = "user_id"
        case weights
    }
}
```

**Step 2: Verify it compiles**

```bash
cd beepbopboop && xcodebuild -scheme beepbopboop -destination 'generic/platform=iOS Simulator' build 2>&1 | tail -5
```
Expected: `** BUILD SUCCEEDED **`

**Step 3: Commit**

```bash
git add beepbopboop/beepbopboop/Models/FeedWeights.swift
git commit -m "feat: add FeedWeights model for iOS weights API"
```

---

### Task 4: iOS — APIService weights methods

**Files:**
- Modify: `beepbopboop/beepbopboop/Services/APIService.swift`

**Step 1: Add `getWeights` and `updateWeights` after `updateSettings`**

Insert after the closing brace of `updateSettings` (line ~125), before `// MARK: - Reactions`:

```swift
    // MARK: - Feed weights

    @MainActor
    func getWeights() async throws -> FeedWeights? {
        let token = authService.getToken()
        guard let url = URL(string: "\(baseURL)/user/weights") else {
            throw APIError.invalidURL
        }
        var request = URLRequest(url: url)
        request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")

        let (data, response) = try await URLSession.shared.data(for: request)
        guard let httpResponse = response as? HTTPURLResponse else {
            throw APIError.invalidResponse
        }
        guard httpResponse.statusCode == 200 else {
            throw APIError.httpError(httpResponse.statusCode)
        }
        let envelope = try JSONDecoder().decode(UserWeightsResponse.self, from: data)
        return envelope.weights
    }

    @MainActor
    func updateWeights(_ weights: FeedWeights) async throws {
        let token = authService.getToken()
        guard let url = URL(string: "\(baseURL)/user/weights") else {
            throw APIError.invalidURL
        }
        var request = URLRequest(url: url)
        request.httpMethod = "PUT"
        request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        request.httpBody = try JSONEncoder().encode(weights)

        let (_, response) = try await URLSession.shared.data(for: request)
        guard let httpResponse = response as? HTTPURLResponse,
              (200...299).contains(httpResponse.statusCode) else {
            throw APIError.httpError((response as? HTTPURLResponse)?.statusCode ?? 0)
        }
    }
```

**Step 2: Verify it compiles**

```bash
cd beepbopboop && xcodebuild -scheme beepbopboop -destination 'generic/platform=iOS Simulator' build 2>&1 | tail -5
```
Expected: `** BUILD SUCCEEDED **`

**Step 3: Commit**

```bash
git add beepbopboop/beepbopboop/Services/APIService.swift
git commit -m "feat: add getWeights/updateWeights to APIService"
```

---

### Task 5: iOS — SettingsViewModel and SettingsView feed tuning sliders

**Files:**
- Modify: `beepbopboop/beepbopboop/Views/SettingsView.swift`

**Step 1: Add weight properties and save logic to `SettingsViewModel`**

In `SettingsViewModel`, add after `@Published var didSave = false`:

```swift
    @Published var geoBias: Double = 0.5
    @Published var freshnessBias: Double = 0.8
    @Published var feedUpdated = false

    private var weightsSaveTask: Task<Void, Never>?
```

Add a `loadWeights()` helper and `scheduleWeightsSave()` method after the existing `save()` method:

```swift
    func loadWeights() async {
        do {
            if let weights = try await apiService.getWeights() {
                geoBias = weights.geoBias
                freshnessBias = weights.freshnessBias
            }
        } catch {
            // Use defaults on failure
        }
    }

    func scheduleWeightsSave() {
        weightsSaveTask?.cancel()
        weightsSaveTask = Task {
            try? await Task.sleep(nanoseconds: 500_000_000)
            guard !Task.isCancelled else { return }
            await saveWeights()
        }
    }

    private func saveWeights() async {
        let weights = FeedWeights(freshnessBias: freshnessBias, geoBias: geoBias)
        do {
            try await apiService.updateWeights(weights)
            feedUpdated = true
            try? await Task.sleep(nanoseconds: 2_000_000_000)
            feedUpdated = false
        } catch {
            // Silent — feed simply won't update until next manual adjustment
        }
    }

    func resetWeightsToDefaults() {
        geoBias = FeedWeights.defaults.geoBias
        freshnessBias = FeedWeights.defaults.freshnessBias
        scheduleWeightsSave()
    }
```

**Step 2: Update `loadSettings()` to also load weights**

In `SettingsViewModel.loadSettings()`, after setting `isLoading = false` at the end (or concurrently), add:

```swift
    func loadSettings() async {
        isLoading = true
        async let settingsTask: () = {
            do {
                let settings = try await apiService.getSettings()
                selectedLocationName = settings.locationName
                selectedLatitude = settings.latitude
                selectedLongitude = settings.longitude
                selectedRadius = settings.radiusKm
                if selectedRadius <= 0 { selectedRadius = 25.0 }
            } catch {}
        }()
        async let weightsTask: () = loadWeights()
        _ = await (settingsTask, weightsTask)
        isLoading = false
    }
```

**Step 3: Add "Tune your feed" section to `SettingsView`**

In the `Form` body of `SettingsView`, add this section after the Radius section (before the Save button section):

```swift
                Section("Tune your feed") {
                    VStack(alignment: .leading, spacing: 12) {
                        HStack {
                            Text("📍")
                            Text("More local")
                                .font(.caption)
                                .foregroundColor(.secondary)
                            Spacer()
                            Text("More global")
                                .font(.caption)
                                .foregroundColor(.secondary)
                            Text("🌍")
                        }
                        Slider(value: $viewModel.geoBias, in: 0...1) { editing in
                            if !editing { viewModel.scheduleWeightsSave() }
                        }
                    }

                    VStack(alignment: .leading, spacing: 12) {
                        HStack {
                            Text("⚡")
                            Text("Live & timely")
                                .font(.caption)
                                .foregroundColor(.secondary)
                            Spacer()
                            Text("Evergreen")
                                .font(.caption)
                                .foregroundColor(.secondary)
                            Text("📚")
                        }
                        Slider(value: $viewModel.freshnessBias, in: 0...1) { editing in
                            if !editing { viewModel.scheduleWeightsSave() }
                        }
                    }

                    HStack {
                        Button("Reset to defaults") {
                            viewModel.resetWeightsToDefaults()
                        }
                        .font(.caption)
                        .foregroundColor(.secondary)

                        Spacer()

                        if viewModel.feedUpdated {
                            HStack(spacing: 4) {
                                Image(systemName: "checkmark.circle.fill")
                                    .foregroundColor(.green)
                                    .font(.caption)
                                Text("Feed updated")
                                    .font(.caption)
                                    .foregroundColor(.green)
                            }
                            .transition(.opacity)
                        }
                    }
                    .animation(.easeInOut(duration: 0.3), value: viewModel.feedUpdated)
                }
```

**Step 4: Verify it compiles**

```bash
cd beepbopboop && xcodebuild -scheme beepbopboop -destination 'generic/platform=iOS Simulator' build 2>&1 | tail -5
```
Expected: `** BUILD SUCCEEDED **`

**Step 5: Commit**

```bash
git add beepbopboop/beepbopboop/Views/SettingsView.swift
git commit -m "feat: add feed tuning sliders to Settings (geoBias + freshnessBias)"
```

---

### Task 6: Full build verification

**Step 1: Run backend tests**

```bash
cd backend && go test ./... -count=1
```
Expected: all pass.

**Step 2: Build iOS app**

```bash
cd beepbopboop && xcodebuild -scheme beepbopboop -destination 'generic/platform=iOS Simulator' build 2>&1 | grep -E "error:|warning:|SUCCEEDED|FAILED"
```
Expected: `** BUILD SUCCEEDED **`, no errors.

---

### Task 7: Create PR and update issue

**Step 1: Push branch**

```bash
git push -u origin claude/affectionate-proskuriakova-a6947e
```

**Step 2: Create PR**

```bash
gh pr create \
  --title "feat: manual feed tuning sliders in Settings (#33)" \
  --body "$(cat <<'EOF'
## Summary

- Adds Firebase-auth `GET/PUT /user/weights` endpoints so iOS can read/write feed weights directly
- New `FeedWeights` model and two `APIService` methods on iOS
- "Tune your feed" section in Settings with two sliders:
  - 📍 More local ↔ 🌍 More global (geoBias, 0–1)
  - ⚡ Live & timely ↔ 📚 Evergreen (freshnessBias, 0–1)
- Sliders auto-save on release (debounced 500ms), show "Feed updated ✓" flash
- Reset to defaults button
- Weights loaded concurrently with location settings on Settings open

Closes #33

## Test plan

- [ ] Backend tests pass: `cd backend && go test ./...`
- [ ] iOS builds: `xcodebuild -scheme beepbopboop -destination 'generic/platform=iOS Simulator' build`
- [ ] Open Settings → "Tune your feed" section appears with two sliders
- [ ] Move a slider → "Feed updated ✓" appears after ~500ms
- [ ] Close and reopen Settings → slider positions are restored from server
- [ ] Reset to defaults → sliders snap to 0.5 / 0.8

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
)"
```

**Step 3: Update issue with progress comment**

```bash
gh issue comment 33 --body "$(cat <<'EOF'
## Implementation complete — PR ready for review

**PR:** [link from step above]

### What was built

**Backend:**
- Added `GetWeightsFirebase` and `UpdateWeightsFirebase` handler methods to `WeightsHandler` — same logic as the existing agent-auth variants but using Firebase UID context
- Registered `GET /user/weights` and `PUT /user/weights` in the Firebase-auth route group so iOS can call them with the existing Bearer token
- Added handler tests covering: no-weights new user, round-trip PUT→GET, invalid JSON rejection

**iOS:**
- `FeedWeights` model + `UserWeightsResponse` envelope (Codable, snake_case keys)
- `getWeights()` and `updateWeights(_:)` in `APIService`
- "Tune your feed" section in Settings with two sliders (geoBias + freshnessBias)
- Auto-save on slider release (debounced 500ms), "Feed updated ✓" flash on success
- Weights load concurrently with location settings; defaults (geo: 0.5, freshness: 0.8) used when no weights set
- Reset to defaults button

### What was deferred
- Per-label interest sliders (optional "Customise further" section from issue) — keeping it to 2 sliders per "don't overwhelm" goal
EOF
)"
```
