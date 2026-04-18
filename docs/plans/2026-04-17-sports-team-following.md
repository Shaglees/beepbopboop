# Sports Team/League Following Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Let users follow specific teams/leagues so those posts rank higher in the ForYou feed.

**Architecture:** `followed_teams JSONB` column on `user_settings`; team keys `"{league}:{abbr_lower}"` like `"nba:lal"`. Backend settings API round-trips the array; `scorePost()` boosts +1.5 per matched team by parsing `external_url` JSON. iOS adds a `SportsSettingsView` navigated from a "Sports & Teams" row in Settings.

**Tech Stack:** Go (database/sql, encoding/json), SwiftUI, PostgreSQL JSONB

---

### Task 1: Backend DB migration + model

**Files:**
- Modify: `backend/internal/database/database.go`
- Modify: `backend/internal/model/model.go`

**Step 1: Add column migration to `database.go`**

After the existing `user_settings` CREATE TABLE block (around line 43), add:

```go
db.Exec("ALTER TABLE user_settings ADD COLUMN IF NOT EXISTS followed_teams JSONB")
```

**Step 2: Add field to `model.UserSettings`**

In `backend/internal/model/model.go`, update `UserSettings`:

```go
type UserSettings struct {
	UserID        string    `json:"user_id"`
	LocationName  string    `json:"location_name,omitempty"`
	Latitude      *float64  `json:"latitude,omitempty"`
	Longitude     *float64  `json:"longitude,omitempty"`
	RadiusKm      float64   `json:"radius_km"`
	FollowedTeams []string  `json:"followed_teams,omitempty"`
	UpdatedAt     time.Time `json:"updated_at"`
}
```

**Step 3: Commit**

```bash
git add backend/internal/database/database.go backend/internal/model/model.go
git commit -m "feat: add followed_teams JSONB column and model field"
```

---

### Task 2: Repository — read/write followed_teams

**Files:**
- Modify: `backend/internal/repository/user_settings_repo.go`

**Step 1: Update `Get()` to scan followed_teams**

Replace the `Get` method. The new query adds `followed_teams` to the SELECT and scans it:

```go
func (r *UserSettingsRepo) Get(userID string) (*model.UserSettings, error) {
	var s model.UserSettings
	var locationName sql.NullString
	var latitude, longitude sql.NullFloat64
	var followedTeamsJSON sql.NullString

	err := r.db.QueryRow(`
		SELECT user_id, location_name, latitude, longitude, radius_km, followed_teams, updated_at
		FROM user_settings WHERE user_id = $1`, userID,
	).Scan(&s.UserID, &locationName, &latitude, &longitude, &s.RadiusKm, &followedTeamsJSON, &s.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query user_settings: %w", err)
	}
	s.LocationName = locationName.String
	if latitude.Valid {
		s.Latitude = &latitude.Float64
	}
	if longitude.Valid {
		s.Longitude = &longitude.Float64
	}
	if followedTeamsJSON.Valid && followedTeamsJSON.String != "" && followedTeamsJSON.String != "null" {
		json.Unmarshal([]byte(followedTeamsJSON.String), &s.FollowedTeams)
	}
	return &s, nil
}
```

Add `"encoding/json"` to the import block if not already present.

**Step 2: Update `Upsert()` to accept and write followed_teams**

Replace the `Upsert` method signature and body:

```go
func (r *UserSettingsRepo) Upsert(userID, locationName string, lat, lon *float64, radiusKm float64, followedTeams []string) (*model.UserSettings, error) {
	var followedTeamsJSON sql.NullString
	if len(followedTeams) > 0 {
		b, err := json.Marshal(followedTeams)
		if err != nil {
			return nil, fmt.Errorf("marshal followed_teams: %w", err)
		}
		followedTeamsJSON = sql.NullString{String: string(b), Valid: true}
	}

	_, err := r.db.Exec(`
		INSERT INTO user_settings (user_id, location_name, latitude, longitude, radius_km, followed_teams, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, CURRENT_TIMESTAMP)
		ON CONFLICT(user_id) DO UPDATE SET
			location_name = excluded.location_name,
			latitude = excluded.latitude,
			longitude = excluded.longitude,
			radius_km = excluded.radius_km,
			followed_teams = excluded.followed_teams,
			updated_at = CURRENT_TIMESTAMP`,
		userID, nullString(locationName), nullFloat64(lat), nullFloat64(lon), radiusKm, followedTeamsJSON,
	)
	if err != nil {
		return nil, fmt.Errorf("upsert user_settings: %w", err)
	}
	return r.Get(userID)
}
```

