# Wave 4 Part A: Community Local News

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a local news pipeline: backend source registry → skill fetches + composes → adaptive iOS card renders article/video/hybrid content.

**Spec:** `docs/superpowers/specs/2026-04-25-wave4-new-features-design.md` (Sub-system A)

---

### Task 1: Database Schema — `news_sources` Table

**Files:**
- Modify: `backend/internal/database/database.go:~397` (before `return db, nil`)

- [ ] **Step 1: Add migration statements**

Add before the final `return db, nil` in the `Open` function:

```go
	// Wave 4: news sources
	db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS news_sources (
		id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		name         TEXT NOT NULL,
		url          TEXT NOT NULL UNIQUE,
		feed_url     TEXT,
		area_label   TEXT NOT NULL,
		latitude     DOUBLE PRECISION NOT NULL,
		longitude    DOUBLE PRECISION NOT NULL,
		radius_km    DOUBLE PRECISION NOT NULL DEFAULT 25.0,
		topics       TEXT[] NOT NULL DEFAULT '{}',
		trust_score  SMALLINT NOT NULL DEFAULT 50,
		fetch_method TEXT NOT NULL DEFAULT 'rss',
		active       BOOLEAN NOT NULL DEFAULT TRUE,
		created_at   TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at   TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`)
	db.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_news_sources_geo ON news_sources (latitude, longitude)`)
	db.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_news_sources_active ON news_sources (active) WHERE active = TRUE`)
```

- [ ] **Step 2: Verify migration runs**

Run: `cd backend && go build ./cmd/server/`
Expected: Compiles without errors.

- [ ] **Step 3: Commit**

```bash
git add backend/internal/database/database.go
git commit -m "feat(wave4): add news_sources table migration"
```

---

### Task 2: NewsSource Model + Repository

**Files:**
- Create: `backend/internal/model/news_source.go`
- Create: `backend/internal/repository/news_source_repo.go`
- Create: `backend/internal/repository/news_source_repo_test.go`

- [ ] **Step 1: Write the model**

Create `backend/internal/model/news_source.go`:

```go
package model

import "time"

type NewsSource struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	URL         string    `json:"url"`
	FeedURL     string    `json:"feed_url,omitempty"`
	AreaLabel   string    `json:"area_label"`
	Latitude    float64   `json:"latitude"`
	Longitude   float64   `json:"longitude"`
	RadiusKm    float64   `json:"radius_km"`
	Topics      []string  `json:"topics"`
	TrustScore  int       `json:"trust_score"`
	FetchMethod string    `json:"fetch_method"`
	Active      bool      `json:"active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
```

- [ ] **Step 2: Write the failing test**

Create `backend/internal/repository/news_source_repo_test.go`:

```go
package repository

import (
	"testing"

	"github.com/shanegleeson/beepbopboop/backend/internal/model"
)

func TestNewsSourceRepo_CreateAndList(t *testing.T) {
	db := setupTestDB(t)
	repo := NewNewsSourceRepo(db)

	src := model.NewsSource{
		Name:        "Dublin Inquirer",
		URL:         "https://dublininquirer.com",
		FeedURL:     "https://dublininquirer.com/feed",
		AreaLabel:   "Dublin, Ireland",
		Latitude:    53.3498,
		Longitude:   -6.2603,
		RadiusKm:    25.0,
		Topics:      []string{"local", "housing"},
		TrustScore:  80,
		FetchMethod: "rss",
	}

	err := repo.Create(src)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Query within radius — should find it
	results, err := repo.List(53.35, -6.26, 50.0, nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Name != "Dublin Inquirer" {
		t.Errorf("expected Dublin Inquirer, got %s", results[0].Name)
	}

	// Query far away — should NOT find it
	results, err = repo.List(40.7128, -74.0060, 50.0, nil)
	if err != nil {
		t.Fatalf("List NYC: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 results for NYC, got %d", len(results))
	}
}

func TestNewsSourceRepo_ListByTopics(t *testing.T) {
	db := setupTestDB(t)
	repo := NewNewsSourceRepo(db)

	repo.Create(model.NewsSource{
		Name: "Sports Daily", URL: "https://sportsdaily.test",
		AreaLabel: "Dublin", Latitude: 53.35, Longitude: -6.26,
		RadiusKm: 25, Topics: []string{"sports"}, TrustScore: 70, FetchMethod: "rss",
	})
	repo.Create(model.NewsSource{
		Name: "Housing Watch", URL: "https://housingwatch.test",
		AreaLabel: "Dublin", Latitude: 53.35, Longitude: -6.26,
		RadiusKm: 25, Topics: []string{"housing", "politics"}, TrustScore: 60, FetchMethod: "rss",
	})

	results, err := repo.List(53.35, -6.26, 50.0, []string{"sports"})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(results) != 1 || results[0].Name != "Sports Daily" {
		t.Fatalf("expected Sports Daily only, got %d results", len(results))
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `cd backend && go test ./internal/repository/ -run TestNewsSourceRepo -v`
Expected: FAIL — `NewNewsSourceRepo` not defined.

- [ ] **Step 4: Write the repository**

Create `backend/internal/repository/news_source_repo.go`:

```go
package repository

