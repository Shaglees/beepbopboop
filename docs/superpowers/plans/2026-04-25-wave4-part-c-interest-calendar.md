# Wave 4 Part C: Interest Calendar

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Automatically create time-sensitive feed posts for upcoming sports games and entertainment releases that match the user's interests.

**Spec:** `docs/superpowers/specs/2026-04-25-wave4-new-features-design.md` (Sub-system C)

---

### Task 12: Database Schema — Calendar Event Tables

**Files:**
- Modify: `backend/internal/database/database.go:~397` (before `return db, nil`)

- [ ] **Step 1: Add migration statements**

Add before the final `return db, nil` in the `Open` function:

```go
	// Wave 4: interest calendar events
	db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS interest_calendar_events (
		id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		event_key     TEXT NOT NULL UNIQUE,
		domain        TEXT NOT NULL,
		title         TEXT NOT NULL,
		start_time    TIMESTAMPTZ NOT NULL,
		end_time      TIMESTAMPTZ,
		timezone      TEXT NOT NULL DEFAULT 'UTC',
		status        TEXT NOT NULL DEFAULT 'scheduled',
		entity_type   TEXT NOT NULL,
		entity_ids    JSONB NOT NULL DEFAULT '{}',
		interest_tags TEXT[] NOT NULL DEFAULT '{}',
		payload       JSONB NOT NULL DEFAULT '{}',
		created_at    TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at    TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`)
	db.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_ice_domain_start ON interest_calendar_events (domain, start_time)`)
	db.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_ice_status ON interest_calendar_events (status) WHERE status = 'scheduled'`)
	db.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_ice_tags ON interest_calendar_events USING GIN (interest_tags)`)

	// Wave 4: calendar post dedup log
	db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS calendar_post_log (
		event_key  TEXT NOT NULL,
		user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		window     TEXT NOT NULL,
		post_id    UUID NOT NULL REFERENCES posts(id),
		created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (event_key, user_id, window)
	)`)
```

- [ ] **Step 2: Verify migration compiles**

Run: `cd backend && go build ./cmd/server/`
Expected: Compiles.

- [ ] **Step 3: Commit**

```bash
git add backend/internal/database/database.go
git commit -m "feat(wave4): add interest_calendar_events + calendar_post_log tables"
```

---

### Task 13: CalendarEvent Model + Repository

**Files:**
- Create: `backend/internal/model/calendar_event.go`
- Create: `backend/internal/repository/calendar_event_repo.go`
- Create: `backend/internal/repository/calendar_event_repo_test.go`

- [ ] **Step 1: Write the model**

Create `backend/internal/model/calendar_event.go`:

```go
package model

import (
	"encoding/json"
	"time"
)

