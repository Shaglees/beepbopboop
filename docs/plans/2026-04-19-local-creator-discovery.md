# Local Creator Discovery Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement a "research-once, serve-many" local creator discovery system — a `local_creators` table, two API endpoints, a new `creator_spotlight` display hint, and an iOS card to render it.

**Architecture:** Agents write discovered creator profiles to the `local_creators` table via `POST /creators` (agent-auth). iOS clients query cached results via `GET /creators/nearby` (Firebase-auth), which returns synthetic `creator_spotlight` posts ordered by distance. Density-aware adaptive radius expands automatically if too few results are found.

**Tech Stack:** Go (chi router, pq), PostgreSQL (inline migration pattern), SwiftUI, Codable, URLSession, AsyncImage.

---

## Context

### How migrations work
All schema changes are inline in `backend/internal/database/database.go` using `db.Exec("ALTER TABLE ... / CREATE TABLE IF NOT EXISTS ...")`. Follow this same pattern — no separate migration files.

### How display hints flow end-to-end
1. Agent calls `POST /posts` with `display_hint: "creator_spotlight"` **or** `POST /creators`.
2. Backend stores in DB, returns post JSON with `"display_hint": "creator_spotlight"`.
3. iOS decodes `displayHint` string → `Post.displayHintValue` enum via the switch in `Post.swift:196`.
4. `FeedItemView.cardContent` switches on `displayHintValue` → renders `CreatorSpotlightCard`.
5. `PostDetailView.detailContent` switches → renders `CreatorSpotlightDetailView`.

### Where data lives in a post for this hint
- `post.title` → creator name
- `post.body` → bio
- `post.imageURL` → creator photo
- `post.locality` → area name (e.g., "Williamsburg, Brooklyn")
- `post.latitude / post.longitude` → coordinates
- `post.externalURL` → JSON payload (see `CreatorData` struct below)

### Adaptive radius logic
`ListNearby` tries three radius tiers: `[baseRadius, baseRadius*3, baseRadius*10]` capped at 100 km. It returns on the first tier that yields ≥ 10 results (or the last tier if none hit threshold).

---

## Task 1: DB Migration — `local_creators` table

**Files:**
- Modify: `backend/internal/database/database.go` (after the `user_feedback` block, ~line 134)

**Step 1: Write the failing test**

Create `backend/internal/database/database_test.go` — check if the table already has tests. If the file exists, append to it. Write:

```go
func TestLocalCreatorsTableExists(t *testing.T) {
    db := database.OpenTestDB(t)
    _, err := db.Exec(`INSERT INTO local_creators (name, designation, source) VALUES ('Test Creator', 'Painter', 'test')`)
    if err != nil {
        t.Fatalf("local_creators table missing or wrong schema: %v", err)
    }
}
```

**Step 2: Run test to verify it fails**

```bash
cd backend && go test ./internal/database/... -run TestLocalCreatorsTableExists -v
```
Expected: FAIL — "relation "local_creators" does not exist"

**Step 3: Add the migration to database.go**

After the `user_feedback` block (after line ~134), add:

```go
// local_creators: cached creator profiles from agent research (research-once, serve-many)
db.Exec(`CREATE TABLE IF NOT EXISTS local_creators (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name          TEXT NOT NULL,
    designation   TEXT NOT NULL,
    bio           TEXT,
    lat           DOUBLE PRECISION,
    lon           DOUBLE PRECISION,
    area_name     TEXT,
    links         JSONB,
    notable_works TEXT,
    tags          TEXT[],
    source        TEXT NOT NULL,
    image_url     TEXT,
    discovered_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    verified_at   TIMESTAMPTZ,
    UNIQUE (name, lat, lon)
)`)
db.Exec("CREATE INDEX IF NOT EXISTS idx_local_creators_geo ON local_creators (lat, lon)")
db.Exec("CREATE INDEX IF NOT EXISTS idx_local_creators_designation ON local_creators (designation)")
```

**Step 4: Run test to verify it passes**

```bash
cd backend && go test ./internal/database/... -run TestLocalCreatorsTableExists -v
```
Expected: PASS

**Step 5: Commit**

```bash
git add backend/internal/database/database.go backend/internal/database/database_test.go
git commit -m "feat: add local_creators table migration"
```

---

## Task 2: Backend Model — `LocalCreator` struct

**Files:**
- Modify: `backend/internal/model/model.go` (append after `FeedbackSummary` block)

**Step 1: Add the struct (no test needed — pure data, no logic)**

Append to `backend/internal/model/model.go`:

```go
// LocalCreator is a cached creator profile discovered by agent research.
type LocalCreator struct {
    ID           string          `json:"id"`
    Name         string          `json:"name"`
    Designation  string          `json:"designation"`
    Bio          string          `json:"bio,omitempty"`
    Lat          *float64        `json:"lat,omitempty"`
    Lon          *float64        `json:"lon,omitempty"`
    AreaName     string          `json:"area_name,omitempty"`
    Links        json.RawMessage `json:"links,omitempty"`
    NotableWorks string          `json:"notable_works,omitempty"`
    Tags         []string        `json:"tags,omitempty"`
    Source       string          `json:"source"`
    ImageURL     string          `json:"image_url,omitempty"`
    DiscoveredAt time.Time       `json:"discovered_at"`
    VerifiedAt   *time.Time      `json:"verified_at,omitempty"`
}

// CreateCreatorRequest is the agent-facing request body for POST /creators.
type CreateCreatorRequest struct {
    Name         string          `json:"name"`
    Designation  string          `json:"designation"`
    Bio          string          `json:"bio"`
    Lat          *float64        `json:"lat"`
    Lon          *float64        `json:"lon"`
    AreaName     string          `json:"area_name"`
    Links        json.RawMessage `json:"links"`
    NotableWorks string          `json:"notable_works"`
    Tags         []string        `json:"tags"`
    Source       string          `json:"source"`
    ImageURL     string          `json:"image_url"`
}
```

**Step 2: Verify it compiles**

```bash
cd backend && go build ./...
```
Expected: no errors

**Step 3: Commit**

```bash
git add backend/internal/model/model.go
git commit -m "feat: add LocalCreator and CreateCreatorRequest models"
```

---

## Task 3: Backend Repository — `LocalCreatorRepo`

