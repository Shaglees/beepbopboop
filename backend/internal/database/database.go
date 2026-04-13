package database

import (
	"database/sql"
	_ "embed"
	"fmt"

	_ "modernc.org/sqlite"
)

//go:embed migrations/001_initial.sql
var migrationSQL string

func Open(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("set WAL mode: %w", err)
	}

	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}

	if _, err := db.Exec(migrationSQL); err != nil {
		db.Close()
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	// Add new columns to existing databases (ignore "duplicate column" errors).
	db.Exec("ALTER TABLE posts ADD COLUMN visibility TEXT NOT NULL DEFAULT 'public'")
	db.Exec("ALTER TABLE posts ADD COLUMN labels TEXT")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_posts_visibility ON posts(visibility, created_at DESC)")

	// User settings table for location preferences
	db.Exec(`CREATE TABLE IF NOT EXISTS user_settings (
		user_id TEXT PRIMARY KEY REFERENCES users(id),
		location_name TEXT,
		latitude REAL,
		longitude REAL,
		radius_km REAL NOT NULL DEFAULT 25.0,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`)

	// Geo index for community feed queries
	db.Exec("CREATE INDEX IF NOT EXISTS idx_posts_geo ON posts(visibility, latitude, longitude, created_at DESC)")

	return db, nil
}
