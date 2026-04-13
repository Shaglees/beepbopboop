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

	return db, nil
}
