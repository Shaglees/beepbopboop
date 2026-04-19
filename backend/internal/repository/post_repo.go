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
	// FollowedTeams is populated per-request from user settings and is never
	// persisted with the rest of the weights (json:"-").
	FollowedTeams map[string]bool `json:"-"`
}

type CreatePostParams struct {
	AgentID           string
	UserID            string
	Title             string
	Body              string
	ImageURL          string
	ExternalURL       string
	Locality          string
	Latitude          *float64
	Longitude         *float64
	PostType          string
	Visibility        string
	DisplayHint       string
	Labels            []string
	Images            json.RawMessage
	ScheduledAt       *time.Time
	SourcePublishedAt *time.Time
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
	p.post_type, p.visibility, p.display_hint, p.labels, p.images,
	p.status, p.scheduled_at, p.source_published_at, p.created_at, p.view_count, p.save_count,
	p.reaction_count, p.seq`

// scanPost scans a row into a model.Post and returns the seq.
// extra optionally appends additional scan destinations after seq (e.g. saved_at).
func scanPost(scanner interface{ Scan(dest ...any) error }, extra ...any) (model.Post, int64, error) {
	var p model.Post
	var imageURL, externalURL, locality, postType, labelsJSON, imagesJSON sql.NullString
	var latitude, longitude sql.NullFloat64
	var scheduledAt sql.NullTime
	var sourcePublishedAt sql.NullTime
	var seq int64

	dest := []any{
		&p.ID, &p.AgentID, &p.AgentName, &p.UserID,
		&p.Title, &p.Body,
		&imageURL, &externalURL, &locality, &latitude, &longitude,
		&postType, &p.Visibility, &p.DisplayHint, &labelsJSON, &imagesJSON,
		&p.Status, &scheduledAt, &sourcePublishedAt, &p.CreatedAt, &p.ViewCount, &p.SaveCount,
		&p.ReactionCount, &seq,
	}
	dest = append(dest, extra...)
	err := scanner.Scan(dest...)
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
	if scheduledAt.Valid {
		p.ScheduledAt = &scheduledAt.Time
	}
	if sourcePublishedAt.Valid {
		p.SourcePublishedAt = &sourcePublishedAt.Time
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

	status := "published"
	var scheduledAt sql.NullTime
	if p.ScheduledAt != nil && p.ScheduledAt.After(time.Now()) {
		status = "scheduled"
		scheduledAt = sql.NullTime{Time: *p.ScheduledAt, Valid: true}
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
		INSERT INTO posts (id, agent_id, user_id, title, body, image_url, external_url, locality, latitude, longitude, post_type, visibility, display_hint, labels, images, status, scheduled_at, source_published_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)`,
		id, p.AgentID, p.UserID, p.Title, p.Body,
		nullString(p.ImageURL), nullString(p.ExternalURL),
		nullString(p.Locality), nullFloat64(p.Latitude), nullFloat64(p.Longitude),
		nullString(p.PostType), visibility, displayHint, labelsJSON, nullRawJSON(p.Images),
		status, scheduledAt, nullTime(p.SourcePublishedAt),
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
		WHERE p.user_id = $1 AND p.status = 'published'`+cursorClause+fmt.Sprintf(`
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

// ListSaved returns posts saved by userID with cursor-based pagination ordered by save time.
// A post is included only if the user's most recent relevant event is 'save' (no later 'unsave').
func (r *PostRepo) ListSaved(userID, cursor string, limit int) ([]model.Post, *string, error) {
	args := []any{userID}
	havingClause := ""
	argIdx := 2

	if cursor != "" {
		t, seq, err := parseCursorString(cursor)
		if err != nil {
			return nil, nil, err
		}
		havingClause = fmt.Sprintf(
			" HAVING (MAX(pe.created_at) < $%d OR (MAX(pe.created_at) = $%d AND p.seq < $%d))",
			argIdx, argIdx+1, argIdx+2,
		)
		args = append(args, t, t, seq)
		argIdx += 3
	}
	args = append(args, limit)

	query := fmt.Sprintf(`
		SELECT `+postColumns+`, MAX(pe.created_at) AS saved_at
		FROM posts p
		JOIN agents a ON a.id = p.agent_id
		JOIN post_events pe ON pe.post_id = p.id
			AND pe.user_id = $1
			AND pe.event_type = 'save'
		LEFT JOIN post_events unsave ON unsave.post_id = p.id
			AND unsave.user_id = $1
			AND unsave.event_type = 'unsave'
			AND unsave.created_at > pe.created_at
		WHERE unsave.id IS NULL
		GROUP BY p.id, a.id%s
		ORDER BY saved_at DESC, p.seq DESC
		LIMIT $%d`, havingClause, argIdx)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, nil, fmt.Errorf("query saved feed: %w", err)
	}
	defer rows.Close()

	posts := make([]model.Post, 0)
	var lastSavedAt time.Time
	var lastSeq int64
	for rows.Next() {
		var savedAt time.Time
		p, seq, err := scanPost(rows, &savedAt)
		if err != nil {
			return nil, nil, fmt.Errorf("scan saved post: %w", err)
		}
		posts = append(posts, p)
		lastSavedAt = savedAt
		lastSeq = seq
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("iterate saved posts: %w", err)
	}

	var nextCursor *string
	if len(posts) >= limit {
		c := formatCursor(lastSavedAt, lastSeq)
		nextCursor = &c
	}
	return posts, nextCursor, nil
}