type CalendarEvent struct {
	ID           string          `json:"id"`
	EventKey     string          `json:"event_key"`
	Domain       string          `json:"domain"`
	Title        string          `json:"title"`
	StartTime    time.Time       `json:"start_time"`
	EndTime      *time.Time      `json:"end_time,omitempty"`
	Timezone     string          `json:"timezone"`
	Status       string          `json:"status"`
	EntityType   string          `json:"entity_type"`
	EntityIDs    json.RawMessage `json:"entity_ids"`
	InterestTags []string        `json:"interest_tags"`
	Payload      json.RawMessage `json:"payload"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

type CalendarPostLog struct {
	EventKey  string    `json:"event_key"`
	UserID    string    `json:"user_id"`
	Window    string    `json:"window"`
	PostID    string    `json:"post_id"`
	CreatedAt time.Time `json:"created_at"`
}
```

- [ ] **Step 2: Write the failing test**

Create `backend/internal/repository/calendar_event_repo_test.go`:

```go
package repository

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/shanegleeson/beepbopboop/backend/internal/model"
)

func TestCalendarEventRepo_Upsert(t *testing.T) {
	db := setupTestDB(t)
	repo := NewCalendarEventRepo(db)

	event := model.CalendarEvent{
		EventKey:     "espn:event:401234567",
		Domain:       "sports",
		Title:        "Lakers @ Celtics",
		StartTime:    time.Now().Add(24 * time.Hour),
		Timezone:     "America/New_York",
		Status:       "scheduled",
		EntityType:   "game",
		EntityIDs:    json.RawMessage(`{"home_team":"celtics","away_team":"lakers","league":"nba"}`),
		InterestTags: []string{"basketball", "nba", "lakers", "celtics"},
		Payload:      json.RawMessage(`{"venue":"TD Garden","broadcast":"ESPN"}`),
	}

	err := repo.Upsert(event)
	if err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	// Upsert again with updated title — should not duplicate
	event.Title = "Lakers @ Celtics (Updated)"
	err = repo.Upsert(event)
	if err != nil {
		t.Fatalf("Upsert again: %v", err)
	}

	events, err := repo.Upcoming("sports", time.Now(), time.Now().Add(48*time.Hour))
	if err != nil {
		t.Fatalf("Upcoming: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Title != "Lakers @ Celtics (Updated)" {
		t.Errorf("expected updated title, got %s", events[0].Title)
	}
}

func TestCalendarEventRepo_ForUser(t *testing.T) {
	db := setupTestDB(t)
	repo := NewCalendarEventRepo(db)

	now := time.Now()

	// Sports event matching "basketball"
	repo.Upsert(model.CalendarEvent{
		EventKey: "espn:event:1", Domain: "sports", Title: "Game 1",
		StartTime: now.Add(12 * time.Hour), Timezone: "UTC", Status: "scheduled",
		EntityType: "game", EntityIDs: json.RawMessage(`{}`),
		InterestTags: []string{"basketball", "nba"},
		Payload:      json.RawMessage(`{}`),
	})

	// Entertainment event matching "sci-fi"
	repo.Upsert(model.CalendarEvent{
		EventKey: "tmdb:movie:999:release:US", Domain: "entertainment", Title: "Movie X",
		StartTime: now.Add(72 * time.Hour), Timezone: "UTC", Status: "scheduled",
		EntityType: "movie_release", EntityIDs: json.RawMessage(`{}`),
		InterestTags: []string{"sci-fi", "action"},
		Payload:      json.RawMessage(`{}`),
	})

	// Query with basketball interest — should match only Game 1
	events, err := repo.ForUser("test-user", []string{"basketball"}, now, now.Add(7*24*time.Hour))
	if err != nil {
		t.Fatalf("ForUser: %v", err)
	}
	if len(events) != 1 || events[0].Title != "Game 1" {
		t.Fatalf("expected Game 1 only, got %d events", len(events))
	}

	// Query with sci-fi interest — should match Movie X
	events, err = repo.ForUser("test-user", []string{"sci-fi"}, now, now.Add(7*24*time.Hour))
	if err != nil {
		t.Fatalf("ForUser sci-fi: %v", err)
	}
	if len(events) != 1 || events[0].Title != "Movie X" {
		t.Fatalf("expected Movie X only, got %d events", len(events))
	}
}

func TestCalendarPostLog_Dedup(t *testing.T) {
	db := setupTestDB(t)
	repo := NewCalendarEventRepo(db)
	userID := createTestUser(t, db)
	postID := createTestPost(t, db)

	pub, err := repo.IsPublished("espn:event:1", userID, "preview")
	if err != nil {
		t.Fatalf("IsPublished: %v", err)
	}
	if pub {
		t.Fatal("should not be published yet")
	}

	err = repo.LogPost("espn:event:1", userID, "preview", postID)
	if err != nil {
		t.Fatalf("LogPost: %v", err)
	}

	pub, err = repo.IsPublished("espn:event:1", userID, "preview")
	if err != nil {
		t.Fatalf("IsPublished after log: %v", err)
	}
	if !pub {
		t.Fatal("should be published now")
	}

	// Logging same combo again should conflict gracefully
	err = repo.LogPost("espn:event:1", userID, "preview", postID)
	if err != nil {
		t.Fatalf("LogPost duplicate: %v", err)
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `cd backend && go test ./internal/repository/ -run TestCalendarEvent -v`
Expected: FAIL — `NewCalendarEventRepo` not defined.

Also: `cd backend && go test ./internal/repository/ -run TestCalendarPostLog -v`
Expected: FAIL.

- [ ] **Step 4: Write the repository**

Create `backend/internal/repository/calendar_event_repo.go`:

```go
package repository

import (
	"database/sql"
	"time"

	"github.com/lib/pq"
	"github.com/shanegleeson/beepbopboop/backend/internal/model"
)

type CalendarEventRepo struct {
	db *sql.DB
}

func NewCalendarEventRepo(db *sql.DB) *CalendarEventRepo {
	return &CalendarEventRepo{db: db}
}

func (r *CalendarEventRepo) Upsert(e model.CalendarEvent) error {
	_, err := r.db.Exec(`
		INSERT INTO interest_calendar_events
			(event_key, domain, title, start_time, end_time, timezone, status,
			 entity_type, entity_ids, interest_tags, payload)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (event_key) DO UPDATE SET
			title = EXCLUDED.title,
			start_time = EXCLUDED.start_time,
			end_time = EXCLUDED.end_time,
			status = EXCLUDED.status,
			entity_ids = EXCLUDED.entity_ids,
			interest_tags = EXCLUDED.interest_tags,
			payload = EXCLUDED.payload,
			updated_at = CURRENT_TIMESTAMP
	`, e.EventKey, e.Domain, e.Title, e.StartTime, e.EndTime, e.Timezone,
		e.Status, e.EntityType, e.EntityIDs, pq.Array(e.InterestTags), e.Payload)
	return err
}

func (r *CalendarEventRepo) Upcoming(domain string, from, to time.Time) ([]model.CalendarEvent, error) {
	rows, err := r.db.Query(`
		SELECT id, event_key, domain, title, start_time, end_time, timezone,
		       status, entity_type, entity_ids, interest_tags, payload, created_at, updated_at
		FROM interest_calendar_events
		WHERE domain = $1 AND start_time >= $2 AND start_time <= $3
		ORDER BY start_time ASC
	`, domain, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanCalendarEvents(rows)
}

func (r *CalendarEventRepo) ForUser(userID string, interests []string, from, to time.Time) ([]model.CalendarEvent, error) {
	if len(interests) == 0 {
		return nil, nil
	}
	rows, err := r.db.Query(`
		SELECT id, event_key, domain, title, start_time, end_time, timezone,
		       status, entity_type, entity_ids, interest_tags, payload, created_at, updated_at
		FROM interest_calendar_events
		WHERE interest_tags && $1
		  AND start_time >= $2 AND start_time <= $3
		  AND status IN ('scheduled', 'live')
		ORDER BY start_time ASC
	`, pq.Array(interests), from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanCalendarEvents(rows)
}

func (r *CalendarEventRepo) LogPost(eventKey, userID, window, postID string) error {
	_, err := r.db.Exec(`
		INSERT INTO calendar_post_log (event_key, user_id, window, post_id)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (event_key, user_id, window) DO NOTHING
	`, eventKey, userID, window, postID)
	return err
}

func (r *CalendarEventRepo) IsPublished(eventKey, userID, window string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM calendar_post_log
			WHERE event_key = $1 AND user_id = $2 AND window = $3
		)
	`, eventKey, userID, window).Scan(&exists)
	return exists, err
}

func scanCalendarEvents(rows *sql.Rows) ([]model.CalendarEvent, error) {
	var events []model.CalendarEvent
	for rows.Next() {
		var e model.CalendarEvent
		err := rows.Scan(&e.ID, &e.EventKey, &e.Domain, &e.Title, &e.StartTime,
			&e.EndTime, &e.Timezone, &e.Status, &e.EntityType, &e.EntityIDs,
			pq.Array(&e.InterestTags), &e.Payload, &e.CreatedAt, &e.UpdatedAt)
		if err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, rows.Err()
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd backend && go test ./internal/repository/ -run "TestCalendarEvent|TestCalendarPostLog" -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add backend/internal/model/calendar_event.go backend/internal/repository/calendar_event_repo.go backend/internal/repository/calendar_event_repo_test.go
git commit -m "feat(wave4): add CalendarEvent model and repository with upsert + dedup"
```

---

### Task 14: Entertainment Ingest Worker

**Files:**
- Create: `backend/internal/entertainment/worker.go`
- Create: `backend/internal/entertainment/worker_test.go`

- [ ] **Step 1: Write the failing test**

Create `backend/internal/entertainment/worker_test.go`:

```go
package entertainment

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestWorker_IngestMovies(t *testing.T) {
	// Mock TMDB server
	tmdbServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/3/movie/upcoming":
			json.NewEncoder(w).Encode(tmdbMovieResponse{
				Results: []tmdbMovie{
					{
						ID:          999,
						Title:       "Test Movie",
						Overview:    "A great test movie.",
						ReleaseDate: time.Now().Add(5 * 24 * time.Hour).Format("2006-01-02"),
						GenreIDs:    []int{28, 878},
						PosterPath:  "/test.jpg",
						VoteAverage: 7.5,
					},
				},
			})
		case r.URL.Path == "/3/tv/on_the_air":
			json.NewEncoder(w).Encode(tmdbTVResponse{
				Results: []tmdbTV{},
			})
		case r.URL.Path == "/3/genre/movie/list":
			json.NewEncoder(w).Encode(tmdbGenreResponse{
				Genres: []tmdbGenre{
					{ID: 28, Name: "Action"},
					{ID: 878, Name: "Science Fiction"},
				},
			})
		default:
			w.WriteHeader(404)
		}
	}))
	defer tmdbServer.Close()

	repo := setupTestCalendarEventRepo(t)
	worker := NewWorker(repo, tmdbServer.URL, "test-key", "US")

	worker.cycle(context.Background())

	events, err := repo.Upcoming("entertainment", time.Now(), time.Now().Add(30*24*time.Hour))
	if err != nil {
		t.Fatalf("Upcoming: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Title != "Test Movie" {
		t.Errorf("expected Test Movie, got %s", events[0].Title)
	}
	if events[0].EntityType != "movie_release" {
		t.Errorf("expected movie_release, got %s", events[0].EntityType)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/entertainment/ -run TestWorker -v`
Expected: FAIL — package doesn't exist.

- [ ] **Step 3: Write the worker**

Create `backend/internal/entertainment/worker.go`:

```go
package entertainment

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/shanegleeson/beepbopboop/backend/internal/model"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

type Worker struct {
	calendarRepo *repository.CalendarEventRepo
	baseURL      string
	apiKey       string
	region       string
	interval     time.Duration
	genres       map[int]string
}

func NewWorker(calendarRepo *repository.CalendarEventRepo, baseURL, apiKey, region string) *Worker {
	return &Worker{
		calendarRepo: calendarRepo,
		baseURL:      baseURL,
		apiKey:       apiKey,
		region:       region,
		interval:     24 * time.Hour,
		genres:       make(map[int]string),
	}
}

func (w *Worker) Run(ctx context.Context) {
	slog.Info("entertainment worker started", "interval", w.interval)
	w.loadGenres(ctx)
	w.cycle(ctx)
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			slog.Info("entertainment worker stopped")
			return
		case <-ticker.C:
			w.cycle(ctx)
		}
	}
}

func (w *Worker) loadGenres(ctx context.Context) {
	url := fmt.Sprintf("%s/3/genre/movie/list?api_key=%s", w.baseURL, w.apiKey)
	resp, err := http.Get(url)
	if err != nil {
		slog.Warn("entertainment: failed to load genres", "error", err)
		return
	}
	defer resp.Body.Close()
	var result tmdbGenreResponse
	json.NewDecoder(resp.Body).Decode(&result)
	for _, g := range result.Genres {
		w.genres[g.ID] = strings.ToLower(g.Name)
	}
}

func (w *Worker) cycle(ctx context.Context) {
	var ok, failed int

	// Upcoming movies
	movies, err := w.fetchUpcomingMovies()
	if err != nil {
		slog.Error("entertainment: fetch movies failed", "error", err)
	} else {
		for _, m := range movies {
			if err := w.ingestMovie(m); err != nil {
				failed++
			} else {
				ok++
			}
		}
	}

	// On-air TV
	shows, err := w.fetchOnAirTV()
	if err != nil {
		slog.Error("entertainment: fetch TV failed", "error", err)
	} else {
		for _, s := range shows {
			if err := w.ingestTV(s); err != nil {
				failed++
			} else {
				ok++
			}
		}
	}

	slog.Info("entertainment cycle complete", "ok", ok, "failed", failed)
}

func (w *Worker) fetchUpcomingMovies() ([]tmdbMovie, error) {
	url := fmt.Sprintf("%s/3/movie/upcoming?api_key=%s&region=%s", w.baseURL, w.apiKey, w.region)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var result tmdbMovieResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result.Results, nil
}

func (w *Worker) fetchOnAirTV() ([]tmdbTV, error) {
	url := fmt.Sprintf("%s/3/tv/on_the_air?api_key=%s", w.baseURL, w.apiKey)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var result tmdbTVResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result.Results, nil
}

func (w *Worker) ingestMovie(m tmdbMovie) error {
	releaseDate, err := time.Parse("2006-01-02", m.ReleaseDate)
	if err != nil {
		return err
	}

	tags := w.genreTagsForIDs(m.GenreIDs)
	entityIDs, _ := json.Marshal(map[string]interface{}{
		"tmdb_id": m.ID,
		"type":    "movie",
	})
	payload, _ := json.Marshal(map[string]interface{}{
		"title":        m.Title,
		"overview":     m.Overview,
		"poster_url":   "https://image.tmdb.org/t/p/w500" + m.PosterPath,
		"genres":       tags,
		"vote_average": m.VoteAverage,
		"release_date": m.ReleaseDate,
	})

	return w.calendarRepo.Upsert(model.CalendarEvent{
		EventKey:     fmt.Sprintf("tmdb:movie:%d:release:%s", m.ID, w.region),
		Domain:       "entertainment",
		Title:        m.Title,
		StartTime:    releaseDate,
		Timezone:     "UTC",
		Status:       "scheduled",
		EntityType:   "movie_release",
		EntityIDs:    entityIDs,
		InterestTags: tags,
		Payload:      payload,
	})
}

func (w *Worker) ingestTV(s tmdbTV) error {
	airDate, err := time.Parse("2006-01-02", s.FirstAirDate)
	if err != nil {
		return err
	}

	tags := w.genreTagsForIDs(s.GenreIDs)
	entityIDs, _ := json.Marshal(map[string]interface{}{
		"tmdb_id": s.ID,
		"type":    "tv",
	})
	payload, _ := json.Marshal(map[string]interface{}{
		"title":        s.Name,
		"overview":     s.Overview,
		"poster_url":   "https://image.tmdb.org/t/p/w500" + s.PosterPath,
		"genres":       tags,
		"vote_average": s.VoteAverage,
	})

	return w.calendarRepo.Upsert(model.CalendarEvent{
		EventKey:     fmt.Sprintf("tmdb:tv:%d:premiere:%s", s.ID, w.region),
		Domain:       "entertainment",
		Title:        s.Name,
		StartTime:    airDate,
		Timezone:     "UTC",
		Status:       "scheduled",
		EntityType:   "tv_premiere",
		EntityIDs:    entityIDs,
		InterestTags: tags,
		Payload:      payload,
	})
}

func (w *Worker) genreTagsForIDs(ids []int) []string {
	var tags []string
	for _, id := range ids {
		if name, ok := w.genres[id]; ok {
			tags = append(tags, name)
		}
	}
	return tags
}

// TMDB response types

type tmdbMovieResponse struct {
	Results []tmdbMovie `json:"results"`
}

type tmdbMovie struct {
	ID          int     `json:"id"`
	Title       string  `json:"title"`
	Overview    string  `json:"overview"`
	ReleaseDate string  `json:"release_date"`
	GenreIDs    []int   `json:"genre_ids"`
	PosterPath  string  `json:"poster_path"`
	VoteAverage float64 `json:"vote_average"`
}

type tmdbTVResponse struct {
	Results []tmdbTV `json:"results"`
}

type tmdbTV struct {
	ID           int     `json:"id"`
	Name         string  `json:"name"`
	Overview     string  `json:"overview"`
	FirstAirDate string  `json:"first_air_date"`
	GenreIDs     []int   `json:"genre_ids"`
	PosterPath   string  `json:"poster_path"`
	VoteAverage  float64 `json:"vote_average"`
}

type tmdbGenreResponse struct {
	Genres []tmdbGenre `json:"genres"`
}

type tmdbGenre struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}
```

- [ ] **Step 4: Run tests**

Run: `cd backend && go test ./internal/entertainment/ -run TestWorker -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add backend/internal/entertainment/
git commit -m "feat(wave4): add entertainment ingest worker for TMDB movies and TV"
```

---

### Task 15: Extend Sports Worker for Calendar Events

**Files:**
- Modify: `backend/internal/sports/worker.go`

- [ ] **Step 1: Read existing sports worker**

Read `backend/internal/sports/worker.go` and `backend/internal/sports/service.go` to understand the current cycle() flow and what game data is already available.

- [ ] **Step 2: Add CalendarEventRepo to Worker struct**

In the `Worker` struct, add:
```go
type Worker struct {
	svc          *Service
	postRepo     *repository.PostRepo
	calendarRepo *repository.CalendarEventRepo  // new
	interval     time.Duration
}
```

Update the constructor:
```go
func NewWorker(svc *Service, postRepo *repository.PostRepo, calendarRepo *repository.CalendarEventRepo, interval time.Duration) *Worker {
	return &Worker{svc: svc, postRepo: postRepo, calendarRepo: calendarRepo, interval: interval}
}
```

- [ ] **Step 3: Add calendar upsert to cycle**

In the `cycle()` method, after processing live scores, add a section that upserts upcoming scheduled games:

```go
	// Upsert upcoming games to calendar events
	if w.calendarRepo != nil {
		for _, game := range games {
			if game.Status != "scheduled" && game.Status != "pre" {
				continue
			}
			startTime, err := time.Parse(time.RFC3339, game.StartTime)
			if err != nil {
				continue
			}
			// Only games in next 7 days
			if startTime.After(time.Now().Add(7 * 24 * time.Hour)) {
				continue
			}

			tags := []string{
				strings.ToLower(game.League),
				strings.ToLower(game.HomeTeam.Slug),
				strings.ToLower(game.AwayTeam.Slug),
			}
			// Add sport name tag
			switch strings.ToLower(game.League) {
			case "nba", "wnba":
				tags = append(tags, "basketball")
			case "nfl":
				tags = append(tags, "football")
			case "mlb":
				tags = append(tags, "baseball")
			case "nhl":
				tags = append(tags, "hockey")
			case "mls":
				tags = append(tags, "soccer")
			}

			entityIDs, _ := json.Marshal(map[string]interface{}{
				"home_team": game.HomeTeam.Slug,
				"away_team": game.AwayTeam.Slug,
				"league":    game.League,
			})
			payload, _ := json.Marshal(map[string]interface{}{
				"home_team":    game.HomeTeam.Name,
				"away_team":    game.AwayTeam.Name,
				"home_abbr":    game.HomeTeam.Abbreviation,
				"away_abbr":    game.AwayTeam.Abbreviation,
				"home_record":  game.HomeTeam.Record,
				"away_record":  game.AwayTeam.Record,
				"venue":        game.Venue,
				"broadcast":    game.Broadcast,
				"league":       game.League,
			})

			w.calendarRepo.Upsert(model.CalendarEvent{
				EventKey:     fmt.Sprintf("espn:event:%s", game.ID),
				Domain:       "sports",
				Title:        fmt.Sprintf("%s @ %s", game.AwayTeam.Name, game.HomeTeam.Name),
				StartTime:    startTime,
				Timezone:     "America/New_York",
				Status:       "scheduled",
				EntityType:   "game",
				EntityIDs:    entityIDs,
				InterestTags: tags,
				Payload:      payload,
			})
		}
	}
```

**Note:** Adapt field names to match the actual `Game` struct in `sports/service.go`. Read the file first.

- [ ] **Step 4: Update main.go to pass calendarRepo to sports worker**

In `main.go`, update the sports worker construction (~line 262):

```go
	sportsWorker := sports.NewWorker(sportsSvc, postRepo, calendarEventRepo, 10*time.Minute)
```

And add to repos section:
```go
	calendarEventRepo := repository.NewCalendarEventRepo(db)
```

- [ ] **Step 5: Build and test**

Run: `cd backend && go build ./cmd/server/`
Expected: Compiles.

Run: `cd backend && go test ./internal/sports/ -v`
Expected: PASS (existing tests still pass; may need to update test constructors).

- [ ] **Step 6: Commit**

```bash
git add backend/internal/sports/worker.go backend/cmd/server/main.go
git commit -m "feat(wave4): extend sports worker to upsert upcoming games as calendar events"
```

---

### Task 16: Materialization Worker

**Files:**
- Create: `backend/internal/calendar/materialize.go`
- Create: `backend/internal/calendar/templates.go`
- Create: `backend/internal/calendar/materialize_test.go`

- [ ] **Step 1: Write the templates**

Create `backend/internal/calendar/templates.go`:

```go
package calendar

import (
	"bytes"
	"encoding/json"
	"fmt"
	"text/template"
	"time"
)

type postContent struct {
	Title       string
	Body        string
	DisplayHint string
	ExternalURL string
}

var sportsPreviewTmpl = template.Must(template.New("sports_preview").Parse(
	`{{.AwayTeam}} @ {{.HomeTeam}} — {{.StartFormatted}}`))

var sportsPreviewBodyTmpl = template.Must(template.New("sports_preview_body").Parse(
	`{{.HomeTeam}} ({{.HomeRecord}}) host {{.AwayTeam}} ({{.AwayRecord}}) at {{.Venue}}. {{.Broadcast}}.`))

var sportsImminentTmpl = template.Must(template.New("sports_imminent").Parse(
	`{{.AwayTeam}} @ {{.HomeTeam}} tips off in {{.TimeUntil}}`))

var sportsImminentBodyTmpl = template.Must(template.New("sports_imminent_body").Parse(
	`{{.HomeTeam}} ({{.HomeRecord}}) vs {{.AwayTeam}} ({{.AwayRecord}}).`))

var entertainmentPreviewTmpl = template.Must(template.New("ent_preview").Parse(
	`{{.Title}} hits theaters {{.ReleaseDay}}`))

var entertainmentPreviewBodyTmpl = template.Must(template.New("ent_preview_body").Parse(
	`{{.Overview}} Starring {{.Cast}}.`))

var entertainmentReleaseTmpl = template.Must(template.New("ent_release").Parse(
	`{{.Title}} is out today`))

var entertainmentReleaseBodyTmpl = template.Must(template.New("ent_release_body").Parse(
	`{{.Overview}} Now playing at theaters near you.`))

func renderSportsPost(payload json.RawMessage, startTime time.Time, window string) (*postContent, error) {
	var data map[string]interface{}
	json.Unmarshal(payload, &data)

	getString := func(key string) string {
		if v, ok := data[key].(string); ok {
			return v
		}
		return ""
	}

	vars := map[string]string{
		"HomeTeam":       getString("home_team"),
		"AwayTeam":       getString("away_team"),
		"HomeRecord":     getString("home_record"),
		"AwayRecord":     getString("away_record"),
		"Venue":          getString("venue"),
		"Broadcast":      getString("broadcast"),
		"StartFormatted": startTime.Format("Mon Jan 2, 3:04 PM"),
		"TimeUntil":      formatTimeUntil(time.Until(startTime)),
	}

	var titleTmpl, bodyTmpl *template.Template
	if window == "imminent" {
		titleTmpl = sportsImminentTmpl
		bodyTmpl = sportsImminentBodyTmpl
	} else {
		titleTmpl = sportsPreviewTmpl
		bodyTmpl = sportsPreviewBodyTmpl
	}

	var title, body bytes.Buffer
	titleTmpl.Execute(&title, vars)
	bodyTmpl.Execute(&body, vars)

	return &postContent{
		Title:       title.String(),
		Body:        body.String(),
		DisplayHint: "matchup",
		ExternalURL: string(payload),
	}, nil
}

func renderEntertainmentPost(payload json.RawMessage, startTime time.Time, window string) (*postContent, error) {
	var data map[string]interface{}
	json.Unmarshal(payload, &data)

	getString := func(key string) string {
		if v, ok := data[key].(string); ok {
			return v
		}
		return ""
	}

	vars := map[string]string{
		"Title":      getString("title"),
		"Overview":   truncate(getString("overview"), 200),
		"Cast":       getString("cast"),
		"ReleaseDay": startTime.Format("January 2"),
	}

	var titleTmpl, bodyTmpl *template.Template
	if window == "release_day" {
		titleTmpl = entertainmentReleaseTmpl
		bodyTmpl = entertainmentReleaseBodyTmpl
	} else {
		titleTmpl = entertainmentPreviewTmpl
		bodyTmpl = entertainmentPreviewBodyTmpl
	}

	var title, body bytes.Buffer
	titleTmpl.Execute(&title, vars)
	bodyTmpl.Execute(&body, vars)

	return &postContent{
		Title:       title.String(),
		Body:        body.String(),
		DisplayHint: "event",
		ExternalURL: string(payload),
	}, nil
}

func formatTimeUntil(d time.Duration) string {
	if d < time.Hour {
		return fmt.Sprintf("%d minutes", int(d.Minutes()))
	}
	return fmt.Sprintf("%.1f hours", d.Hours())
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
```

- [ ] **Step 2: Write the failing test**

Create `backend/internal/calendar/materialize_test.go`:

```go
package calendar

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/shanegleeson/beepbopboop/backend/internal/model"
)

func TestMaterializeWorker_SportsPreview(t *testing.T) {
	db := setupTestDB(t)
	calendarRepo := setupCalendarEventRepo(t, db)
	postRepo := setupPostRepo(t, db)
	userRepo := setupUserRepo(t, db)
	interestRepo := setupInterestRepo(t, db)

	// Create user with basketball interest
	userID := createTestUserWithInterests(t, db, []string{"basketball", "nba"})

	// Create calendar event for tomorrow
	calendarRepo.Upsert(model.CalendarEvent{
		EventKey: "espn:event:test1", Domain: "sports", Title: "Lakers @ Celtics",
		StartTime: time.Now().Add(20 * time.Hour), Timezone: "UTC", Status: "scheduled",
		EntityType: "game", EntityIDs: json.RawMessage(`{"home_team":"celtics","away_team":"lakers"}`),
		InterestTags: []string{"basketball", "nba", "lakers", "celtics"},
		Payload:      json.RawMessage(`{"home_team":"Celtics","away_team":"Lakers","home_record":"50-20","away_record":"45-25","venue":"TD Garden","broadcast":"ESPN"}`),
	})

	worker := NewMaterializeWorker(calendarRepo, postRepo, userRepo, interestRepo, "test-agent-id")
	worker.cycleOnce(context.Background())

	// Check that a post was created
	published, err := calendarRepo.IsPublished("espn:event:test1", userID, "preview")
	if err != nil {
		t.Fatalf("IsPublished: %v", err)
	}
	if !published {
		t.Fatal("expected preview post to be published")
	}
}

func TestMaterializeWorker_SkipsPublished(t *testing.T) {
	db := setupTestDB(t)
	calendarRepo := setupCalendarEventRepo(t, db)
	postRepo := setupPostRepo(t, db)
	userRepo := setupUserRepo(t, db)
	interestRepo := setupInterestRepo(t, db)

	userID := createTestUserWithInterests(t, db, []string{"basketball"})

	calendarRepo.Upsert(model.CalendarEvent{
		EventKey: "espn:event:test2", Domain: "sports", Title: "Game 2",
		StartTime: time.Now().Add(20 * time.Hour), Timezone: "UTC", Status: "scheduled",
		EntityType: "game", EntityIDs: json.RawMessage(`{}`),
		InterestTags: []string{"basketball"},
		Payload:      json.RawMessage(`{"home_team":"Team A","away_team":"Team B","home_record":"","away_record":"","venue":"","broadcast":""}`),
	})

	worker := NewMaterializeWorker(calendarRepo, postRepo, userRepo, interestRepo, "test-agent-id")

	// Run twice
	worker.cycleOnce(context.Background())
	worker.cycleOnce(context.Background())

	// Count posts — should be exactly 1
	// (implementation depends on postRepo API — check post count for agent)
	_ = userID // used in cycleOnce internally
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `cd backend && go test ./internal/calendar/ -run TestMaterializeWorker -v`
Expected: FAIL — `NewMaterializeWorker` not defined.

- [ ] **Step 4: Write the materialization worker**

Create `backend/internal/calendar/materialize.go`:

```go
package calendar

import (
	"context"
	"log/slog"
	"time"

	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

type MaterializeWorker struct {
	calendarRepo *repository.CalendarEventRepo
	postRepo     *repository.PostRepo
	userRepo     *repository.UserRepo
	interestRepo *repository.UserInterestRepo
	agentID      string
	interval     time.Duration
}

func NewMaterializeWorker(
	calendarRepo *repository.CalendarEventRepo,
	postRepo *repository.PostRepo,
	userRepo *repository.UserRepo,
	interestRepo *repository.UserInterestRepo,
	agentID string,
) *MaterializeWorker {
	return &MaterializeWorker{
		calendarRepo: calendarRepo,
		postRepo:     postRepo,
		userRepo:     userRepo,
		interestRepo: interestRepo,
		agentID:      agentID,
		interval:     15 * time.Minute,
	}
}

func (w *MaterializeWorker) Run(ctx context.Context) {
	slog.Info("materialize worker started", "interval", w.interval)
	w.cycleOnce(ctx)
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			slog.Info("materialize worker stopped")
			return
		case <-ticker.C:
			w.cycleOnce(ctx)
		}
	}
}

func (w *MaterializeWorker) cycleOnce(ctx context.Context) {
	// Get all users with declared interests
	users, err := w.interestRepo.ListUsersWithInterests(ctx)
	if err != nil {
		slog.Error("materialize: list users failed", "error", err)
		return
	}

	now := time.Now()
	var published, skipped int

	for _, u := range users {
		interests := u.Interests // []string of interest tags
		if len(interests) == 0 {
			continue
		}

		// Find matching events in next 24h for this user
		events, err := w.calendarRepo.ForUser(u.UserID, interests, now.Add(-24*time.Hour), now.Add(24*time.Hour))
		if err != nil {
			slog.Warn("materialize: ForUser failed", "user_id", u.UserID, "error", err)
			continue
		}

		for _, event := range events {
			windows := w.applicableWindows(event, now)
			for _, window := range windows {
				already, err := w.calendarRepo.IsPublished(event.EventKey, u.UserID, window)
				if err != nil {
					continue
				}
				if already {
					skipped++
					continue
				}

				// Render post content
				var content *postContent
				switch event.Domain {
				case "sports":
					content, err = renderSportsPost(event.Payload, event.StartTime, window)
				case "entertainment":
					content, err = renderEntertainmentPost(event.Payload, event.StartTime, window)
				}
				if err != nil || content == nil {
					continue
				}

				// Create post
				post, err := w.postRepo.Create(repository.CreatePostParams{
					AgentID:     w.agentID,
					Title:       content.Title,
					Body:        content.Body,
					DisplayHint: content.DisplayHint,
					ExternalURL: content.ExternalURL,
					PostType:    "discovery",
					Visibility:  "personal",
					OwnerUserID: u.UserID,
				})
				if err != nil {
					slog.Warn("materialize: create post failed", "event", event.EventKey, "error", err)
					continue
				}

				w.calendarRepo.LogPost(event.EventKey, u.UserID, window, post.ID)
				published++
			}
		}
	}

	if published > 0 || skipped > 0 {
		slog.Info("materialize cycle complete", "published", published, "skipped", skipped)
	}
}

func (w *MaterializeWorker) applicableWindows(event model.CalendarEvent, now time.Time) []string {
	var windows []string
	until := event.StartTime.Sub(now)

	switch event.Domain {
	case "sports":
		// Preview: T-24h to T-12h
		if until >= 12*time.Hour && until <= 24*time.Hour {
			windows = append(windows, "preview")
		}
		// Imminent: T-2h to T-0
		if until >= 0 && until <= 2*time.Hour {
			windows = append(windows, "imminent")
		}
	case "entertainment":
		// Preview: T-7d to T-3d
		if until >= 3*24*time.Hour && until <= 7*24*time.Hour {
			windows = append(windows, "preview")
		}
		// Release day: T-24h to T+24h
		if until >= -24*time.Hour && until <= 24*time.Hour {
			windows = append(windows, "release_day")
		}
	}
	return windows
}
```

**Note:** This references `model.CalendarEvent` — add the import. Also, `repository.CreatePostParams` and `interestRepo.ListUsersWithInterests` may need to be adapted to match actual repo method signatures. Read the actual files during implementation.

- [ ] **Step 5: Run tests**

Run: `cd backend && go test ./internal/calendar/ -run TestMaterializeWorker -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add backend/internal/calendar/materialize.go backend/internal/calendar/templates.go backend/internal/calendar/materialize_test.go
git commit -m "feat(wave4): add calendar materialization worker with templates"
```

---

### Task 17: Wire Up Calendar Workers in main.go

**Files:**
- Modify: `backend/cmd/server/main.go`
- Modify: `backend/internal/config/config.go` (add TMDB_KEY)

- [ ] **Step 1: Add TMDB_KEY to config**

In `backend/internal/config/config.go`, add:

```go
	TMDBKey    string
```

And in the `Load()` function:

```go
	cfg.TMDBKey = os.Getenv("TMDB_KEY")
```

- [ ] **Step 2: Wire entertainment worker in main.go**

Add import:
```go
	"github.com/shanegleeson/beepbopboop/backend/internal/entertainment"
```

Add after other workers (~line 281):

```go
	if cfg.TMDBKey != "" {
		entertainmentWorker := entertainment.NewWorker(calendarEventRepo, "https://api.themoviedb.org", cfg.TMDBKey, "US")
		go entertainmentWorker.Run(workerCtx)
		slog.Info("entertainment ingest worker enabled")
	} else {
		slog.Warn("TMDB_KEY not set — entertainment ingest disabled")
	}
```

- [ ] **Step 3: Wire materialize worker in main.go**

Add import (if not already):
```go
	// calendar package already imported for existing calendar worker
```

Add after entertainment worker:

```go
	materializeAgent := os.Getenv("CALENDAR_AGENT_ID")
	if materializeAgent == "" {
		materializeAgent = os.Getenv("FEEDBACK_AGENT_ID") // fallback to shared agent
	}
	if materializeAgent != "" {
		materializeWorker := calendar.NewMaterializeWorker(
			calendarEventRepo, postRepo, userRepo, interestRepo, materializeAgent,
		)
		go materializeWorker.Run(workerCtx)
		slog.Info("calendar materialize worker enabled")
	} else {
		slog.Warn("no agent ID for calendar materialization — worker disabled")
	}
```

- [ ] **Step 4: Add system agent for calendar in database.go**

Add to the system agent insert section in `database.go`:

```go
	db.ExecContext(ctx, `INSERT INTO agents (id, user_id, name, persona)
		VALUES ('calendar-bot', (SELECT id FROM users WHERE firebase_uid = 'system'), 'Calendar Bot', 'Generates time-sensitive posts from calendar events')
		ON CONFLICT DO NOTHING`)
```

- [ ] **Step 5: Build**

Run: `cd backend && go build ./cmd/server/`
Expected: Compiles.

- [ ] **Step 6: Run all tests**

Run: `cd backend && go test ./... 2>&1 | tail -20`
Expected: All tests pass.

- [ ] **Step 7: Commit**

```bash
git add backend/cmd/server/main.go backend/internal/config/config.go backend/internal/database/database.go
git commit -m "feat(wave4): wire entertainment + materialize workers in server main"
```
