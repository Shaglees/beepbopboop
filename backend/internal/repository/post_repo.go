package repository

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"strings"
	"time"

	"github.com/shanegleeson/beepbopboop/backend/internal/geo"
	"github.com/shanegleeson/beepbopboop/backend/internal/model"
)

// FeedWeights holds parsed preference weights for scored ranking.
type FeedWeights struct {
	LabelWeights  map[string]float64 `json:"label_weights"`
	TypeWeights   map[string]float64 `json:"type_weights"`
	FreshnessBias float64            `json:"freshness_bias"`
	GeoBias       float64            `json:"geo_bias"`
}

type CreatePostParams struct {
	AgentID     string
	UserID      string
	Title       string
	Body        string
	ImageURL    string
	ExternalURL string
	Locality    string
	Latitude    *float64
	Longitude   *float64
	PostType    string
	Visibility  string
	DisplayHint string
	Labels      []string
	Images      json.RawMessage
}

type PostRepo struct {
	db *sql.DB
}

func NewPostRepo(db *sql.DB) *PostRepo {
	return &PostRepo{db: db}
}

// postColumns is the shared SELECT column list. seq is last for cursor pagination.
const postColumns = `p.id, p.agent_id, a.name, p.user_id, p.title, p.body,
	p.image_url, p.external_url, p.locality, p.latitude, p.longitude,
	p.post_type, p.visibility, p.display_hint, p.labels, p.images, p.created_at, p.seq`

// scanPost scans a row into a model.Post and returns the seq.
func scanPost(scanner interface{ Scan(dest ...any) error }) (model.Post, int64, error) {
	var p model.Post
	var imageURL, externalURL, locality, postType, labelsJSON, imagesJSON sql.NullString
	var latitude, longitude sql.NullFloat64
	var seq int64

	err := scanner.Scan(&p.ID, &p.AgentID, &p.AgentName, &p.UserID,
		&p.Title, &p.Body,
		&imageURL, &externalURL, &locality, &latitude, &longitude,
		&postType, &p.Visibility, &p.DisplayHint, &labelsJSON, &imagesJSON, &p.CreatedAt, &seq)
	if err != nil {
		return p, 0, err
	}
	p.ImageURL = imageURL.String
	p.ExternalURL = externalURL.String
	p.Locality = locality.String
	if latitude.Valid {
		p.Latitude = &latitude.Float64
	}
	if longitude.Valid {
		p.Longitude = &longitude.Float64
	}
	p.PostType = postType.String
	if labelsJSON.Valid {
		json.Unmarshal([]byte(labelsJSON.String), &p.Labels)
	}
	if imagesJSON.Valid {
		p.Images = json.RawMessage(imagesJSON.String)
	}
	return p, seq, nil
}

func (r *PostRepo) Create(p CreatePostParams) (*model.Post, error) {
	id, err := generateID()
	if err != nil {
		return nil, fmt.Errorf("generate id: %w", err)
	}

	visibility := p.Visibility
	if visibility == "" {
		visibility = "public"
	}

	displayHint := p.DisplayHint
	if displayHint == "" {
		displayHint = "card"
	}

	var labelsJSON sql.NullString
	if len(p.Labels) > 0 {
		b, err := json.Marshal(p.Labels)
		if err != nil {
			return nil, fmt.Errorf("marshal labels: %w", err)
		}
		labelsJSON = sql.NullString{String: string(b), Valid: true}
	}

	_, err = r.db.Exec(`
		INSERT INTO posts (id, agent_id, user_id, title, body, image_url, external_url, locality, latitude, longitude, post_type, visibility, display_hint, labels, images)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)`,
		id, p.AgentID, p.UserID, p.Title, p.Body,
		nullString(p.ImageURL), nullString(p.ExternalURL),
		nullString(p.Locality), nullFloat64(p.Latitude), nullFloat64(p.Longitude),
		nullString(p.PostType), visibility, displayHint, labelsJSON, nullRawJSON(p.Images),
	)
	if err != nil {
		return nil, fmt.Errorf("insert post: %w", err)
	}

	return r.GetByID(id)
}

func (r *PostRepo) GetByID(id string) (*model.Post, error) {
	row := r.db.QueryRow(`
		SELECT `+postColumns+`
		FROM posts p
		JOIN agents a ON a.id = p.agent_id
		WHERE p.id = $1`, id)
	p, _, err := scanPost(row)
	if err != nil {
		return nil, fmt.Errorf("query post: %w", err)
	}
	return &p, nil
}