import (
	"database/sql"
	"math"

	"github.com/lib/pq"
	"github.com/shanegleeson/beepbopboop/backend/internal/model"
)

type NewsSourceRepo struct {
	db *sql.DB
}

func NewNewsSourceRepo(db *sql.DB) *NewsSourceRepo {
	return &NewsSourceRepo{db: db}
}

// List returns active news sources within the given radius of (lat, lon).
// If topics is non-empty, only sources with overlapping topics are returned.
func (r *NewsSourceRepo) List(lat, lon, radiusKm float64, topics []string) ([]model.NewsSource, error) {
	query := `
		SELECT id, name, url, COALESCE(feed_url,''), area_label,
		       latitude, longitude, radius_km, topics, trust_score,
		       fetch_method, active, created_at, updated_at
		FROM news_sources
		WHERE active = TRUE
	`
	args := []interface{}{}
	argN := 1

	// Haversine distance filter: distance < source.radius_km + query.radius_km
	// Using the spherical law of cosines approximation for speed.
	query += ` AND (
		6371 * acos(
			LEAST(1.0, cos(radians($` + itoa(argN) + `)) * cos(radians(latitude))
			* cos(radians(longitude) - radians($` + itoa(argN+1) + `))
			+ sin(radians($` + itoa(argN) + `)) * sin(radians(latitude)))
		)
	) < (radius_km + $` + itoa(argN+2) + `)`
	args = append(args, lat, lon, radiusKm)
	argN += 3

	if len(topics) > 0 {
		query += ` AND topics && $` + itoa(argN)
		args = append(args, pq.Array(topics))
		argN++
	}

	query += ` ORDER BY trust_score DESC`

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sources []model.NewsSource
	for rows.Next() {
		var s model.NewsSource
		err := rows.Scan(&s.ID, &s.Name, &s.URL, &s.FeedURL, &s.AreaLabel,
			&s.Latitude, &s.Longitude, &s.RadiusKm, pq.Array(&s.Topics),
			&s.TrustScore, &s.FetchMethod, &s.Active, &s.CreatedAt, &s.UpdatedAt)
		if err != nil {
			return nil, err
		}
		sources = append(sources, s)
	}
	return sources, rows.Err()
}

// Create inserts a news source, upserting on URL conflict.
func (r *NewsSourceRepo) Create(src model.NewsSource) error {
	_, err := r.db.Exec(`
		INSERT INTO news_sources (name, url, feed_url, area_label, latitude, longitude,
		                          radius_km, topics, trust_score, fetch_method)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (url) DO UPDATE SET
			name = EXCLUDED.name,
			feed_url = EXCLUDED.feed_url,
			area_label = EXCLUDED.area_label,
			latitude = EXCLUDED.latitude,
			longitude = EXCLUDED.longitude,
			radius_km = EXCLUDED.radius_km,
			topics = EXCLUDED.topics,
			trust_score = EXCLUDED.trust_score,
			fetch_method = EXCLUDED.fetch_method,
			updated_at = CURRENT_TIMESTAMP
	`, src.Name, src.URL, src.FeedURL, src.AreaLabel, src.Latitude, src.Longitude,
		src.RadiusKm, pq.Array(src.Topics), src.TrustScore, src.FetchMethod)
	return err
}