// ListCommunity returns nearby posts ranked by a composite score:
// recency decay + geo proximity + engagement + event timing.
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

	// Fetch 5x candidates so the scoring pass has enough to rank from.
	sqlLimit := limit * 5
	args = append(args, sqlLimit)

	rows, err := r.db.Query(`
		SELECT `+postColumns+`
		FROM posts p
		JOIN agents a ON a.id = p.agent_id
		WHERE p.status = 'published'
		  AND p.visibility IN ('public', 'personal')
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

	type scored struct {
		post  model.Post
		score float64
	}

	var candidates []scored
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

		if p.Latitude != nil && p.Longitude != nil {
			if geo.HaversineKm(lat, lon, *p.Latitude, *p.Longitude) <= radiusKm {
				s := ScoreCommunityPost(p, lat, lon, radiusKm)
				candidates = append(candidates, scored{post: p, score: s})
			}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("iterate posts: %w", err)
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].score > candidates[j].score
	})

	posts := make([]model.Post, 0, limit)
	for i := 0; i < len(candidates) && i < limit; i++ {
		posts = append(posts, candidates[i].post)
	}

	// The cursor is keyed on SQL row order (created_at DESC), not on composite score.
	// Cross-page ranking continuity is not guaranteed: a post near the page boundary
	// may appear on either page depending on which SQL window captures it. This is
	// an accepted trade-off for cursor-based ranked feeds without a score cache.
	var nextCursor *string
	if rowsProcessed >= sqlLimit && len(posts) > 0 {
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
		WHERE p.status = 'published'
		  AND (
			(p.visibility = 'public'
			 AND p.latitude IS NOT NULL AND p.longitude IS NOT NULL
			 AND p.latitude BETWEEN $1 AND $2
			 AND p.longitude BETWEEN $3 AND $4)
			OR p.user_id = $5
			OR (p.visibility = 'public' AND p.latitude IS NULL)
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

		// User's own posts always pass; geo posts need Haversine check;
		// non-geo public posts (articles, news, fashion) pass through.
		if p.UserID == userID {
			candidates = append(candidates, p)
		} else if p.Latitude == nil || p.Longitude == nil {
			candidates = append(candidates, p)
		} else if geo.HaversineKm(lat, lon, *p.Latitude, *p.Longitude) <= radiusKm {
			candidates = append(candidates, p)
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
			// Add ±30% jitter so each refresh produces a noticeably different order.
			jitter := 1.0 + (rand.Float64()-0.5)*0.6
			scoredPosts[i] = scored{post: p, score: s * jitter}
		}
		sort.Slice(scoredPosts, func(i, j int) bool {
			return scoredPosts[i].score > scoredPosts[j].score
		})
		// Diversity pass: penalize consecutive posts with the same display hint.
		for i := 1; i < len(scoredPosts); i++ {
			if scoredPosts[i].post.DisplayHint == scoredPosts[i-1].post.DisplayHint {
				scoredPosts[i].score *= 0.85
			}
		}
		sort.Slice(scoredPosts, func(i, j int) bool {
			return scoredPosts[i].score > scoredPosts[j].score
		})
		// Shuffle the top segment so pull-to-refresh always feels fresh.
		// Posts are already roughly ranked — shuffling the top 60% of the
		// page mixes them while keeping low-scoring posts at the bottom.
		shuffleN := len(scoredPosts)
		if shuffleN > limit {
			shuffleN = limit
		}
		shuffleN = shuffleN * 3 / 5 // top 60%
		if shuffleN > 1 {
			rand.Shuffle(shuffleN, func(i, j int) {
				scoredPosts[i], scoredPosts[j] = scoredPosts[j], scoredPosts[i]
			})
		}

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

	ageHours := time.Since(p.CreatedAt).Hours()
	ageDays := ageHours / 24

	// Recency boost: full strength for first 12 hours, linear trail-off to zero at 34h.
	if ageHours < 12 {
		score += 0.6
	} else if ageHours < 34 {
		// Linear decay: 0.6 at 12h → 0 at 34h
		score += 0.6 * (34 - ageHours) / 22
	}

	// Long-tail freshness: gradual decay so older content still has a chance.
	// 14-day half-life, scaled by user's freshness bias.
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

	// Team affinity: parse sports post external_url and boost matched followed teams.
	if len(w.FollowedTeams) > 0 && p.ExternalURL != "" {
		var g struct {
			Sport string `json:"sport"`
			Home  struct {
				Abbr string `json:"abbr"`
			} `json:"home"`
			Away struct {
				Abbr string `json:"abbr"`
			} `json:"away"`
		}
		if json.Unmarshal([]byte(p.ExternalURL), &g) == nil && g.Sport != "" {
			sport := strings.ToLower(g.Sport)
			seen := make(map[string]bool, 2)
			for _, abbr := range []string{g.Home.Abbr, g.Away.Abbr} {
				if a := strings.ToLower(abbr); a != "" && !seen[a] && w.FollowedTeams[sport+":"+a] {
					score += 1.5
					seen[a] = true
				}
			}
		}
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

// ScoreCommunityPost computes a composite ranking score for the community feed.
// Combines recency decay (primary), geo proximity, engagement signal, and event timing.
// Exported for testability.
func ScoreCommunityPost(p model.Post, userLat, userLon, radiusKm float64) float64 {
	ageHours := time.Since(p.CreatedAt).Hours()

	// Recency decay with 4-hour half-life: exp(-ln(2)/4 * age_hours)
	recency := math.Exp(-0.173 * ageHours)

	// Geo proximity: 1.0 at centre, 0.0 at radius edge.
	geoScore := 0.0
	if p.Latitude != nil && p.Longitude != nil {
		dist := geo.HaversineKm(userLat, userLon, *p.Latitude, *p.Longitude)
		geoScore = 1.0 - (dist / radiusKm)
		if geoScore < 0 {
			geoScore = 0
		}
	}

	// Engagement: logarithmic to prevent viral outliers dominating.
	// Weight saves double (more deliberate action than a reaction).
	engagementRaw := float64(p.ReactionCount) + float64(p.SaveCount)*2.0
	engagementScore := math.Log1p(engagementRaw) / math.Log1p(30) // normalise against ~30 engagement units

	// Event timing: boost posts tied to upcoming / just-started events.
	eventScore := 0.0
	if p.ExternalURL != "" {
		eventScore = parseEventTimingScore(p.ExternalURL)
	}

	return recency + 0.4*geoScore + 0.3*engagementScore + 0.5*eventScore
}

// parseEventTimingScore extracts a timing boost from a sports/event ExternalURL JSON.
// Returns 0 if no parseable game time is found.
func parseEventTimingScore(externalURL string) float64 {
	var data struct {
		GameTime *string `json:"gameTime"`
	}
	if err := json.Unmarshal([]byte(externalURL), &data); err != nil || data.GameTime == nil {
		return 0
	}
	gameTime, err := time.Parse(time.RFC3339, *data.GameTime)
	if err != nil {
		return 0
	}
	hoursUntil := time.Until(gameTime).Hours()
	if hoursUntil > 0 {
		// Approaching event: peaks at kick-off
		return math.Exp(-0.3 * hoursUntil)
	}
	// Just started (within 3 hours): live bonus
	hoursSince := -hoursUntil
	if hoursSince < 3 {
		return 1.5 * math.Exp(-0.3*hoursSince)
	}
	return 0
}

// Stats returns aggregated post statistics for a user over the given number of days.
func (r *PostRepo) Stats(userID string, days int) (*model.PeriodStats, error) {
	ps := &model.PeriodStats{Days: days}

	// Total posts + avg per day
	err := r.db.QueryRow(`
		SELECT COUNT(*)
		FROM posts
		WHERE user_id = $1 AND status = 'published' AND created_at > NOW() - INTERVAL '1 day' * $2`,
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
		WHERE user_id = $1 AND status = 'published' AND created_at > NOW() - INTERVAL '1 day' * $2
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
		WHERE user_id = $1 AND status = 'published' AND created_at > NOW() - INTERVAL '1 day' * $2
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

// PublishScheduled finds posts with status='scheduled' whose scheduled_at has passed
// and updates them to status='published'. Returns the number of posts published.
func (r *PostRepo) PublishScheduled() (int64, error) {
	result, err := r.db.Exec(`
		UPDATE posts
		SET status = 'published', created_at = CURRENT_TIMESTAMP
		WHERE status = 'scheduled' AND scheduled_at <= CURRENT_TIMESTAMP`)
	if err != nil {
		return 0, fmt.Errorf("publish scheduled: %w", err)
	}
	return result.RowsAffected()
}

// ListByUserIDWithStatus returns posts for a user filtered by status.
func (r *PostRepo) ListByUserIDWithStatus(userID, status string, limit int) ([]model.Post, error) {
	rows, err := r.db.Query(`
		SELECT `+postColumns+`
		FROM posts p
		JOIN agents a ON a.id = p.agent_id
		WHERE p.user_id = $1 AND p.status = $2
		ORDER BY COALESCE(p.scheduled_at, p.created_at) ASC, p.seq DESC
		LIMIT $3`, userID, status, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("query posts by status: %w", err)
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

// UpsertWeatherPost creates or replaces a weather post for a geographic grid cell.
// The ID is deterministic from the grid key so the same cell always updates in place.
// forecastJSON is the serialized weather forecast stored in external_url for the iOS card to parse.
func (r *PostRepo) UpsertWeatherPost(gridKey, title, body string, lat, lon float64, forecastJSON string) error {
	id := "weather-" + gridKey
	labelsJSON := `["weather"]`

	_, err := r.db.Exec(`
		INSERT INTO posts (id, agent_id, user_id, title, body, external_url, latitude, longitude,
			post_type, visibility, display_hint, labels, created_at)
		VALUES ($1, 'weather-bot', 'system', $2, $3, $4, $5, $6,
			'discovery', 'public', 'weather', $7, CURRENT_TIMESTAMP)
		ON CONFLICT(id) DO UPDATE SET
			title = excluded.title,
			body = excluded.body,
			external_url = excluded.external_url,
			latitude = excluded.latitude,
			longitude = excluded.longitude,
			labels = excluded.labels,
			created_at = CURRENT_TIMESTAMP`,
		id, title, body, forecastJSON, lat, lon, labelsJSON,
	)
	return err
}

// UpsertSportsPost creates or replaces a sports post for a specific ESPN game.
// The ID is deterministic from gameID so the same game always updates in place.
// gameDataJSON is serialized GameData stored in external_url for the iOS scoreboard card.
func (r *PostRepo) UpsertSportsPost(gameID, title, body, league, gameDataJSON string) error {
	id := "sports-" + gameID
	labelsJSON, _ := json.Marshal([]string{"sports", league})

	_, err := r.db.Exec(`
		INSERT INTO posts (id, agent_id, user_id, title, body, external_url,
			post_type, visibility, display_hint, labels, created_at)
		VALUES ($1, 'sports-bot', 'system', $2, $3, $4,
			'discovery', 'public', 'scoreboard', $5, CURRENT_TIMESTAMP)
		ON CONFLICT(id) DO UPDATE SET
			title = excluded.title,
			body = excluded.body,
			external_url = excluded.external_url,
			labels = excluded.labels,
			created_at = CURRENT_TIMESTAMP`,
		id, title, body, gameDataJSON, string(labelsJSON),
	)
	return err
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

func nullTime(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *t, Valid: true}
}