**Files:**
- Create: `backend/internal/repository/creator_repo.go`
- Create: `backend/internal/repository/creator_repo_test.go`

**Step 1: Write the failing tests**

Create `backend/internal/repository/creator_repo_test.go`:

```go
package repository_test

import (
    "fmt"
    "testing"

    "github.com/shanegleeson/beepbopboop/backend/internal/database"
    "github.com/shanegleeson/beepbopboop/backend/internal/model"
    "github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

func floatPtr(f float64) *float64 { return &f }

func TestLocalCreatorRepo_UpsertAndListNearby(t *testing.T) {
    db := database.OpenTestDB(t)
    repo := repository.NewLocalCreatorRepo(db)

    lat, lon := 40.7128, -74.0060
    got, err := repo.Upsert(model.CreateCreatorRequest{
        Name:        "Maria Chen",
        Designation: "Painter",
        Bio:         "Brooklyn-based oil painter.",
        Lat:         floatPtr(lat),
        Lon:         floatPtr(lon),
        AreaName:    "Brooklyn, NY",
        Source:      "Brooklyn Rail",
    })
    if err != nil {
        t.Fatalf("Upsert failed: %v", err)
    }
    if got.ID == "" {
        t.Error("expected non-empty ID after upsert")
    }
    if got.Name != "Maria Chen" {
        t.Errorf("expected name Maria Chen, got %s", got.Name)
    }

    creators, usedRadius, err := repo.ListNearby(lat, lon, 25.0, 20)
    if err != nil {
        t.Fatalf("ListNearby failed: %v", err)
    }
    if len(creators) != 1 {
        t.Errorf("expected 1 creator, got %d", len(creators))
    }
    if creators[0].Name != "Maria Chen" {
        t.Errorf("expected Maria Chen, got %s", creators[0].Name)
    }
    if usedRadius <= 0 {
        t.Error("expected positive usedRadius")
    }
}

func TestLocalCreatorRepo_Upsert_Idempotent(t *testing.T) {
    db := database.OpenTestDB(t)
    repo := repository.NewLocalCreatorRepo(db)

    lat, lon := 40.7128, -74.0060
    req := model.CreateCreatorRequest{
        Name:        "Jane Doe",
        Designation: "Sculptor",
        Lat:         floatPtr(lat),
        Lon:         floatPtr(lon),
        Source:      "test",
    }

    first, err := repo.Upsert(req)
    if err != nil {
        t.Fatalf("first upsert failed: %v", err)
    }

    // Update bio on second upsert
    req.Bio = "Updated bio"
    second, err := repo.Upsert(req)
    if err != nil {
        t.Fatalf("second upsert failed: %v", err)
    }

    if first.ID != second.ID {
        t.Errorf("expected same ID on upsert, got %s vs %s", first.ID, second.ID)
    }
    if second.Bio != "Updated bio" {
        t.Errorf("expected updated bio, got %q", second.Bio)
    }
}

func TestLocalCreatorRepo_ListNearby_AdaptiveRadius(t *testing.T) {
    db := database.OpenTestDB(t)
    repo := repository.NewLocalCreatorRepo(db)

    baseLat, baseLon := 40.7128, -74.0060

    // 5 creators within 1km
    for i := 0; i < 5; i++ {
        _, err := repo.Upsert(model.CreateCreatorRequest{
            Name:        fmt.Sprintf("Near Creator %d", i),
            Designation: "Painter",
            Lat:         floatPtr(baseLat + float64(i)*0.001),
            Lon:         floatPtr(baseLon),
            Source:      "test",
        })
        if err != nil {
            t.Fatalf("upsert %d: %v", i, err)
        }
    }

    // 10 more creators ~30km away (lat +0.27 ≈ 30km)
    for i := 0; i < 10; i++ {
        _, err := repo.Upsert(model.CreateCreatorRequest{
            Name:        fmt.Sprintf("Far Creator %d", i),
            Designation: "Musician",
            Lat:         floatPtr(baseLat + 0.27 + float64(i)*0.001),
            Lon:         floatPtr(baseLon),
            Source:      "test",
        })
        if err != nil {
            t.Fatalf("upsert far %d: %v", i, err)
        }
    }

    // Starting radius 5km — only 5 near results, below threshold of 10.
    // Should expand to 15km, still only 5. Then 50km, catches all 15.
    creators, usedRadius, err := repo.ListNearby(baseLat, baseLon, 5.0, 30)
    if err != nil {
        t.Fatalf("ListNearby: %v", err)
    }
    if usedRadius <= 5.0 {
        t.Errorf("expected radius to expand beyond 5km, got %.1f", usedRadius)
    }
    if len(creators) < 10 {
        t.Errorf("expected ≥10 creators after radius expansion, got %d", len(creators))
    }
}

func TestLocalCreatorRepo_ListNearby_OutOfRange(t *testing.T) {
    db := database.OpenTestDB(t)
    repo := repository.NewLocalCreatorRepo(db)

    // Creator in Brooklyn
    _, err := repo.Upsert(model.CreateCreatorRequest{
        Name:        "Local Artist",
        Designation: "Painter",
        Lat:         floatPtr(40.7128),
        Lon:         floatPtr(-74.0060),
        Source:      "test",
    })
    if err != nil {
        t.Fatalf("upsert: %v", err)
    }

    // Query from London — should return zero results even after expansion
    creators, _, err := repo.ListNearby(51.5074, -0.1278, 25.0, 20)
    if err != nil {
        t.Fatalf("ListNearby: %v", err)
    }
    if len(creators) != 0 {
        t.Errorf("expected 0 creators from London, got %d", len(creators))
    }
}
```

**Step 2: Run tests to verify they fail**

```bash
cd backend && go test ./internal/repository/... -run TestLocalCreatorRepo -v
```
Expected: FAIL — "NewLocalCreatorRepo undefined"

**Step 3: Create `backend/internal/repository/creator_repo.go`**

