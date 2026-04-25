package repository

import (
	"database/sql"
	"fmt"
	"strconv"

	"github.com/lib/pq"
	"github.com/shanegleeson/beepbopboop/backend/internal/model"
)

type NewsSourceRepo struct {
	db *sql.DB
}

func NewNewsSourceRepo(db *sql.DB) *NewsSourceRepo {
	return &NewsSourceRepo{db: db}
}

// List returns news sources whose coverage area overlaps with the query circle.
// The overlap condition is: haversine_distance(source, query) < source.radius_km + query.radius_km.
// Optionally filtered by topics array overlap when topics is non-empty.
// Results are ordered by trust_score DESC.
func (r *NewsSourceRepo) List(lat, lon, radiusKm float64, topics []string) ([]model.NewsSource, error) {
	args := []any{lat, lon, radiusKm}
	argIdx := 4

	whereTopics := ""
	if len(topics) > 0 {
		args = append(args, pq.Array(topics))
		whereTopics = fmt.Sprintf(" AND topics && $%s", strconv.Itoa(argIdx))
		argIdx++
	}

	// Haversine distance in km between query point and source location.
	// Match when distance < source.radius_km + query.radius_km.
	query := fmt.Sprintf(`
		SELECT
			id, name, url, COALESCE(feed_url, ''), area_label,
			latitude, longitude, radius_km, topics,
			trust_score, fetch_method, active, created_at, updated_at
		FROM news_sources
		WHERE active = TRUE
		  AND 6371 * acos(LEAST(1.0,
		        cos(radians($1)) * cos(radians(latitude)) * cos(radians(longitude) - radians($2))
		        + sin(radians($1)) * sin(radians(latitude))
		      )) < radius_km + $3
		%s
		ORDER BY trust_score DESC`, whereTopics)

	_ = argIdx // suppress unused warning when topics is empty

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list news sources: %w", err)
	}
	defer rows.Close()

	var result []model.NewsSource
	for rows.Next() {
		var src model.NewsSource
		if err := rows.Scan(
			&src.ID, &src.Name, &src.URL, &src.FeedURL, &src.AreaLabel,
			&src.Latitude, &src.Longitude, &src.RadiusKm, pq.Array(&src.Topics),
			&src.TrustScore, &src.FetchMethod, &src.Active, &src.CreatedAt, &src.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan news source: %w", err)
		}
		result = append(result, src)
	}
	return result, rows.Err()
}

// Create inserts a new news source. If a source with the same URL already exists,
// it updates all fields (upsert by URL).
func (r *NewsSourceRepo) Create(src model.NewsSource) error {
	_, err := r.db.Exec(`
		INSERT INTO news_sources
			(name, url, feed_url, area_label, latitude, longitude, radius_km, topics, trust_score, fetch_method, active)
		VALUES
			($1, $2, NULLIF($3, ''), $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (url) DO UPDATE SET
			name         = EXCLUDED.name,
			feed_url     = EXCLUDED.feed_url,
			area_label   = EXCLUDED.area_label,
			latitude     = EXCLUDED.latitude,
			longitude    = EXCLUDED.longitude,
			radius_km    = EXCLUDED.radius_km,
			topics       = EXCLUDED.topics,
			trust_score  = EXCLUDED.trust_score,
			fetch_method = EXCLUDED.fetch_method,
			active       = EXCLUDED.active,
			updated_at   = CURRENT_TIMESTAMP`,
		src.Name, src.URL, src.FeedURL, src.AreaLabel,
		src.Latitude, src.Longitude, src.RadiusKm, pq.Array(src.Topics),
		src.TrustScore, src.FetchMethod, src.Active,
	)
	if err != nil {
		return fmt.Errorf("create news source: %w", err)
	}
	return nil
}

// Get returns the news source with the given ID, or nil if not found.
func (r *NewsSourceRepo) Get(id string) (*model.NewsSource, error) {
	var src model.NewsSource
	err := r.db.QueryRow(`
		SELECT
			id, name, url, COALESCE(feed_url, ''), area_label,
			latitude, longitude, radius_km, topics,
			trust_score, fetch_method, active, created_at, updated_at
		FROM news_sources
		WHERE id = $1`, id,
	).Scan(
		&src.ID, &src.Name, &src.URL, &src.FeedURL, &src.AreaLabel,
		&src.Latitude, &src.Longitude, &src.RadiusKm, pq.Array(&src.Topics),
		&src.TrustScore, &src.FetchMethod, &src.Active, &src.CreatedAt, &src.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get news source: %w", err)
	}
	return &src, nil
}
