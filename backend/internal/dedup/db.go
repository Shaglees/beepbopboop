package dedup

import (
	"database/sql"
	_ "embed"
	"fmt"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

//go:embed migration.sql
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

	return db, nil
}

func SavePost(db *sql.DB, entry PostEntry) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	res, err := tx.Exec(
		`INSERT INTO posts (title, external_url, post_type, locality, latitude, longitude) VALUES (?, ?, ?, ?, ?, ?)`,
		entry.Title, nilIfEmpty(entry.ExternalURL), entry.PostType,
		nilIfEmpty(entry.Locality), entry.Latitude, entry.Longitude,
	)
	if err != nil {
		return fmt.Errorf("insert post: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("last insert id: %w", err)
	}

	if err := insertLabels(tx, id, entry.Labels); err != nil {
		return err
	}

	return tx.Commit()
}

func SavePosts(db *sql.DB, entries []PostEntry) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	for _, entry := range entries {
		res, err := tx.Exec(
			`INSERT INTO posts (title, external_url, post_type, locality, latitude, longitude) VALUES (?, ?, ?, ?, ?, ?)`,
			entry.Title, nilIfEmpty(entry.ExternalURL), entry.PostType,
			nilIfEmpty(entry.Locality), entry.Latitude, entry.Longitude,
		)
		if err != nil {
			return fmt.Errorf("insert post %q: %w", entry.Title, err)
		}

		id, err := res.LastInsertId()
		if err != nil {
			return fmt.Errorf("last insert id: %w", err)
		}

		if err := insertLabels(tx, id, entry.Labels); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func ListRecent(db *sql.DB, ttlDays int) ([]PostEntry, error) {
	cutoff := time.Now().AddDate(0, 0, -ttlDays)

	rows, err := db.Query(
		`SELECT id, title, COALESCE(external_url, ''), post_type, COALESCE(locality, ''), latitude, longitude, created_at
		 FROM posts WHERE created_at >= ? ORDER BY created_at DESC`, cutoff,
	)
	if err != nil {
		return nil, fmt.Errorf("query posts: %w", err)
	}
	defer rows.Close()

	var posts []PostEntry
	for rows.Next() {
		var p PostEntry
		if err := rows.Scan(&p.ID, &p.Title, &p.ExternalURL, &p.PostType, &p.Locality, &p.Latitude, &p.Longitude, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan post: %w", err)
		}
		posts = append(posts, p)
	}

	// Load labels for each post
	for i := range posts {
		labels, err := loadLabels(db, posts[i].ID)
		if err != nil {
			return nil, err
		}
		posts[i].Labels = labels
	}

	return posts, nil
}

func Prune(db *sql.DB, ttlDays int) (int, error) {
	cutoff := time.Now().AddDate(0, 0, -ttlDays)
	res, err := db.Exec(`DELETE FROM posts WHERE created_at < ?`, cutoff)
	if err != nil {
		return 0, fmt.Errorf("prune: %w", err)
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

func insertLabels(tx *sql.Tx, postID int64, labels []string) error {
	if len(labels) == 0 {
		return nil
	}
	// Build batch insert
	vals := make([]string, len(labels))
	args := make([]any, 0, len(labels)*2)
	for i, l := range labels {
		vals[i] = "(?, ?)"
		args = append(args, postID, strings.ToLower(strings.TrimSpace(l)))
	}
	_, err := tx.Exec(
		"INSERT OR IGNORE INTO post_labels (post_id, label) VALUES "+strings.Join(vals, ", "),
		args...,
	)
	if err != nil {
		return fmt.Errorf("insert labels: %w", err)
	}
	return nil
}

func loadLabels(db *sql.DB, postID int64) ([]string, error) {
	rows, err := db.Query(`SELECT label FROM post_labels WHERE post_id = ?`, postID)
	if err != nil {
		return nil, fmt.Errorf("query labels: %w", err)
	}
	defer rows.Close()

	var labels []string
	for rows.Next() {
		var l string
		if err := rows.Scan(&l); err != nil {
			return nil, fmt.Errorf("scan label: %w", err)
		}
		labels = append(labels, l)
	}
	return labels, nil
}

func nilIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