```go
package repository

import (
    "database/sql"
    "encoding/json"
    "time"

    "github.com/shanegleeson/beepbopboop/backend/internal/geo"
    "github.com/shanegleeson/beepbopboop/backend/internal/model"
)

const (
    nearbyMinResults = 10
    nearbyMaxRadius  = 100.0
)

type LocalCreatorRepo struct {
    db *sql.DB
}

func NewLocalCreatorRepo(db *sql.DB) *LocalCreatorRepo {
    return &LocalCreatorRepo{db: db}
}

func (r *LocalCreatorRepo) Upsert(req model.CreateCreatorRequest) (model.LocalCreator, error) {
    linksJSON, err := json.Marshal(req.Links)
    if err != nil {
        linksJSON = []byte("null")
    }
    if req.Links == nil {
        linksJSON = []byte("null")
    }

    var tagsJSON []byte
    if len(req.Tags) > 0 {
        tagsJSON, err = json.Marshal(req.Tags)
        if err != nil {
            tagsJSON = []byte("null")
        }
    }

    row := r.db.QueryRow(`
        INSERT INTO local_creators (name, designation, bio, lat, lon, area_name, links, notable_works, tags, source, image_url)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9::text[]::text[], $10, $11)
        ON CONFLICT (name, lat, lon) DO UPDATE SET
            designation   = EXCLUDED.designation,
            bio           = EXCLUDED.bio,
            area_name     = EXCLUDED.area_name,
            links         = EXCLUDED.links,
            notable_works = EXCLUDED.notable_works,
            tags          = EXCLUDED.tags,
            source        = EXCLUDED.source,
            image_url     = EXCLUDED.image_url
        RETURNING id, name, designation, bio, lat, lon, area_name, links, notable_works, tags, source, image_url, discovered_at, verified_at`,
        req.Name, req.Designation, req.Bio, req.Lat, req.Lon, req.AreaName,
        linksJSON, req.NotableWorks, tagsJSON, req.Source, req.ImageURL,
    )
    return scanCreator(row)
}

func (r *LocalCreatorRepo) ListNearby(lat, lon, baseRadiusKm float64, limit int) ([]model.LocalCreator, float64, error) {
    tiers := []float64{baseRadiusKm, baseRadiusKm * 3, baseRadiusKm * 10}
    for i, radius := range tiers {
        if radius > nearbyMaxRadius {
            radius = nearbyMaxRadius
            tiers[i] = radius
        }
        creators, err := r.queryWithRadius(lat, lon, radius, limit)
        if err != nil {
            return nil, 0, err
        }
        if len(creators) >= nearbyMinResults || radius >= nearbyMaxRadius {
            return creators, radius, nil
        }
    }
    return nil, 0, nil
}

func (r *LocalCreatorRepo) queryWithRadius(lat, lon, radiusKm float64, limit int) ([]model.LocalCreator, error) {
    minLat, maxLat, minLon, maxLon := geo.BoundingBox(lat, lon, radiusKm)
    rows, err := r.db.Query(`
        SELECT id, name, designation, bio, lat, lon, area_name, links, notable_works, tags, source, image_url, discovered_at, verified_at
        FROM local_creators
        WHERE lat BETWEEN $1 AND $2
          AND lon BETWEEN $3 AND $4
        ORDER BY discovered_at DESC
        LIMIT $5`,
        minLat, maxLat, minLon, maxLon, limit*5,
    )
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var results []model.LocalCreator
    for rows.Next() {
        c, err := scanCreatorRow(rows)
        if err != nil {
            return nil, err
        }
        if c.Lat != nil && c.Lon != nil {
            if geo.HaversineKm(lat, lon, *c.Lat, *c.Lon) > radiusKm {
                continue
            }
        }
        results = append(results, c)
        if len(results) >= limit {
            break
        }
    }
    return results, rows.Err()
}

func scanCreator(row *sql.Row) (model.LocalCreator, error) {
    var c model.LocalCreator
    var bio, areaName, notableWorks, source, imageURL sql.NullString
    var linksJSON []byte
    var tagsJSON []byte
    var verifiedAt sql.NullTime

    err := row.Scan(
        &c.ID, &c.Name, &c.Designation, &bio, &c.Lat, &c.Lon,
        &areaName, &linksJSON, &notableWorks, &tagsJSON,
        &source, &imageURL, &c.DiscoveredAt, &verifiedAt,
    )
    if err != nil {
        return model.LocalCreator{}, err
    }
    c.Bio = bio.String
    c.AreaName = areaName.String
    c.NotableWorks = notableWorks.String
    c.Source = source.String
    c.ImageURL = imageURL.String
    if verifiedAt.Valid {
        t := verifiedAt.Time
        c.VerifiedAt = &t
    }
    if len(linksJSON) > 0 && string(linksJSON) != "null" {
        c.Links = json.RawMessage(linksJSON)
    }
    if len(tagsJSON) > 0 && string(tagsJSON) != "null" {
        json.Unmarshal(tagsJSON, &c.Tags)
    }
    return c, nil
}

func scanCreatorRow(rows *sql.Rows) (model.LocalCreator, error) {
    var c model.LocalCreator
    var bio, areaName, notableWorks, source, imageURL sql.NullString
    var linksJSON []byte
    var tagsJSON []byte
    var verifiedAt sql.NullTime
    var discoveredAt time.Time

    err := rows.Scan(
        &c.ID, &c.Name, &c.Designation, &bio, &c.Lat, &c.Lon,
        &areaName, &linksJSON, &notableWorks, &tagsJSON,
        &source, &imageURL, &discoveredAt, &verifiedAt,
    )
    if err != nil {
        return model.LocalCreator{}, err
    }
    c.DiscoveredAt = discoveredAt
    c.Bio = bio.String
    c.AreaName = areaName.String
    c.NotableWorks = notableWorks.String
    c.Source = source.String
    c.ImageURL = imageURL.String
    if verifiedAt.Valid {
        t := verifiedAt.Time
        c.VerifiedAt = &t
    }
    if len(linksJSON) > 0 && string(linksJSON) != "null" {
        c.Links = json.RawMessage(linksJSON)
    }
    if len(tagsJSON) > 0 && string(tagsJSON) != "null" {
        json.Unmarshal(tagsJSON, &c.Tags)
    }
    return c, nil
}
```

**Step 4: Run tests to verify they pass**

```bash
cd backend && go test ./internal/repository/... -run TestLocalCreatorRepo -v
```
Expected: all 4 PASS

**Step 5: Commit**

```bash
git add backend/internal/repository/creator_repo.go backend/internal/repository/creator_repo_test.go
git commit -m "feat: add LocalCreatorRepo with adaptive-radius ListNearby"
```