**Step 3: Commit**

```bash
git add backend/internal/repository/user_settings_repo.go
git commit -m "feat: persist followed_teams in user_settings repo"
```

---

### Task 3: Settings handler — accept followed_teams in PUT

**Files:**
- Modify: `backend/internal/handler/settings.go`

**Step 1: Add field to request struct**

Update `updateSettingsRequest`:

```go
type updateSettingsRequest struct {
	LocationName  string   `json:"location_name"`
	Latitude      *float64 `json:"latitude"`
	Longitude     *float64 `json:"longitude"`
	RadiusKm      float64  `json:"radius_km"`
	FollowedTeams []string `json:"followed_teams"`
}
```

**Step 2: Pass followed_teams to Upsert**

Update the `Upsert` call in `UpdateSettings` (line ~79):

```go
settings, err := h.userSettingsRepo.Upsert(user.ID, req.LocationName, req.Latitude, req.Longitude, req.RadiusKm, req.FollowedTeams)
```

**Step 3: Verify the project builds**

```bash
cd backend && go build ./...
```

Expected: no errors.

**Step 4: Commit**

```bash
git add backend/internal/handler/settings.go
git commit -m "feat: settings handler accepts and returns followed_teams"
```

---

### Task 4: Feed scoring — boost followed teams in ForYou

**Files:**
- Modify: `backend/internal/repository/post_repo.go`
- Modify: `backend/internal/handler/multi_feed.go`

**Step 1: Add FollowedTeams to FeedWeights**

In `post_repo.go`, update `FeedWeights`:

```go
type FeedWeights struct {
	LabelWeights  map[string]float64 `json:"label_weights"`
	TypeWeights   map[string]float64 `json:"type_weights"`
	FreshnessBias float64            `json:"freshness_bias"`
	GeoBias       float64            `json:"geo_bias"`
	FollowedTeams map[string]bool    `json:"-"`
}
```

**Step 2: Add team boost to scorePost()**

At the end of `scorePost()`, before the `return score` statement, add:

```go
// Team affinity: parse sports post external_url and boost matched followed teams.
if len(w.FollowedTeams) > 0 && p.ExternalURL != "" {
	var g struct {
		Sport string `json:"sport"`
		Home  struct{ Abbr string `json:"abbr"` } `json:"home"`
		Away  struct{ Abbr string `json:"abbr"` } `json:"away"`
	}
	if json.Unmarshal([]byte(p.ExternalURL), &g) == nil && g.Sport != "" {
		sport := strings.ToLower(g.Sport)
		if abbr := strings.ToLower(g.Home.Abbr); abbr != "" && w.FollowedTeams[sport+":"+abbr] {
			score += 1.5
		}
		if abbr := strings.ToLower(g.Away.Abbr); abbr != "" && w.FollowedTeams[sport+":"+abbr] {
			score += 1.5
		}
	}
}
```

Both `"encoding/json"` and `"strings"` are already imported in post_repo.go.

**Step 3: Populate FollowedTeams in GetForYou**

In `multi_feed.go`, after `feedWeights` is set (after the `weightsRepo.GetOrCompute` block, before the `postRepo.ListForYou` call), add:

```go
if settings != nil && len(settings.FollowedTeams) > 0 {
	feedWeights.FollowedTeams = make(map[string]bool, len(settings.FollowedTeams))
	for _, t := range settings.FollowedTeams {
		feedWeights.FollowedTeams[t] = true
	}
}
```

**Step 4: Verify build**

```bash
cd backend && go build ./...
```

**Step 5: Commit**

```bash
git add backend/internal/repository/post_repo.go backend/internal/handler/multi_feed.go
git commit -m "feat: boost followed teams +1.5 in ForYou feed scoring"
```

---

### Task 5: Backend tests for settings

**Files:**
- Create: `backend/internal/handler/settings_test.go`

**Step 1: Write failing tests**

Create the file:

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

