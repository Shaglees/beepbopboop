# Wave 3: Feed Architecture & Skill Cleanup Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add user-configurable content-mix spread settings with a full dashboard (backend + iOS), and complete skill decomposition (#180 Phase 3).

**Architecture:** New `spread_targets` JSONB column on `user_settings` + `spread_history` table. Three new endpoints (`GET/PUT /settings/spread`, `GET /settings/spread/history`). iOS slider-based Content Mix section in Settings. Parallel track: `_shared/SPORTS_COMMON.md`, INIT_WIZARD rename, hints cache.

**Tech Stack:** Go 1.22, PostgreSQL (JSONB), Swift/SwiftUI, chi router, testcontainers-go

---

## File Structure

### Backend (Go) — New Files
- `backend/internal/handler/spread.go` — SpreadHandler with GET/PUT spread + GET history
- `backend/internal/handler/spread_test.go` — Tests for all spread endpoints
- `backend/internal/repository/spread_repo.go` — SpreadRepo for spread_targets + spread_history queries

### Backend (Go) — Modified Files
- `backend/internal/database/database.go` — Add `spread_targets` column + `spread_history` table
- `backend/internal/model/model.go` — Add SpreadTargets and SpreadHistory models
- `backend/cmd/server/main.go` — Wire SpreadHandler + routes

### iOS — New Files
- `beepbopboop/beepbopboop/Views/ContentMixView.swift` — Content Mix settings section
- `beepbopboop/beepbopboop/ViewModels/ContentMixViewModel.swift` — ViewModel for spread settings

### iOS — Modified Files
- `beepbopboop/beepbopboop/Services/APIService.swift` — Add spread API methods
- `beepbopboop/beepbopboop/Views/SettingsView.swift` — Add Content Mix section

### Skills — New Files
- `.claude/skills/_shared/SPORTS_COMMON.md` — Shared sport-skill patterns

### Skills — Modified Files
- `.claude/skills/_shared/CONTEXT_BOOTSTRAP.md` — Add hints cache pattern
- `.claude/skills/beepbopboop-post/SKILL.md` — Update INIT_WIZARD → MODE_INIT references
- `.claude/skills/beepbopboop-post/MODE_BATCH.md` — Reference spread API in BT2
- `.claude/skills/beepbopboop-post/WEIGHT_COMPUTATION.md` — Document auto-adjust
- `.claude/skills/beepbopboop-post/INIT_WIZARD.md` → renamed to `MODE_INIT.md`
- `.claude/skills/beepbopboop-soccer/SKILL.md` — Reference SPORTS_COMMON.md
- `.claude/skills/beepbopboop-basketball/SKILL.md` — Reference SPORTS_COMMON.md
- `.claude/skills/beepbopboop-football/SKILL.md` — Reference SPORTS_COMMON.md
- `.claude/skills/beepbopboop-baseball/SKILL.md` — Reference SPORTS_COMMON.md

---

### Task 1: Database Schema — Add spread_targets Column & spread_history Table

**Files:**
- Modify: `backend/internal/database/database.go`
- Modify: `backend/internal/model/model.go`

- [ ] **Step 1: Add migrations to database.go**

Open `backend/internal/database/database.go`. After the existing `ALTER TABLE user_settings ADD COLUMN IF NOT EXISTS` lines (around line 46), add:

```go
db.Exec("ALTER TABLE user_settings ADD COLUMN IF NOT EXISTS spread_targets JSONB")

db.Exec(`CREATE TABLE IF NOT EXISTS spread_history (
	user_id    TEXT NOT NULL REFERENCES users(id),
	date       DATE NOT NULL,
	targets    JSONB NOT NULL,
	actuals    JSONB NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
	PRIMARY KEY (user_id, date)
)`)
```

- [ ] **Step 2: Add SpreadTargets model to model.go**

Open `backend/internal/model/model.go`. Add after the `UserSettings` struct:

```go
// SpreadVertical holds a single vertical's weight and pin state.
type SpreadVertical struct {
	Weight float64 `json:"weight"`
	Pinned bool    `json:"pinned"`
}

// SpreadTargets is the JSONB stored in user_settings.spread_targets.
type SpreadTargets struct {
	Verticals  map[string]SpreadVertical `json:"verticals"`
	Omega      string                    `json:"omega"`
	AutoAdjust bool                      `json:"auto_adjust"`
	UpdatedAt  time.Time                 `json:"updated_at"`
}

// SpreadResponse is what GET /settings/spread returns.
type SpreadResponse struct {
	Targets    map[string]float64 `json:"targets"`
	Omega      string             `json:"omega"`
	Pinned     []string           `json:"pinned"`
	AutoAdjust bool               `json:"auto_adjust"`
	Actual30d  map[string]float64 `json:"actual_30d"`
	Status     map[string]string  `json:"status"`
}

// SpreadHistoryDay is one day's snapshot.
type SpreadHistoryDay struct {
	Date   string             `json:"date"`
	Target map[string]float64 `json:"target"`
	Actual map[string]float64 `json:"actual"`
}
```

- [ ] **Step 3: Verify the app compiles**

Run: `cd /Users/shanegleeson/Repos/beepbopboop/backend && go build ./...`
Expected: Compiles with no errors.

- [ ] **Step 4: Commit**

```bash
git add backend/internal/database/database.go backend/internal/model/model.go
git commit -m "feat(db): add spread_targets column and spread_history table (#185)"
```

---

### Task 2: SpreadRepo — Repository for Spread Data

**Files:**
- Create: `backend/internal/repository/spread_repo.go`

- [ ] **Step 1: Create SpreadRepo**

Create `backend/internal/repository/spread_repo.go`:

```go
package repository

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/shanegleeson/beepbopboop/backend/internal/model"
)

type SpreadRepo struct {
	db *sql.DB
}

func NewSpreadRepo(db *sql.DB) *SpreadRepo {
	return &SpreadRepo{db: db}
}

// DefaultTargets returns an even distribution across all known verticals.
func DefaultTargets() *model.SpreadTargets {
	verticals := []string{"sports", "food", "music", "travel", "science", "gaming", "creators", "fashion", "movies", "pets", "news"}
	weight := 1.0 / float64(len(verticals))
	m := make(map[string]model.SpreadVertical, len(verticals))
	for _, v := range verticals {
		m[v] = model.SpreadVertical{Weight: weight, Pinned: false}
	}
	return &model.SpreadTargets{
		Verticals:  m,
		Omega:      "sports",
		AutoAdjust: true,
		UpdatedAt:  time.Now(),
	}
}

// GetTargets returns the user's spread targets, or nil when none are stored.
func (r *SpreadRepo) GetTargets(userID string) (*model.SpreadTargets, error) {
	var raw sql.NullString
	err := r.db.QueryRow(
		`SELECT spread_targets FROM user_settings WHERE user_id = $1`, userID,
	).Scan(&raw)
	if err == sql.ErrNoRows || !raw.Valid {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query spread_targets: %w", err)
	}
	var st model.SpreadTargets
	if err := json.Unmarshal([]byte(raw.String), &st); err != nil {
		return nil, fmt.Errorf("unmarshal spread_targets: %w", err)
	}
	return &st, nil
}

// UpsertTargets stores the user's spread targets.
func (r *SpreadRepo) UpsertTargets(userID string, st *model.SpreadTargets) error {
	st.UpdatedAt = time.Now()
	b, err := json.Marshal(st)
	if err != nil {
		return fmt.Errorf("marshal spread_targets: %w", err)
	}
	_, err = r.db.Exec(`
		INSERT INTO user_settings (user_id, spread_targets, updated_at)
		VALUES ($1, $2, CURRENT_TIMESTAMP)
		ON CONFLICT(user_id) DO UPDATE SET
			spread_targets = excluded.spread_targets,
			updated_at     = CURRENT_TIMESTAMP`,
		userID, string(b))
	if err != nil {
		return fmt.Errorf("upsert spread_targets: %w", err)
	}
	return nil
}

// Actual30d computes the actual allocation over the last 30 days from posts.
// It groups posts by their first label and returns the fraction for each label.
func (r *SpreadRepo) Actual30d(userID string) (map[string]float64, error) {
	rows, err := r.db.Query(`
		SELECT labels FROM posts
		WHERE agent_id IN (SELECT id FROM agents WHERE user_id = $1)
		  AND created_at > NOW() - INTERVAL '30 days'
		  AND visibility = 'public'`,
		userID)
	if err != nil {
		return nil, fmt.Errorf("query posts for actual_30d: %w", err)
	}
	defer rows.Close()

	counts := make(map[string]int)
	total := 0
	for rows.Next() {
		var labelsRaw sql.NullString
		if err := rows.Scan(&labelsRaw); err != nil {
			return nil, fmt.Errorf("scan labels: %w", err)
		}
		if !labelsRaw.Valid || labelsRaw.String == "" {
			continue
		}
		// Labels stored as comma-separated string; first label is the vertical.
		primary := firstLabel(labelsRaw.String)
		if primary != "" {
			counts[primary]++
			total++
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	result := make(map[string]float64, len(counts))
	if total == 0 {
		return result, nil
	}
	for k, v := range counts {
		result[k] = float64(v) / float64(total)
	}
	return result, nil
}

// firstLabel extracts the first label from a comma-separated labels string.
func firstLabel(labels string) string {
	for i, c := range labels {
		if c == ',' {
			return labels[:i]
		}
	}
	return labels
}

// InsertHistory writes a daily snapshot row.
func (r *SpreadRepo) InsertHistory(userID string, date string, targets, actuals map[string]float64) error {
	targetsJSON, _ := json.Marshal(targets)
	actualsJSON, _ := json.Marshal(actuals)
	_, err := r.db.Exec(`
		INSERT INTO spread_history (user_id, date, targets, actuals)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id, date) DO UPDATE SET
			targets = excluded.targets,
			actuals = excluded.actuals`,
		userID, date, string(targetsJSON), string(actualsJSON))
	if err != nil {
		return fmt.Errorf("insert spread_history: %w", err)
	}
	return nil
}

// GetHistory returns the last N days of spread history.
func (r *SpreadRepo) GetHistory(userID string, days int) ([]model.SpreadHistoryDay, error) {
	rows, err := r.db.Query(`
		SELECT date, targets, actuals FROM spread_history
		WHERE user_id = $1 AND date > CURRENT_DATE - $2::INT
		ORDER BY date ASC`, userID, days)
	if err != nil {
		return nil, fmt.Errorf("query spread_history: %w", err)
	}
	defer rows.Close()

	var result []model.SpreadHistoryDay
	for rows.Next() {
		var d model.SpreadHistoryDay
		var targetsRaw, actualsRaw string
		if err := rows.Scan(&d.Date, &targetsRaw, &actualsRaw); err != nil {
			return nil, fmt.Errorf("scan spread_history: %w", err)
		}
		json.Unmarshal([]byte(targetsRaw), &d.Target)
		json.Unmarshal([]byte(actualsRaw), &d.Actual)
		result = append(result, d)
	}
	return result, rows.Err()
}
```

- [ ] **Step 2: Verify it compiles**

Run: `cd /Users/shanegleeson/Repos/beepbopboop/backend && go build ./...`
Expected: Compiles with no errors.

- [ ] **Step 3: Commit**

```bash
git add backend/internal/repository/spread_repo.go
git commit -m "feat(repo): add SpreadRepo for spread targets and history (#185)"
```

---

### Task 3: SpreadHandler — GET/PUT Endpoints + Tests

**Files:**
- Create: `backend/internal/handler/spread.go`
- Create: `backend/internal/handler/spread_test.go`

- [ ] **Step 1: Write the failing tests**

Create `backend/internal/handler/spread_test.go`:

```go
package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/shanegleeson/beepbopboop/backend/internal/model"
)

func TestSpreadHandler_GetDefault(t *testing.T) {
	db := setupTestDB(t)
	userID := createTestUser(t, db)
	h := newTestSpreadHandler(t, db)

	req := httptest.NewRequest("GET", "/settings/spread", nil)
	req = withFirebaseContext(req, userID)
	w := httptest.NewRecorder()

	h.GetSpread(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d: %s", w.Code, w.Body.String())
	}

	var resp model.SpreadResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if len(resp.Targets) == 0 {
		t.Fatal("expected default targets, got empty")
	}
	if resp.Omega == "" {
		t.Fatal("expected omega to be set")
	}
	if resp.AutoAdjust != true {
		t.Fatal("expected auto_adjust to default to true")
	}
}

func TestSpreadHandler_PutAndGet(t *testing.T) {
	db := setupTestDB(t)
	userID := createTestUser(t, db)
	h := newTestSpreadHandler(t, db)

	body := `{
		"targets": {"sports": 0.4, "food": 0.3, "music": 0.3},
		"omega": "sports",
		"pinned": ["sports"],
		"auto_adjust": true
	}`
	req := httptest.NewRequest("PUT", "/settings/spread", bytes.NewBufferString(body))
	req = withFirebaseContext(req, userID)
	w := httptest.NewRecorder()

	h.PutSpread(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("PUT expected 200 got %d: %s", w.Code, w.Body.String())
	}

	// GET it back
	req2 := httptest.NewRequest("GET", "/settings/spread", nil)
	req2 = withFirebaseContext(req2, userID)
	w2 := httptest.NewRecorder()
	h.GetSpread(w2, req2)

	var resp model.SpreadResponse
	json.NewDecoder(w2.Body).Decode(&resp)

	if resp.Targets["sports"] != 0.4 {
		t.Fatalf("expected sports=0.4 got %f", resp.Targets["sports"])
	}
	if resp.Omega != "sports" {
		t.Fatalf("expected omega=sports got %s", resp.Omega)
	}
}

func TestSpreadHandler_PutValidation_BadSum(t *testing.T) {
	db := setupTestDB(t)
	userID := createTestUser(t, db)
	h := newTestSpreadHandler(t, db)

	body := `{
		"targets": {"sports": 0.5, "food": 0.6},
		"omega": "sports",
		"pinned": [],
		"auto_adjust": true
	}`
	req := httptest.NewRequest("PUT", "/settings/spread", bytes.NewBufferString(body))
	req = withFirebaseContext(req, userID)
	w := httptest.NewRecorder()

	h.PutSpread(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d: %s", w.Code, w.Body.String())
	}
}

func TestSpreadHandler_PutValidation_MissingOmega(t *testing.T) {
	db := setupTestDB(t)
	userID := createTestUser(t, db)
	h := newTestSpreadHandler(t, db)

	body := `{
		"targets": {"sports": 0.5, "food": 0.5},
		"omega": "gaming",
		"pinned": [],
		"auto_adjust": true
	}`
	req := httptest.NewRequest("PUT", "/settings/spread", bytes.NewBufferString(body))
	req = withFirebaseContext(req, userID)
	w := httptest.NewRecorder()

	h.PutSpread(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d: %s", w.Code, w.Body.String())
	}
}

func TestSpreadHandler_PutValidation_AllPinnedWithAutoAdjust(t *testing.T) {
	db := setupTestDB(t)
	userID := createTestUser(t, db)
	h := newTestSpreadHandler(t, db)

	body := `{
		"targets": {"sports": 0.5, "food": 0.5},
		"omega": "sports",
		"pinned": ["sports", "food"],
		"auto_adjust": true
	}`
	req := httptest.NewRequest("PUT", "/settings/spread", bytes.NewBufferString(body))
	req = withFirebaseContext(req, userID)
	w := httptest.NewRecorder()

	h.PutSpread(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d: %s", w.Code, w.Body.String())
	}
}

func TestSpreadHandler_History(t *testing.T) {
	db := setupTestDB(t)
	userID := createTestUser(t, db)
	h := newTestSpreadHandler(t, db)

	// Insert a history row directly
	spreadRepo := h.spreadRepo
	spreadRepo.InsertHistory(userID, "2026-04-23",
		map[string]float64{"sports": 0.5, "food": 0.5},
		map[string]float64{"sports": 0.45, "food": 0.55},
	)

	req := httptest.NewRequest("GET", "/settings/spread/history", nil)
	req = withFirebaseContext(req, userID)
	w := httptest.NewRecorder()
	h.GetHistory(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Days []model.SpreadHistoryDay `json:"days"`
	}
	json.NewDecoder(w.Body).Decode(&resp)

	if len(resp.Days) == 0 {
		t.Fatal("expected at least 1 history day")
	}
}
```

- [ ] **Step 2: Write test helpers**

Add at the bottom of `spread_test.go`:

```go
func newTestSpreadHandler(t *testing.T, db *sql.DB) *SpreadHandler {
	t.Helper()
	userRepo := repository.NewUserRepo(db)
	spreadRepo := repository.NewSpreadRepo(db)
	return NewSpreadHandler(userRepo, spreadRepo)
}
```

Add these imports to the import block: `"database/sql"`, `"github.com/shanegleeson/beepbopboop/backend/internal/repository"`.

- [ ] **Step 3: Run tests to verify they fail**

Run: `cd /Users/shanegleeson/Repos/beepbopboop/backend && go test ./internal/handler/ -run "TestSpread" -v -count=1`
Expected: FAIL (SpreadHandler does not exist yet)

- [ ] **Step 4: Implement SpreadHandler**

Create `backend/internal/handler/spread.go`:

```go
package handler

import (
	"encoding/json"
	"math"
	"net/http"

	"github.com/shanegleeson/beepbopboop/backend/internal/middleware"
	"github.com/shanegleeson/beepbopboop/backend/internal/model"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

type SpreadHandler struct {
	userRepo   *repository.UserRepo
	spreadRepo *repository.SpreadRepo
}

func NewSpreadHandler(userRepo *repository.UserRepo, spreadRepo *repository.SpreadRepo) *SpreadHandler {
	return &SpreadHandler{userRepo: userRepo, spreadRepo: spreadRepo}
}

func (h *SpreadHandler) GetSpread(w http.ResponseWriter, r *http.Request) {
	uid := middleware.FirebaseUIDFromContext(r.Context())
	user, err := h.userRepo.FindOrCreateByFirebaseUID(uid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to resolve user"})
		return
	}

	st, err := h.spreadRepo.GetTargets(user.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load spread targets"})
		return
	}
	if st == nil {
		st = repository.DefaultTargets()
	}

	actual, err := h.spreadRepo.Actual30d(user.ID)
	if err != nil {
		actual = make(map[string]float64)
	}

	resp := model.SpreadResponse{
		Targets:    make(map[string]float64, len(st.Verticals)),
		Omega:      st.Omega,
		AutoAdjust: st.AutoAdjust,
		Actual30d:  actual,
		Status:     make(map[string]string, len(st.Verticals)),
	}

	for k, v := range st.Verticals {
		resp.Targets[k] = v.Weight
		if v.Pinned {
			resp.Pinned = append(resp.Pinned, k)
		}
		a := actual[k]
		diff := a - v.Weight
		if math.Abs(diff) <= 0.03 {
			resp.Status[k] = "on_target"
		} else if diff < 0 {
			resp.Status[k] = "below_target"
		} else {
			resp.Status[k] = "above_target"
		}
	}

	writeJSON(w, http.StatusOK, resp)
}

type putSpreadRequest struct {
	Targets    map[string]float64 `json:"targets"`
	Omega      string             `json:"omega"`
	Pinned     []string           `json:"pinned"`
	AutoAdjust bool               `json:"auto_adjust"`
}

func (h *SpreadHandler) PutSpread(w http.ResponseWriter, r *http.Request) {
	uid := middleware.FirebaseUIDFromContext(r.Context())
	user, err := h.userRepo.FindOrCreateByFirebaseUID(uid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to resolve user"})
		return
	}

	var req putSpreadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	// Validate weights sum to 1.0 (±0.01).
	sum := 0.0
	for _, w := range req.Targets {
		if w < 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "weights must be >= 0"})
			return
		}
		sum += w
	}
	if math.Abs(sum-1.0) > 0.01 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "weights must sum to 1.0"})
		return
	}

	// Validate omega exists in targets.
	if _, ok := req.Targets[req.Omega]; !ok {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "omega must be a key in targets"})
		return
	}

	// Validate not all pinned when auto_adjust is on.
	pinnedSet := make(map[string]bool, len(req.Pinned))
	for _, p := range req.Pinned {
		pinnedSet[p] = true
	}
	if req.AutoAdjust {
		allPinned := true
		for k := range req.Targets {
			if !pinnedSet[k] {
				allPinned = false
				break
			}
		}
		if allPinned {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "at least one vertical must be unpinned when auto_adjust is enabled"})
			return
		}
	}

	// Build SpreadTargets.
	st := &model.SpreadTargets{
		Verticals:  make(map[string]model.SpreadVertical, len(req.Targets)),
		Omega:      req.Omega,
		AutoAdjust: req.AutoAdjust,
	}
	for k, w := range req.Targets {
		st.Verticals[k] = model.SpreadVertical{Weight: w, Pinned: pinnedSet[k]}
	}

	if err := h.spreadRepo.UpsertTargets(user.ID, st); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save spread targets"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *SpreadHandler) GetHistory(w http.ResponseWriter, r *http.Request) {
	uid := middleware.FirebaseUIDFromContext(r.Context())
	user, err := h.userRepo.FindOrCreateByFirebaseUID(uid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to resolve user"})
		return
	}

	days, err := h.spreadRepo.GetHistory(user.ID, 30)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load history"})
		return
	}
	if days == nil {
		days = []model.SpreadHistoryDay{}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"days": days})
}
```

**Note:** There is a variable shadowing issue in `PutSpread` — the `w` in `for _, w := range req.Targets` shadows the `http.ResponseWriter w`. Rename the loop variable to `wt`:

```go
	sum := 0.0
	for _, wt := range req.Targets {
		if wt < 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "weights must be >= 0"})
			return
		}
		sum += wt
	}
```

And similarly in the Build SpreadTargets section:

```go
	for k, wt := range req.Targets {
		st.Verticals[k] = model.SpreadVertical{Weight: wt, Pinned: pinnedSet[k]}
	}
```

- [ ] **Step 5: Wire handler in main.go**

Open `backend/cmd/server/main.go`. After `settingsH` initialization (around line 135), add:

```go
spreadRepo := repository.NewSpreadRepo(db)
spreadH := handler.NewSpreadHandler(userRepo, spreadRepo)
```

In the Firebase-auth route group (after line 191), add:

```go
r.Get("/settings/spread", spreadH.GetSpread)
r.Put("/settings/spread", spreadH.PutSpread)
r.Get("/settings/spread/history", spreadH.GetHistory)
```

- [ ] **Step 6: Run tests to verify they pass**

Run: `cd /Users/shanegleeson/Repos/beepbopboop/backend && go test ./internal/handler/ -run "TestSpread" -v -count=1`
Expected: All 5 tests PASS.

- [ ] **Step 7: Commit**

```bash
git add backend/internal/handler/spread.go backend/internal/handler/spread_test.go backend/cmd/server/main.go
git commit -m "feat(spread): add GET/PUT /settings/spread endpoints with validation (#185)"
```

---

### Task 4: iOS — SpreadTargets Model & API Methods

**Files:**
- Modify: `beepbopboop/beepbopboop/Services/APIService.swift`

- [ ] **Step 1: Add SpreadTargets model**

Add to `APIService.swift` (or a new Models file if the project uses separate model files):