---

## Task 4: Backend Handler — `CreatorsHandler`

**Files:**
- Create: `backend/internal/handler/creators.go`
- Create: `backend/internal/handler/creators_test.go`

**Step 1: Write failing tests**

Create `backend/internal/handler/creators_test.go`:

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
    "github.com/shanegleeson/beepbopboop/backend/internal/model"
    "github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

func TestCreatorsHandler_Create(t *testing.T) {
    db := database.OpenTestDB(t)
    userRepo := repository.NewUserRepo(db)
    agentRepo := repository.NewAgentRepo(db)
    creatorRepo := repository.NewLocalCreatorRepo(db)
    userSettingsRepo := repository.NewUserSettingsRepo(db)

    user, _ := userRepo.FindOrCreateByFirebaseUID("firebase-creator-test")
    agent, _ := agentRepo.Create(user.ID, "Local Scout")

    h := handler.NewCreatorsHandler(creatorRepo, userSettingsRepo)

    body := `{"name":"Maria Chen","designation":"Painter","bio":"Oil painter.","lat":40.7128,"lon":-74.0060,"area_name":"Brooklyn, NY","source":"Brooklyn Rail"}`
    req := httptest.NewRequest("POST", "/creators", bytes.NewBufferString(body))
    req = req.WithContext(middleware.WithAgentID(req.Context(), agent.ID))
    rec := httptest.NewRecorder()

    h.Create(rec, req)

    if rec.Code != http.StatusCreated {
        t.Errorf("expected 201, got %d: %s", rec.Code, rec.Body.String())
    }

    var resp model.LocalCreator
    json.NewDecoder(rec.Body).Decode(&resp)
    if resp.Name != "Maria Chen" {
        t.Errorf("expected Maria Chen, got %q", resp.Name)
    }
    if resp.ID == "" {
        t.Error("expected non-empty ID")
    }
}

func TestCreatorsHandler_Create_MissingRequired(t *testing.T) {
    db := database.OpenTestDB(t)
    userRepo := repository.NewUserRepo(db)
    agentRepo := repository.NewAgentRepo(db)
    creatorRepo := repository.NewLocalCreatorRepo(db)
    userSettingsRepo := repository.NewUserSettingsRepo(db)

    user, _ := userRepo.FindOrCreateByFirebaseUID("firebase-creator-test2")
    agent, _ := agentRepo.Create(user.ID, "Local Scout 2")

    h := handler.NewCreatorsHandler(creatorRepo, userSettingsRepo)

    // Missing required: name, designation, source
    body := `{"bio":"missing name and designation"}`
    req := httptest.NewRequest("POST", "/creators", bytes.NewBufferString(body))
    req = req.WithContext(middleware.WithAgentID(req.Context(), agent.ID))
    rec := httptest.NewRecorder()

    h.Create(rec, req)

    if rec.Code != http.StatusBadRequest {
        t.Errorf("expected 400, got %d", rec.Code)
    }
}

func TestCreatorsHandler_GetNearby(t *testing.T) {
    db := database.OpenTestDB(t)
    userRepo := repository.NewUserRepo(db)
    userSettingsRepo := repository.NewUserSettingsRepo(db)
    creatorRepo := repository.NewLocalCreatorRepo(db)

    user, _ := userRepo.FindOrCreateByFirebaseUID("firebase-nearby-test")

    lat, lon := 40.7128, -74.0060
    userSettingsRepo.Upsert(user.ID, model.UserSettings{
        UserID:    user.ID,
        Latitude:  &lat,
        Longitude: &lon,
        RadiusKm:  25.0,
    })

    creatorRepo.Upsert(model.CreateCreatorRequest{
        Name:        "Maria Chen",
        Designation: "Painter",
        Lat:         &lat,
        Lon:         &lon,
        Source:      "Brooklyn Rail",
    })

    h := handler.NewCreatorsHandler(creatorRepo, userSettingsRepo)

    req := httptest.NewRequest("GET", "/creators/nearby", nil)
    req = req.WithContext(middleware.WithUserID(req.Context(), user.ID))
    rec := httptest.NewRecorder()

    h.GetNearby(rec, req)

    if rec.Code != http.StatusOK {
        t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
    }

    var resp model.FeedResponse
    json.NewDecoder(rec.Body).Decode(&resp)
    if len(resp.Posts) != 1 {
        t.Errorf("expected 1 post, got %d", len(resp.Posts))
    }
    if resp.Posts[0].Title != "Maria Chen" {
        t.Errorf("expected title Maria Chen, got %q", resp.Posts[0].Title)
    }
    if resp.Posts[0].DisplayHint != "creator_spotlight" {
        t.Errorf("expected creator_spotlight, got %q", resp.Posts[0].DisplayHint)
    }
}

func TestCreatorsHandler_GetNearby_NoLocation(t *testing.T) {
    db := database.OpenTestDB(t)
    userRepo := repository.NewUserRepo(db)
    userSettingsRepo := repository.NewUserSettingsRepo(db)
    creatorRepo := repository.NewLocalCreatorRepo(db)

    user, _ := userRepo.FindOrCreateByFirebaseUID("firebase-noloc-test")
    // No location set in settings

    h := handler.NewCreatorsHandler(creatorRepo, userSettingsRepo)

    req := httptest.NewRequest("GET", "/creators/nearby", nil)
    req = req.WithContext(middleware.WithUserID(req.Context(), user.ID))
    rec := httptest.NewRecorder()

    h.GetNearby(rec, req)

    if rec.Code != http.StatusUnprocessableEntity {
        t.Errorf("expected 422 when no location set, got %d", rec.Code)
    }
}
```

**Step 2: Run tests to verify they fail**

```bash
cd backend && go test ./internal/handler/... -run TestCreatorsHandler -v
```
Expected: FAIL — "NewCreatorsHandler undefined"

**Step 3: Create `backend/internal/handler/creators.go`**

```go
package handler

