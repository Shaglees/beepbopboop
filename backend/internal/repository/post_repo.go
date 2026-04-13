package repository

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/shanegleeson/beepbopboop/backend/internal/geo"
	"github.com/shanegleeson/beepbopboop/backend/internal/model"
)

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
	Labels      []string
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
	p.post_type, p.visibility, p.labels, p.created_at, p.seq`

// scanPost scans a row into a model.Post and returns the seq.
func scanPost(scanner interface{ Scan(dest ...any) error }) (model.Post, int64, error) {
	var p model.Post
	var imageURL, externalURL, locality, postType, labelsJSON sql.NullString
	var latitude, longitude sql.NullFloat64
	var seq int64

	err := scanner.Scan(&p.ID, &p.AgentID, &p.AgentName, &p.UserID,
		&p.Title, &p.Body,
		&imageURL, &externalURL, &locality, &latitude, &longitude,
		&postType, &p.Visibility, &labelsJSON, &p.CreatedAt, &seq)
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

	var labelsJSON sql.NullString
	if len(p.Labels) > 0 {
		b, err := json.Marshal(p.Labels)
		if err != nil {
			return nil, fmt.Errorf("marshal labels: %w", err)
		}
		labelsJSON = sql.NullString{String: string(b), Valid: true}
	}

	_, err = r.db.Exec(`
		INSERT INTO posts (id, agent_id, user_id, title, body, image_url, external_url, locality, latitude, longitude, post_type, visibility, labels)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
		id, p.AgentID, p.UserID, p.Title, p.Body,
		nullString(p.ImageURL), nullString(p.ExternalURL),
		nullString(p.Locality), nullFloat64(p.Latitude), nullFloat64(p.Longitude),
		nullString(p.PostType), visibility, labelsJSON,
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
func (r *PostRepo) ListForYou(userID string, lat, lon, radiusKm float64, cursor string, limit int) ([]model.Post, *string, error) {
	minLat, maxLat, minLon, maxLon := geo.BoundingBox(lat, lon, radiusKm)

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

	sqlLimit := limit * 3
	args = append(args, sqlLimit)

	rows, err := r.db.Query(`
		SELECT `+postColumns+`
		FROM posts p
		JOIN agents a ON a.id = p.agent_id
		WHERE p.visibility IN ('public', 'personal')
		  AND (
			(p.latitude IS NOT NULL AND p.longitude IS NOT NULL
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

		// User's own posts always pass; community posts need Haversine check
		if p.UserID == userID {
			posts = append(posts, p)
		} else if p.Latitude != nil && p.Longitude != nil {
			if geo.HaversineKm(lat, lon, *p.Latitude, *p.Longitude) <= radiusKm {
				posts = append(posts, p)
			}
		}
		if len(posts) >= limit {
			break
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