```swift
struct SpreadTargets: Codable {
    let targets: [String: Double]
    let omega: String
    let pinned: [String]
    let autoAdjust: Bool
    let actual30d: [String: Double]
    let status: [String: String]

    enum CodingKeys: String, CodingKey {
        case targets, omega, pinned
        case autoAdjust = "auto_adjust"
        case actual30d = "actual_30d"
        case status
    }
}

struct SpreadHistoryResponse: Codable {
    let days: [SpreadHistoryDay]
}

struct SpreadHistoryDay: Codable {
    let date: String
    let target: [String: Double]
    let actual: [String: Double]
}

struct PutSpreadRequest: Codable {
    let targets: [String: Double]
    let omega: String
    let pinned: [String]
    let autoAdjust: Bool

    enum CodingKeys: String, CodingKey {
        case targets, omega, pinned
        case autoAdjust = "auto_adjust"
    }
}
```

- [ ] **Step 2: Add API methods to APIService**

Add to the `APIService` class:

```swift
func fetchSpreadTargets() async throws -> SpreadTargets {
    let url = baseURL.appendingPathComponent("settings/spread")
    var request = URLRequest(url: url)
    request.setValue("Bearer \(authToken)", forHTTPHeaderField: "Authorization")
    let (data, _) = try await URLSession.shared.data(for: request)
    return try JSONDecoder().decode(SpreadTargets.self, from: data)
}

func updateSpreadTargets(_ spread: PutSpreadRequest) async throws {
    let url = baseURL.appendingPathComponent("settings/spread")
    var request = URLRequest(url: url)
    request.httpMethod = "PUT"
    request.setValue("Bearer \(authToken)", forHTTPHeaderField: "Authorization")
    request.setValue("application/json", forHTTPHeaderField: "Content-Type")
    request.httpBody = try JSONEncoder().encode(spread)
    let (_, response) = try await URLSession.shared.data(for: request)
    guard let httpResponse = response as? HTTPURLResponse, httpResponse.statusCode == 200 else {
        throw URLError(.badServerResponse)
    }
}

func fetchSpreadHistory() async throws -> SpreadHistoryResponse {
    let url = baseURL.appendingPathComponent("settings/spread/history")
    var request = URLRequest(url: url)
    request.setValue("Bearer \(authToken)", forHTTPHeaderField: "Authorization")
    let (data, _) = try await URLSession.shared.data(for: request)
    return try JSONDecoder().decode(SpreadHistoryResponse.self, from: data)
}
```

- [ ] **Step 3: Build to verify**

Run: `xcodebuild -project beepbopboop/beepbopboop.xcodeproj -scheme beepbopboop -sdk iphonesimulator -destination 'platform=iOS Simulator,name=iPhone 17 Pro' build 2>&1 | tail -5`
Expected: BUILD SUCCEEDED

- [ ] **Step 4: Commit**

```bash
git add beepbopboop/beepbopboop/Services/APIService.swift
git commit -m "feat(ios): add SpreadTargets model and API methods (#185)"
```

---

### Task 5: iOS — ContentMixView & ViewModel

**Files:**
- Create: `beepbopboop/beepbopboop/ViewModels/ContentMixViewModel.swift`
- Create: `beepbopboop/beepbopboop/Views/ContentMixView.swift`
- Modify: `beepbopboop/beepbopboop/Views/SettingsView.swift`

- [ ] **Step 1: Create ContentMixViewModel**

Create `beepbopboop/beepbopboop/ViewModels/ContentMixViewModel.swift`:

```swift
import Foundation

@MainActor
class ContentMixViewModel: ObservableObject {
    @Published var targets: [String: Double] = [:]
    @Published var omega: String = ""
    @Published var pinned: Set<String> = []
    @Published var autoAdjust: Bool = true
    @Published var actual30d: [String: Double] = [:]
    @Published var status: [String: String] = [:]
    @Published var isLoading = false
    @Published var error: String?

    private let apiService: APIService

    static let verticalInfo: [(key: String, emoji: String, name: String)] = [
        ("sports", "🏀", "Sports"),
        ("food", "🍕", "Food"),
        ("music", "🎵", "Music"),
        ("travel", "✈️", "Travel"),
        ("science", "🔬", "Science"),
        ("gaming", "🎮", "Gaming"),
        ("creators", "🎨", "Creators"),
        ("fashion", "👗", "Fashion"),
        ("movies", "🎬", "Movies"),
        ("pets", "🐾", "Pets"),
        ("news", "📰", "News"),
    ]

    static let verticalColors: [String: String] = [
        "sports": "#4CAF50", "food": "#FF9800", "music": "#2196F3",
        "travel": "#9C27B0", "science": "#F44336", "gaming": "#00BCD4",
        "creators": "#795548", "fashion": "#E91E63", "movies": "#607D8B",
        "pets": "#8BC34A", "news": "#FF5722",
    ]

    init(apiService: APIService) {
        self.apiService = apiService
    }

    func load() async {
        isLoading = true
        error = nil
        do {
            let spread = try await apiService.fetchSpreadTargets()
            targets = spread.targets
            omega = spread.omega
            pinned = Set(spread.pinned)
            autoAdjust = spread.autoAdjust
            actual30d = spread.actual30d
            status = spread.status
        } catch {
            self.error = "Failed to load content mix"
        }
        isLoading = false
    }

    func save() async {
        let req = PutSpreadRequest(
            targets: targets,
            omega: omega,
            pinned: Array(pinned),
            autoAdjust: autoAdjust
        )
        do {
            try await apiService.updateSpreadTargets(req)
        } catch {
            self.error = "Failed to save"
        }
    }

    func togglePin(_ vertical: String) {
        if pinned.contains(vertical) {
            pinned.remove(vertical)
        } else {
            pinned.insert(vertical)
        }
        Task { await save() }
    }

    func updateWeight(_ vertical: String, newWeight: Double) {
        let oldWeight = targets[vertical] ?? 0
        let diff = newWeight - oldWeight
        targets[vertical] = newWeight

        // Re-normalize non-pinned, non-changed verticals proportionally.
        let adjustable = targets.keys.filter { $0 != vertical && !pinned.contains($0) }
        let adjustableSum = adjustable.reduce(0.0) { $0 + (targets[$1] ?? 0) }

        if adjustableSum > 0 {
            for key in adjustable {
                let proportion = (targets[key] ?? 0) / adjustableSum
                targets[key] = max(0, (targets[key] ?? 0) - diff * proportion)
            }
        }

        // Normalize to exactly 1.0.
        let total = targets.values.reduce(0, +)
        if total > 0 {
            for key in targets.keys {
                targets[key] = (targets[key] ?? 0) / total
            }
        }
    }
}
```