import (
    "encoding/json"
    "net/http"

    "github.com/shanegleeson/beepbopboop/backend/internal/middleware"
    "github.com/shanegleeson/beepbopboop/backend/internal/model"
    "github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

type CreatorsHandler struct {
    creatorRepo      *repository.LocalCreatorRepo
    userSettingsRepo *repository.UserSettingsRepo
}

func NewCreatorsHandler(creatorRepo *repository.LocalCreatorRepo, userSettingsRepo *repository.UserSettingsRepo) *CreatorsHandler {
    return &CreatorsHandler{
        creatorRepo:      creatorRepo,
        userSettingsRepo: userSettingsRepo,
    }
}

// Create handles POST /creators (agent-auth). Upserts a creator profile.
func (h *CreatorsHandler) Create(w http.ResponseWriter, r *http.Request) {
    var req model.CreateCreatorRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "invalid JSON", http.StatusBadRequest)
        return
    }
    if req.Name == "" || req.Designation == "" || req.Source == "" {
        http.Error(w, "name, designation, and source are required", http.StatusBadRequest)
        return
    }

    creator, err := h.creatorRepo.Upsert(req)
    if err != nil {
        http.Error(w, "failed to save creator", http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(creator)
}

// GetNearby handles GET /creators/nearby (Firebase-auth). Returns cached creators
// near the user's stored location as creator_spotlight posts.
func (h *CreatorsHandler) GetNearby(w http.ResponseWriter, r *http.Request) {
    userID := middleware.UserIDFromContext(r.Context())

    settings, err := h.userSettingsRepo.Get(userID)
    if err != nil || settings.Latitude == nil || settings.Longitude == nil {
        http.Error(w, "location not set — update your settings first", http.StatusUnprocessableEntity)
        return
    }

    radius := settings.RadiusKm
    if radius <= 0 {
        radius = 25.0
    }

    creators, _, err := h.creatorRepo.ListNearby(*settings.Latitude, *settings.Longitude, radius, 50)
    if err != nil {
        http.Error(w, "failed to query creators", http.StatusInternalServerError)
        return
    }

    posts := make([]model.Post, 0, len(creators))
    for _, c := range creators {
        posts = append(posts, creatorToPost(c))
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(model.FeedResponse{Posts: posts})
}

func creatorToPost(c model.LocalCreator) model.Post {
    payload := map[string]any{
        "designation":   c.Designation,
        "notable_works": c.NotableWorks,
        "tags":          c.Tags,
        "source":        c.Source,
        "area_name":     c.AreaName,
    }
    if len(c.Links) > 0 {
        var links any
        json.Unmarshal(c.Links, &links)
        payload["links"] = links
    }
    externalJSON, _ := json.Marshal(payload)

    p := model.Post{
        ID:          c.ID,
        AgentID:     "system",
        AgentName:   "Local Creators",
        UserID:      "system",
        Title:       c.Name,
        Body:        c.Bio,
        ImageURL:    c.ImageURL,
        Locality:    c.AreaName,
        Latitude:    c.Lat,
        Longitude:   c.Lon,
        PostType:    "discovery",
        Visibility:  "public",
        DisplayHint: "creator_spotlight",
        ExternalURL: string(externalJSON),
        Status:      "published",
        CreatedAt:   c.DiscoveredAt,
    }
    return p
}
```

**Step 4: Run tests to verify they pass**

```bash
cd backend && go test ./internal/handler/... -run TestCreatorsHandler -v
```
Expected: all 4 PASS

**Note on `userSettingsRepo.Upsert`:** If the settings repo doesn't have an `Upsert` method, use the existing `Update` or `Save` method — check `backend/internal/repository/user_settings_repo.go` for the correct method name and adapt the test accordingly.

**Step 5: Commit**

```bash
git add backend/internal/handler/creators.go backend/internal/handler/creators_test.go
git commit -m "feat: add CreatorsHandler for POST /creators and GET /creators/nearby"
```

---

## Task 5: Backend Wiring — Routes + Display Hint

**Files:**
- Modify: `backend/internal/handler/post.go` (add `creator_spotlight` to `ValidDisplayHints`)
- Modify: `backend/cmd/server/main.go` (add routes + handler instantiation)

**Step 1: Add `creator_spotlight` to valid display hints**

In `backend/internal/handler/post.go`, add to the `ValidDisplayHints` map:

```go
"creator_spotlight": true,
```

Add it after `"feedback": true` at line ~61.

**Step 2: Verify hint validation test exists or add one**

Check `backend/internal/handler/post_test.go` for a `TestCreatePost_InvalidDisplayHint` test. The `creator_spotlight` hint should now pass validation. Run:

```bash
cd backend && go test ./internal/handler/... -run TestPostHandler_ValidDisplayHint -v
```

If no such test exists, verify manually that `go build ./...` passes.

**Step 3: Wire `CreatorsHandler` into routes**

In `backend/cmd/server/main.go`:

1. Find where other handlers are instantiated (around line 70–100). Add:
```go
creatorRepo := repository.NewLocalCreatorRepo(db)
creatorsH := handler.NewCreatorsHandler(creatorRepo, userSettingsRepo)
```

2. In the Firebase-authenticated group (around line 126), add:
```go
r.Get("/creators/nearby", creatorsH.GetNearby)
```

3. In the agent-authenticated group (around line 143), add:
```go
r.Post("/creators", creatorsH.Create)
```

**Step 4: Build to verify compilation**

```bash
cd backend && go build ./...
```
Expected: no errors

**Step 5: Run all backend tests**

```bash
cd backend && go test ./...
```
Expected: all PASS

**Step 6: Commit**

```bash
git add backend/internal/handler/post.go backend/cmd/server/main.go
git commit -m "feat: wire creators routes and add creator_spotlight display hint"
```

---

## Task 6: iOS Model — `CreatorData` + `Post.swift` updates

**Files:**
- Create: `beepbopboop/beepbopboop/Models/CreatorData.swift`
- Modify: `beepbopboop/beepbopboop/Models/Post.swift`

**Step 1: Create `CreatorData.swift`**

```swift
import Foundation

struct CreatorData: Codable {
    let designation: String
    let links: CreatorLinks?
    let notableWorks: String?
    let tags: [String]?
    let source: String?
    let areaName: String?

    enum CodingKeys: String, CodingKey {
        case designation
        case links
        case notableWorks = "notable_works"
        case tags
        case source
        case areaName = "area_name"
    }
}

struct CreatorLinks: Codable {
    let website: String?
    let instagram: String?
    let bandcamp: String?
    let etsy: String?
    let substack: String?
    let soundcloud: String?
    let behance: String?
}
```

**Step 2: Update `Post.swift`**

**2a. Add `creatorSpotlight` to `DisplayHintValue` enum** (line 193, after `feedback`):

```swift
case creatorSpotlight
```

**2b. Add the switch case in `displayHintValue`** (after `"feedback"` case, before `default`):

```swift
case "creator_spotlight": return .creatorSpotlight
```

**2c. Add `creatorData` computed property** (after the `feedbackData` property, ~line 583):

```swift
var creatorData: CreatorData? {
    guard displayHintValue == .creatorSpotlight,
          let json = externalURL,
          let data = json.data(using: .utf8) else { return nil }
    return try? JSONDecoder().decode(CreatorData.self, from: data)
}
```

**2d. Add hint display properties** — find the `hintColor` switch in Post.swift (~line 279) and add:
```swift
case .creatorSpotlight: return Color(red: 0.541, green: 0.169, blue: 0.886)  // indigo-purple
```

Find the `hintIcon` switch and add:
```swift
case .creatorSpotlight: return "paintpalette"
```

Find the `hintLabel` switch and add:
```swift
case .creatorSpotlight: return "Local Creator"
```

**Step 3: Build the iOS project to verify compilation**

```bash
cd beepbopboop && xcodebuild -scheme beepbopboop -destination 'platform=iOS Simulator,name=iPhone 16' build CODE_SIGNING_ALLOWED=NO 2>&1 | grep -E "(error:|warning:|Build succeeded|BUILD SUCCEEDED)"
```
Expected: Build succeeded / BUILD SUCCEEDED

**Step 4: Commit**

```bash
git add beepbopboop/beepbopboop/Models/CreatorData.swift beepbopboop/beepbopboop/Models/Post.swift
git commit -m "feat: add CreatorData model and creatorSpotlight display hint to Post"
```

---

## Task 7: iOS Card — `CreatorSpotlightCard` + `CreatorSpotlightDetailView`

**Files:**
- Create: `beepbopboop/beepbopboop/Views/CreatorCards.swift`

**Step 1: Create the card file**

```swift
import SwiftUI

// MARK: - Palette

private let creatorIndigo  = Color(red: 0.541, green: 0.169, blue: 0.886)  // #8A2BE2
private let creatorPurple  = Color(red: 0.686, green: 0.400, blue: 0.961)  // #AF66F5
private let creatorCream   = Color(red: 0.976, green: 0.969, blue: 1.0)    // #F9F7FF

// MARK: - Feed Card

struct CreatorSpotlightCard: View {
    let post: Post
    let creator: CreatorData

    init?(post: Post) {
        guard post.displayHintValue == .creatorSpotlight,
              let cd = post.creatorData else { return nil }
        self.post = post
        self.creator = cd
    }

    var body: some View {
        VStack(spacing: 0) {
            photoSection
            infoSection
        }
        .background(creatorCream)
    }

    // MARK: Photo

    private var photoSection: some View {
        ZStack(alignment: .bottomLeading) {
            creatorPhoto
            designationBadge
                .padding(10)
        }
        .frame(height: 200)
        .clipped()
    }

    @ViewBuilder
    private var creatorPhoto: some View {
        if let urlStr = post.imageURL, let url = URL(string: urlStr) {
            AsyncImage(url: url) { phase in
                switch phase {
                case .success(let image):
                    image
                        .resizable()
                        .aspectRatio(contentMode: .fill)
                        .frame(maxWidth: .infinity, maxHeight: 200)
                        .clipped()
                case .failure:
                    placeholderBackground
                default:
                    Color(red: 0.93, green: 0.90, blue: 0.97)
                        .overlay(ProgressView().tint(creatorIndigo))
                }
            }
        } else {
            placeholderBackground
        }
    }

    private var placeholderBackground: some View {
        creatorIndigo.opacity(0.15)
            .overlay(
                Image(systemName: "paintpalette")
                    .font(.system(size: 48))
                    .foregroundColor(creatorIndigo.opacity(0.5))
            )
    }

    private var designationBadge: some View {
        Text(creator.designation)
            .font(.caption.weight(.semibold))
            .foregroundColor(.white)
            .padding(.horizontal, 10)
            .padding(.vertical, 5)
            .background(creatorIndigo)
            .clipShape(Capsule())
    }

    // MARK: Info

    private var infoSection: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(post.title)
                .font(.headline)
                .foregroundColor(.primary)
                .lineLimit(1)

            if !post.body.isEmpty {
                Text(post.body)
                    .font(.subheadline)
                    .foregroundColor(.secondary)
                    .lineLimit(3)
            }

            if let locality = post.locality, !locality.isEmpty {
                HStack(spacing: 4) {
                    Image(systemName: "mappin.circle.fill")
                        .font(.caption)
                        .foregroundColor(creatorPurple)
                    Text(locality)
                        .font(.caption)
                        .foregroundColor(.secondary)
                }
            }

            if let source = creator.source, !source.isEmpty {
                Text("Found via \(source)")
                    .font(.caption2)
                    .foregroundColor(creatorPurple)
            }
        }
        .padding(12)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(creatorCream)
    }
}

