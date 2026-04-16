package database

import (
	"database/sql"
	_ "embed"
	"fmt"

	_ "github.com/lib/pq"
)

//go:embed migrations/001_initial.sql
var migrationSQL string

func Open(url string) (*sql.DB, error) {
	db, err := sql.Open("postgres", url)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	if _, err := db.Exec(migrationSQL); err != nil {
		db.Close()
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	// Add new columns to existing databases (ignore "already exists" errors).
	db.Exec("ALTER TABLE posts ADD COLUMN IF NOT EXISTS visibility TEXT NOT NULL DEFAULT 'public'")
	db.Exec("ALTER TABLE posts ADD COLUMN IF NOT EXISTS labels TEXT")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_posts_visibility ON posts(visibility, created_at DESC)")

	// User settings table for location preferences
	db.Exec(`CREATE TABLE IF NOT EXISTS user_settings (
		user_id TEXT PRIMARY KEY REFERENCES users(id),
		location_name TEXT,
		latitude DOUBLE PRECISION,
		longitude DOUBLE PRECISION,
		radius_km DOUBLE PRECISION NOT NULL DEFAULT 25.0,
		updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`)

	// Geo index for community feed queries
	db.Exec("CREATE INDEX IF NOT EXISTS idx_posts_geo ON posts(visibility, latitude, longitude, created_at DESC)")

	// Engagement events table
	db.Exec(`CREATE TABLE IF NOT EXISTS post_events (
		id         BIGSERIAL PRIMARY KEY,
		post_id    TEXT NOT NULL REFERENCES posts(id),
		user_id    TEXT NOT NULL REFERENCES users(id),
		event_type TEXT NOT NULL,
		dwell_ms   INTEGER,
		created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`)
	db.Exec("CREATE INDEX IF NOT EXISTS idx_post_events_post ON post_events(post_id, event_type)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_post_events_user ON post_events(user_id, created_at DESC)")

	// Display hint for post rendering
	db.Exec("ALTER TABLE posts ADD COLUMN IF NOT EXISTS display_hint TEXT NOT NULL DEFAULT 'card'")

	// Images JSONB for multi-image posts (outfit collages, product thumbnails)
	db.Exec("ALTER TABLE posts ADD COLUMN IF NOT EXISTS images JSONB")

	// Custom display templates per user
	db.Exec(`CREATE TABLE IF NOT EXISTS display_templates (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL REFERENCES users(id),
		hint_name TEXT NOT NULL,
		definition JSONB NOT NULL,
		created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(user_id, hint_name)
	)`)

	// User preference weights (pushed by agent, applied in ForYou feed)
	db.Exec(`CREATE TABLE IF NOT EXISTS user_weights (
		user_id    TEXT NOT NULL REFERENCES users(id) PRIMARY KEY,
		weights    JSONB NOT NULL,
		updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`)

	// System user + weather agent for server-generated posts.
	db.Exec("INSERT INTO users (id, firebase_uid) VALUES ('system', 'system') ON CONFLICT DO NOTHING")
	db.Exec("INSERT INTO agents (id, user_id, name, status) VALUES ('weather-bot', 'system', 'Weather', 'active') ON CONFLICT DO NOTHING")

	return db, nil
}