func (r *PostRepo) ListByUserID(userID string, limit int) ([]model.Post, error) {
	rows, err := r.db.Query(`
		SELECT `+postColumns+`
		FROM posts p
		JOIN agents a ON a.id = p.agent_id
		WHERE p.user_id = $1
		ORDER BY p.created_at DESC, p.seq DESC
		LIMIT $2`, userID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("query posts: %w", err)
	}
	defer rows.Close()

	posts := make([]model.Post, 0)
	for rows.Next() {
		p, _, err := scanPost(rows)
		if err != nil {
			return nil, fmt.Errorf("scan post: %w", err)
		}
		posts = append(posts, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate posts: %w", err)
	}
	return posts, nil
}

// --- Cursor pagination helpers ---

// parseCursorString decodes a cursor string of format "2006-01-02T15:04:05Z|42".
func parseCursorString(raw string) (time.Time, int64, error) {
	parts := strings.SplitN(raw, "|", 2)
	if len(parts) != 2 {
		return time.Time{}, 0, fmt.Errorf("invalid cursor format")
	}
	t, err := time.Parse(time.RFC3339, parts[0])
	if err != nil {
		return time.Time{}, 0, fmt.Errorf("invalid cursor time: %w", err)
	}
	var seq int64
	if _, err := fmt.Sscanf(parts[1], "%d", &seq); err != nil {
		return time.Time{}, 0, fmt.Errorf("invalid cursor seq: %w", err)
	}
	return t, seq, nil
}

func formatCursor(t time.Time, seq int64) string {
	return fmt.Sprintf("%s|%d", t.UTC().Format(time.RFC3339), seq)
}

// --- Multi-feed list methods ---

// ListPersonal returns the user's own posts with cursor-based pagination.
func (r *PostRepo) ListPersonal(userID, cursor string, limit int) ([]model.Post, *string, error) {
	args := []any{userID}
	cursorClause := ""
	argIdx := 2

	if cursor != "" {
		t, seq, err := parseCursorString(cursor)
		if err != nil {
			return nil, nil, err
		}
		cursorClause = fmt.Sprintf(" AND (p.created_at < $%d OR (p.created_at = $%d AND p.seq < $%d))", argIdx, argIdx+1, argIdx+2)
		args = append(args, t, t, seq)
		argIdx += 3
	}
	args = append(args, limit)

	rows, err := r.db.Query(`
		SELECT `+postColumns+`
		FROM posts p
		JOIN agents a ON a.id = p.agent_id
		WHERE p.user_id = $1`+cursorClause+fmt.Sprintf(`
		ORDER BY p.created_at DESC, p.seq DESC
		LIMIT $%d`, argIdx), args...,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("query personal feed: %w", err)
	}
	defer rows.Close()

	posts := make([]model.Post, 0)
	var lastCreatedAt time.Time
	var lastSeq int64
	for rows.Next() {
		p, seq, err := scanPost(rows)
		if err != nil {
			return nil, nil, fmt.Errorf("scan post: %w", err)
		}
		posts = append(posts, p)
		lastCreatedAt = p.CreatedAt
		lastSeq = seq
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("iterate posts: %w", err)
	}

	var nextCursor *string
	if len(posts) >= limit {
		c := formatCursor(lastCreatedAt, lastSeq)
		nextCursor = &c
	}
	return posts, nextCursor, nil
}

// ListCommunity returns nearby posts with cursor-based pagination.
// Uses a bounding-box SQL pre-filter then Haversine in Go.
func (r *PostRepo) ListCommunity(lat, lon, radiusKm float64, cursor string, limit int) ([]model.Post, *string, error) {
	minLat, maxLat, minLon, maxLon := geo.BoundingBox(lat, lon, radiusKm)

	args := []any{minLat, maxLat, minLon, maxLon}
	cursorClause := ""
	argIdx := 5

	if cursor != "" {
		t, seq, err := parseCursorString(cursor)
		if err != nil {
			return nil, nil, err
		}
		cursorClause = fmt.Sprintf(" AND (p.created_at < $%d OR (p.created_at = $%d AND p.seq < $%d))", argIdx, argIdx+1, argIdx+2)
		args = append(args, t, t, seq)
		argIdx += 3
	}

	sqlLimit := limit * 3
	args = append(args, sqlLimit)

	rows, err := r.db.Query(`
		SELECT `+postColumns+`
		FROM posts p
		JOIN agents a ON a.id = p.agent_id
		WHERE p.visibility IN ('public', 'personal')
		  AND p.latitude IS NOT NULL AND p.longitude IS NOT NULL
		  AND p.latitude BETWEEN $1 AND $2
		  AND p.longitude BETWEEN $3 AND $4`+cursorClause+fmt.Sprintf(`
		ORDER BY p.created_at DESC, p.seq DESC
		LIMIT $%d`, argIdx), args...,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("query community feed: %w", err)
	}
	defer rows.Close()

	posts := make([]model.Post, 0, limit)
	var lastCreatedAt time.Time
	var lastSeq int64
	rowsProcessed := 0

	for rows.Next() {
		p, seq, err := scanPost(rows)
		if err != nil {
			return nil, nil, fmt.Errorf("scan post: %w", err)
		}
		lastCreatedAt = p.CreatedAt
		lastSeq = seq
		rowsProcessed++

		// Haversine check
		if p.Latitude != nil && p.Longitude != nil {
			if geo.HaversineKm(lat, lon, *p.Latitude, *p.Longitude) <= radiusKm {
				posts = append(posts, p)
				if len(posts) >= limit {
					break
				}
			}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("iterate posts: %w", err)
	}

	var nextCursor *string
	if rowsProcessed >= limit && len(posts) > 0 {
		c := formatCursor(lastCreatedAt, lastSeq)
		nextCursor = &c
	}
	return posts, nextCursor, nil
}

// ListForYou returns community + user's own posts with cursor-based pagination.
// If weights is non-nil, posts are scored and ranked instead of pure recency.
func (r *PostRepo) ListForYou(userID string, lat, lon, radiusKm float64, cursor string, limit int, weights *FeedWeights) ([]model.Post, *string, error) {
	minLat, maxLat, minLon, maxLon := geo.BoundingBox(lat, lon, radiusKm)

	// When scoring, fetch more candidates so we have enough to rank.
	fetchLimit := limit * 3
	if weights != nil {
		fetchLimit = limit * 5
	}

	args := []any{minLat, maxLat, minLon, maxLon, userID}
	cursorClause := ""
	argIdx := 6

	if cursor != "" {
		t, seq, err := parseCursorString(cursor)
		if err != nil {
			return nil, nil, err
		}
		cursorClause = fmt.Sprintf(" AND (p.created_at < $%d OR (p.created_at = $%d AND p.seq < $%d))", argIdx, argIdx+1, argIdx+2)
		args = append(args, t, t, seq)
		argIdx += 3
	}

	args = append(args, fetchLimit)

	rows, err := r.db.Query(`
		SELECT `+postColumns+`
		FROM posts p
		JOIN agents a ON a.id = p.agent_id
		WHERE (
			(p.visibility IN ('public', 'personal')
			 AND p.latitude IS NOT NULL AND p.longitude IS NOT NULL
			 AND p.latitude BETWEEN $1 AND $2
			 AND p.longitude BETWEEN $3 AND $4)
			OR p.user_id = $5
		  )`+cursorClause+fmt.Sprintf(`
		ORDER BY p.created_at DESC, p.seq DESC
		LIMIT $%d`, argIdx), args...,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("query foryou feed: %w", err)
	}
	defer rows.Close()

	// Collect all candidates that pass geo filter.
	candidates := make([]model.Post, 0, fetchLimit)
	var lastCreatedAt time.Time
	var lastSeq int64
	rowsProcessed := 0

	for rows.Next() {
		p, seq, err := scanPost(rows)
		if err != nil {
			return nil, nil, fmt.Errorf("scan post: %w", err)
		}
		lastCreatedAt = p.CreatedAt
		lastSeq = seq
		rowsProcessed++

		// User's own posts always pass; community posts need Haversine check
		if p.UserID == userID {
			candidates = append(candidates, p)
		} else if p.Latitude != nil && p.Longitude != nil {
			if geo.HaversineKm(lat, lon, *p.Latitude, *p.Longitude) <= radiusKm {
				candidates = append(candidates, p)
			}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("iterate posts: %w", err)
	}

	// Apply weighted scoring if weights are available.
	if weights != nil && len(candidates) > 0 {
		type scored struct {
			post  model.Post
			score float64
		}
		scoredPosts := make([]scored, len(candidates))
		for i, p := range candidates {
			s := scorePost(p, lat, lon, radiusKm, weights)
			// Add ±5% jitter so near-equal posts shuffle between requests.
			jitter := 1.0 + (rand.Float64()-0.5)*0.1
			scoredPosts[i] = scored{post: p, score: s * jitter}
		}
		sort.Slice(scoredPosts, func(i, j int) bool {
			return scoredPosts[i].score > scoredPosts[j].score
		})

		posts := make([]model.Post, 0, limit)
		for i := 0; i < len(scoredPosts) && i < limit; i++ {
			posts = append(posts, scoredPosts[i].post)
		}

		var nextCursor *string
		if rowsProcessed >= limit && len(posts) > 0 {
			c := formatCursor(lastCreatedAt, lastSeq)
			nextCursor = &c
		}
		return posts, nextCursor, nil
	}

	// No weights: return in recency order (original behavior).
	posts := candidates
	if len(posts) > limit {
		posts = posts[:limit]
	}

	var nextCursor *string
	if rowsProcessed >= limit && len(posts) > 0 {
		c := formatCursor(lastCreatedAt, lastSeq)
		nextCursor = &c
	}
	return posts, nextCursor, nil
}

// scorePost computes a weighted relevance score for a post.
func scorePost(p model.Post, userLat, userLon, radiusKm float64, w *FeedWeights) float64 {
	var score float64

	ageDays := time.Since(p.CreatedAt).Hours() / 24

	// Baseline freshness floor: ensures new content surfaces regardless of weights.
	// 0.5 for brand-new posts, decaying with 7-day half-life.
	score += math.Exp(-0.099*ageDays) * 0.5

	// Weighted freshness: 14-day half-life, clamped bias.
	freshnessBias := clamp(w.FreshnessBias, 0, 1.0)
	freshness := math.Exp(-0.0495 * ageDays) // ln(2)/14 ≈ 0.0495
	score += freshnessBias * freshness

	// Geo proximity: 1.0 at center, 0.0 at radius edge.
	if p.Latitude != nil && p.Longitude != nil {
		dist := geo.HaversineKm(userLat, userLon, *p.Latitude, *p.Longitude)
		geoScore := 1.0 - (dist / radiusKm)
		if geoScore < 0 {
			geoScore = 0
		}
		score += clamp(w.GeoBias, 0, 1.0) * geoScore
	}

	// Label affinity: average of matching label weights (each clamped).
	if len(p.Labels) > 0 && len(w.LabelWeights) > 0 {
		var labelSum float64
		var labelCount int
		for _, label := range p.Labels {
			if wt, ok := w.LabelWeights[label]; ok {
				labelSum += clamp(wt, -1.0, 1.0)
				labelCount++
			}
		}
		if labelCount > 0 {
			score += labelSum / float64(labelCount)
		}
	}

	// Type affinity (clamped).
	if wt, ok := w.TypeWeights[p.PostType]; ok {
		score += clamp(wt, -1.0, 1.0)
	}

	return score
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// Stats returns aggregated post statistics for a user over the given number of days.
func (r *PostRepo) Stats(userID string, days int) (*model.PeriodStats, error) {
	ps := &model.PeriodStats{Days: days}

	// Total posts + avg per day
	err := r.db.QueryRow(`
		SELECT COUNT(*)
		FROM posts
		WHERE user_id = $1 AND created_at > NOW() - INTERVAL '1 day' * $2`,
		userID, days,
	).Scan(&ps.TotalPosts)
	if err != nil {
		return nil, fmt.Errorf("count posts: %w", err)
	}
	if days > 0 {
		ps.AvgPerDay = float64(ps.TotalPosts) / float64(days)
	}

	// Type counts + last posted
	rows, err := r.db.Query(`
		SELECT COALESCE(post_type, 'unknown'),
			COUNT(*) AS count,
			EXTRACT(DAY FROM NOW() - MAX(created_at))::int AS last_days_ago
		FROM posts
		WHERE user_id = $1 AND created_at > NOW() - INTERVAL '1 day' * $2
		GROUP BY post_type
		ORDER BY count DESC`,
		userID, days,
	)
	if err != nil {
		return nil, fmt.Errorf("query type stats: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var tc model.TypeCount
		if err := rows.Scan(&tc.Type, &tc.Count, &tc.LastDaysAgo); err != nil {
			return nil, fmt.Errorf("scan type count: %w", err)
		}
		ps.ByType = append(ps.ByType, tc)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate type stats: %w", err)
	}

	// Top labels (unnest JSON array)
	rows2, err := r.db.Query(`
		SELECT label, COUNT(*) AS count
		FROM posts,
		LATERAL jsonb_array_elements_text(labels::jsonb) AS label
		WHERE user_id = $1 AND created_at > NOW() - INTERVAL '1 day' * $2
		  AND labels IS NOT NULL AND labels != 'null'
		GROUP BY label
		ORDER BY count DESC
		LIMIT 15`,
		userID, days,
	)
	if err != nil {
		return nil, fmt.Errorf("query label stats: %w", err)
	}
	defer rows2.Close()

	for rows2.Next() {
		var lc model.LabelCount
		if err := rows2.Scan(&lc.Label, &lc.Count); err != nil {
			return nil, fmt.Errorf("scan label count: %w", err)
		}
		ps.TopLabels = append(ps.TopLabels, lc)
	}
	if err := rows2.Err(); err != nil {
		return nil, fmt.Errorf("iterate label stats: %w", err)
	}

	return ps, nil
}

func nullRawJSON(j json.RawMessage) sql.NullString {
	if len(j) == 0 || string(j) == "null" {
		return sql.NullString{}
	}
	return sql.NullString{String: string(j), Valid: true}
}

func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

func nullFloat64(f *float64) sql.NullFloat64 {
	if f == nil {
		return sql.NullFloat64{}
	}
	return sql.NullFloat64{Float64: *f, Valid: true}
}