func TestSettingsHandler_GetSettings_Default(t *testing.T) {
	db := database.OpenTestDB(t)

	userRepo := repository.NewUserRepo(db)
	userSettingsRepo := repository.NewUserSettingsRepo(db)
	h := handler.NewSettingsHandler(userRepo, userSettingsRepo)

	req := httptest.NewRequest("GET", "/user/settings", nil)
	req = req.WithContext(middleware.WithFirebaseUID(req.Context(), "firebase-abc"))
	rec := httptest.NewRecorder()

	h.GetSettings(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var settings map[string]any
	json.NewDecoder(rec.Body).Decode(&settings)
	if settings["radius_km"] != 25.0 {
		t.Errorf("expected default radius_km 25, got %v", settings["radius_km"])
	}
}

func TestSettingsHandler_UpdateSettings_FollowedTeams(t *testing.T) {
	db := database.OpenTestDB(t)

	userRepo := repository.NewUserRepo(db)
	userSettingsRepo := repository.NewUserSettingsRepo(db)
	h := handler.NewSettingsHandler(userRepo, userSettingsRepo)

	body := `{
		"location_name": "Toronto",
		"latitude": 43.651070,
		"longitude": -79.347015,
		"radius_km": 25,
		"followed_teams": ["nhl:tor", "nba:lal"]
	}`
	req := httptest.NewRequest("PUT", "/user/settings", bytes.NewBufferString(body))
	req = req.WithContext(middleware.WithFirebaseUID(req.Context(), "firebase-abc"))
	rec := httptest.NewRecorder()

	h.UpdateSettings(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("PUT expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// Round-trip: GET should return the same followed_teams
	req2 := httptest.NewRequest("GET", "/user/settings", nil)
	req2 = req2.WithContext(middleware.WithFirebaseUID(req2.Context(), "firebase-abc"))
	rec2 := httptest.NewRecorder()
	h.GetSettings(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Fatalf("GET expected 200, got %d", rec2.Code)
	}

	var settings map[string]any
	json.NewDecoder(rec2.Body).Decode(&settings)
	teams, ok := settings["followed_teams"].([]any)
	if !ok {
		t.Fatalf("expected followed_teams array, got %T: %v", settings["followed_teams"], settings["followed_teams"])
	}
	if len(teams) != 2 {
		t.Errorf("expected 2 followed teams, got %d: %v", len(teams), teams)
	}
}

func TestSettingsHandler_UpdateSettings_ClearsFollowedTeams(t *testing.T) {
	db := database.OpenTestDB(t)

	userRepo := repository.NewUserRepo(db)
	userSettingsRepo := repository.NewUserSettingsRepo(db)
	h := handler.NewSettingsHandler(userRepo, userSettingsRepo)

	// First: set some teams
	body1 := `{"location_name":"Toronto","latitude":43.65,"longitude":-79.35,"radius_km":25,"followed_teams":["nhl:tor"]}`
	req1 := httptest.NewRequest("PUT", "/user/settings", bytes.NewBufferString(body1))
	req1 = req1.WithContext(middleware.WithFirebaseUID(req1.Context(), "firebase-clear"))
	h.UpdateSettings(httptest.NewRecorder(), req1)

	// Second: clear teams by sending empty array
	body2 := `{"location_name":"Toronto","latitude":43.65,"longitude":-79.35,"radius_km":25,"followed_teams":[]}`
	req2 := httptest.NewRequest("PUT", "/user/settings", bytes.NewBufferString(body2))
	req2 = req2.WithContext(middleware.WithFirebaseUID(req2.Context(), "firebase-clear"))
	rec2 := httptest.NewRecorder()
	h.UpdateSettings(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec2.Code)
	}

	req3 := httptest.NewRequest("GET", "/user/settings", nil)
	req3 = req3.WithContext(middleware.WithFirebaseUID(req3.Context(), "firebase-clear"))
	rec3 := httptest.NewRecorder()
	h.GetSettings(rec3, req3)

	var settings map[string]any
	json.NewDecoder(rec3.Body).Decode(&settings)
	if _, exists := settings["followed_teams"]; exists {
		t.Errorf("expected followed_teams to be absent after clearing, got %v", settings["followed_teams"])
	}
}
```

**Step 2: Run tests**

```bash
cd backend && go test ./internal/handler/... -run TestSettingsHandler -v -timeout 120s
```

Expected: All 3 tests PASS (they require Docker for the test DB container).

**Step 3: Commit**

```bash
git add backend/internal/handler/settings_test.go
git commit -m "test: settings handler GET/PUT round-trip with followed_teams"
```

---

### Task 6: iOS — UserSettings model update

**Files:**
- Modify: `beepbopboop/beepbopboop/Models/UserSettings.swift`

**Step 1: Add followedTeams field**

Replace the entire file:

```swift
import Foundation

struct UserSettings: Codable {
    var locationName: String?
    var latitude: Double?
    var longitude: Double?
    var radiusKm: Double
    var followedTeams: [String]?

    var hasLocation: Bool {
        latitude != nil && longitude != nil
    }

    enum CodingKeys: String, CodingKey {
        case locationName = "location_name"
        case latitude
        case longitude
        case radiusKm = "radius_km"
        case followedTeams = "followed_teams"
    }
}
```

**Step 2: Commit**

```bash
git add beepbopboop/beepbopboop/Models/UserSettings.swift
git commit -m "feat(ios): add followedTeams to UserSettings model"
```

---

### Task 7: iOS — SportsSettingsView

**Files:**
- Create: `beepbopboop/beepbopboop/Views/SportsSettingsView.swift`

**Step 1: Create the view**

```swift
import SwiftUI

private struct SportsTeam {
    let name: String
    let abbr: String
}

private struct SportsLeague {
    let name: String
    let id: String
    let teams: [SportsTeam]
}

private let allLeagues: [SportsLeague] = [
    SportsLeague(name: "NBA", id: "nba", teams: [
        SportsTeam(name: "Boston Celtics", abbr: "bos"),
        SportsTeam(name: "Chicago Bulls", abbr: "chi"),
        SportsTeam(name: "Denver Nuggets", abbr: "den"),
        SportsTeam(name: "Golden State Warriors", abbr: "gsw"),
        SportsTeam(name: "Los Angeles Lakers", abbr: "lal"),
        SportsTeam(name: "Miami Heat", abbr: "mia"),
        SportsTeam(name: "Milwaukee Bucks", abbr: "mil"),
        SportsTeam(name: "New York Knicks", abbr: "nyk"),
        SportsTeam(name: "Oklahoma City Thunder", abbr: "okc"),
        SportsTeam(name: "San Antonio Spurs", abbr: "sas"),
    ]),
    SportsLeague(name: "NHL", id: "nhl", teams: [
        SportsTeam(name: "Boston Bruins", abbr: "bos"),
        SportsTeam(name: "Calgary Flames", abbr: "cgy"),
        SportsTeam(name: "Chicago Blackhawks", abbr: "chi"),
        SportsTeam(name: "Edmonton Oilers", abbr: "edm"),
        SportsTeam(name: "Montreal Canadiens", abbr: "mtl"),
        SportsTeam(name: "New York Rangers", abbr: "nyr"),
        SportsTeam(name: "Ottawa Senators", abbr: "ott"),
        SportsTeam(name: "Toronto Maple Leafs", abbr: "tor"),
        SportsTeam(name: "Vancouver Canucks", abbr: "van"),
        SportsTeam(name: "Winnipeg Jets", abbr: "wpg"),
    ]),
    SportsLeague(name: "MLB", id: "mlb", teams: [
        SportsTeam(name: "Atlanta Braves", abbr: "atl"),
        SportsTeam(name: "Boston Red Sox", abbr: "bos"),
        SportsTeam(name: "Chicago Cubs", abbr: "chc"),
        SportsTeam(name: "Houston Astros", abbr: "hou"),
        SportsTeam(name: "Los Angeles Dodgers", abbr: "lad"),
        SportsTeam(name: "New York Yankees", abbr: "nyy"),
        SportsTeam(name: "San Francisco Giants", abbr: "sfg"),
        SportsTeam(name: "St. Louis Cardinals", abbr: "stl"),
        SportsTeam(name: "Toronto Blue Jays", abbr: "tor"),
        SportsTeam(name: "Cincinnati Reds", abbr: "cin"),
    ]),
    SportsLeague(name: "NFL", id: "nfl", teams: [
        SportsTeam(name: "Buffalo Bills", abbr: "buf"),
        SportsTeam(name: "Chicago Bears", abbr: "chi"),
        SportsTeam(name: "Dallas Cowboys", abbr: "dal"),
        SportsTeam(name: "Denver Broncos", abbr: "den"),
        SportsTeam(name: "Green Bay Packers", abbr: "gb"),
        SportsTeam(name: "Kansas City Chiefs", abbr: "kc"),
        SportsTeam(name: "New England Patriots", abbr: "ne"),
        SportsTeam(name: "Philadelphia Eagles", abbr: "phi"),
        SportsTeam(name: "San Francisco 49ers", abbr: "sf"),
        SportsTeam(name: "Seattle Seahawks", abbr: "sea"),
    ]),
]

struct SportsSettingsView: View {
    @Binding var followedTeams: Set<String>

    var body: some View {
        List {
            ForEach(allLeagues, id: \.id) { league in
                Section(league.name) {
                    ForEach(league.teams, id: \.abbr) { team in
                        let key = "\(league.id):\(team.abbr)"
                        Button {
                            if followedTeams.contains(key) {
                                followedTeams.remove(key)
                            } else {
                                followedTeams.insert(key)
                            }
                        } label: {
                            HStack {
                                Text(team.name)
                                    .foregroundColor(.primary)
                                Spacer()
                                if followedTeams.contains(key) {
                                    Image(systemName: "checkmark")
                                        .foregroundColor(.accentColor)
                                }
                            }
                        }
                    }
                }
            }
        }
        .navigationTitle("Sports & Teams")
        .navigationBarTitleDisplayMode(.inline)
    }
}
```

**Step 2: Commit**

```bash
git add beepbopboop/beepbopboop/Views/SportsSettingsView.swift
git commit -m "feat(ios): add SportsSettingsView with league/team picker"
```

---

### Task 8: iOS — Wire SettingsView and ViewModel

**Files:**
- Modify: `beepbopboop/beepbopboop/Views/SettingsView.swift`

**Step 1: Add followedTeams to SettingsViewModel**

In the `SettingsViewModel` class, add a published property after the `didSave` property:

```swift
@Published var followedTeams: Set<String> = []
```

**Step 2: Load followedTeams in loadSettings()**

In `loadSettings()`, after setting `selectedRadius`, add:

```swift
followedTeams = Set(settings.followedTeams ?? [])
```

**Step 3: Include followedTeams in save()**

In `save()`, update the `UserSettings` construction. Find the block:

```swift
let settings = UserSettings(
    locationName: selectedLocationName,
    latitude: selectedLatitude,
    longitude: selectedLongitude,
    radiusKm: selectedRadius
)
```

Replace with:

```swift
let settings = UserSettings(
    locationName: selectedLocationName,
    latitude: selectedLatitude,
    longitude: selectedLongitude,
    radiusKm: selectedRadius,
    followedTeams: followedTeams.isEmpty ? nil : Array(followedTeams)
)
```

**Step 4: Add "Sports & Teams" row to SettingsView**

In `SettingsView.body`, in the `Form`, add a new `Section` after the Radius section (before the Save button section):

```swift
Section("Sports") {
    NavigationLink("Sports & Teams") {
        SportsSettingsView(followedTeams: $viewModel.followedTeams)
    }
}
```

**Step 5: Commit**

```bash
git add beepbopboop/beepbopboop/Views/SettingsView.swift
git commit -m "feat(ios): wire Sports & Teams picker into SettingsView"
```

---

### Task 9: Build check

**Step 1: Build backend**

```bash
cd backend && go build ./...
```

Expected: no errors.

**Step 2: Build iOS**

```bash
cd beepbopboop && xcodebuild -scheme beepbopboop -destination 'generic/platform=iOS Simulator' build 2>&1 | tail -5
```

Expected: `** BUILD SUCCEEDED **`

**Step 3: Fix any compilation errors before continuing**

---

### Task 10: Create PR and update issue

**Step 1: Push branch**

```bash
git push -u origin claude/distracted-ritchie-2415f5
```

**Step 2: Create PR**

```bash
gh pr create \
  --title "feat: sports team/league following (#30)" \
  --body "$(cat <<'EOF'
## Summary

- Adds `followed_teams JSONB` to `user_settings` (format: `"{league}:{abbr_lower}"` e.g. `"nba:lal"`)
- GET/PUT `/user/settings` round-trips the array
- `scorePost()` boosts +1.5 per matched team by parsing `external_url` JSON in sports posts
- iOS: "Sports & Teams" screen navigable from Settings — league sections with team toggles
- Three backend handler tests covering default GET, PUT round-trip, and clearing teams

Closes #30

## Test plan
- [ ] `go test ./internal/handler/... -run TestSettingsHandler` passes
- [ ] `go build ./...` succeeds
- [ ] iOS builds without errors
- [ ] Follow a team in Settings → ForYou feed surfaces that team's games higher

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
)"
```

**Step 3: Comment on issue #30**

```bash
gh issue comment 30 --body "$(cat <<'EOF'
## Implementation complete — PR raised

**What was built:**

- `followed_teams JSONB` column on `user_settings`; keys are `"{league}:{abbr_lower}"` (e.g. `"nba:lal"`, `"nhl:tor"`)
- GET/PUT `/user/settings` extended to round-trip the array
- `scorePost()` in ForYou feed boosts matched teams by +1.5 by parsing `external_url.sport`/`home.abbr`/`away.abbr`
- iOS: "Sports & Teams" screen with NHL/NBA/MLB/NFL league sections and team toggles, navigated from Settings
- 3 backend handler tests (default, round-trip, clear)

Users with no followed teams see unchanged ranking behaviour.
EOF
)"
```