- [ ] **Step 2: Create ContentMixView**

Create `beepbopboop/beepbopboop/Views/ContentMixView.swift`:

```swift
import SwiftUI

struct ContentMixView: View {
    @StateObject private var viewModel: ContentMixViewModel

    init(apiService: APIService) {
        _viewModel = StateObject(wrappedValue: ContentMixViewModel(apiService: apiService))
    }

    var body: some View {
        Section("Content Mix") {
            if viewModel.isLoading {
                ProgressView()
            } else {
                // Summary bar
                GeometryReader { geo in
                    HStack(spacing: 0) {
                        ForEach(ContentMixViewModel.verticalInfo, id: \.key) { info in
                            let weight = viewModel.targets[info.key] ?? 0
                            if weight > 0 {
                                Rectangle()
                                    .fill(Color(hex: ContentMixViewModel.verticalColors[info.key] ?? "#888"))
                                    .frame(width: geo.size.width * weight)
                            }
                        }
                    }
                    .clipShape(RoundedRectangle(cornerRadius: 4))
                }
                .frame(height: 8)
                .listRowBackground(Color.clear)

                // Vertical rows
                ForEach(ContentMixViewModel.verticalInfo, id: \.key) { info in
                    HStack {
                        Text(info.emoji)
                        Text(info.name)
                            .fontWeight(.medium)

                        if info.key == viewModel.omega {
                            Text("Ω")
                                .font(.caption2)
                                .fontWeight(.bold)
                                .foregroundColor(.white)
                                .padding(.horizontal, 6)
                                .padding(.vertical, 2)
                                .background(Color.green)
                                .clipShape(Capsule())
                        }

                        Spacer()

                        // Status dot
                        let st = viewModel.status[info.key] ?? "on_target"
                        Circle()
                            .fill(st == "on_target" ? Color.green : st == "below_target" ? Color.orange : Color.blue)
                            .frame(width: 6, height: 6)

                        Text("\(Int((viewModel.targets[info.key] ?? 0) * 100))%")
                            .foregroundColor(.secondary)
                            .monospacedDigit()

                        Button {
                            viewModel.togglePin(info.key)
                        } label: {
                            Image(systemName: viewModel.pinned.contains(info.key) ? "pin.fill" : "pin")
                                .foregroundColor(viewModel.pinned.contains(info.key) ? .primary : .secondary.opacity(0.3))
                        }
                        .buttonStyle(.plain)
                    }
                }

                // Auto-adjust toggle
                Toggle("Auto-adjust from engagement", isOn: $viewModel.autoAdjust)
                    .onChange(of: viewModel.autoAdjust) { _ in
                        Task { await viewModel.save() }
                    }
            }
        }
        .task { await viewModel.load() }
    }
}
```

- [ ] **Step 3: Add ContentMixView to SettingsView**

Open `beepbopboop/beepbopboop/Views/SettingsView.swift`. Find the `Form` or `List` body and add the Content Mix section. Add it after the existing sections (like location, radius, sports):

```swift
ContentMixView(apiService: apiService)
```

Where `apiService` is the `@EnvironmentObject` or passed-in APIService instance — match the existing pattern in SettingsView.

- [ ] **Step 4: Add Color(hex:) extension if not present**

Check if a `Color(hex:)` initializer exists. If not, add to a `Color+Extensions.swift` or at the bottom of `ContentMixView.swift`:

```swift
extension Color {
    init(hex: String) {
        let hex = hex.trimmingCharacters(in: CharacterSet.alphanumerics.inverted)
        var int: UInt64 = 0
        Scanner(string: hex).scanHexInt64(&int)
        let r, g, b: Double
        r = Double((int >> 16) & 0xFF) / 255.0
        g = Double((int >> 8) & 0xFF) / 255.0
        b = Double(int & 0xFF) / 255.0
        self.init(red: r, green: g, blue: b)
    }
}
```

- [ ] **Step 5: Build to verify**

Run: `xcodebuild -project beepbopboop/beepbopboop.xcodeproj -scheme beepbopboop -sdk iphonesimulator -destination 'platform=iOS Simulator,name=iPhone 17 Pro' build 2>&1 | tail -5`
Expected: BUILD SUCCEEDED

- [ ] **Step 6: Commit**

```bash
git add beepbopboop/beepbopboop/ViewModels/ContentMixViewModel.swift beepbopboop/beepbopboop/Views/ContentMixView.swift beepbopboop/beepbopboop/Views/SettingsView.swift
git commit -m "feat(ios): add Content Mix settings UI with sliders and status (#185)"
```

---

### Task 6: Update MODE_BATCH.md — Reference Spread API

**Files:**
- Modify: `.claude/skills/beepbopboop-post/MODE_BATCH.md`

- [ ] **Step 1: Update BT2/BT3 to reference spread API**

Open `.claude/skills/beepbopboop-post/MODE_BATCH.md`. Find the section where BT2 sets the target post count and BT3 builds the content plan. Add a new step between BT2 and BT3 (call it BT2.5):