// MARK: - Detail View

struct CreatorSpotlightDetailView: View {
    let post: Post
    @Environment(\.openURL) private var openURL

    var creator: CreatorData? { post.creatorData }

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 0) {
                heroSection
                    .frame(height: 280)
                    .clipped()

                VStack(alignment: .leading, spacing: 20) {
                    nameSection
                    if !post.body.isEmpty {
                        bioSection
                    }
                    if let links = creator?.links {
                        linksSection(links)
                    }
                    if let tags = creator?.tags, !tags.isEmpty {
                        tagsSection(tags)
                    }
                    if let works = creator?.notableWorks, !works.isEmpty {
                        worksSection(works)
                    }
                    sourceSection
                }
                .padding(20)
            }
        }
        .ignoresSafeArea(edges: .top)
    }

    // MARK: Hero

    @ViewBuilder
    private var heroSection: some View {
        if let urlStr = post.imageURL, let url = URL(string: urlStr) {
            AsyncImage(url: url) { phase in
                switch phase {
                case .success(let image):
                    image.resizable().aspectRatio(contentMode: .fill)
                default:
                    creatorIndigo.opacity(0.2)
                        .overlay(Image(systemName: "paintpalette").font(.system(size: 64)).foregroundColor(creatorIndigo.opacity(0.4)))
                }
            }
        } else {
            creatorIndigo.opacity(0.2)
                .overlay(Image(systemName: "paintpalette").font(.system(size: 64)).foregroundColor(creatorIndigo.opacity(0.4)))
        }
    }

    // MARK: Name + Designation

    private var nameSection: some View {
        VStack(alignment: .leading, spacing: 6) {
            HStack(alignment: .firstTextBaseline, spacing: 10) {
                Text(post.title)
                    .font(.title2.bold())
                if let d = creator?.designation {
                    Text(d)
                        .font(.subheadline.weight(.medium))
                        .foregroundColor(.white)
                        .padding(.horizontal, 10)
                        .padding(.vertical, 4)
                        .background(creatorIndigo)
                        .clipShape(Capsule())
                }
            }
            if let locality = post.locality, !locality.isEmpty {
                HStack(spacing: 4) {
                    Image(systemName: "mappin.circle.fill").foregroundColor(creatorPurple)
                    Text(locality).font(.subheadline).foregroundColor(.secondary)
                }
            }
        }
    }

    // MARK: Bio

    private var bioSection: some View {
        VStack(alignment: .leading, spacing: 6) {
            Text("About")
                .font(.subheadline.weight(.semibold))
                .foregroundColor(.secondary)
            Text(post.body)
                .font(.body)
        }
    }

    // MARK: Links

    private func linksSection(_ links: CreatorLinks) -> some View {
        VStack(alignment: .leading, spacing: 10) {
            Text("Links")
                .font(.subheadline.weight(.semibold))
                .foregroundColor(.secondary)
            FlowLayout(spacing: 8) {
                if let url = links.website {
                    linkChip(label: "Website", icon: "globe", urlString: url)
                }
                if let ig = links.instagram {
                    linkChip(label: "Instagram", icon: "camera", urlString: ig)
                }
                if let bc = links.bandcamp {
                    linkChip(label: "Bandcamp", icon: "music.note", urlString: bc)
                }
                if let etsy = links.etsy {
                    linkChip(label: "Etsy", icon: "bag", urlString: etsy)
                }
                if let sub = links.substack {
                    linkChip(label: "Substack", icon: "envelope", urlString: sub)
                }
                if let sc = links.soundcloud {
                    linkChip(label: "SoundCloud", icon: "waveform", urlString: sc)
                }
                if let bh = links.behance {
                    linkChip(label: "Behance", icon: "photo.on.rectangle", urlString: bh)
                }
            }
        }
    }

    private func linkChip(label: String, icon: String, urlString: String) -> some View {
        Button {
            if let url = URL(string: urlString) {
                openURL(url)
            }
        } label: {
            HStack(spacing: 5) {
                Image(systemName: icon).font(.caption)
                Text(label).font(.subheadline.weight(.medium))
            }
            .foregroundColor(creatorIndigo)
            .padding(.horizontal, 12)
            .padding(.vertical, 7)
            .background(creatorIndigo.opacity(0.1))
            .clipShape(Capsule())
        }
    }

    // MARK: Tags

    private func tagsSection(_ tags: [String]) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            Text("Tags")
                .font(.subheadline.weight(.semibold))
                .foregroundColor(.secondary)
            FlowLayout(spacing: 6) {
                ForEach(tags, id: \.self) { tag in
                    Text(tag)
                        .font(.caption.weight(.medium))
                        .foregroundColor(creatorPurple)
                        .padding(.horizontal, 10)
                        .padding(.vertical, 5)
                        .background(creatorPurple.opacity(0.12))
                        .clipShape(Capsule())
                }
            }
        }
    }

    // MARK: Notable Works

    private func worksSection(_ works: String) -> some View {
        VStack(alignment: .leading, spacing: 6) {
            Text("Notable Works")
                .font(.subheadline.weight(.semibold))
                .foregroundColor(.secondary)
            Text(works)
                .font(.body)
        }
    }

    // MARK: Source

    private var sourceSection: some View {
        Group {
            if let source = creator?.source, !source.isEmpty {
                HStack(spacing: 4) {
                    Image(systemName: "magnifyingglass").font(.caption)
                    Text("Discovered via \(source)")
                        .font(.caption)
                }
                .foregroundColor(.secondary)
            }
        }
    }
}