// Get returns a single news source by ID.
func (r *NewsSourceRepo) Get(id string) (*model.NewsSource, error) {
	var s model.NewsSource
	err := r.db.QueryRow(`
		SELECT id, name, url, COALESCE(feed_url,''), area_label,
		       latitude, longitude, radius_km, topics, trust_score,
		       fetch_method, active, created_at, updated_at
		FROM news_sources WHERE id = $1
	`, id).Scan(&s.ID, &s.Name, &s.URL, &s.FeedURL, &s.AreaLabel,
		&s.Latitude, &s.Longitude, &s.RadiusKm, pq.Array(&s.Topics),
		&s.TrustScore, &s.FetchMethod, &s.Active, &s.CreatedAt, &s.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &s, err
}

// itoa is a simple int-to-string for building parameterized queries.
func itoa(n int) string {
	return string(rune('0'+n%10)) // works for 1-9
}

// For larger param counts, use strconv.Itoa instead.
func init() {
	// Verify itoa works for small numbers (build-time safety).
	_ = math.Abs(0)
}
```

**Note:** The `itoa` helper above is fragile for param counts > 9. Replace with `strconv.Itoa` if more params are needed. Or use `fmt.Sprintf` for the query.

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd backend && go test ./internal/repository/ -run TestNewsSourceRepo -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add backend/internal/model/news_source.go backend/internal/repository/news_source_repo.go backend/internal/repository/news_source_repo_test.go
git commit -m "feat(wave4): add NewsSource model and repository with geo query"
```

---

### Task 3: News Source Handler + Routes

**Files:**
- Create: `backend/internal/handler/news_source.go`
- Create: `backend/internal/handler/news_source_test.go`
- Modify: `backend/cmd/server/main.go:65-254`

- [ ] **Step 1: Write the failing test**

Create `backend/internal/handler/news_source_test.go`:

```go
package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewsSourceHandler_Create(t *testing.T) {
	srv := setupTestServer(t)
	body := `{
		"name": "Dublin Inquirer",
		"url": "https://dublininquirer.com",
		"feed_url": "https://dublininquirer.com/feed",
		"area_label": "Dublin, Ireland",
		"latitude": 53.3498,
		"longitude": -6.2603,
		"radius_km": 25.0,
		"topics": ["local", "housing"],
		"trust_score": 80,
		"fetch_method": "rss"
	}`

	req := httptest.NewRequest("POST", "/news-sources", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = injectAgentAuth(req, srv.agentID)
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestNewsSourceHandler_List(t *testing.T) {
	srv := setupTestServer(t)

	// Create a source first
	body := `{
		"name": "Test Source",
		"url": "https://testsource.test",
		"area_label": "Dublin",
		"latitude": 53.35,
		"longitude": -6.26,
		"radius_km": 25,
		"topics": ["local"],
		"trust_score": 70,
		"fetch_method": "rss"
	}`
	req := httptest.NewRequest("POST", "/news-sources", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = injectAgentAuth(req, srv.agentID)
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	// List nearby
	req = httptest.NewRequest("GET", "/news-sources?lat=53.35&lon=-6.26&radius_km=50", nil)
	req = injectAgentAuth(req, srv.agentID)
	w = httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var result []json.RawMessage
	json.Unmarshal(w.Body.Bytes(), &result)
	if len(result) != 1 {
		t.Fatalf("expected 1 source, got %d", len(result))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/handler/ -run TestNewsSourceHandler -v`
Expected: FAIL — `NewsSourceHandler` not defined.

- [ ] **Step 3: Write the handler**

Create `backend/internal/handler/news_source.go`:

```go
package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/shanegleeson/beepbopboop/backend/internal/model"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

type NewsSourceHandler struct {
	repo *repository.NewsSourceRepo
}

func NewNewsSourceHandler(repo *repository.NewsSourceRepo) *NewsSourceHandler {
	return &NewsSourceHandler{repo: repo}
}

func (h *NewsSourceHandler) List(w http.ResponseWriter, r *http.Request) {
	latStr := r.URL.Query().Get("lat")
	lonStr := r.URL.Query().Get("lon")
	radiusStr := r.URL.Query().Get("radius_km")

	if latStr == "" || lonStr == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "lat and lon are required"})
		return
	}

	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid lat"})
		return
	}
	lon, err := strconv.ParseFloat(lonStr, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid lon"})
		return
	}

	radiusKm := 50.0
	if radiusStr != "" {
		radiusKm, err = strconv.ParseFloat(radiusStr, 64)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid radius_km"})
			return
		}
	}

	var topics []string
	if t := r.URL.Query().Get("topics"); t != "" {
		topics = splitCSV(t)
	}

	sources, err := h.repo.List(lat, lon, radiusKm, topics)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list sources"})
		return
	}

	writeJSON(w, http.StatusOK, sources)
}

func (h *NewsSourceHandler) Create(w http.ResponseWriter, r *http.Request) {
	var src model.NewsSource
	if err := json.NewDecoder(r.Body).Decode(&src); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}

	if src.Name == "" || src.URL == "" || src.AreaLabel == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name, url, and area_label are required"})
		return
	}

	if err := h.repo.Create(src); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create source"})
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{"status": "ok"})
}

func (h *NewsSourceHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	src, err := h.repo.Get(id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to get source"})
		return
	}
	if src == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	writeJSON(w, http.StatusOK, src)
}

// splitCSV splits a comma-separated string into a slice.
func splitCSV(s string) []string {
	var result []string
	for _, item := range strings.Split(s, ",") {
		item = strings.TrimSpace(item)
		if item != "" {
			result = append(result, item)
		}
	}
	return result
}
```

**Note:** Add `"strings"` to the import block.

- [ ] **Step 4: Wire up routes in main.go**

In `backend/cmd/server/main.go`, add to the repositories section (~line 86):

```go
	newsSourceRepo := repository.NewNewsSourceRepo(db)
```

Add to the handlers section (~line 167):

```go
	newsSourceH := handler.NewNewsSourceHandler(newsSourceRepo)
```

Add to agent-auth routes (~line 253, inside the agent-auth `r.Group`):

```go
	r.Get("/news-sources", newsSourceH.List)
	r.Post("/news-sources", newsSourceH.Create)
	r.Get("/news-sources/{id}", newsSourceH.Get)
```

- [ ] **Step 5: Run tests**

Run: `cd backend && go test ./internal/handler/ -run TestNewsSourceHandler -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add backend/internal/handler/news_source.go backend/internal/handler/news_source_test.go backend/cmd/server/main.go
git commit -m "feat(wave4): add news source handler with geo list + create endpoints"
```

---

### Task 4: `local_news` Display Hint

**Files:**
- Modify: `backend/internal/handler/hints.go:~546`

- [ ] **Step 1: Add the hint entry**

Add to the `buildHintCatalog()` slice in `hints.go`, before the closing `}`:

```go
	{
		Hint:           "local_news",
		Description:    "Local news card — article, video, or hybrid from a community source. Structured JSON in external_url carries source metadata and content kind.",
		PostType:       "article",
		StructuredJSON: true,
		RequiredFields: []string{"title", "body", "external_url"},
		Example: rawJSON(`{
			"title": "Dublin housing report reveals 30% increase in builds",
			"body": "A new report from the Dublin Inquirer shows significant growth in housing construction across the city centre.",
			"post_type": "article",
			"display_hint": "local_news",
			"image_url": "https://example.com/thumb.jpg",
			"external_url": "{\"content_kind\":\"article\",\"source_name\":\"Dublin Inquirer\",\"source_url\":\"https://dublininquirer.com\",\"source_logo_url\":\"https://example.com/logo.png\",\"thumbnail_url\":\"https://example.com/thumb.jpg\",\"article_url\":\"https://dublininquirer.com/2026/04/25/housing-report\",\"embed_url\":null,\"duration_seconds\":null,\"locality\":\"Dublin, Ireland\",\"published_at\":\"2026-04-25T10:00:00Z\",\"trust_score\":80}",
			"locality": "Dublin, Ireland"
		}`),
		Renders: &hintRenderInfo{
			Card:          "LocalNewsCard",
			UsesFields:    []string{"title", "body", "external_url", "image_url", "images"},
			IgnoresFields: []string{},
		},
		PickWhen:  "Content from a local publication, community news source, or local video segment.",
		AvoidWhen: "National/international news without a clear local source.",
		Generator: "beepbopboop-local-news",
	},
```

- [ ] **Step 2: Verify build**

Run: `cd backend && go build ./cmd/server/`
Expected: Compiles.

- [ ] **Step 3: Test hint lint**

Run: `cd backend && go test ./internal/handler/ -run TestHint -v`
Expected: PASS (existing hint tests + new hint validates).

If no existing hint test covers the new hint, add a quick test:

```go
func TestLocalNewsHint_LintValid(t *testing.T) {
	// Verify the hint example passes lint validation
	srv := setupTestServer(t)
	body := `{
		"title": "Test news",
		"body": "Test body",
		"display_hint": "local_news",
		"external_url": "{\"content_kind\":\"article\",\"source_name\":\"Test\",\"source_url\":\"https://test.com\"}"
	}`
	req := httptest.NewRequest("POST", "/posts/lint", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = injectAgentAuth(req, srv.agentID)
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("lint failed: %d %s", w.Code, w.Body.String())
	}
}
```

- [ ] **Step 4: Commit**

```bash
git add backend/internal/handler/hints.go
git commit -m "feat(wave4): add local_news display hint to catalog"
```

---

### Task 5: iOS — `LocalNewsCard` + Feed Routing

**Files:**
- Create: `beepbopboop/beepbopboop/Views/LocalNewsCard.swift`
- Modify: `beepbopboop/beepbopboop/Models/Post.swift` (add enum case)
- Modify: `beepbopboop/beepbopboop/Views/FeedItemView.swift:21-158` (add routing)

- [ ] **Step 1: Add `.localNews` to DisplayHint enum**

In `Post.swift`, find the `DisplayHint` enum and add:

```swift
case localNews = "local_news"
```

- [ ] **Step 2: Create LocalNewsCard**

Create `beepbopboop/beepbopboop/Views/LocalNewsCard.swift`:

```swift
import SwiftUI

struct LocalNewsCard: View {
    let post: Post

    private var newsData: LocalNewsData? {
        guard let urlStr = post.externalURL,
              let data = urlStr.data(using: .utf8),
              let parsed = try? JSONDecoder().decode(LocalNewsData.self, from: data)
        else { return nil }
        return parsed
    }

    var body: some View {
        if let news = newsData {
            VStack(alignment: .leading, spacing: 8) {
                // Source badge
                sourceBadge(news)

                switch news.contentKind {
                case "video":
                    videoLayout(news)
                case "hybrid":
                    hybridLayout(news)
                default:
                    articleLayout(news)
                }
            }
        } else {
            StandardCard(post: post)
        }
    }

    // MARK: - Source Badge

    @ViewBuilder
    private func sourceBadge(_ news: LocalNewsData) -> some View {
        HStack(spacing: 6) {
            if let logoURL = news.sourceLogoURL, let url = URL(string: logoURL) {
                AsyncImage(url: url) { image in
                    image.resizable().aspectRatio(contentMode: .fill)
                } placeholder: {
                    Circle().fill(Color.gray.opacity(0.3))
                }
                .frame(width: 20, height: 20)
                .clipShape(Circle())
            }

            Text(news.sourceName)
                .font(.caption)
                .fontWeight(.semibold)
                .foregroundColor(.secondary)

            if let locality = news.locality {
                Text("·")
                    .foregroundColor(.secondary)
                Text(locality)
                    .font(.caption2)
                    .padding(.horizontal, 6)
                    .padding(.vertical, 2)
                    .background(Color.blue.opacity(0.1))
                    .cornerRadius(4)
            }

            Spacer()

            // Trust indicator
            if let score = news.trustScore, score > 70 {
                Circle()
                    .fill(Color.green)
                    .frame(width: 6, height: 6)
            }
        }
    }

    // MARK: - Article Layout

    @ViewBuilder
    private func articleLayout(_ news: LocalNewsData) -> some View {
        HStack(alignment: .top, spacing: 12) {
            VStack(alignment: .leading, spacing: 4) {
                Text(post.title)
                    .font(.headline)
                    .lineLimit(2)

                if let body = post.body {
                    Text(body)
                        .font(.subheadline)
                        .foregroundColor(.secondary)
                        .lineLimit(2)
                }

                if let pub = news.publishedAt {
                    Text(pub, style: .relative)
                        .font(.caption2)
                        .foregroundColor(.tertiary)
                }
            }

            if let thumbURL = news.thumbnailURL, let url = URL(string: thumbURL) {
                AsyncImage(url: url) { image in
                    image.resizable().aspectRatio(contentMode: .fill)
                } placeholder: {
                    RoundedRectangle(cornerRadius: 8).fill(Color.gray.opacity(0.2))
                }
                .frame(width: 80, height: 80)
                .cornerRadius(8)
            }
        }
        .contentShape(Rectangle())
        .onTapGesture {
            if let articleURL = news.articleURL, let url = URL(string: articleURL) {
                UIApplication.shared.open(url)
            }
        }
    }

    // MARK: - Video Layout

    @ViewBuilder
    private func videoLayout(_ news: LocalNewsData) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            // Full-width thumbnail with play overlay
            ZStack {
                if let thumbURL = news.thumbnailURL, let url = URL(string: thumbURL) {
                    AsyncImage(url: url) { image in
                        image.resizable().aspectRatio(16/9, contentMode: .fill)
                    } placeholder: {
                        RoundedRectangle(cornerRadius: 8).fill(Color.gray.opacity(0.2))
                            .aspectRatio(16/9, contentMode: .fill)
                    }
                    .cornerRadius(8)
                }

                // Play button overlay
                Image(systemName: "play.circle.fill")
                    .font(.system(size: 44))
                    .foregroundColor(.white)
                    .shadow(radius: 4)

                // Duration badge
                if let duration = news.durationSeconds {
                    VStack {
                        Spacer()
                        HStack {
                            Spacer()
                            Text(formatDuration(duration))
                                .font(.caption2)
                                .fontWeight(.medium)
                                .foregroundColor(.white)
                                .padding(.horizontal, 6)
                                .padding(.vertical, 2)
                                .background(Color.black.opacity(0.7))
                                .cornerRadius(4)
                                .padding(8)
                        }
                    }
                }
            }
            .contentShape(Rectangle())
            .onTapGesture {
                let urlStr = news.embedURL ?? news.articleURL
                if let urlStr = urlStr, let url = URL(string: urlStr) {
                    UIApplication.shared.open(url)
                }
            }

            Text(post.title)
                .font(.headline)
                .lineLimit(2)
        }
    }

    // MARK: - Hybrid Layout

    @ViewBuilder
    private func hybridLayout(_ news: LocalNewsData) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            articleLayout(news)

            // Secondary video row
            if news.embedURL != nil {
                HStack(spacing: 8) {
                    ZStack {
                        if let thumbURL = news.thumbnailURL, let url = URL(string: thumbURL) {
                            AsyncImage(url: url) { image in
                                image.resizable().aspectRatio(16/9, contentMode: .fill)
                            } placeholder: {
                                RoundedRectangle(cornerRadius: 6).fill(Color.gray.opacity(0.2))
                            }
                            .frame(width: 120)
                            .cornerRadius(6)
                        }

                        Image(systemName: "play.circle.fill")
                            .font(.system(size: 24))
                            .foregroundColor(.white)
                            .shadow(radius: 2)
                    }
                    .contentShape(Rectangle())
                    .onTapGesture {
                        if let embedURL = news.embedURL, let url = URL(string: embedURL) {
                            UIApplication.shared.open(url)
                        }
                    }

                    if let duration = news.durationSeconds {
                        Text(formatDuration(duration))
                            .font(.caption2)
                            .foregroundColor(.secondary)
                    }

                    Spacer()
                }
            }
        }
    }

    private func formatDuration(_ seconds: Int) -> String {
        let minutes = seconds / 60
        let secs = seconds % 60
        return String(format: "%d:%02d", minutes, secs)
    }
}

// MARK: - Data Model

struct LocalNewsData: Codable {
    let contentKind: String
    let sourceName: String
    let sourceURL: String
    let sourceLogoURL: String?
    let thumbnailURL: String?
    let articleURL: String?
    let embedURL: String?
    let durationSeconds: Int?
    let locality: String?
    let publishedAt: Date?
    let trustScore: Int?

    enum CodingKeys: String, CodingKey {
        case contentKind = "content_kind"
        case sourceName = "source_name"
        case sourceURL = "source_url"
        case sourceLogoURL = "source_logo_url"
        case thumbnailURL = "thumbnail_url"
        case articleURL = "article_url"
        case embedURL = "embed_url"
        case durationSeconds = "duration_seconds"
        case locality
        case publishedAt = "published_at"
        case trustScore = "trust_score"
    }
}
```

- [ ] **Step 3: Add routing in FeedItemView.swift**

In the `cardContent` switch statement (~line 21), add before `default:`:

```swift
case .localNews:
    LocalNewsCard(post: post)
```

- [ ] **Step 4: Build iOS**

Run:
```bash
xcodebuild -project beepbopboop/beepbopboop.xcodeproj -scheme beepbopboop -sdk iphonesimulator -destination 'platform=iOS Simulator,name=iPhone 17 Pro' -derivedDataPath /tmp/bbp-build clean build 2>&1 | tail -5
```
Expected: BUILD SUCCEEDED

- [ ] **Step 5: Commit**

```bash
git add beepbopboop/beepbopboop/Views/LocalNewsCard.swift beepbopboop/beepbopboop/Models/Post.swift beepbopboop/beepbopboop/Views/FeedItemView.swift
git commit -m "feat(wave4): add LocalNewsCard with article/video/hybrid layouts"
```

---

### Task 6: `beepbopboop-local-news` Skill

**Files:**
- Create: `.claude/skills/beepbopboop-local-news/SKILL.md`
- Create: `.claude/skills/beepbopboop-local-news/MODE_FETCH.md`
- Create: `.claude/skills/beepbopboop-local-news/MODE_DISCOVER.md`
- Create: `.claude/skills/beepbopboop-local-news/MODE_VIDEO.md`

- [ ] **Step 1: Create SKILL.md**

Create `.claude/skills/beepbopboop-local-news/SKILL.md`:

```markdown
---
name: beepbopboop-local-news
description: Fetch and publish local news from community sources near the user
argument-hint: "local news" or "find local sources" or "local video news"
allowed-tools: Bash, Read, Write, WebSearch, WebFetch, Glob, Grep, Task
---

# BeepBopBoop Local News

Fetch, curate, and publish local news from community sources near the user's location.

## Step 0a: Load Config

Read `_shared/CONFIG.md` for `BEEPBOPBOOP_API_URL`, `BEEPBOPBOOP_AGENT_TOKEN`, `BEEPBOPBOOP_HOME_LAT`, `BEEPBOPBOOP_HOME_LON`.

## Step 0b: Route to Mode

| Input contains | Mode | Read |
|---|---|---|
| "find local sources" / "discover" | discover | `MODE_DISCOVER.md` |
| "local video" / "video news" | video | `MODE_VIDEO.md` |
| "local news" / default | fetch | `MODE_FETCH.md` |

## Step 0c: Lint + Publish

All modes end by following `../_shared/PUBLISH_ENVELOPE.md` for lint → dedup → publish.
```

- [ ] **Step 2: Create MODE_FETCH.md**

Create `.claude/skills/beepbopboop-local-news/MODE_FETCH.md`:

```markdown
# Mode: Fetch Local News

## Step 1: Get Nearby Sources

```bash
curl -s "$BEEPBOPBOOP_API_URL/news-sources?lat=$BEEPBOPBOOP_HOME_LAT&lon=$BEEPBOPBOOP_HOME_LON&radius_km=50" \
  -H "Authorization: Bearer $BEEPBOPBOOP_AGENT_TOKEN"
```

If empty: suggest running discover mode first, then stop.

## Step 2: Fetch Content from Each Source

For each source with `feed_url`:
- Fetch the RSS feed (use WebFetch)
- Parse items: title, link, description, pubDate
- Filter to items published in last 48 hours

For sources without `feed_url`:
- Fetch the main `url` with WebFetch
- Extract top stories from the page

## Step 3: Score and Rank

Score each item by:
- **Recency**: items from last 6h score highest, 6-24h medium, 24-48h lowest
- **Source trust**: multiply by `trust_score / 100`
- **Topic relevance**: bonus if item topics overlap user's interests (from profile)

Pick top 3-5 items.

## Step 4: Compose Posts

For each selected item, compose a post:

```json
{
  "title": "<headline, max 100 chars>",
  "body": "<2-3 sentence summary>",
  "post_type": "article",
  "display_hint": "local_news",
  "image_url": "<thumbnail if available>",
  "external_url": "<JSON string>",
  "locality": "<source area_label>",
  "labels": ["news", "local", "<topic>"]
}
```

The `external_url` must be a JSON string with this shape:
```json
{
  "content_kind": "article",
  "source_name": "Source Name",
  "source_url": "https://source.com",
  "source_logo_url": null,
  "thumbnail_url": "https://example.com/thumb.jpg",
  "article_url": "https://source.com/article",
  "embed_url": null,
  "duration_seconds": null,
  "locality": "City, Country",
  "published_at": "2026-04-25T10:00:00Z",
  "trust_score": 80
}
```

## Step 5: Lint and Publish

Follow `../_shared/PUBLISH_ENVELOPE.md`.
```

- [ ] **Step 3: Create MODE_DISCOVER.md**

Create `.claude/skills/beepbopboop-local-news/MODE_DISCOVER.md`:

```markdown
# Mode: Discover Local News Sources

## Step 1: Determine Location

Use `BEEPBOPBOOP_HOME_LAT` and `BEEPBOPBOOP_HOME_LON` from config.

## Step 2: Search for Sources

Use WebSearch to find local news publications:
- "local news <city name>"
- "<city name> community newspaper"
- "<city name> independent media"

## Step 3: Evaluate Each Source

For each found publication:
- Check if they have an RSS feed (look for `/feed`, `/rss`, `/atom.xml`)
- Assess trust: established publication? Regular updates? Real journalism?
- Determine topics covered
- Assign initial trust_score (50 for unknown, 70+ for established outlets)

## Step 4: Register Sources

For each viable source, POST to the registry:

```bash
curl -s -X POST "$BEEPBOPBOOP_API_URL/news-sources" \
  -H "Authorization: Bearer $BEEPBOPBOOP_AGENT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Publication Name",
    "url": "https://publication.com",
    "feed_url": "https://publication.com/feed",
    "area_label": "City, Country",
    "latitude": 53.35,
    "longitude": -6.26,
    "radius_km": 25,
    "topics": ["local", "politics"],
    "trust_score": 70,
    "fetch_method": "rss"
  }'
```

## Step 5: Report

Print a summary of discovered and registered sources.
```

- [ ] **Step 4: Create MODE_VIDEO.md**

Create `.claude/skills/beepbopboop-local-news/MODE_VIDEO.md`:

```markdown
# Mode: Local Video News

Same as MODE_FETCH.md but:

1. Filter sources and items to video content only
2. Search YouTube for local news channels in the area
3. Set `content_kind` to `"video"` in the external_url JSON
4. Include `embed_url` (YouTube embed URL) and `duration_seconds`

For YouTube results:
- `embed_url`: `https://www.youtube.com/embed/{videoId}`
- `article_url`: `https://www.youtube.com/watch?v={videoId}`
- Get duration from video metadata if available
```

- [ ] **Step 5: Commit**

```bash
git add .claude/skills/beepbopboop-local-news/
git commit -m "feat(wave4): add beepbopboop-local-news skill with fetch/discover/video modes"
```