```markdown
### BT2.5: Load spread targets

Fetch the user's content-mix preferences:

```bash
SPREAD=$(curl -s -H "$AUTH" "$API/settings/spread")
```

If the endpoint returns an error or is unavailable, fall back to even distribution across all verticals.

Use the `targets` map to allocate BT2's post count across verticals:

1. **Omega** vertical (`echo "$SPREAD" | jq -r '.omega'`) always gets at least 1 slot.
2. Remaining slots are distributed proportionally by weight: `slots[v] = round(weight[v] * (total - 1))`.
3. If a vertical's weight is 0, skip it entirely.
4. Validate: if the sum of allocated slots exceeds the total, trim the lowest-weight verticals first.

When building the content plan in BT3, use these per-vertical slot counts instead of the hardcoded category defaults. The `status` field shows which verticals are under/over-represented — prioritize `below_target` verticals when filling remaining slots.
```

- [ ] **Step 2: Commit**

```bash
git add .claude/skills/beepbopboop-post/MODE_BATCH.md
git commit -m "feat(skills): reference spread API in batch mode allocation (#185)"
```

---

### Task 7: Update WEIGHT_COMPUTATION.md — Document Auto-Adjust

**Files:**
- Modify: `.claude/skills/beepbopboop-post/WEIGHT_COMPUTATION.md`

- [ ] **Step 1: Add spread-aware section**

Open `.claude/skills/beepbopboop-post/WEIGHT_COMPUTATION.md`. Add a new section at the end:

```markdown
## Spread-Aware Weight Adjustment

When `GET /settings/spread` returns `auto_adjust: true`, Lobs should also nudge the spread targets:

1. Read `GET /settings/spread` to get current targets and `actual_30d`.
2. For each non-pinned vertical:
   - If `actual_30d[v]` < `targets[v]` and positive engagement signals exist (saves, more reactions) → nudge weight **up** by 2%.
   - If `actual_30d[v]` > `targets[v]` and negative signals exist (less, not_for_me reactions) → nudge weight **down** by 2%.
3. Maximum shift per vertical per run: ±2%.
4. Re-normalize all non-pinned weights so they sum to 1.0 minus the sum of pinned weights.
5. `PUT /settings/spread` with the updated targets.

**Pinned verticals** are locked — never adjust their weights. The user explicitly chose them.

**Auto-adjust disabled:** If `auto_adjust: false`, skip this entire section. Only the user can change weights manually.
```

- [ ] **Step 2: Commit**

```bash
git add .claude/skills/beepbopboop-post/WEIGHT_COMPUTATION.md
git commit -m "docs(skills): document spread-aware auto-adjustment in Lobs (#185)"
```

---

### Task 8: Create SPORTS_COMMON.md — Shared Sport Patterns

**Files:**
- Create: `.claude/skills/_shared/SPORTS_COMMON.md`
- Modify: `.claude/skills/beepbopboop-soccer/SKILL.md`
- Modify: `.claude/skills/beepbopboop-basketball/SKILL.md`
- Modify: `.claude/skills/beepbopboop-football/SKILL.md`
- Modify: `.claude/skills/beepbopboop-baseball/SKILL.md`

- [ ] **Step 1: Create SPORTS_COMMON.md**

Create `.claude/skills/_shared/SPORTS_COMMON.md`:

```markdown
# Sports Common Patterns

Shared conventions for all `beepbopboop-*` sport skills. Read this file once at Step 0 alongside CONFIG.md.

## Source Rules

- **Never hallucinate stats, scores, or records.** Every number must come from an API response or a cited web source.
- Use official league APIs, ESPN, or team sites as primary sources. Fan sites and social media are secondary.
- If a stat cannot be verified, omit it rather than guess.

## Display Hints

Sport skills produce posts with these structured display hints:

| Hint | When to use | Required external_url fields |
|------|-------------|------------------------------|
| `scoreboard` | Live or final game scores | `home_team`, `away_team`, `home_score`, `away_score`, `status`, `period` |
| `matchup` | Upcoming game previews | `home_team`, `away_team`, `date`, `venue`, `odds` (optional) |
| `standings` | League/division standings | `entries[]` with `team`, `wins`, `losses`, `pct`, `gb` |
| `box_score` | Detailed post-game stats | `home_team`, `away_team`, `home_score`, `away_score`, `leaders[]` |
| `player_spotlight` | Individual player features | `name`, `team`, `position`, `stats{}`, `headline` |

All `external_url` values must be JSON strings (not raw objects). Use the canonical pattern from `PUBLISH_ENVELOPE.md`:
```bash
EXTERNAL_URL=$(echo "$DATA_JSON" | jq -c . | jq -Rs .)
```

## Labels

Every sport post must include:
1. The sport label: `sports` (always first)
2. The league: `nba`, `nfl`, `mlb`, `premier-league`, `champions-league`, etc.
3. Team slugs if applicable: `nba:lal`, `nfl:sf`, `mlb:nyy`

## Team Data

If `BEEPBOPBOOP_SPORTS_TEAMS` is set in config, prioritize those teams. Format: comma-separated league:abbrev pairs (e.g., `nba:lal,nfl:sf,mlb:nyy`).

## Publishing

After building the post payload, follow `../_shared/PUBLISH_ENVELOPE.md` for lint + publish. Always lint before POST.
```

- [ ] **Step 2: Add reference line to each sport skill**

For each of these 4 files, add a line after their Step 0 config loading section:

**`.claude/skills/beepbopboop-soccer/SKILL.md`** — after the config loading step, add:
```markdown
Read `../_shared/SPORTS_COMMON.md` for shared sport conventions (source rules, display hints, labels, team data, publishing).
```

**`.claude/skills/beepbopboop-basketball/SKILL.md`** — same line after config loading.

**`.claude/skills/beepbopboop-football/SKILL.md`** — same line after config loading.

**`.claude/skills/beepbopboop-baseball/SKILL.md`** — same line after config loading.

- [ ] **Step 3: Commit**

```bash
git add .claude/skills/_shared/SPORTS_COMMON.md .claude/skills/beepbopboop-soccer/SKILL.md .claude/skills/beepbopboop-basketball/SKILL.md .claude/skills/beepbopboop-football/SKILL.md .claude/skills/beepbopboop-baseball/SKILL.md
git commit -m "feat(skills): extract shared sport patterns to SPORTS_COMMON.md (#180)"
```

