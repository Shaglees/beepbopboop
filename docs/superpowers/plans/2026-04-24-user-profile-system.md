# User Profile System Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a user profile system with identity fields, rich interests, lifestyle tags, and content preferences — consumed by both iOS and skills.

**Architecture:** Extend the `users` table with identity columns. Add three new tables (`user_interests`, `user_lifestyle_tags`, `user_content_prefs`). New `ProfileHandler` serves `GET/PUT /user/profile` for both Firebase and agent auth. iOS onboarding flow populates the profile. Skills fetch the profile in their bootstrap.

**Tech Stack:** Go (chi router, database/sql, pq), SwiftUI, PostgreSQL

**Spec:** `docs/superpowers/specs/2026-04-24-user-profile-system-design.md`

---

## File Structure

### Backend — New Files
| File | Responsibility |
|------|---------------|
| `backend/internal/model/profile.go` | `UserInterest`, `LifestyleTag`, `ContentPref`, `UserProfile` structs |
| `backend/internal/repository/user_interest_repo.go` | CRUD for `user_interests` table |
| `backend/internal/repository/user_lifestyle_repo.go` | CRUD for `user_lifestyle_tags` table |
| `backend/internal/repository/user_content_prefs_repo.go` | CRUD for `user_content_prefs` table |
| `backend/internal/handler/profile.go` | All profile endpoints (GET/PUT profile, interests, lifestyle, prefs) |
| `backend/internal/handler/profile_test.go` | HTTP handler tests |
| `backend/internal/interest/worker.go` | Background interest inference worker |
| `backend/internal/interest/worker_test.go` | Worker tests |
| `backend/internal/interest/decay.go` | Interest decay checker — generates feedback posts for disengaged interests |
| `backend/internal/interest/decay_test.go` | Decay checker tests |

### Backend — Modified Files
| File | Changes |
|------|---------|
| `backend/internal/model/model.go` | Add profile fields to `User` struct |
| `backend/internal/repository/user_repo.go` | Update queries to include new columns, add `UpdateProfile` method |
| `backend/internal/database/database.go` | Add migration statements for new tables and columns |
| `backend/internal/handler/onboarding.go` | Also write plaintext interests to `user_interests` table |
| `backend/internal/handler/post.go` | Update `GetPostStats` to include user's `max_per_day` from content prefs |
| `backend/cmd/server/main.go` | Register new routes, instantiate handler, start worker |

### iOS — New Files
| File | Responsibility |
|------|---------------|
| `beepbopboop/beepbopboop/Models/UserProfile.swift` | `UserProfile`, `UserInterest`, `LifestyleTag`, `ContentPref` Codable structs |
| `beepbopboop/beepbopboop/Views/Onboarding/OnboardingView.swift` | Container view managing 7 onboarding steps |
| `beepbopboop/beepbopboop/Views/Onboarding/OnboardingNameView.swift` | Step 1: Name & avatar |
| `beepbopboop/beepbopboop/Views/Onboarding/OnboardingLocationView.swift` | Step 2: Location & timezone |
| `beepbopboop/beepbopboop/Views/Onboarding/OnboardingNotificationsView.swift` | Step 3: Notifications setup |
| `beepbopboop/beepbopboop/Views/Onboarding/OnboardingInterestsView.swift` | Step 4: Interest grid with card carousel |
| `beepbopboop/beepbopboop/Views/Onboarding/OnboardingFrequencyView.swift` | Step 5: Content frequency |
| `beepbopboop/beepbopboop/Views/Onboarding/OnboardingLifestyleView.swift` | Step 6: Lifestyle tags |
| `beepbopboop/beepbopboop/Views/Onboarding/OnboardingPrefsView.swift` | Step 7: Content preferences |
| `beepbopboop/beepbopboop/Views/ProfileView.swift` | Profile display and editing screen |

### iOS — Modified Files
| File | Changes |
|------|---------|
| `beepbopboop/beepbopboop/Services/APIService.swift` | Add profile fetch/update methods |
| `beepbopboop/beepbopboop/Services/AuthService.swift` | Fetch profile after sign-in, expose `profileInitialized` |
| `beepbopboop/beepbopboop/Views/FeedListView.swift` | Gate on `profileInitialized`, show onboarding if false |

### Skills — Modified Files
| File | Changes |
|------|---------|
| `.claude/skills/_shared/CONTEXT_BOOTSTRAP.md` | Add `GET /user/profile` as 5th parallel fetch |
| `.claude/skills/beepbopboop-post/MODE_BATCH.md` | Read profile for interest-driven content planning |
| `.claude/skills/beepbopboop-post/BASE_LOCAL.md` | Read profile for contextual depth |
| `.claude/skills/beepbopboop-post/SKILL.md` | Pin profile into working memory |

---

## Task 1: Database Schema — New Tables & Columns

**Files:**
- Modify: `backend/internal/database/database.go:320-332` (before `return db, nil`)

- [ ] **Step 1: Add migration statements to database.go**

Add these statements before the final `return db, nil` in the `Open` function at the end of `database.go`:

```go
	// User profile identity fields
	db.Exec("ALTER TABLE users ADD COLUMN IF NOT EXISTS display_name TEXT NOT NULL DEFAULT ''")
	db.Exec("ALTER TABLE users ADD COLUMN IF NOT EXISTS avatar_url TEXT NOT NULL DEFAULT ''")
	db.Exec("ALTER TABLE users ADD COLUMN IF NOT EXISTS timezone TEXT NOT NULL DEFAULT 'UTC+0'")
	db.Exec("ALTER TABLE users ADD COLUMN IF NOT EXISTS home_location TEXT NOT NULL DEFAULT ''")
	db.Exec("ALTER TABLE users ADD COLUMN IF NOT EXISTS home_lat DOUBLE PRECISION")
	db.Exec("ALTER TABLE users ADD COLUMN IF NOT EXISTS home_lon DOUBLE PRECISION")
	db.Exec("ALTER TABLE users ADD COLUMN IF NOT EXISTS profile_updated_at TIMESTAMPTZ")

	// Rich user interests
	db.Exec(`CREATE TABLE IF NOT EXISTS user_interests (
		id            TEXT PRIMARY KEY,
		user_id       TEXT NOT NULL REFERENCES users(id),
		category      TEXT NOT NULL,
		topic         TEXT NOT NULL,
		source        TEXT NOT NULL CHECK (source IN ('user', 'inferred')),
		confidence    DOUBLE PRECISION NOT NULL DEFAULT 1.0,
		dismissed     BOOLEAN NOT NULL DEFAULT FALSE,
		paused_until  TIMESTAMPTZ,
		last_asked_at TIMESTAMPTZ,
		times_asked   INT NOT NULL DEFAULT 0,
		created_at    TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at    TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`)
	db.Exec("CREATE INDEX IF NOT EXISTS idx_user_interests_user ON user_interests(user_id)")

	// User lifestyle tags
	db.Exec(`CREATE TABLE IF NOT EXISTS user_lifestyle_tags (
		id           TEXT PRIMARY KEY,
		user_id      TEXT NOT NULL REFERENCES users(id),
		tag_category TEXT NOT NULL,
		tag_value    TEXT NOT NULL,
		created_at   TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(user_id, tag_category, tag_value)
	)`)
	db.Exec("CREATE INDEX IF NOT EXISTS idx_user_lifestyle_user ON user_lifestyle_tags(user_id)")

	// User content preferences
	db.Exec(`CREATE TABLE IF NOT EXISTS user_content_prefs (
		id          TEXT PRIMARY KEY,
		user_id     TEXT NOT NULL REFERENCES users(id),
		category    TEXT,
		depth       TEXT NOT NULL DEFAULT 'standard',
		tone        TEXT NOT NULL DEFAULT 'casual',
		max_per_day INT,
		updated_at  TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(user_id, category)
	)`)
	db.Exec("CREATE INDEX IF NOT EXISTS idx_user_content_prefs_user ON user_content_prefs(user_id)")
```

- [ ] **Step 2: Verify migration runs**

Run: `cd backend && go run ./cmd/server`

Expected: Server starts, no migration errors. Check logs for any SQL errors. Ctrl+C to stop.

- [ ] **Step 3: Verify tables exist**

Run: `psql beepbopboop -c "\dt user_*"`

Expected: `user_interests`, `user_lifestyle_tags`, `user_content_prefs` tables listed alongside existing `user_settings`.

- [ ] **Step 4: Verify columns exist**

Run: `psql beepbopboop -c "\d users" | grep -E "display_name|avatar_url|timezone|home_location|home_lat|home_lon|profile_updated_at"`

Expected: All 7 new columns visible.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/database/database.go
git commit -m "feat(db): add user profile tables and columns

Add identity fields to users table (display_name, avatar_url, timezone,
home_location, home_lat/lon, profile_updated_at). Create user_interests,
user_lifestyle_tags, and user_content_prefs tables."
```

---

## Task 2: Profile Models

**Files:**
- Create: `backend/internal/model/profile.go`
- Modify: `backend/internal/model/model.go:8-12`

- [ ] **Step 1: Create profile models file**

Create `backend/internal/model/profile.go`:

```go
package model

import "time"

// UserInterest represents a user's declared or inferred interest.
type UserInterest struct {
	ID          string     `json:"id"`
	UserID      string     `json:"user_id"`
	Category    string     `json:"category"`
	Topic       string     `json:"topic"`
	Source      string     `json:"source"`
	Confidence  float64    `json:"confidence"`
	Dismissed   bool       `json:"-"`
	PausedUntil *time.Time `json:"paused_until"`
	LastAskedAt *time.Time `json:"-"`
	TimesAsked  int        `json:"-"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// LifestyleTag represents a structured lifestyle attribute.
type LifestyleTag struct {
	ID       string `json:"id,omitempty"`
	Category string `json:"category"`
	Value    string `json:"value"`
}

// ContentPref represents content delivery preferences, global or per-category.
type ContentPref struct {
	ID        string  `json:"id,omitempty"`
	Category  *string `json:"category"`
	Depth     string  `json:"depth"`
	Tone      string  `json:"tone"`
	MaxPerDay *int    `json:"max_per_day"`
}

// UserProfileIdentity holds the identity portion of a user profile.
type UserProfileIdentity struct {
	DisplayName  string   `json:"display_name"`
	AvatarURL    string   `json:"avatar_url"`
	Timezone     string   `json:"timezone"`
	HomeLocation string   `json:"home_location"`
	HomeLat      *float64 `json:"home_lat"`
	HomeLon      *float64 `json:"home_lon"`
}

// UserProfile is the full profile response returned by GET /user/profile.
type UserProfile struct {
	Identity           UserProfileIdentity `json:"identity"`
	Interests          []UserInterest      `json:"interests"`
	Lifestyle          []LifestyleTag      `json:"lifestyle"`
	ContentPrefs       []ContentPref       `json:"content_prefs"`
	ProfileInitialized bool                `json:"profile_initialized"`
}
```

- [ ] **Step 2: Extend User struct**

In `backend/internal/model/model.go`, replace the `User` struct (lines 8-12):

```go
type User struct {
	ID               string     `json:"id"`
	FirebaseUID      string     `json:"firebase_uid"`
	DisplayName      string     `json:"display_name"`
	AvatarURL        string     `json:"avatar_url"`
	Timezone         string     `json:"timezone"`
	HomeLocation     string     `json:"home_location"`
	HomeLat          *float64   `json:"home_lat,omitempty"`
	HomeLon          *float64   `json:"home_lon,omitempty"`
	ProfileUpdatedAt *time.Time `json:"profile_updated_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
}
```

- [ ] **Step 3: Verify compilation**

Run: `cd backend && go build ./...`

Expected: Compilation errors in `user_repo.go` (Scan calls don't match new fields). This is expected — we'll fix in Task 3.

- [ ] **Step 4: Commit**

```bash
git add backend/internal/model/profile.go backend/internal/model/model.go
git commit -m "feat(model): add UserInterest, LifestyleTag, ContentPref, UserProfile models

Extend User struct with identity fields. Create profile.go with rich
interest objects, lifestyle tags, content prefs, and composite profile."
```

---

## Task 3: User Repo — Profile Field Support

**Files:**
- Modify: `backend/internal/repository/user_repo.go`

- [ ] **Step 1: Write the failing test**

Create `backend/internal/repository/user_repo_profile_test.go`:

```go
package repository_test

import (
	"testing"

	"github.com/shanegleeson/beepbopboop/backend/internal/database"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

func TestUpdateProfile(t *testing.T) {
	db := database.OpenTestDB(t)
	repo := repository.NewUserRepo(db)

	user, err := repo.FindOrCreateByFirebaseUID("firebase-profile-test")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	lat := 37.77
	lon := -122.42
	err = repo.UpdateProfile(user.ID, "Shane", "", "UTC-7", "San Francisco", &lat, &lon)
	if err != nil {
		t.Fatalf("update profile: %v", err)
	}

	updated, err := repo.FindOrCreateByFirebaseUID("firebase-profile-test")
	if err != nil {
		t.Fatalf("refetch user: %v", err)
	}

	if updated.DisplayName != "Shane" {
		t.Errorf("display_name = %q, want %q", updated.DisplayName, "Shane")
	}
	if updated.Timezone != "UTC-7" {
		t.Errorf("timezone = %q, want %q", updated.Timezone, "UTC-7")
	}
	if updated.HomeLocation != "San Francisco" {
		t.Errorf("home_location = %q, want %q", updated.HomeLocation, "San Francisco")
	}
	if updated.HomeLat == nil || *updated.HomeLat != 37.77 {
		t.Errorf("home_lat = %v, want 37.77", updated.HomeLat)
	}
	if updated.ProfileUpdatedAt == nil {
		t.Error("profile_updated_at should be set after update")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/repository/ -run TestUpdateProfile -v`

Expected: Compilation error — `UpdateProfile` and new User fields don't exist yet in repo.

- [ ] **Step 3: Update user_repo.go**

Replace the full content of `backend/internal/repository/user_repo.go`:

```go
package repository

import (
	"database/sql"
	"fmt"

	"github.com/shanegleeson/beepbopboop/backend/internal/model"
)

type UserRepo struct {
	db *sql.DB
}

func NewUserRepo(db *sql.DB) *UserRepo {
	return &UserRepo{db: db}
}

const userColumns = `id, firebase_uid, display_name, avatar_url, timezone,
	home_location, home_lat, home_lon, profile_updated_at, created_at`

func scanUser(row interface{ Scan(...any) error }) (*model.User, error) {
	var u model.User
	err := row.Scan(
		&u.ID, &u.FirebaseUID, &u.DisplayName, &u.AvatarURL, &u.Timezone,
		&u.HomeLocation, &u.HomeLat, &u.HomeLon, &u.ProfileUpdatedAt, &u.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepo) FindOrCreateByFirebaseUID(firebaseUID string) (*model.User, error) {
	row := r.db.QueryRow(
		"SELECT "+userColumns+" FROM users WHERE firebase_uid = $1",
		firebaseUID,
	)
	u, err := scanUser(row)
	if err == nil {
		return u, nil
	}
	if err != sql.ErrNoRows {
		return nil, fmt.Errorf("query user by firebase_uid: %w", err)
	}

	id, err := generateID()
	if err != nil {
		return nil, fmt.Errorf("generate id: %w", err)
	}
	_, err = r.db.Exec(
		"INSERT INTO users (id, firebase_uid) VALUES ($1, $2)",
		id, firebaseUID,
	)
	if err != nil {
		return nil, fmt.Errorf("insert user: %w", err)
	}
	return r.FindOrCreateByFirebaseUID(firebaseUID)
}

func (r *UserRepo) GetByID(id string) (*model.User, error) {
	row := r.db.QueryRow(
		"SELECT "+userColumns+" FROM users WHERE id = $1", id,
	)
	return scanUser(row)
}

func (r *UserRepo) UpdateProfile(userID, displayName, avatarURL, timezone, homeLocation string, homeLat, homeLon *float64) error {
	_, err := r.db.Exec(`
		UPDATE users SET
			display_name = $2,
			avatar_url = $3,
			timezone = $4,
			home_location = $5,
			home_lat = $6,
			home_lon = $7,
			profile_updated_at = CURRENT_TIMESTAMP
		WHERE id = $1`,
		userID, displayName, avatarURL, timezone, homeLocation, homeLat, homeLon,
	)
	if err != nil {
		return fmt.Errorf("update profile: %w", err)
	}
	return nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend && go test ./internal/repository/ -run TestUpdateProfile -v`

Expected: PASS

- [ ] **Step 5: Run all repository tests to check for regressions**

Run: `cd backend && go test ./internal/repository/ -v`

Expected: All tests pass. If any fail due to the new Scan columns, the tests need the same column list — fix by using `scanUser` or updating test queries.

- [ ] **Step 6: Commit**

```bash
git add backend/internal/repository/user_repo.go backend/internal/repository/user_repo_profile_test.go
git commit -m "feat(repo): add UpdateProfile and extend user queries with profile fields"
```

---

## Task 4: User Interest Repository

**Files:**
- Create: `backend/internal/repository/user_interest_repo.go`

- [ ] **Step 1: Write the failing test**

Create `backend/internal/repository/user_interest_repo_test.go`:

```go
package repository_test

import (
	"testing"

	"github.com/shanegleeson/beepbopboop/backend/internal/database"
	"github.com/shanegleeson/beepbopboop/backend/internal/model"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

func TestUserInterestRepo_BulkSetAndList(t *testing.T) {
	db := database.OpenTestDB(t)
	userRepo := repository.NewUserRepo(db)
	interestRepo := repository.NewUserInterestRepo(db)

	user, _ := userRepo.FindOrCreateByFirebaseUID("firebase-interest-test")

	interests := []model.UserInterest{
		{Category: "sports", Topic: "NBA", Source: "user", Confidence: 1.0},
		{Category: "food", Topic: "ramen", Source: "user", Confidence: 1.0},
	}

	err := interestRepo.BulkSetUser(user.ID, interests)
	if err != nil {
		t.Fatalf("BulkSetUser: %v", err)
	}

	got, err := interestRepo.ListActive(user.ID)
	if err != nil {
		t.Fatalf("ListActive: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d interests, want 2", len(got))
	}
	if got[0].Category != "food" && got[0].Category != "sports" {
		t.Errorf("unexpected category: %q", got[0].Category)
	}
}

func TestUserInterestRepo_BulkSetReplacesExisting(t *testing.T) {
	db := database.OpenTestDB(t)
	userRepo := repository.NewUserRepo(db)
	interestRepo := repository.NewUserInterestRepo(db)

	user, _ := userRepo.FindOrCreateByFirebaseUID("firebase-interest-replace")

	first := []model.UserInterest{
		{Category: "sports", Topic: "NBA", Source: "user", Confidence: 1.0},
		{Category: "food", Topic: "ramen", Source: "user", Confidence: 1.0},
	}
	interestRepo.BulkSetUser(user.ID, first)

	second := []model.UserInterest{
		{Category: "music", Topic: "indie rock", Source: "user", Confidence: 1.0},
	}
	interestRepo.BulkSetUser(user.ID, second)

	got, _ := interestRepo.ListActive(user.ID)
	if len(got) != 1 {
		t.Fatalf("got %d interests, want 1 (replaced)", len(got))
	}
	if got[0].Category != "music" {
		t.Errorf("category = %q, want music", got[0].Category)
	}
}

func TestUserInterestRepo_PauseAndDismiss(t *testing.T) {
	db := database.OpenTestDB(t)
	userRepo := repository.NewUserRepo(db)
	interestRepo := repository.NewUserInterestRepo(db)

	user, _ := userRepo.FindOrCreateByFirebaseUID("firebase-interest-pause")

	interests := []model.UserInterest{
		{Category: "sports", Topic: "NFL", Source: "user", Confidence: 1.0},
	}
	interestRepo.BulkSetUser(user.ID, interests)

	all, _ := interestRepo.ListActive(user.ID)
	if len(all) != 1 {
		t.Fatalf("setup: got %d, want 1", len(all))
	}

	// Pause for 120 days
	err := interestRepo.Pause(all[0].ID, 120)
	if err != nil {
		t.Fatalf("Pause: %v", err)
	}

	// Should be excluded from active list
	active, _ := interestRepo.ListActive(user.ID)
	if len(active) != 0 {
		t.Errorf("paused interest should be excluded from active, got %d", len(active))
	}

	// ListAll should include it
	allInc, _ := interestRepo.ListAll(user.ID)
	if len(allInc) != 1 {
		t.Fatalf("ListAll should include paused, got %d", len(allInc))
	}
	if allInc[0].PausedUntil == nil {
		t.Error("PausedUntil should be set")
	}
}

func TestUserInterestRepo_InferredPreservation(t *testing.T) {
	db := database.OpenTestDB(t)
	userRepo := repository.NewUserRepo(db)
	interestRepo := repository.NewUserInterestRepo(db)

	user, _ := userRepo.FindOrCreateByFirebaseUID("firebase-interest-inferred")

	// Add an inferred interest directly
	err := interestRepo.UpsertInferred(user.ID, "travel", "Japan", 0.8)
	if err != nil {
		t.Fatalf("UpsertInferred: %v", err)
	}

	// BulkSetUser (user-declared) should NOT remove inferred ones
	userInterests := []model.UserInterest{
		{Category: "sports", Topic: "NBA", Source: "user", Confidence: 1.0},
	}
	interestRepo.BulkSetUser(user.ID, userInterests)

	all, _ := interestRepo.ListActive(user.ID)
	if len(all) != 2 {
		t.Fatalf("got %d interests, want 2 (1 user + 1 inferred)", len(all))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/repository/ -run TestUserInterestRepo -v`

Expected: Compilation error — `UserInterestRepo` doesn't exist yet.

- [ ] **Step 3: Implement user_interest_repo.go**

Create `backend/internal/repository/user_interest_repo.go`:

```go
package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/shanegleeson/beepbopboop/backend/internal/model"
)

type UserInterestRepo struct {
	db *sql.DB
}

func NewUserInterestRepo(db *sql.DB) *UserInterestRepo {
	return &UserInterestRepo{db: db}
}

// BulkSetUser replaces all source='user' interests for the given user.
// Does NOT touch source='inferred' rows.
func (r *UserInterestRepo) BulkSetUser(userID string, interests []model.UserInterest) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.Exec("DELETE FROM user_interests WHERE user_id = $1 AND source = 'user'", userID)
	if err != nil {
		return fmt.Errorf("delete existing user interests: %w", err)
	}

	for _, i := range interests {
		id, err := generateID()
		if err != nil {
			return fmt.Errorf("generate id: %w", err)
		}
		_, err = tx.Exec(`
			INSERT INTO user_interests (id, user_id, category, topic, source, confidence)
			VALUES ($1, $2, $3, $4, 'user', $5)`,
			id, userID, i.Category, i.Topic, i.Confidence,
		)
		if err != nil {
			return fmt.Errorf("insert interest: %w", err)
		}
	}

	return tx.Commit()
}

// UpsertInferred inserts or updates an inferred interest.
func (r *UserInterestRepo) UpsertInferred(userID, category, topic string, confidence float64) error {
	id, err := generateID()
	if err != nil {
		return fmt.Errorf("generate id: %w", err)
	}
	_, err = r.db.Exec(`
		INSERT INTO user_interests (id, user_id, category, topic, source, confidence)
		VALUES ($1, $2, $3, $4, 'inferred', $5)
		ON CONFLICT (user_id, category, topic) DO UPDATE SET
			confidence = EXCLUDED.confidence,
			updated_at = CURRENT_TIMESTAMP
		WHERE user_interests.source = 'inferred'`,
		id, userID, category, topic, confidence,
	)
	if err != nil {
		return fmt.Errorf("upsert inferred interest: %w", err)
	}
	return nil
}

// ListActive returns non-dismissed, non-paused interests.
func (r *UserInterestRepo) ListActive(userID string) ([]model.UserInterest, error) {
	return r.list(userID, true)
}

// ListAll returns all interests including paused and dismissed.
func (r *UserInterestRepo) ListAll(userID string) ([]model.UserInterest, error) {
	return r.list(userID, false)
}

func (r *UserInterestRepo) list(userID string, activeOnly bool) ([]model.UserInterest, error) {
	query := `SELECT id, user_id, category, topic, source, confidence, dismissed,
		paused_until, last_asked_at, times_asked, created_at, updated_at
		FROM user_interests WHERE user_id = $1`
	if activeOnly {
		query += ` AND dismissed = FALSE AND (paused_until IS NULL OR paused_until < NOW())`
	}
	query += ` ORDER BY category, topic`

	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("list interests: %w", err)
	}
	defer rows.Close()

	var result []model.UserInterest
	for rows.Next() {
		var i model.UserInterest
		err := rows.Scan(
			&i.ID, &i.UserID, &i.Category, &i.Topic, &i.Source, &i.Confidence,
			&i.Dismissed, &i.PausedUntil, &i.LastAskedAt, &i.TimesAsked,
			&i.CreatedAt, &i.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan interest: %w", err)
		}
		result = append(result, i)
	}
	return result, rows.Err()
}

// Promote changes an inferred interest to user-declared.
func (r *UserInterestRepo) Promote(id string) error {
	_, err := r.db.Exec(`
		UPDATE user_interests SET source = 'user', confidence = 1.0, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1 AND source = 'inferred'`, id)
	if err != nil {
		return fmt.Errorf("promote interest: %w", err)
	}
	return nil
}

// Dismiss marks an inferred interest as dismissed.
func (r *UserInterestRepo) Dismiss(id string) error {
	_, err := r.db.Exec(`
		UPDATE user_interests SET dismissed = TRUE, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("dismiss interest: %w", err)
	}
	return nil
}

// Pause sets paused_until to N days from now.
func (r *UserInterestRepo) Pause(id string, days int) error {
	pauseUntil := time.Now().AddDate(0, 0, days)
	_, err := r.db.Exec(`
		UPDATE user_interests SET paused_until = $2, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1`, id, pauseUntil)
	if err != nil {
		return fmt.Errorf("pause interest: %w", err)
	}
	return nil
}

// MarkAsked records that the system asked the user about a declining interest.
func (r *UserInterestRepo) MarkAsked(id string) error {
	_, err := r.db.Exec(`
		UPDATE user_interests SET last_asked_at = CURRENT_TIMESTAMP,
			times_asked = times_asked + 1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("mark asked: %w", err)
	}
	return nil
}

// Delete removes an interest permanently.
func (r *UserInterestRepo) Delete(id string) error {
	_, err := r.db.Exec("DELETE FROM user_interests WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete interest: %w", err)
	}
	return nil
}
```

**Note:** The `ON CONFLICT` in `UpsertInferred` requires a unique constraint on `(user_id, category, topic)`. Add this to `database.go` migration. This must be a regular (non-partial) unique index — PostgreSQL's `ON CONFLICT` cannot target partial indexes:

```go
db.Exec("ALTER TABLE user_interests ADD CONSTRAINT uq_user_interests_user_cat_topic UNIQUE (user_id, category, topic)")
```

- [ ] **Step 4: Add the unique index to database.go**

In `backend/internal/database/database.go`, add after the `idx_user_interests_user` index:

```go
	db.Exec("ALTER TABLE user_interests ADD CONSTRAINT IF NOT EXISTS uq_user_interests_user_cat_topic UNIQUE (user_id, category, topic)")
```

Note: PostgreSQL doesn't support `IF NOT EXISTS` on `ADD CONSTRAINT`, so wrap in an idempotent pattern:

```go
	db.Exec(`DO $$ BEGIN
		ALTER TABLE user_interests ADD CONSTRAINT uq_user_interests_user_cat_topic UNIQUE (user_id, category, topic);
	EXCEPTION WHEN duplicate_object THEN NULL; END $$`)
```

- [ ] **Step 5: Run tests**

Run: `cd backend && go test ./internal/repository/ -run TestUserInterestRepo -v`

Expected: All 4 tests pass.

- [ ] **Step 6: Commit**

```bash
git add backend/internal/repository/user_interest_repo.go backend/internal/repository/user_interest_repo_test.go backend/internal/database/database.go
git commit -m "feat(repo): add UserInterestRepo with bulk set, pause, promote, inferred upsert"
```

---

## Task 5: Lifestyle Tag & Content Prefs Repositories

**Files:**
- Create: `backend/internal/repository/user_lifestyle_repo.go`
- Create: `backend/internal/repository/user_content_prefs_repo.go`

- [ ] **Step 1: Write the failing test**

Create `backend/internal/repository/user_lifestyle_repo_test.go`:

```go
package repository_test

import (
	"testing"

	"github.com/shanegleeson/beepbopboop/backend/internal/database"
	"github.com/shanegleeson/beepbopboop/backend/internal/model"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

func TestUserLifestyleRepo_BulkSetAndList(t *testing.T) {
	db := database.OpenTestDB(t)
	userRepo := repository.NewUserRepo(db)
	lifestyleRepo := repository.NewUserLifestyleRepo(db)

	user, _ := userRepo.FindOrCreateByFirebaseUID("firebase-lifestyle-test")

	tags := []model.LifestyleTag{
		{Category: "diet", Value: "vegetarian"},
		{Category: "pets", Value: "dog_owner"},
	}

	err := lifestyleRepo.BulkSet(user.ID, tags)
	if err != nil {
		t.Fatalf("BulkSet: %v", err)
	}

	got, err := lifestyleRepo.List(user.ID)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d tags, want 2", len(got))
	}
}
```

- [ ] **Step 2: Write the content prefs test**

Create `backend/internal/repository/user_content_prefs_repo_test.go`:

```go
package repository_test

import (
	"testing"

	"github.com/shanegleeson/beepbopboop/backend/internal/database"
	"github.com/shanegleeson/beepbopboop/backend/internal/model"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

func TestUserContentPrefsRepo_SetAndList(t *testing.T) {
	db := database.OpenTestDB(t)
	userRepo := repository.NewUserRepo(db)
	prefsRepo := repository.NewUserContentPrefsRepo(db)

	user, _ := userRepo.FindOrCreateByFirebaseUID("firebase-prefs-test")

	sports := "sports"
	maxFive := 5
	prefs := []model.ContentPref{
		{Category: nil, Depth: "standard", Tone: "casual", MaxPerDay: nil},
		{Category: &sports, Depth: "detailed", Tone: "informative", MaxPerDay: &maxFive},
	}

	err := prefsRepo.BulkSet(user.ID, prefs)
	if err != nil {
		t.Fatalf("BulkSet: %v", err)
	}

	got, err := prefsRepo.List(user.ID)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d prefs, want 2", len(got))
	}
}
```

- [ ] **Step 3: Implement user_lifestyle_repo.go**

Create `backend/internal/repository/user_lifestyle_repo.go`:

```go
package repository

import (
	"database/sql"
	"fmt"

	"github.com/shanegleeson/beepbopboop/backend/internal/model"
)

type UserLifestyleRepo struct {
	db *sql.DB
}

func NewUserLifestyleRepo(db *sql.DB) *UserLifestyleRepo {
	return &UserLifestyleRepo{db: db}
}

func (r *UserLifestyleRepo) BulkSet(userID string, tags []model.LifestyleTag) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.Exec("DELETE FROM user_lifestyle_tags WHERE user_id = $1", userID)
	if err != nil {
		return fmt.Errorf("delete existing tags: %w", err)
	}

	for _, tag := range tags {
		id, err := generateID()
		if err != nil {
			return fmt.Errorf("generate id: %w", err)
		}
		_, err = tx.Exec(`
			INSERT INTO user_lifestyle_tags (id, user_id, tag_category, tag_value)
			VALUES ($1, $2, $3, $4)`,
			id, userID, tag.Category, tag.Value,
		)
		if err != nil {
			return fmt.Errorf("insert tag: %w", err)
		}
	}

	return tx.Commit()
}

func (r *UserLifestyleRepo) List(userID string) ([]model.LifestyleTag, error) {
	rows, err := r.db.Query(`
		SELECT id, tag_category, tag_value FROM user_lifestyle_tags
		WHERE user_id = $1 ORDER BY tag_category, tag_value`, userID)
	if err != nil {
		return nil, fmt.Errorf("list lifestyle tags: %w", err)
	}
	defer rows.Close()

	var result []model.LifestyleTag
	for rows.Next() {
		var tag model.LifestyleTag
		if err := rows.Scan(&tag.ID, &tag.Category, &tag.Value); err != nil {
			return nil, fmt.Errorf("scan tag: %w", err)
		}
		result = append(result, tag)
	}
	return result, rows.Err()
}
```

- [ ] **Step 4: Implement user_content_prefs_repo.go**

Create `backend/internal/repository/user_content_prefs_repo.go`:

```go
package repository

import (
	"database/sql"
	"fmt"

	"github.com/shanegleeson/beepbopboop/backend/internal/model"
)

type UserContentPrefsRepo struct {
	db *sql.DB
}

func NewUserContentPrefsRepo(db *sql.DB) *UserContentPrefsRepo {
	return &UserContentPrefsRepo{db: db}
}

func (r *UserContentPrefsRepo) BulkSet(userID string, prefs []model.ContentPref) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.Exec("DELETE FROM user_content_prefs WHERE user_id = $1", userID)
	if err != nil {
		return fmt.Errorf("delete existing prefs: %w", err)
	}

	for _, p := range prefs {
		id, err := generateID()
		if err != nil {
			return fmt.Errorf("generate id: %w", err)
		}
		_, err = tx.Exec(`
			INSERT INTO user_content_prefs (id, user_id, category, depth, tone, max_per_day)
			VALUES ($1, $2, $3, $4, $5, $6)`,
			id, userID, p.Category, p.Depth, p.Tone, p.MaxPerDay,
		)
		if err != nil {
			return fmt.Errorf("insert pref: %w", err)
		}
	}

	return tx.Commit()
}

func (r *UserContentPrefsRepo) List(userID string) ([]model.ContentPref, error) {
	rows, err := r.db.Query(`
		SELECT id, category, depth, tone, max_per_day FROM user_content_prefs
		WHERE user_id = $1 ORDER BY category NULLS FIRST`, userID)
	if err != nil {
		return nil, fmt.Errorf("list content prefs: %w", err)
	}
	defer rows.Close()

	var result []model.ContentPref
	for rows.Next() {
		var p model.ContentPref
		if err := rows.Scan(&p.ID, &p.Category, &p.Depth, &p.Tone, &p.MaxPerDay); err != nil {
			return nil, fmt.Errorf("scan pref: %w", err)
		}
		result = append(result, p)
	}
	return result, rows.Err()
}
```

- [ ] **Step 5: Run tests**

Run: `cd backend && go test ./internal/repository/ -run "TestUserLifestyleRepo|TestUserContentPrefsRepo" -v`

Expected: All tests pass.

- [ ] **Step 6: Commit**

```bash
git add backend/internal/repository/user_lifestyle_repo.go backend/internal/repository/user_lifestyle_repo_test.go backend/internal/repository/user_content_prefs_repo.go backend/internal/repository/user_content_prefs_repo_test.go
git commit -m "feat(repo): add UserLifestyleRepo and UserContentPrefsRepo with bulk set/list"
```

---

## Task 6: Profile Handler

**Files:**
- Create: `backend/internal/handler/profile.go`
- Create: `backend/internal/handler/profile_test.go`

- [ ] **Step 1: Write the failing test for GET /user/profile (Firebase auth)**

Create `backend/internal/handler/profile_test.go`:

```go
package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/shanegleeson/beepbopboop/backend/internal/database"
	"github.com/shanegleeson/beepbopboop/backend/internal/handler"
	"github.com/shanegleeson/beepbopboop/backend/internal/middleware"
	"github.com/shanegleeson/beepbopboop/backend/internal/model"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

func TestGetProfile_Empty(t *testing.T) {
	db := database.OpenTestDB(t)
	userRepo := repository.NewUserRepo(db)
	interestRepo := repository.NewUserInterestRepo(db)
	lifestyleRepo := repository.NewUserLifestyleRepo(db)
	prefsRepo := repository.NewUserContentPrefsRepo(db)
	agentRepo := repository.NewAgentRepo(db)

	h := handler.NewProfileHandler(userRepo, agentRepo, interestRepo, lifestyleRepo, prefsRepo)

	req := httptest.NewRequest("GET", "/user/profile", nil)
	ctx := middleware.WithFirebaseUID(req.Context(), "firebase-profile-empty")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	h.GetProfileFirebase(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	var profile model.UserProfile
	json.NewDecoder(w.Body).Decode(&profile)

	if profile.ProfileInitialized {
		t.Error("profile_initialized should be false for new user")
	}
	if profile.Identity.Timezone != "UTC+0" {
		t.Errorf("timezone = %q, want UTC+0", profile.Identity.Timezone)
	}
}

func TestGetProfile_WithData(t *testing.T) {
	db := database.OpenTestDB(t)
	userRepo := repository.NewUserRepo(db)
	interestRepo := repository.NewUserInterestRepo(db)
	lifestyleRepo := repository.NewUserLifestyleRepo(db)
	prefsRepo := repository.NewUserContentPrefsRepo(db)
	agentRepo := repository.NewAgentRepo(db)

	h := handler.NewProfileHandler(userRepo, agentRepo, interestRepo, lifestyleRepo, prefsRepo)

	// Setup user with profile data
	user, _ := userRepo.FindOrCreateByFirebaseUID("firebase-profile-data")
	lat := 37.77
	lon := -122.42
	userRepo.UpdateProfile(user.ID, "Shane", "", "UTC-7", "San Francisco", &lat, &lon)

	interestRepo.BulkSetUser(user.ID, []model.UserInterest{
		{Category: "sports", Topic: "NBA", Confidence: 1.0},
	})
	lifestyleRepo.BulkSet(user.ID, []model.LifestyleTag{
		{Category: "diet", Value: "vegetarian"},
	})

	req := httptest.NewRequest("GET", "/user/profile", nil)
	ctx := middleware.WithFirebaseUID(req.Context(), "firebase-profile-data")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	h.GetProfileFirebase(w, req)

	var profile model.UserProfile
	json.NewDecoder(w.Body).Decode(&profile)

	if !profile.ProfileInitialized {
		t.Error("profile_initialized should be true")
	}
	if profile.Identity.DisplayName != "Shane" {
		t.Errorf("display_name = %q, want Shane", profile.Identity.DisplayName)
	}
	if len(profile.Interests) != 1 {
		t.Errorf("got %d interests, want 1", len(profile.Interests))
	}
	if len(profile.Lifestyle) != 1 {
		t.Errorf("got %d lifestyle tags, want 1", len(profile.Lifestyle))
	}
}
```

- [ ] **Step 2: Add WithFirebaseUID test helper to middleware**

Check if `middleware.WithFirebaseUID` exists. If not, add to `backend/internal/middleware/firebase_auth.go`:

```go
// WithFirebaseUID sets the firebase UID in context (for tests).
func WithFirebaseUID(ctx context.Context, uid string) context.Context {
	return context.WithValue(ctx, firebaseUIDKey, uid)
}
```

- [ ] **Step 3: Implement profile.go handler**

Create `backend/internal/handler/profile.go`:

```go
package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/shanegleeson/beepbopboop/backend/internal/middleware"
	"github.com/shanegleeson/beepbopboop/backend/internal/model"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

type ProfileHandler struct {
	userRepo      *repository.UserRepo
	agentRepo     *repository.AgentRepo
	interestRepo  *repository.UserInterestRepo
	lifestyleRepo *repository.UserLifestyleRepo
	prefsRepo     *repository.UserContentPrefsRepo
}

func NewProfileHandler(
	userRepo *repository.UserRepo,
	agentRepo *repository.AgentRepo,
	interestRepo *repository.UserInterestRepo,
	lifestyleRepo *repository.UserLifestyleRepo,
	prefsRepo *repository.UserContentPrefsRepo,
) *ProfileHandler {
	return &ProfileHandler{
		userRepo:      userRepo,
		agentRepo:     agentRepo,
		interestRepo:  interestRepo,
		lifestyleRepo: lifestyleRepo,
		prefsRepo:     prefsRepo,
	}
}

func (h *ProfileHandler) buildProfile(userID string, includeInactive bool) (*model.UserProfile, error) {
	user, err := h.userRepo.GetByID(userID)
	if err != nil {
		return nil, err
	}

	var interests []model.UserInterest
	if includeInactive {
		interests, err = h.interestRepo.ListAll(userID)
	} else {
		interests, err = h.interestRepo.ListActive(userID)
	}
	if err != nil {
		return nil, err
	}
	if interests == nil {
		interests = []model.UserInterest{}
	}

	lifestyle, err := h.lifestyleRepo.List(userID)
	if err != nil {
		return nil, err
	}
	if lifestyle == nil {
		lifestyle = []model.LifestyleTag{}
	}

	prefs, err := h.prefsRepo.List(userID)
	if err != nil {
		return nil, err
	}
	if prefs == nil {
		prefs = []model.ContentPref{}
	}

	hasUserInterest := false
	for _, i := range interests {
		if i.Source == "user" {
			hasUserInterest = true
			break
		}
	}

	return &model.UserProfile{
		Identity: model.UserProfileIdentity{
			DisplayName:  user.DisplayName,
			AvatarURL:    user.AvatarURL,
			Timezone:     user.Timezone,
			HomeLocation: user.HomeLocation,
			HomeLat:      user.HomeLat,
			HomeLon:      user.HomeLon,
		},
		Interests:          interests,
		Lifestyle:          lifestyle,
		ContentPrefs:       prefs,
		ProfileInitialized: user.DisplayName != "" && hasUserInterest,
	}, nil
}

// GetProfileFirebase handles GET /user/profile (Firebase auth).
func (h *ProfileHandler) GetProfileFirebase(w http.ResponseWriter, r *http.Request) {
	uid := middleware.FirebaseUIDFromContext(r.Context())
	user, err := h.userRepo.FindOrCreateByFirebaseUID(uid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to resolve user"})
		return
	}

	includeInactive := r.URL.Query().Get("include_inactive") == "true"
	profile, err := h.buildProfile(user.ID, includeInactive)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to build profile"})
		return
	}

	writeJSON(w, http.StatusOK, profile)
}

// GetProfileAgent handles GET /user/profile (Agent auth).
func (h *ProfileHandler) GetProfileAgent(w http.ResponseWriter, r *http.Request) {
	agentID := middleware.AgentIDFromContext(r.Context())
	agent, err := h.agentRepo.GetByID(agentID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to resolve agent"})
		return
	}

	profile, err := h.buildProfile(agent.UserID, false)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to build profile"})
		return
	}

	writeJSON(w, http.StatusOK, profile)
}

// UpdateProfileFirebase handles PUT /user/profile (Firebase auth).
func (h *ProfileHandler) UpdateProfileFirebase(w http.ResponseWriter, r *http.Request) {
	uid := middleware.FirebaseUIDFromContext(r.Context())
	user, err := h.userRepo.FindOrCreateByFirebaseUID(uid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to resolve user"})
		return
	}

	var req struct {
		DisplayName  string   `json:"display_name"`
		AvatarURL    string   `json:"avatar_url"`
		Timezone     string   `json:"timezone"`
		HomeLocation string   `json:"home_location"`
		HomeLat      *float64 `json:"home_lat"`
		HomeLon      *float64 `json:"home_lon"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.DisplayName == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "display_name is required"})
		return
	}
	if req.Timezone != "" && !strings.HasPrefix(req.Timezone, "UTC") {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "timezone must be UTC offset (e.g. UTC-7)"})
		return
	}

	err = h.userRepo.UpdateProfile(user.ID, req.DisplayName, req.AvatarURL, req.Timezone, req.HomeLocation, req.HomeLat, req.HomeLon)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to update profile"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// SetInterests handles PUT /user/interests (Firebase auth).
func (h *ProfileHandler) SetInterests(w http.ResponseWriter, r *http.Request) {
	uid := middleware.FirebaseUIDFromContext(r.Context())
	user, err := h.userRepo.FindOrCreateByFirebaseUID(uid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to resolve user"})
		return
	}

	var req struct {
		Interests []model.UserInterest `json:"interests"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	err = h.interestRepo.BulkSetUser(user.ID, req.Interests)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to set interests"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// PromoteInterest handles POST /user/interests/{id}/promote.
func (h *ProfileHandler) PromoteInterest(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.interestRepo.Promote(id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to promote interest"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// DismissInterest handles POST /user/interests/{id}/dismiss.
func (h *ProfileHandler) DismissInterest(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.interestRepo.Dismiss(id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to dismiss interest"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// PauseInterest handles POST /user/interests/{id}/pause.
func (h *ProfileHandler) PauseInterest(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req struct {
		Days int `json:"days"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Days <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "days must be a positive integer"})
		return
	}
	if err := h.interestRepo.Pause(id, req.Days); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to pause interest"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// SetLifestyle handles PUT /user/lifestyle (Firebase auth).
func (h *ProfileHandler) SetLifestyle(w http.ResponseWriter, r *http.Request) {
	uid := middleware.FirebaseUIDFromContext(r.Context())
	user, err := h.userRepo.FindOrCreateByFirebaseUID(uid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to resolve user"})
		return
	}

	var req struct {
		Tags []model.LifestyleTag `json:"tags"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if err := h.lifestyleRepo.BulkSet(user.ID, req.Tags); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to set lifestyle tags"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// SetContentPrefs handles PUT /user/content-prefs (Firebase auth).
func (h *ProfileHandler) SetContentPrefs(w http.ResponseWriter, r *http.Request) {
	uid := middleware.FirebaseUIDFromContext(r.Context())
	user, err := h.userRepo.FindOrCreateByFirebaseUID(uid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to resolve user"})
		return
	}

	var req struct {
		Prefs []model.ContentPref `json:"prefs"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if err := h.prefsRepo.BulkSet(user.ID, req.Prefs); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to set content prefs"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
```

- [ ] **Step 4: Run tests**

Run: `cd backend && go test ./internal/handler/ -run TestGetProfile -v`

Expected: Both tests pass.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/handler/profile.go backend/internal/handler/profile_test.go backend/internal/middleware/firebase_auth.go
git commit -m "feat(handler): add ProfileHandler with GET/PUT profile, interests, lifestyle, prefs"
```

---

## Task 7: Route Registration & Server Wiring

**Files:**
- Modify: `backend/cmd/server/main.go`

- [ ] **Step 1: Add repository instantiation**

In `main.go`, after the existing repository block (around line 81), add:

```go
	interestRepo := repository.NewUserInterestRepo(db)
	lifestyleRepo := repository.NewUserLifestyleRepo(db)
	contentPrefsRepo := repository.NewUserContentPrefsRepo(db)
```

- [ ] **Step 2: Add handler instantiation**

After the existing handler block (around line 159), add:

```go
	profileH := handler.NewProfileHandler(userRepo, agentRepo, interestRepo, lifestyleRepo, contentPrefsRepo)
```

- [ ] **Step 3: Register Firebase-auth routes**

Inside the Firebase-auth group (after `r.Post("/user/interests", onboardingH.SubmitInterests)`), add:

```go
		r.Get("/user/profile", profileH.GetProfileFirebase)
		r.Put("/user/profile", profileH.UpdateProfileFirebase)
		r.Put("/user/interests/declared", profileH.SetInterests)
		r.Post("/user/interests/{id}/promote", profileH.PromoteInterest)
		r.Post("/user/interests/{id}/dismiss", profileH.DismissInterest)
		r.Post("/user/interests/{id}/pause", profileH.PauseInterest)
		r.Put("/user/lifestyle", profileH.SetLifestyle)
		r.Put("/user/content-prefs", profileH.SetContentPrefs)
```

Note: Using `/user/interests/declared` to avoid conflict with existing `POST /user/interests` (onboarding embedding endpoint).

- [ ] **Step 4: Register Agent-auth route**

Inside the Agent-auth group, add:

```go
		r.Get("/user/profile", profileH.GetProfileAgent)
```

- [ ] **Step 5: Build and verify**

Run: `cd backend && go build ./cmd/server`

Expected: Compiles with no errors.

- [ ] **Step 6: Start server and test endpoint**

Run: `cd backend && go run ./cmd/server &`

Then test:

```bash
curl -s http://localhost:8080/user/profile \
  -H "Authorization: Bearer bbp_7f57b11dc3776bdb8440d0e6f2070eee48a475bd46694a7a22da2d958b750a77" | jq .
```

Expected: JSON response with `profile_initialized: false`, empty interests/lifestyle/prefs arrays.

- [ ] **Step 7: Commit**

```bash
git add backend/cmd/server/main.go
git commit -m "feat(server): wire profile handler routes for Firebase and Agent auth"
```

---

## Task 8: Update Onboarding to Write Plaintext Interests

**Files:**
- Modify: `backend/internal/handler/onboarding.go`

- [ ] **Step 1: Write the failing test**

Add to `backend/internal/handler/onboarding_test.go`:

```go
func TestSubmitInterests_WritesPlaintext(t *testing.T) {
	db := database.OpenTestDB(t)
	userRepo := repository.NewUserRepo(db)
	interestRepo := repository.NewUserInterestRepo(db)
	userEmbRepo := repository.NewUserEmbeddingRepo(db)
	protoStore := embedding.NewPrototypeStore(db)

	// Seed post data for prototypes (needed for embedding pipeline)
	for i, label := range []string{"sports", "music"} {
		postID := fmt.Sprintf("proto-seed-%d", i)
		db.Exec(`INSERT INTO posts (id, agent_id, user_id, title, body, labels, status, display_hint)
			VALUES ($1, 'agent1', 'user1', 'test', 'body', $2, 'published', 'card')`,
			postID, fmt.Sprintf(`["%s"]`, label))
	}
	protoStore.Compute(context.Background())

	h := handler.NewOnboardingHandler(userRepo, protoStore, userEmbRepo, interestRepo)

	body := strings.NewReader(`{"interests":["Sports","Music"]}`)
	req := httptest.NewRequest("POST", "/user/interests", body)
	ctx := middleware.WithFirebaseUID(req.Context(), "firebase-onboard-plaintext")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	h.SubmitInterests(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body: %s", w.Code, w.Body.String())
	}

	user, _ := userRepo.FindOrCreateByFirebaseUID("firebase-onboard-plaintext")
	interests, err := interestRepo.ListActive(user.ID)
	if err != nil {
		t.Fatalf("list interests: %v", err)
	}
	if len(interests) < 2 {
		t.Errorf("got %d plaintext interests, want at least 2", len(interests))
	}
}
```

- [ ] **Step 2: Modify onboarding.go**

Add `interestRepo` to `OnboardingHandler` and update `SubmitInterests` to also write plaintext:

In the struct, add the field:

```go
type OnboardingHandler struct {
	userRepo     *repository.UserRepo
	prototypes   *embedding.PrototypeStore
	userEmbRepo  *repository.UserEmbeddingRepo
	interestRepo *repository.UserInterestRepo
}
```

Update the constructor to accept `interestRepo`.

In `SubmitInterests`, after the embedding upsert succeeds, add:

```go
	// Write plaintext interests for profile display and skill access
	var userInterests []model.UserInterest
	for _, name := range req.Interests {
		userInterests = append(userInterests, model.UserInterest{
			Category: name,
			Topic:    name,
			Source:   "user",
			Confidence: 1.0,
		})
	}
	if err := h.interestRepo.BulkSetUser(user.ID, userInterests); err != nil {
		slog.Warn("failed to write plaintext interests", "error", err)
		// Non-fatal — embedding was already written
	}
```

- [ ] **Step 3: Update main.go constructor call**

In `cmd/server/main.go`, update the `NewOnboardingHandler` call to pass `interestRepo`:

```go
	onboardingH := handler.NewOnboardingHandler(userRepo, prototypeStore, userEmbeddingRepo, interestRepo)
```

- [ ] **Step 4: Build and test**

Run: `cd backend && go build ./cmd/server && go test ./internal/handler/ -v`

Expected: All tests pass, compiles cleanly.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/handler/onboarding.go backend/cmd/server/main.go
git commit -m "feat(onboarding): write plaintext interests alongside embeddings"
```

---

## Task 9: iOS Profile Model

**Files:**
- Create: `beepbopboop/beepbopboop/Models/UserProfile.swift`

- [ ] **Step 1: Create the UserProfile model**

Create `beepbopboop/beepbopboop/Models/UserProfile.swift`:

```swift
import Foundation

struct UserProfileIdentity: Codable {
    var displayName: String
    var avatarUrl: String
    var timezone: String
    var homeLocation: String
    var homeLat: Double?
    var homeLon: Double?

    enum CodingKeys: String, CodingKey {
        case displayName = "display_name"
        case avatarUrl = "avatar_url"
        case timezone
        case homeLocation = "home_location"
        case homeLat = "home_lat"
        case homeLon = "home_lon"
    }
}

struct UserInterest: Codable, Identifiable {
    let id: String
    var category: String
    var topic: String
    var source: String
    var confidence: Double
    var pausedUntil: String?

    enum CodingKeys: String, CodingKey {
        case id, category, topic, source, confidence
        case pausedUntil = "paused_until"
    }
}

struct LifestyleTag: Codable {
    var category: String
    var value: String
}

struct ContentPref: Codable {
    var category: String?
    var depth: String
    var tone: String
    var maxPerDay: Int?

    enum CodingKeys: String, CodingKey {
        case category, depth, tone
        case maxPerDay = "max_per_day"
    }
}

struct UserProfile: Codable {
    var identity: UserProfileIdentity
    var interests: [UserInterest]
    var lifestyle: [LifestyleTag]
    var contentPrefs: [ContentPref]
    var profileInitialized: Bool

    enum CodingKeys: String, CodingKey {
        case identity, interests, lifestyle
        case contentPrefs = "content_prefs"
        case profileInitialized = "profile_initialized"
    }
}
```

- [ ] **Step 2: Build to verify**

Run: `python3 scripts/build_and_test.py --project beepbopboop/beepbopboop.xcodeproj --scheme beepbopboop`

Expected: 0 errors.

- [ ] **Step 3: Commit**

```bash
git add beepbopboop/beepbopboop/Models/UserProfile.swift
git commit -m "feat(ios): add UserProfile, UserInterest, LifestyleTag, ContentPref models"
```

---

## Task 10: iOS API Service — Profile Methods

**Files:**
- Modify: `beepbopboop/beepbopboop/Services/APIService.swift`

- [ ] **Step 1: Add profile fetch method**

Add to `APIService.swift`:

```swift
    func getProfile() async throws -> UserProfile {
        let url = URL(string: "\(baseURL)/user/profile")!
        var request = URLRequest(url: url)
        request.setValue("Bearer \(authToken)", forHTTPHeaderField: "Authorization")

        let (data, response) = try await URLSession.shared.data(for: request)
        guard let httpResponse = response as? HTTPURLResponse,
              (200...299).contains(httpResponse.statusCode) else {
            throw APIError.serverError
        }

        return try JSONDecoder().decode(UserProfile.self, from: data)
    }

    func updateProfile(identity: UserProfileIdentity) async throws {
        let url = URL(string: "\(baseURL)/user/profile")!
        var request = URLRequest(url: url)
        request.httpMethod = "PUT"
        request.setValue("Bearer \(authToken)", forHTTPHeaderField: "Authorization")
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        request.httpBody = try JSONEncoder().encode(identity)

        let (_, response) = try await URLSession.shared.data(for: request)
        guard let httpResponse = response as? HTTPURLResponse,
              (200...299).contains(httpResponse.statusCode) else {
            throw APIError.serverError
        }
    }

    func setInterests(_ interests: [UserInterest]) async throws {
        let url = URL(string: "\(baseURL)/user/interests/declared")!
        var request = URLRequest(url: url)
        request.httpMethod = "PUT"
        request.setValue("Bearer \(authToken)", forHTTPHeaderField: "Authorization")
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")

        let body = ["interests": interests]
        request.httpBody = try JSONEncoder().encode(body)

        let (_, response) = try await URLSession.shared.data(for: request)
        guard let httpResponse = response as? HTTPURLResponse,
              (200...299).contains(httpResponse.statusCode) else {
            throw APIError.serverError
        }
    }

    func setLifestyle(_ tags: [LifestyleTag]) async throws {
        let url = URL(string: "\(baseURL)/user/lifestyle")!
        var request = URLRequest(url: url)
        request.httpMethod = "PUT"
        request.setValue("Bearer \(authToken)", forHTTPHeaderField: "Authorization")
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")

        let body = ["tags": tags]
        request.httpBody = try JSONEncoder().encode(body)

        let (_, response) = try await URLSession.shared.data(for: request)
        guard let httpResponse = response as? HTTPURLResponse,
              (200...299).contains(httpResponse.statusCode) else {
            throw APIError.serverError
        }
    }

    func setContentPrefs(_ prefs: [ContentPref]) async throws {
        let url = URL(string: "\(baseURL)/user/content-prefs")!
        var request = URLRequest(url: url)
        request.httpMethod = "PUT"
        request.setValue("Bearer \(authToken)", forHTTPHeaderField: "Authorization")
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")

        let body = ["prefs": prefs]
        request.httpBody = try JSONEncoder().encode(body)

        let (_, response) = try await URLSession.shared.data(for: request)
        guard let httpResponse = response as? HTTPURLResponse,
              (200...299).contains(httpResponse.statusCode) else {
            throw APIError.serverError
        }
    }
```

- [ ] **Step 2: Build to verify**

Run: `python3 scripts/build_and_test.py --project beepbopboop/beepbopboop.xcodeproj --scheme beepbopboop`

Expected: 0 errors.

- [ ] **Step 3: Commit**

```bash
git add beepbopboop/beepbopboop/Services/APIService.swift
git commit -m "feat(ios): add profile API methods (get, update, interests, lifestyle, prefs)"
```

---

## Task 11: iOS Onboarding Flow

**Files:**
- Create: `beepbopboop/beepbopboop/Views/Onboarding/OnboardingView.swift`
- Modify: `beepbopboop/beepbopboop/Services/AuthService.swift`
- Modify: `beepbopboop/beepbopboop/Views/FeedListView.swift`

This is a large task — the onboarding UI involves 7 steps. The plan provides the container view and the integration points. Each step view follows the same pattern and should be built iteratively in sub-steps.

- [ ] **Step 1: Create OnboardingView container**

Create `beepbopboop/beepbopboop/Views/Onboarding/OnboardingView.swift`:

```swift
import SwiftUI

struct OnboardingView: View {
    @EnvironmentObject var authService: AuthService
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

    let onComplete: () -> Void

    private let totalSteps = 7

    var body: some View {
        VStack(spacing: 0) {
            // Progress bar
            ProgressView(value: Double(currentStep + 1), total: Double(totalSteps))
                .tint(.accentColor)
                .padding(.horizontal)
                .padding(.top, 8)

            // Step content
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
        // Save all data via API
        Task {
            let api = APIService(baseURL: Config.backendBaseURL, authToken: authService.getToken())
            try? await api.updateProfile(identity: profile)
            try? await api.setInterests(interests)
            if !lifestyle.isEmpty {
                try? await api.setLifestyle(lifestyle)
            }

            // Build content prefs — include frequency as global max_per_day
            var finalPrefs = contentPrefs
            if let freq = targetFrequency {
                // Find or create the global (nil category) pref
                if let idx = finalPrefs.firstIndex(where: { $0.category == nil }) {
                    finalPrefs[idx].maxPerDay = freq
                } else {
                    finalPrefs.append(ContentPref(category: nil, depth: "standard", tone: "casual", maxPerDay: freq))
                }
            }
            if !finalPrefs.isEmpty {
                try? await api.setContentPrefs(finalPrefs)
            }
            onComplete()
        }
    }
}
```

- [ ] **Step 2: Create placeholder step views**

Create each step view as a placeholder that compiles. Each follows this pattern (example for Name):

Create `beepbopboop/beepbopboop/Views/Onboarding/OnboardingNameView.swift`:

```swift
import SwiftUI

struct OnboardingNameView: View {
    @Binding var profile: UserProfileIdentity
    let onNext: () -> Void

    var body: some View {
        VStack(spacing: 24) {
            Spacer()
            Text("What should we call you?")
                .font(.system(size: 28, weight: .bold, design: .serif))
            TextField("Your name", text: $profile.displayName)
                .textFieldStyle(.roundedBorder)
                .padding(.horizontal, 40)
            Spacer()
            Button("Continue") { onNext() }
                .disabled(profile.displayName.isEmpty)
                .buttonStyle(.borderedProminent)
                .padding(.bottom, 40)
        }
    }
}
```

Create `beepbopboop/beepbopboop/Views/Onboarding/OnboardingLocationView.swift`:

```swift
import SwiftUI
import CoreLocation

struct OnboardingLocationView: View {
    @Binding var profile: UserProfileIdentity
    let onNext: () -> Void
    @StateObject private var locationManager = LocationHelper()

    var body: some View {
        VStack(spacing: 24) {
            Spacer()
            Text("Where are you based?")
                .font(.system(size: 28, weight: .bold, design: .serif))
            TextField("City or neighborhood", text: $profile.homeLocation)
                .textFieldStyle(.roundedBorder)
                .padding(.horizontal, 40)
            Text("Timezone: \(profile.timezone)")
                .font(.system(size: 13, design: .monospaced))
                .foregroundStyle(.secondary)
            Button("Use my location") {
                locationManager.requestLocation { lat, lon, name, tz in
                    profile.homeLat = lat
                    profile.homeLon = lon
                    if let name { profile.homeLocation = name }
                    if let tz { profile.timezone = tz }
                }
            }
            .buttonStyle(.bordered)
            Spacer()
            Button("Continue") { onNext() }
                .buttonStyle(.borderedProminent)
                .padding(.bottom, 40)
        }
        .onAppear {
            let tz = TimeZone.current
            let seconds = tz.secondsFromGMT()
            let hours = seconds / 3600
            let mins = abs(seconds % 3600) / 60
            if mins == 0 {
                profile.timezone = "UTC\(hours >= 0 ? "+" : "")\(hours)"
            } else {
                profile.timezone = "UTC\(hours >= 0 ? "+" : "")\(hours):\(String(format: "%02d", mins))"
            }
        }
    }
}

class LocationHelper: NSObject, ObservableObject, CLLocationManagerDelegate {
    private let manager = CLLocationManager()
    private var completion: ((Double, Double, String?, String?) -> Void)?

    func requestLocation(completion: @escaping (Double, Double, String?, String?) -> Void) {
        self.completion = completion
        manager.delegate = self
        manager.requestWhenInUseAuthorization()
        manager.requestLocation()
    }

    func locationManager(_ manager: CLLocationManager, didUpdateLocations locations: [CLLocation]) {
        guard let loc = locations.first else { return }
        let geocoder = CLGeocoder()
        geocoder.reverseGeocodeLocation(loc) { placemarks, _ in
            let name = placemarks?.first?.locality
            let tz = placemarks?.first?.timeZone
            var tzString: String? = nil
            if let tz {
                let s = tz.secondsFromGMT()
                let h = s / 3600
                let m = abs(s % 3600) / 60
                tzString = m == 0 ? "UTC\(h >= 0 ? "+" : "")\(h)" : "UTC\(h >= 0 ? "+" : "")\(h):\(String(format: "%02d", m))"
            }
            self.completion?(loc.coordinate.latitude, loc.coordinate.longitude, name, tzString)
        }
    }

    func locationManager(_ manager: CLLocationManager, didFailWithError error: Error) {}
}
```

Create `beepbopboop/beepbopboop/Views/Onboarding/OnboardingNotificationsView.swift`:

```swift
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
```

Create `beepbopboop/beepbopboop/Views/Onboarding/OnboardingInterestsView.swift`:

```swift
import SwiftUI

struct InterestCategory: Identifiable {
    let id: String
    let name: String
    let icon: String
    let topics: [String]
    let previewHints: [String] // display_hints to show in carousel
}

private let categories: [InterestCategory] = [
    InterestCategory(id: "sports", name: "Sports", icon: "sportscourt", topics: ["NBA", "NFL", "MLB", "Premier League", "MLS"], previewHints: ["scoreboard", "matchup", "player_spotlight"]),
    InterestCategory(id: "food", name: "Food", icon: "fork.knife", topics: ["Ramen", "Italian", "Vegan", "Coffee", "Bakeries"], previewHints: ["restaurant", "deal"]),
    InterestCategory(id: "music", name: "Music", icon: "music.note", topics: ["Indie Rock", "Hip Hop", "Electronic", "Jazz", "Classical"], previewHints: ["album", "concert"]),
    InterestCategory(id: "science", name: "Science", icon: "atom", topics: ["Space", "Biology", "Climate", "AI", "Physics"], previewHints: ["science"]),
    InterestCategory(id: "travel", name: "Travel", icon: "airplane", topics: ["Europe", "Asia", "Budget", "Road Trips", "Adventure"], previewHints: ["destination"]),
    InterestCategory(id: "fitness", name: "Fitness", icon: "figure.run", topics: ["Running", "Cycling", "Yoga", "Gym", "Swimming"], previewHints: ["fitness"]),
    InterestCategory(id: "pets", name: "Pets", icon: "pawprint", topics: ["Dogs", "Cats", "Adoption", "Training"], previewHints: ["pet_spotlight"]),
    InterestCategory(id: "fashion", name: "Fashion", icon: "tshirt", topics: ["Streetwear", "Minimalist", "Vintage", "Sustainable"], previewHints: ["outfit"]),
    InterestCategory(id: "entertainment", name: "Entertainment", icon: "film", topics: ["Movies", "TV Shows", "Podcasts", "Gaming"], previewHints: ["movie", "show"]),
    InterestCategory(id: "tech", name: "Tech", icon: "desktopcomputer", topics: ["AI", "Startups", "Open Source", "Gadgets"], previewHints: ["article"]),
]

struct OnboardingInterestsView: View {
    @Binding var interests: [UserInterest]
    let onNext: () -> Void
    @State private var selectedCategories: Set<String> = []
    @State private var expandedCategory: String? = nil

    var body: some View {
        ScrollView {
            VStack(spacing: 24) {
                Text("What interests you?")
                    .font(.system(size: 28, weight: .bold, design: .serif))
                    .padding(.top, 20)
                Text("Pick at least 3. Tap to preview what you'll see.")
                    .font(.system(size: 15))
                    .foregroundStyle(.secondary)

                LazyVGrid(columns: [GridItem(.adaptive(minimum: 100), spacing: 12)], spacing: 12) {
                    ForEach(categories) { cat in
                        Button {
                            if expandedCategory == cat.id {
                                expandedCategory = nil
                            } else {
                                expandedCategory = cat.id
                                selectedCategories.insert(cat.id)
                            }
                        } label: {
                            VStack(spacing: 6) {
                                Image(systemName: cat.icon)
                                    .font(.title2)
                                Text(cat.name)
                                    .font(.system(size: 13, weight: .medium))
                            }
                            .frame(maxWidth: .infinity)
                            .padding(.vertical, 16)
                            .background(selectedCategories.contains(cat.id) ? Color.accentColor.opacity(0.15) : Color(.systemGray6))
                            .clipShape(RoundedRectangle(cornerRadius: 12))
                            .overlay(
                                RoundedRectangle(cornerRadius: 12)
                                    .stroke(selectedCategories.contains(cat.id) ? Color.accentColor : .clear, lineWidth: 2)
                            )
                        }
                        .buttonStyle(.plain)
                    }
                }
                .padding(.horizontal)

                // Card preview carousel for expanded category
                if let catID = expandedCategory, let cat = categories.first(where: { $0.id == catID }) {
                    VStack(alignment: .leading, spacing: 8) {
                        Text("\(cat.name) cards you'll see:")
                            .font(.system(size: 13, weight: .medium, design: .monospaced))
                            .foregroundStyle(.secondary)
                            .padding(.horizontal)

                        ScrollView(.horizontal, showsIndicators: false) {
                            HStack(spacing: 12) {
                                ForEach(cat.previewHints, id: \.self) { hint in
                                    RoundedRectangle(cornerRadius: 16)
                                        .fill(Color(.systemGray5))
                                        .frame(width: 240, height: 160)
                                        .overlay(
                                            Text(hint.replacingOccurrences(of: "_", with: " ").capitalized)
                                                .font(.system(size: 15, weight: .semibold, design: .serif))
                                        )
                                }
                            }
                            .padding(.horizontal)
                        }

                        // Topic chips within the category
                        FlowLayout(spacing: 8) {
                            ForEach(cat.topics, id: \.self) { topic in
                                let isSelected = interests.contains(where: { $0.category == cat.id && $0.topic == topic })
                                Button {
                                    if isSelected {
                                        interests.removeAll { $0.category == cat.id && $0.topic == topic }
                                    } else {
                                        interests.append(UserInterest(id: UUID().uuidString, category: cat.id, topic: topic, source: "user", confidence: 1.0, pausedUntil: nil))
                                    }
                                } label: {
                                    Text(topic)
                                        .font(.system(size: 13))
                                        .padding(.horizontal, 12)
                                        .padding(.vertical, 6)
                                        .background(isSelected ? Color.accentColor.opacity(0.2) : Color(.systemGray6))
                                        .clipShape(Capsule())
                                }
                                .buttonStyle(.plain)
                            }
                        }
                        .padding(.horizontal)
                    }
                }

                Button("Continue") {
                    // Ensure at least the category-level interests are added
                    for catID in selectedCategories {
                        if !interests.contains(where: { $0.category == catID }) {
                            interests.append(UserInterest(id: UUID().uuidString, category: catID, topic: catID, source: "user", confidence: 1.0, pausedUntil: nil))
                        }
                    }
                    onNext()
                }
                .disabled(selectedCategories.count < 3)
                .buttonStyle(.borderedProminent)
                .padding(.bottom, 40)
            }
        }
    }
}

// Simple flow layout for topic chips
struct FlowLayout: Layout {
    var spacing: CGFloat = 8
    func sizeThatFits(proposal: ProposedViewSize, subviews: Subviews, cache: inout ()) -> CGSize {
        var width: CGFloat = 0
        var height: CGFloat = 0
        var rowHeight: CGFloat = 0
        var rowWidth: CGFloat = 0
        let maxWidth = proposal.width ?? .infinity
        for sub in subviews {
            let size = sub.sizeThatFits(.unspecified)
            if rowWidth + size.width > maxWidth {
                width = max(width, rowWidth - spacing)
                height += rowHeight + spacing
                rowWidth = 0; rowHeight = 0
            }
            rowWidth += size.width + spacing
            rowHeight = max(rowHeight, size.height)
        }
        height += rowHeight
        return CGSize(width: max(width, rowWidth - spacing), height: height)
    }
    func placeSubviews(in bounds: CGRect, proposal: ProposedViewSize, subviews: Subviews, cache: inout ()) {
        var x = bounds.minX; var y = bounds.minY; var rowHeight: CGFloat = 0
        for sub in subviews {
            let size = sub.sizeThatFits(.unspecified)
            if x + size.width > bounds.maxX {
                x = bounds.minX; y += rowHeight + spacing; rowHeight = 0
            }
            sub.place(at: CGPoint(x: x, y: y), proposal: .unspecified)
            x += size.width + spacing
            rowHeight = max(rowHeight, size.height)
        }
    }
}
```

Create `beepbopboop/beepbopboop/Views/Onboarding/OnboardingFrequencyView.swift`:

```swift
import SwiftUI

struct OnboardingFrequencyView: View {
    @Binding var targetFrequency: Int?
    let onNext: () -> Void
    @State private var sliderValue: Double = 10

    private let labels = ["Light (5/day)", "Moderate (10/day)", "Full (15/day)", "Max (20/day)"]

    var body: some View {
        VStack(spacing: 24) {
            Spacer()
            Text("How much content?")
                .font(.system(size: 28, weight: .bold, design: .serif))
            Text("You can change this anytime in settings.")
                .font(.system(size: 15))
                .foregroundStyle(.secondary)
            VStack(spacing: 8) {
                Text("\(Int(sliderValue)) posts per day")
                    .font(.system(size: 20, weight: .semibold, design: .monospaced))
                Slider(value: $sliderValue, in: 3...25, step: 1)
                    .padding(.horizontal, 40)
            }
            Spacer()
            Button("Continue") {
                targetFrequency = Int(sliderValue)
                onNext()
            }
            .buttonStyle(.borderedProminent)
            .padding(.bottom, 40)
        }
    }
}
```

Create `beepbopboop/beepbopboop/Views/Onboarding/OnboardingLifestyleView.swift`:

```swift
import SwiftUI

private struct TagOption: Identifiable {
    let id: String
    let category: String
    let value: String
    let label: String
}

private let tagOptions: [TagOption] = [
    TagOption(id: "diet-veg", category: "diet", value: "vegetarian", label: "Vegetarian"),
    TagOption(id: "diet-vegan", category: "diet", value: "vegan", label: "Vegan"),
    TagOption(id: "diet-gf", category: "diet", value: "gluten_free", label: "Gluten-free"),
    TagOption(id: "diet-halal", category: "diet", value: "halal", label: "Halal"),
    TagOption(id: "diet-kosher", category: "diet", value: "kosher", label: "Kosher"),
    TagOption(id: "fit-run", category: "fitness", value: "runner", label: "Runner"),
    TagOption(id: "fit-cycle", category: "fitness", value: "cyclist", label: "Cyclist"),
    TagOption(id: "fit-gym", category: "fitness", value: "gym", label: "Gym"),
    TagOption(id: "fit-yoga", category: "fitness", value: "yoga", label: "Yoga"),
    TagOption(id: "fit-swim", category: "fitness", value: "swimmer", label: "Swimmer"),
    TagOption(id: "pet-dog", category: "pets", value: "dog_owner", label: "Dog owner"),
    TagOption(id: "pet-cat", category: "pets", value: "cat_owner", label: "Cat owner"),
    TagOption(id: "fam-parent", category: "family", value: "parent", label: "Parent"),
    TagOption(id: "fam-couple", category: "family", value: "couple", label: "Couple"),
]

struct OnboardingLifestyleView: View {
    @Binding var lifestyle: [LifestyleTag]
    let onNext: () -> Void
    @State private var selected: Set<String> = []

    var body: some View {
        ScrollView {
            VStack(spacing: 24) {
                Text("Tell us about you")
                    .font(.system(size: 28, weight: .bold, design: .serif))
                    .padding(.top, 20)
                Text("This helps personalize your feed. Skip if you prefer.")
                    .font(.system(size: 15))
                    .foregroundStyle(.secondary)

                ForEach(["diet", "fitness", "pets", "family"], id: \.self) { category in
                    VStack(alignment: .leading, spacing: 8) {
                        Text(category.capitalized)
                            .font(.system(size: 11, weight: .medium, design: .monospaced))
                            .foregroundStyle(.secondary)
                        FlowLayout(spacing: 8) {
                            ForEach(tagOptions.filter { $0.category == category }) { opt in
                                Button {
                                    if selected.contains(opt.id) {
                                        selected.remove(opt.id)
                                    } else {
                                        selected.insert(opt.id)
                                    }
                                } label: {
                                    Text(opt.label)
                                        .font(.system(size: 14))
                                        .padding(.horizontal, 14)
                                        .padding(.vertical, 8)
                                        .background(selected.contains(opt.id) ? Color.accentColor.opacity(0.2) : Color(.systemGray6))
                                        .clipShape(Capsule())
                                }
                                .buttonStyle(.plain)
                            }
                        }
                    }
                    .padding(.horizontal)
                }

                Button("Continue") {
                    lifestyle = tagOptions
                        .filter { selected.contains($0.id) }
                        .map { LifestyleTag(category: $0.category, value: $0.value) }
                    onNext()
                }
                .buttonStyle(.borderedProminent)
                .padding(.bottom, 40)
            }
        }
    }
}
```

Create `beepbopboop/beepbopboop/Views/Onboarding/OnboardingPrefsView.swift`:

```swift
import SwiftUI

struct OnboardingPrefsView: View {
    @Binding var contentPrefs: [ContentPref]
    let onComplete: () -> Void
    @State private var depth = "standard"
    @State private var tone = "casual"

    var body: some View {
        VStack(spacing: 24) {
            Spacer()
            Text("Content style")
                .font(.system(size: 28, weight: .bold, design: .serif))
            Text("How should your feed feel?")
                .font(.system(size: 15))
                .foregroundStyle(.secondary)

            VStack(alignment: .leading, spacing: 12) {
                Text("DEPTH")
                    .font(.system(size: 11, weight: .medium, design: .monospaced))
                    .foregroundStyle(.secondary)
                Picker("Depth", selection: $depth) {
                    Text("Brief").tag("brief")
                    Text("Standard").tag("standard")
                    Text("Detailed").tag("detailed")
                }
                .pickerStyle(.segmented)
            }
            .padding(.horizontal, 40)

            VStack(alignment: .leading, spacing: 12) {
                Text("TONE")
                    .font(.system(size: 11, weight: .medium, design: .monospaced))
                    .foregroundStyle(.secondary)
                Picker("Tone", selection: $tone) {
                    Text("Casual").tag("casual")
                    Text("Informative").tag("informative")
                    Text("Playful").tag("playful")
                }
                .pickerStyle(.segmented)
            }
            .padding(.horizontal, 40)

            Spacer()
            Button("Finish setup") {
                contentPrefs = [ContentPref(category: nil, depth: depth, tone: tone, maxPerDay: nil)]
                onComplete()
            }
            .buttonStyle(.borderedProminent)
            .padding(.bottom, 40)
        }
    }
}
```

- [ ] **Step 3: Add profile state to AuthService**

In `beepbopboop/beepbopboop/Services/AuthService.swift`, add:

```swift
    @Published var profileInitialized: Bool = false
    @Published var isLoadingProfile: Bool = false

    func checkProfile() async {
        isLoadingProfile = true
        defer { isLoadingProfile = false }

        let api = APIService(baseURL: Config.backendBaseURL, authToken: getToken())
        do {
            let profile = try await api.getProfile()
            profileInitialized = profile.profileInitialized
        } catch {
            profileInitialized = false
        }
    }
```

- [ ] **Step 4: Gate feed on profile initialization**

In `beepbopboop/beepbopboop/Views/FeedListView.swift`, wrap the main content:

```swift
if authService.isLoadingProfile {
    ProgressView()
} else if !authService.profileInitialized {
    OnboardingView {
        authService.profileInitialized = true
    }
} else {
    // existing feed content
}
```

And in the `.onAppear` or `.task` modifier of the feed view, add:

```swift
.task {
    if authService.isSignedIn && !authService.profileInitialized {
        await authService.checkProfile()
    }
}
```

- [ ] **Step 5: Build and verify**

Run: `python3 scripts/build_and_test.py --project beepbopboop/beepbopboop.xcodeproj --scheme beepbopboop`

Expected: 0 errors.

- [ ] **Step 6: Commit**

```bash
git add beepbopboop/beepbopboop/Views/Onboarding/ beepbopboop/beepbopboop/Services/AuthService.swift beepbopboop/beepbopboop/Views/FeedListView.swift
git commit -m "feat(ios): add onboarding flow with 7 steps, gate feed on profile_initialized"
```

---

## Task 12: Skill Bootstrap Update

**Files:**
- Modify: `.claude/skills/_shared/CONTEXT_BOOTSTRAP.md`
- Modify: `.claude/skills/beepbopboop-post/SKILL.md`
- Modify: `.claude/skills/beepbopboop-post/MODE_BATCH.md`
- Modify: `.claude/skills/beepbopboop-post/BASE_LOCAL.md`

- [ ] **Step 1: Add profile fetch to CONTEXT_BOOTSTRAP.md**

In the Step 0d section where the 4 parallel GETs are listed, add a 5th:

```bash
PROFILE=$(curl -s -H "$AUTH" "$API/user/profile")
```

Add to the "Pin into working memory" section:

```
6. User profile: display_name, timezone, home_location, active interests (category + topic),
   lifestyle tags, content_prefs (depth, tone, max_per_day per category).
   If PROFILE fetch fails or returns empty, fall back to config keys
   (BEEPBOPBOOP_INTERESTS, BEEPBOPBOOP_HOME_ADDRESS, etc.)
```

- [ ] **Step 2: Update SKILL.md**

In `SKILL.md` Step 0, add after the config load:

```
After CONTEXT_BOOTSTRAP.md Step 0d, pin the user profile:
- Use profile.identity.timezone for all time references
- Use profile.identity.home_location as default locality (overrides BEEPBOPBOOP_DEFAULT_LOCATION)
- Use profile.interests[].category + topic as primary interest signal (overrides BEEPBOPBOOP_INTERESTS)
- Use profile.lifestyle[] for content filtering
- Use profile.content_prefs[] for depth/tone shaping
```

- [ ] **Step 3: Update MODE_BATCH.md**

In BT1 (Load schedule & engagement signals), add a new sub-step BT1d:

```
**BT1d: Load user profile** — from PROFILE (pinned in Step 0d)

- **Interest-driven fill:** Use `profile.interests[]` where `source="user"` and `confidence >= 0.5`
  as the primary category list for Phase 2 fill. These REPLACE `BEEPBOPBOOP_INTERESTS`.
- **Lifestyle filtering:** Check `profile.lifestyle[]` before generating:
  - `diet=vegetarian` → food skill skips meat-focused restaurants
  - `pets=dog_owner` → pet skill prefers dog content
  - `family=parent_of_Nyo` → kid-friendly content, age-appropriate suggestions
- **Content prefs:** Read `profile.content_prefs[]`:
  - Global `max_per_day` replaces `BATCH_MIN`/`BATCH_MAX`
  - Per-category `depth` and `tone` shape each post's writing
- **Fallback:** If profile.interests is empty, fall back to `BEEPBOPBOOP_INTERESTS` config key
```

- [ ] **Step 4: Update BASE_LOCAL.md**

In Step 1 (Resolve location), add before the existing priority order:

```
0. Profile home location — if `PROFILE.identity.home_lat` and `PROFILE.identity.home_lon`
   are set, use those. Overrides `BEEPBOPBOOP_HOME_LAT`/`BEEPBOPBOOP_HOME_LON`.
```

In Step 4 (Generate post content), add:

```
Before generating, check PROFILE for contextual enrichment:
- `profile.lifestyle[]` tags inform the angle (e.g. parent → kid-friendly framing)
- `profile.content_prefs[]` depth/tone shape the writing style
- `profile.interests[]` topics add relevant cross-references
```

- [ ] **Step 5: Commit**

```bash
git add .claude/skills/_shared/CONTEXT_BOOTSTRAP.md .claude/skills/beepbopboop-post/SKILL.md .claude/skills/beepbopboop-post/MODE_BATCH.md .claude/skills/beepbopboop-post/BASE_LOCAL.md
git commit -m "feat(skills): add user profile to bootstrap, batch planning, and local post context"
```

---

## Task 13: Interest Inference Worker

**Files:**
- Create: `backend/internal/interest/worker.go`
- Create: `backend/internal/interest/worker_test.go`

- [ ] **Step 1: Write the failing test**

Create `backend/internal/interest/worker_test.go`:

```go
package interest_test

import (
	"context"
	"testing"

	"github.com/shanegleeson/beepbopboop/backend/internal/database"
	"github.com/shanegleeson/beepbopboop/backend/internal/interest"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

func TestWorker_InfersFromEngagement(t *testing.T) {
	db := database.OpenTestDB(t)
	userRepo := repository.NewUserRepo(db)
	interestRepo := repository.NewUserInterestRepo(db)

	user, _ := userRepo.FindOrCreateByFirebaseUID("firebase-infer-test")

	// Seed engagement: user saved 10 sports posts in last 30 days
	for i := 0; i < 10; i++ {
		postID, _ := repository.GenerateTestID()
		db.Exec(`INSERT INTO posts (id, agent_id, user_id, title, body, labels, status, display_hint)
			VALUES ($1, 'agent1', $2, 'test', 'body', '["sports"]', 'published', 'card')`,
			postID, user.ID)
		db.Exec(`INSERT INTO post_events (id, post_id, user_id, event_type, created_at)
			VALUES ($1, $2, $3, 'save', NOW() - INTERVAL '1 day')`,
			postID+"evt", postID, user.ID)
	}

	w := interest.NewWorker(db, interestRepo)
	err := w.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce: %v", err)
	}

	interests, _ := interestRepo.ListActive(user.ID)
	found := false
	for _, i := range interests {
		if i.Category == "sports" && i.Source == "inferred" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected inferred 'sports' interest from engagement data")
	}
}
```

- [ ] **Step 2: Implement worker.go**

Create `backend/internal/interest/worker.go`:

```go
package interest

import (
	"context"
	"database/sql"
	"log/slog"
	"time"

	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

type Worker struct {
	db           *sql.DB
	interestRepo *repository.UserInterestRepo
}

func NewWorker(db *sql.DB, interestRepo *repository.UserInterestRepo) *Worker {
	return &Worker{db: db, interestRepo: interestRepo}
}

func (w *Worker) Run(ctx context.Context, interval time.Duration) {
	slog.Info("interest inference worker started", "interval", interval)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.RunOnce(ctx); err != nil {
				slog.Warn("interest inference failed", "error", err)
			}
		}
	}
}

type labelEngagement struct {
	UserID string
	Label  string
	Saves  int
}

func (w *Worker) RunOnce(ctx context.Context) error {
	// Find labels with high save engagement per user over last 30 days
	rows, err := w.db.QueryContext(ctx, `
		SELECT pe.user_id, unnest(string_to_array(trim(both '[]"' from p.labels), '","')) AS label,
			COUNT(*) AS saves
		FROM post_events pe
		JOIN posts p ON p.id = pe.post_id
		WHERE pe.event_type = 'save'
		  AND pe.created_at > NOW() - INTERVAL '30 days'
		  AND p.labels IS NOT NULL AND p.labels != ''
		GROUP BY pe.user_id, label
		HAVING COUNT(*) >= 3
		ORDER BY saves DESC`)
	if err != nil {
		return err
	}
	defer rows.Close()

	var engagements []labelEngagement
	for rows.Next() {
		var e labelEngagement
		if err := rows.Scan(&e.UserID, &e.Label, &e.Saves); err != nil {
			continue
		}
		engagements = append(engagements, e)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for _, e := range engagements {
		confidence := float64(e.Saves) / 20.0 // 20 saves = 1.0 confidence
		if confidence > 1.0 {
			confidence = 1.0
		}
		if err := w.interestRepo.UpsertInferred(e.UserID, e.Label, e.Label, confidence); err != nil {
			slog.Warn("failed to upsert inferred interest",
				"user_id", e.UserID, "label", e.Label, "error", err)
		}
	}

	slog.Info("interest inference complete", "engagements_processed", len(engagements))
	return nil
}
```

- [ ] **Step 3: Export GenerateTestID for tests**

If `repository.GenerateTestID` doesn't exist, add to `backend/internal/repository/user_repo.go`:

```go
// GenerateTestID is exported for use in tests.
func GenerateTestID() (string, error) {
	return generateID()
}
```

- [ ] **Step 4: Run tests**

Run: `cd backend && go test ./internal/interest/ -run TestWorker -v`

Expected: PASS (may need to adjust SQL for your labels column format — labels are stored as JSON text `'["sports","nba"]'`).

- [ ] **Step 5: Wire worker in main.go**

In `cmd/server/main.go`, add import and start the worker:

```go
import "github.com/shanegleeson/beepbopboop/backend/internal/interest"
```

After the existing worker starts (around line 251):

```go
	interestWorker := interest.NewWorker(db, interestRepo)
	go interestWorker.Run(workerCtx, 24*time.Hour)
```

- [ ] **Step 6: Commit**

```bash
git add backend/internal/interest/ backend/internal/repository/user_repo.go backend/cmd/server/main.go
git commit -m "feat(worker): add interest inference worker — infers interests from 30-day engagement"
```

---

## Task 14: Run All Backend Tests

- [ ] **Step 1: Run full test suite**

Run: `cd backend && go test ./... -v`

Expected: All tests pass. Fix any failures from the User struct changes (Scan column count mismatches in other test files).

- [ ] **Step 2: Build the server**

Run: `cd backend && go build ./cmd/server`

Expected: 0 errors.

- [ ] **Step 3: Build iOS app**

Run: `python3 scripts/build_and_test.py --project beepbopboop/beepbopboop.xcodeproj --scheme beepbopboop`

Expected: 0 errors.

---

## Task 15: Interest Decay Checker

**Files:**
- Create: `backend/internal/interest/decay.go`
- Create: `backend/internal/interest/decay_test.go`

- [ ] **Step 1: Write the failing test**

Create `backend/internal/interest/decay_test.go`:

```go
package interest_test

import (
	"context"
	"testing"

	"github.com/shanegleeson/beepbopboop/backend/internal/database"
	"github.com/shanegleeson/beepbopboop/backend/internal/interest"
	"github.com/shanegleeson/beepbopboop/backend/internal/model"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

func TestDecayChecker_GeneratesFeedbackPost(t *testing.T) {
	db := database.OpenTestDB(t)
	userRepo := repository.NewUserRepo(db)
	interestRepo := repository.NewUserInterestRepo(db)
	postRepo := repository.NewPostRepo(db)

	user, _ := userRepo.FindOrCreateByFirebaseUID("firebase-decay-test")

	// Create agent for the user (needed for post creation)
	agentRepo := repository.NewAgentRepo(db)
	agent, _ := agentRepo.Create(user.ID, "Briefing")

	// Add a user-declared interest with no engagement for 30+ days
	interestRepo.BulkSetUser(user.ID, []model.UserInterest{
		{Category: "sports", Topic: "NFL", Confidence: 1.0},
	})

	// Backdate the interest to 45 days ago
	db.Exec("UPDATE user_interests SET created_at = NOW() - INTERVAL '45 days' WHERE user_id = $1", user.ID)

	// No post_events for this user/label at all (complete disengagement)

	checker := interest.NewDecayChecker(db, interestRepo, postRepo, agent.ID)
	err := checker.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce: %v", err)
	}

	// Should have generated a feedback post
	posts, _ := postRepo.ListByAgent(agent.ID, 10, "")
	found := false
	for _, p := range posts {
		if p.DisplayHint != nil && *p.DisplayHint == "feedback" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected a feedback post for the disengaged interest")
	}

	// Should have updated last_asked_at
	interests, _ := interestRepo.ListAll(user.ID)
	if len(interests) == 0 {
		t.Fatal("no interests found")
	}
	if interests[0].TimesAsked != 1 {
		t.Errorf("times_asked = %d, want 1", interests[0].TimesAsked)
	}
}

func TestDecayChecker_RespectsBackoff(t *testing.T) {
	db := database.OpenTestDB(t)
	userRepo := repository.NewUserRepo(db)
	interestRepo := repository.NewUserInterestRepo(db)
	postRepo := repository.NewPostRepo(db)
	agentRepo := repository.NewAgentRepo(db)

	user, _ := userRepo.FindOrCreateByFirebaseUID("firebase-decay-backoff")
	agent, _ := agentRepo.Create(user.ID, "Briefing")

	interestRepo.BulkSetUser(user.ID, []model.UserInterest{
		{Category: "sports", Topic: "NFL", Confidence: 1.0},
	})
	db.Exec("UPDATE user_interests SET created_at = NOW() - INTERVAL '45 days' WHERE user_id = $1", user.ID)

	// Mark as already asked recently (within 90 day backoff)
	db.Exec("UPDATE user_interests SET last_asked_at = NOW() - INTERVAL '10 days', times_asked = 1 WHERE user_id = $1", user.ID)

	checker := interest.NewDecayChecker(db, interestRepo, postRepo, agent.ID)
	checker.RunOnce(context.Background())

	// Should NOT have generated a new feedback post (backoff not expired)
	posts, _ := postRepo.ListByAgent(agent.ID, 10, "")
	for _, p := range posts {
		if p.DisplayHint != nil && *p.DisplayHint == "feedback" {
			t.Error("should not generate feedback post during backoff period")
		}
	}
}

func TestDecayChecker_StopsAfterMaxAsks(t *testing.T) {
	db := database.OpenTestDB(t)
	userRepo := repository.NewUserRepo(db)
	interestRepo := repository.NewUserInterestRepo(db)
	postRepo := repository.NewPostRepo(db)
	agentRepo := repository.NewAgentRepo(db)

	user, _ := userRepo.FindOrCreateByFirebaseUID("firebase-decay-maxask")
	agent, _ := agentRepo.Create(user.ID, "Briefing")

	interestRepo.BulkSetUser(user.ID, []model.UserInterest{
		{Category: "sports", Topic: "NFL", Confidence: 1.0},
	})
	db.Exec("UPDATE user_interests SET created_at = NOW() - INTERVAL '45 days', times_asked = 3, last_asked_at = NOW() - INTERVAL '100 days' WHERE user_id = $1", user.ID)

	checker := interest.NewDecayChecker(db, interestRepo, postRepo, agent.ID)
	checker.RunOnce(context.Background())

	// Should NOT generate — max 3 asks reached
	posts, _ := postRepo.ListByAgent(agent.ID, 10, "")
	for _, p := range posts {
		if p.DisplayHint != nil && *p.DisplayHint == "feedback" {
			t.Error("should not generate feedback post after max asks (3)")
		}
	}
}
```

- [ ] **Step 2: Implement decay.go**

Create `backend/internal/interest/decay.go`:

```go
package interest

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

const (
	disengagementDays = 30
	backoffDays       = 90
	maxAsks           = 3
)

type DecayChecker struct {
	db           *sql.DB
	interestRepo *repository.UserInterestRepo
	postRepo     *repository.PostRepo
	agentID      string // system agent that posts feedback panels
}

func NewDecayChecker(db *sql.DB, interestRepo *repository.UserInterestRepo, postRepo *repository.PostRepo, agentID string) *DecayChecker {
	return &DecayChecker{db: db, interestRepo: interestRepo, postRepo: postRepo, agentID: agentID}
}

type decayCandidate struct {
	InterestID string
	UserID     string
	Category   string
	Topic      string
	TimesAsked int
	LastAsked  *time.Time
}

func (d *DecayChecker) RunOnce(ctx context.Context) error {
	// Find user-declared interests older than 30 days with low engagement
	rows, err := d.db.QueryContext(ctx, `
		SELECT ui.id, ui.user_id, ui.category, ui.topic, ui.times_asked, ui.last_asked_at
		FROM user_interests ui
		WHERE ui.source = 'user'
		  AND ui.dismissed = FALSE
		  AND (ui.paused_until IS NULL OR ui.paused_until < NOW())
		  AND ui.created_at < NOW() - INTERVAL '30 days'
		  AND ui.times_asked < $1
		  AND (ui.last_asked_at IS NULL OR ui.last_asked_at < NOW() - INTERVAL '90 days')
		  AND NOT EXISTS (
			SELECT 1 FROM post_events pe
			JOIN posts p ON p.id = pe.post_id
			WHERE pe.user_id = ui.user_id
			  AND pe.event_type IN ('save', 'dwell')
			  AND pe.created_at > NOW() - INTERVAL '30 days'
			  AND p.labels LIKE '%%' || ui.category || '%%'
		  )`, maxAsks)
	if err != nil {
		return fmt.Errorf("decay query: %w", err)
	}
	defer rows.Close()

	var candidates []decayCandidate
	for rows.Next() {
		var c decayCandidate
		if err := rows.Scan(&c.InterestID, &c.UserID, &c.Category, &c.Topic, &c.TimesAsked, &c.LastAsked); err != nil {
			continue
		}
		candidates = append(candidates, c)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("decay iterate: %w", err)
	}

	for _, c := range candidates {
		if err := d.generateFeedbackPost(ctx, c); err != nil {
			slog.Warn("decay: failed to generate feedback post",
				"user_id", c.UserID, "interest", c.Category, "error", err)
			continue
		}
		if err := d.interestRepo.MarkAsked(c.InterestID); err != nil {
			slog.Warn("decay: failed to mark asked", "interest_id", c.InterestID, "error", err)
		}
	}

	slog.Info("interest decay check complete", "candidates_checked", len(candidates))
	return nil
}

func (d *DecayChecker) generateFeedbackPost(ctx context.Context, c decayCandidate) error {
	feedbackData := map[string]interface{}{
		"feedback_type": "interest_check",
		"interest_id":   c.InterestID,
		"question":      fmt.Sprintf("You haven't been engaging with %s posts recently. What would you like to do?", c.Topic),
		"options": []map[string]string{
			{"key": "still_interested", "label": "Still interested"},
			{"key": "pause", "label": "Pause for a while"},
			{"key": "less", "label": "Less of this"},
			{"key": "remove", "label": "Remove it"},
		},
	}

	externalURL, _ := json.Marshal(feedbackData)

	_, err := d.postRepo.Create(
		d.agentID,
		c.UserID,
		fmt.Sprintf("Still interested in %s?", c.Topic),
		fmt.Sprintf("We noticed you haven't been engaging with %s content lately. Let us know how you'd like to adjust.", c.Topic),
		"",                  // image_url
		string(externalURL), // external_url (structured JSON)
		"",                  // locality
		nil,                 // latitude
		nil,                 // longitude
		"discovery",         // post_type
		"personal",          // visibility
		"feedback",          // display_hint
		[]string{c.Category, "feedback"},
		nil, // images
		"",  // status
		"",  // scheduled_at
		"",  // source_published_at
	)
	return err
}
```

- [ ] **Step 3: Wire decay checker in main.go**

In `cmd/server/main.go`, find or create a system agent ID for feedback posts, and start the decay checker alongside the interest worker:

```go
	decayChecker := interest.NewDecayChecker(db, interestRepo, postRepo, os.Getenv("FEEDBACK_AGENT_ID"))
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-workerCtx.Done():
				return
			case <-ticker.C:
				if err := decayChecker.RunOnce(workerCtx); err != nil {
					slog.Warn("interest decay check failed", "error", err)
				}
			}
		}
	}()
```

- [ ] **Step 4: Run tests**

Run: `cd backend && go test ./internal/interest/ -v`

Expected: All decay tests pass.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/interest/decay.go backend/internal/interest/decay_test.go backend/cmd/server/main.go
git commit -m "feat(interest): add decay checker — generates feedback posts for disengaged interests

Respects 90-day backoff between asks, max 3 asks per interest.
Checks for zero engagement over 30 days before prompting."
```

---

## Task 16: Update GetPostStats to Include max_per_day

**Files:**
- Modify: `backend/internal/handler/post.go` (GetPostStats method)

- [ ] **Step 1: Find the GetPostStats handler**

Read `backend/internal/handler/post.go` and find the `GetPostStats` method. It currently returns posting stats by type/hint/label across 7/30/90-day windows.

- [ ] **Step 2: Add content prefs repo to PostHandler**

Add `contentPrefsRepo *repository.UserContentPrefsRepo` to the `PostHandler` struct and constructor. Update `main.go` to pass it.

- [ ] **Step 3: Include target_frequency in stats response**

In `GetPostStats`, after building the existing response, add:

```go
	// Include user's target frequency from content prefs
	prefs, err := h.contentPrefsRepo.List(agent.UserID)
	if err == nil {
		for _, p := range prefs {
			if p.Category == nil && p.MaxPerDay != nil {
				resp["target_posts_per_day"] = *p.MaxPerDay
				break
			}
		}
	}
```

This adds `target_posts_per_day` to the stats response so batch mode can read it.

- [ ] **Step 4: Build and verify**

Run: `cd backend && go build ./cmd/server`

Expected: Compiles cleanly.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/handler/post.go backend/cmd/server/main.go
git commit -m "feat(stats): include target_posts_per_day in GET /posts/stats response"
```

---

## Task 17: iOS ProfileView

**Files:**
- Create: `beepbopboop/beepbopboop/Views/ProfileView.swift`

- [ ] **Step 1: Create the profile view**

Create `beepbopboop/beepbopboop/Views/ProfileView.swift`:

```swift
import SwiftUI

struct ProfileView: View {
    @EnvironmentObject var authService: AuthService
    @State private var profile: UserProfile?
    @State private var isLoading = true
    @State private var isEditing = false

    var body: some View {
        NavigationStack {
            Group {
                if isLoading {
                    ProgressView()
                } else if let profile {
                    ScrollView {
                        VStack(alignment: .leading, spacing: 24) {
                            // Identity section
                            VStack(alignment: .leading, spacing: 8) {
                                Text("PROFILE")
                                    .font(.system(size: 11, weight: .medium, design: .monospaced))
                                    .foregroundStyle(.secondary)
                                HStack(spacing: 12) {
                                    Circle()
                                        .fill(Color(.systemGray4))
                                        .frame(width: 48, height: 48)
                                        .overlay(
                                            Text(String(profile.identity.displayName.prefix(1)).uppercased())
                                                .font(.system(size: 20, weight: .bold, design: .serif))
                                        )
                                    VStack(alignment: .leading, spacing: 2) {
                                        Text(profile.identity.displayName)
                                            .font(.system(size: 17, weight: .semibold))
                                        Text("\(profile.identity.homeLocation) · \(profile.identity.timezone)")
                                            .font(.system(size: 13, design: .monospaced))
                                            .foregroundStyle(.secondary)
                                    }
                                }
                            }
                            .padding(.horizontal)

                            // Interests section
                            if !profile.interests.isEmpty {
                                VStack(alignment: .leading, spacing: 8) {
                                    Text("INTERESTS")
                                        .font(.system(size: 11, weight: .medium, design: .monospaced))
                                        .foregroundStyle(.secondary)
                                    FlowLayout(spacing: 8) {
                                        ForEach(profile.interests) { interest in
                                            HStack(spacing: 4) {
                                                Text(interest.topic)
                                                    .font(.system(size: 13))
                                                if interest.source == "inferred" {
                                                    Image(systemName: "sparkles")
                                                        .font(.system(size: 9))
                                                        .foregroundStyle(.secondary)
                                                }
                                                if interest.pausedUntil != nil {
                                                    Image(systemName: "pause.circle")
                                                        .font(.system(size: 9))
                                                        .foregroundStyle(.orange)
                                                }
                                            }
                                            .padding(.horizontal, 12)
                                            .padding(.vertical, 6)
                                            .background(Color(.systemGray6))
                                            .clipShape(Capsule())
                                        }
                                    }
                                }
                                .padding(.horizontal)
                            }

                            // Lifestyle section
                            if !profile.lifestyle.isEmpty {
                                VStack(alignment: .leading, spacing: 8) {
                                    Text("LIFESTYLE")
                                        .font(.system(size: 11, weight: .medium, design: .monospaced))
                                        .foregroundStyle(.secondary)
                                    FlowLayout(spacing: 8) {
                                        ForEach(profile.lifestyle, id: \.value) { tag in
                                            Text(tag.value.replacingOccurrences(of: "_", with: " ").capitalized)
                                                .font(.system(size: 13))
                                                .padding(.horizontal, 12)
                                                .padding(.vertical, 6)
                                                .background(Color(.systemGray6))
                                                .clipShape(Capsule())
                                        }
                                    }
                                }
                                .padding(.horizontal)
                            }

                            // Content prefs section
                            if !profile.contentPrefs.isEmpty {
                                VStack(alignment: .leading, spacing: 8) {
                                    Text("CONTENT PREFERENCES")
                                        .font(.system(size: 11, weight: .medium, design: .monospaced))
                                        .foregroundStyle(.secondary)
                                    ForEach(profile.contentPrefs, id: \.depth) { pref in
                                        HStack {
                                            Text(pref.category ?? "Global")
                                                .font(.system(size: 14, weight: .medium))
                                            Spacer()
                                            Text("\(pref.depth) · \(pref.tone)")
                                                .font(.system(size: 13, design: .monospaced))
                                                .foregroundStyle(.secondary)
                                            if let max = pref.maxPerDay {
                                                Text("≤\(max)/day")
                                                    .font(.system(size: 13, design: .monospaced))
                                                    .foregroundStyle(.secondary)
                                            }
                                        }
                                    }
                                }
                                .padding(.horizontal)
                            }
                        }
                        .padding(.vertical)
                    }
                } else {
                    Text("Failed to load profile")
                        .foregroundStyle(.secondary)
                }
            }
            .navigationTitle("Profile")
            .task { await loadProfile() }
        }
    }

    private func loadProfile() async {
        isLoading = true
        defer { isLoading = false }
        let api = APIService(baseURL: Config.backendBaseURL, authToken: authService.getToken())
        profile = try? await api.getProfile()
    }
}
```

- [ ] **Step 2: Build to verify**

Run: `python3 scripts/build_and_test.py --project beepbopboop/beepbopboop.xcodeproj --scheme beepbopboop`

Expected: 0 errors. (Note: `FlowLayout` is defined in `OnboardingInterestsView.swift` — if it's not accessible, move it to a shared file.)

- [ ] **Step 3: Commit**

```bash
git add beepbopboop/beepbopboop/Views/ProfileView.swift
git commit -m "feat(ios): add ProfileView showing identity, interests, lifestyle, prefs"
```

---

## Task 18: Update Interest Inference Worker to Include Dwell Time and Reactions

**Files:**
- Modify: `backend/internal/interest/worker.go`

- [ ] **Step 1: Update the RunOnce query**

Replace the saves-only query with one that includes dwell time events and "more" reactions:

```go
func (w *Worker) RunOnce(ctx context.Context) error {
	rows, err := w.db.QueryContext(ctx, `
		WITH engagement AS (
			SELECT pe.user_id,
				unnest(string_to_array(trim(both '[]"' from p.labels), '","')) AS label,
				CASE
					WHEN pe.event_type = 'save' THEN 3.0
					WHEN pe.event_type = 'dwell' THEN 1.0
					ELSE 0.0
				END AS weight
			FROM post_events pe
			JOIN posts p ON p.id = pe.post_id
			WHERE pe.created_at > NOW() - INTERVAL '30 days'
			  AND p.labels IS NOT NULL AND p.labels != ''

			UNION ALL

			SELECT pr.user_id,
				unnest(string_to_array(trim(both '[]"' from p.labels), '","')) AS label,
				CASE WHEN pr.reaction = 'more' THEN 5.0 ELSE 0.0 END AS weight
			FROM post_reactions pr
			JOIN posts p ON p.id = pr.post_id
			WHERE pr.reaction = 'more'
			  AND pr.created_at > NOW() - INTERVAL '30 days'
			  AND p.labels IS NOT NULL AND p.labels != ''
		)
		SELECT user_id, label, SUM(weight) AS score
		FROM engagement
		GROUP BY user_id, label
		HAVING SUM(weight) >= 5.0
		ORDER BY score DESC`)
	if err != nil {
		return err
	}
	defer rows.Close()

	var count int
	for rows.Next() {
		var userID, label string
		var score float64
		if err := rows.Scan(&userID, &label, &score); err != nil {
			continue
		}
		confidence := score / 30.0 // 30 weighted points = 1.0 confidence
		if confidence > 1.0 {
			confidence = 1.0
		}
		if err := w.interestRepo.UpsertInferred(userID, label, label, confidence); err != nil {
			slog.Warn("failed to upsert inferred interest",
				"user_id", userID, "label", label, "error", err)
		}
		count++
	}

	slog.Info("interest inference complete", "interests_upserted", count)
	return rows.Err()
}
```

- [ ] **Step 2: Run tests**

Run: `cd backend && go test ./internal/interest/ -v`

Expected: All tests pass.

- [ ] **Step 3: Commit**

```bash
git add backend/internal/interest/worker.go
git commit -m "feat(interest): include dwell time and 'more' reactions in inference scoring

Weighted scoring: save=3, dwell=1, 'more' reaction=5. Threshold of 5
weighted points to trigger an inferred interest."
```

---

## Spec Deviations

**`PUT /user/interests` → `PUT /user/interests/declared`**: The spec defines the endpoint path as `PUT /user/interests`. The plan registers it as `PUT /user/interests/declared` to avoid a conflict with the existing `POST /user/interests` (onboarding embedding endpoint). Both the backend and iOS use the `/declared` path consistently. The spec should be updated to reflect this.

---

## Summary

| Task | What | Backend | iOS | Skills |
|------|------|---------|-----|--------|
| 1 | Database schema | x | | |
| 2 | Profile models | x | | |
| 3 | User repo profile support | x | | |
| 4 | Interest repo | x | | |
| 5 | Lifestyle + prefs repos | x | | |
| 6 | Profile handler | x | | |
| 7 | Route registration | x | | |
| 8 | Onboarding writes plaintext | x | | |
| 9 | iOS profile model | | x | |
| 10 | iOS API methods | | x | |
| 11 | iOS onboarding flow | | x | |
| 12 | Skill bootstrap update | | | x |
| 13 | Interest inference worker | x | | |
| 14 | Full test pass | x | x | |
| 15 | Interest decay checker | x | | |
| 16 | Stats include max_per_day | x | | |
| 17 | iOS ProfileView | | x | |
| 18 | Worker: dwell + reactions | x | | |