// MARK: - FlowLayout

/// Simple left-to-right wrapping layout for chips.
private struct FlowLayout: Layout {
    var spacing: CGFloat = 8

    func sizeThatFits(proposal: ProposedViewSize, subviews: Subviews, cache: inout ()) -> CGSize {
        let width = proposal.width ?? .infinity
        var x: CGFloat = 0
        var y: CGFloat = 0
        var rowHeight: CGFloat = 0
        for subview in subviews {
            let size = subview.sizeThatFits(.unspecified)
            if x + size.width > width && x > 0 {
                x = 0
                y += rowHeight + spacing
                rowHeight = 0
            }
            x += size.width + spacing
            rowHeight = max(rowHeight, size.height)
        }
        return CGSize(width: width, height: y + rowHeight)
    }

    func placeSubviews(in bounds: CGRect, proposal: ProposedViewSize, subviews: Subviews, cache: inout ()) {
        var x = bounds.minX
        var y = bounds.minY
        var rowHeight: CGFloat = 0
        for subview in subviews {
            let size = subview.sizeThatFits(.unspecified)
            if x + size.width > bounds.maxX && x > bounds.minX {
                x = bounds.minX
                y += rowHeight + spacing
                rowHeight = 0
            }
            subview.place(at: CGPoint(x: x, y: y), proposal: ProposedViewSize(size))
            x += size.width + spacing
            rowHeight = max(rowHeight, size.height)
        }
    }
}
```

**Step 2: Build to verify compilation**

```bash
cd beepbopboop && xcodebuild -scheme beepbopboop -destination 'platform=iOS Simulator,name=iPhone 16' build CODE_SIGNING_ALLOWED=NO 2>&1 | grep -E "(error:|Build succeeded|BUILD SUCCEEDED)"
```
Expected: BUILD SUCCEEDED

**Step 3: Commit**

```bash
git add beepbopboop/beepbopboop/Views/CreatorCards.swift
git commit -m "feat: add CreatorSpotlightCard and CreatorSpotlightDetailView"
```

---

## Task 8: iOS Wiring — `FeedItemView` + `PostDetailView`

**Files:**
- Modify: `beepbopboop/beepbopboop/Views/FeedItemView.swift`
- Modify: `beepbopboop/beepbopboop/Views/PostDetailView.swift`

**Step 1: Add `creatorSpotlight` to the styled card list in `FeedItemView`**

In `FeedItemView.swift` at line 16, find the array of display hint values that get the shadow treatment:

```swift
if [.outfit, .weather, ..., .feedback].contains(post.displayHintValue) {
```

Add `.creatorSpotlight` to this array.

**Step 2: Add switch case in `FeedItemView.cardContent`**

After the `.feedback` case (~line 146) and before `default:`, add:

```swift
case .creatorSpotlight:
    if let card = CreatorSpotlightCard(post: post) {
        card
    } else {
        StandardCard(post: post)
    }
```

**Step 3: Add switch case in `PostDetailView.detailContent`**

After the `.feedback` case (~line 75) and before `default:`, add:

```swift
case .creatorSpotlight:
    CreatorSpotlightDetailView(post: post)
```

**Step 4: Build to verify compilation**

```bash
cd beepbopboop && xcodebuild -scheme beepbopboop -destination 'platform=iOS Simulator,name=iPhone 16' build CODE_SIGNING_ALLOWED=NO 2>&1 | grep -E "(error:|Build succeeded|BUILD SUCCEEDED)"
```
Expected: BUILD SUCCEEDED

**Step 5: Commit**

```bash
git add beepbopboop/beepbopboop/Views/FeedItemView.swift beepbopboop/beepbopboop/Views/PostDetailView.swift
git commit -m "feat: wire creatorSpotlight into FeedItemView and PostDetailView"
```

---

## Task 9: APIService — `fetchNearbyCreators`

**Files:**
- Modify: `beepbopboop/beepbopboop/Services/APIService.swift`

**Step 1: Add the method**

Find the existing feed-fetching methods in `APIService.swift`. Following the same pattern, add:

```swift
func fetchNearbyCreators() async throws -> FeedResponse {
    let url = baseURL.appendingPathComponent("creators/nearby")
    var request = URLRequest(url: url)
    request.httpMethod = "GET"
    try await addAuthHeaders(&request)
    let (data, response) = try await URLSession.shared.data(for: request)
    try validateResponse(response)
    return try JSONDecoder().decode(FeedResponse.self, from: data)
}
```

Note: Look at an existing feed method (e.g., `fetchCommunityFeed`) to copy the exact pattern for `addAuthHeaders` and `validateResponse` — these may have different names in this codebase. Match the existing pattern exactly.

**Step 2: Build to verify compilation**

```bash
cd beepbopboop && xcodebuild -scheme beepbopboop -destination 'platform=iOS Simulator,name=iPhone 16' build CODE_SIGNING_ALLOWED=NO 2>&1 | grep -E "(error:|Build succeeded|BUILD SUCCEEDED)"
```
Expected: BUILD SUCCEEDED

**Step 3: Commit**

```bash
git add beepbopboop/beepbopboop/Services/APIService.swift
git commit -m "feat: add fetchNearbyCreators to APIService"
```

---

## Task 10: Final Validation

**Step 1: Run all backend tests**

```bash
cd backend && go test ./... -v 2>&1 | tail -20
```
Expected: all PASS, no FAIL

**Step 2: Build iOS**

```bash
cd beepbopboop && xcodebuild -scheme beepbopboop -destination 'platform=iOS Simulator,name=iPhone 16' build CODE_SIGNING_ALLOWED=NO 2>&1 | grep -E "(error:|Build succeeded|BUILD SUCCEEDED)"
```
Expected: BUILD SUCCEEDED

**Step 3: Create the PR**

```bash
gh pr create \
  --title "feat: local creator discovery (#84)" \
  --body "$(cat <<'EOF'
## What

Implements [#84](https://github.com/Shaglees/beepbopboop/issues/84) — local creator discovery.

**Backend:**
- `local_creators` table (UUID PK, geo index, UNIQUE on name+lat+lon)
- `LocalCreatorRepo` with `Upsert` (ON CONFLICT DO UPDATE) and `ListNearby` (density-aware adaptive radius: tries baseRadius → 3x → 10x, capped at 100km, stops when ≥10 results)
- `CreatorsHandler` — `POST /creators` (agent-auth) + `GET /creators/nearby` (Firebase-auth)
- `creator_spotlight` added to `ValidDisplayHints`

**iOS:**
- `CreatorData` / `CreatorLinks` Codable structs (parsed from `externalURL`)
- `creatorSpotlight` case in `DisplayHintValue` enum
- `CreatorSpotlightCard` — photo hero, designation badge, bio, area, source
- `CreatorSpotlightDetailView` — full detail with tappable link chips, tags, notable works
- `FlowLayout` for wrapping chip rows
- `fetchNearbyCreators()` in `APIService`

## Test plan

- [ ] `go test ./...` passes
- [ ] iOS builds clean (no errors)
- [ ] `POST /creators` with agent token creates a record and returns 201
- [ ] `GET /creators/nearby` with Firebase token and user location set returns creator_spotlight posts
- [ ] `GET /creators/nearby` with no location set returns 422
- [ ] Adaptive radius expands when < 10 results
- [ ] `creator_spotlight` card renders in feed with photo, name, designation badge
- [ ] Tapping card opens CreatorSpotlightDetailView
- [ ] Link chips open URLs
- [ ] Fallback to StandardCard when `creatorData` is nil

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
)"
```

---

## Out of Scope (follow-up issues)

- **Staleness / re-verify logic** — 60-day re-verify, 6-month full re-research (issue mentions these)
- **Fuzzy dedup** — embedding-similarity dedup for near-duplicate names within a radius
- **In-feed UI section** — a "Local Creators" section in the Community or ForYou feed that auto-calls `fetchNearbyCreators`
- **Proactive research trigger** — agent detects strong location + arts interest and pre-seeds the DB
- **Opt-out mechanism** — `opt_out_requests` table for creators who want to be removed
- **`box_score` hint** — already in the iOS enum but missing from `ValidDisplayHints` (pre-existing)