---

### Task 9: Rename INIT_WIZARD.md → MODE_INIT.md

**Files:**
- Rename: `.claude/skills/beepbopboop-post/INIT_WIZARD.md` → `.claude/skills/beepbopboop-post/MODE_INIT.md`
- Modify: `.claude/skills/beepbopboop-post/SKILL.md`
- Modify: `.claude/skills/_shared/CONFIG.md`

- [ ] **Step 1: Rename the file**

```bash
mv .claude/skills/beepbopboop-post/INIT_WIZARD.md .claude/skills/beepbopboop-post/MODE_INIT.md
```

- [ ] **Step 2: Update references in SKILL.md**

Open `.claude/skills/beepbopboop-post/SKILL.md`. Replace all occurrences of `INIT_WIZARD.md` with `MODE_INIT.md`. This includes the Step 0 reference (line 56):

Before: `If the required keys are missing, jump to the Init Wizard (read `INIT_WIZARD.md`), then return here.`
After: `If the required keys are missing, jump to the Init Wizard (read `MODE_INIT.md`), then return here.`

Also update the mode table if INIT_WIZARD appears there.

- [ ] **Step 3: Update references in CONFIG.md**

Open `.claude/skills/_shared/CONFIG.md`. Replace all occurrences of `INIT_WIZARD.md` with `MODE_INIT.md`.

- [ ] **Step 4: Commit**

```bash
git add .claude/skills/beepbopboop-post/MODE_INIT.md .claude/skills/beepbopboop-post/SKILL.md .claude/skills/_shared/CONFIG.md
git rm .claude/skills/beepbopboop-post/INIT_WIZARD.md
git commit -m "refactor(skills): rename INIT_WIZARD.md → MODE_INIT.md for consistency (#180)"
```

---

### Task 10: Client-Side Hints Cache

**Files:**
- Modify: `.claude/skills/_shared/CONTEXT_BOOTSTRAP.md`

- [ ] **Step 1: Add cache logic to CONTEXT_BOOTSTRAP.md**

Open `.claude/skills/_shared/CONTEXT_BOOTSTRAP.md`. Find the section where `HINTS` is fetched (the `curl` to `/posts/hints`). Replace the direct fetch with a cache-aware version:

Add a new subsection before the parallel fetch block:

```markdown
### Hints cache

The hints catalog changes rarely (only when new display hints are added). To avoid fetching on every invocation, use a local cache:

```bash
HINTS_CACHE="$HOME/.cache/beepbopboop/hints.json"
HINTS_STALE=true

if [ -f "$HINTS_CACHE" ]; then
  FETCHED_AT=$(jq -r '.fetched_at // empty' "$HINTS_CACHE" 2>/dev/null)
  if [ -n "$FETCHED_AT" ]; then
    # Check if cache is < 24 hours old
    CACHE_AGE=$(( $(date +%s) - $(date -d "$FETCHED_AT" +%s 2>/dev/null || date -j -f "%Y-%m-%dT%H:%M:%S" "${FETCHED_AT%%.*}" +%s 2>/dev/null || echo 0) ))
    if [ "$CACHE_AGE" -lt 86400 ]; then
      HINTS=$(jq '.hints' "$HINTS_CACHE")
      HINTS_STALE=false
    fi
  fi
fi

if [ "$HINTS_STALE" = true ]; then
  HINTS=$(curl -s -H "$AUTH" "$API/posts/hints")
  mkdir -p "$(dirname "$HINTS_CACHE")"
  echo "{\"fetched_at\": \"$(date -u +%Y-%m-%dT%H:%M:%SZ)\", \"hints\": $HINTS}" > "$HINTS_CACHE"
fi
```

When the cache is fresh, skip the `/posts/hints` fetch in the parallel block below. The other three fetches (`/posts/stats`, `/reactions/summary`, `/events/summary`) always run fresh — they contain time-sensitive data.
```

- [ ] **Step 2: Commit**

```bash
git add .claude/skills/_shared/CONTEXT_BOOTSTRAP.md
git commit -m "feat(skills): add client-side hints cache with 24h TTL (#180)"
```

---

### Task 11: Close #188 + Final Integration Test

**Files:** None (test + issue management only)

- [ ] **Step 1: Run full Go test suite**

Run: `cd /Users/shanegleeson/Repos/beepbopboop/backend && go test ./internal/handler/ -v -count=1`
Expected: All tests pass (including new Spread tests).

- [ ] **Step 2: Verify iOS builds**

Run: `xcodebuild -project beepbopboop/beepbopboop.xcodeproj -scheme beepbopboop -sdk iphonesimulator -destination 'platform=iOS Simulator,name=iPhone 17 Pro' build 2>&1 | tail -5`
Expected: BUILD SUCCEEDED

- [ ] **Step 3: Verify all skill files exist and reference correctly**

```bash
ls .claude/skills/_shared/SPORTS_COMMON.md
ls .claude/skills/beepbopboop-post/MODE_INIT.md
test ! -f .claude/skills/beepbopboop-post/INIT_WIZARD.md && echo "INIT_WIZARD.md removed OK"
grep -l "SPORTS_COMMON" .claude/skills/beepbopboop-soccer/SKILL.md .claude/skills/beepbopboop-basketball/SKILL.md .claude/skills/beepbopboop-football/SKILL.md .claude/skills/beepbopboop-baseball/SKILL.md
grep -l "MODE_INIT" .claude/skills/beepbopboop-post/SKILL.md
grep -l "hints.json" .claude/skills/_shared/CONTEXT_BOOTSTRAP.md
```

Expected: All files exist, all references found.

- [ ] **Step 4: Close #188**

```bash
gh issue close 188 --comment "Feed consolidation complete — iOS shows 3 tabs (For You, Following, Community). Personal/Saved are programmatic-only. No further work needed."
```

- [ ] **Step 5: Close #180 and #185 via PR**

These will be closed when the PR is merged with `Closes #185, #180` in the body.
