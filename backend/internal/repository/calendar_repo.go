package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/shanegleeson/beepbopboop/backend/internal/model"
)

type CalendarRepo struct {
	db *sql.DB
}

func NewCalendarRepo(db *sql.DB) *CalendarRepo {
	return &CalendarRepo{db: db}
}

// UpsertEvents replaces the synced calendar events for a user. Existing rows for
// the user are deleted first so stale events (cancelled, moved) are removed.
func (r *CalendarRepo) UpsertEvents(userID string, events []model.CalendarEvent) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM calendar_events WHERE user_id = $1`, userID); err != nil {
		return fmt.Errorf("delete calendar events: %w", err)
	}

	for _, e := range events {
		var endTime sql.NullTime
		if e.EndTime != nil {
			endTime = sql.NullTime{Time: *e.EndTime, Valid: true}
		}
		_, err := tx.Exec(`
			INSERT INTO calendar_events (id, user_id, title, start_time, end_time, location, notes, synced_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, CURRENT_TIMESTAMP)
			ON CONFLICT (user_id, id) DO UPDATE SET
				title      = excluded.title,
				start_time = excluded.start_time,
				end_time   = excluded.end_time,
				location   = excluded.location,
				notes      = excluded.notes,
				synced_at  = CURRENT_TIMESTAMP`,
			e.ID, userID, e.Title, e.StartTime, endTime,
			nullString(e.Location), nullString(e.Notes),
		)
		if err != nil {
			return fmt.Errorf("upsert event %s: %w", e.ID, err)
		}
	}

	return tx.Commit()
}

// UpcomingEvents returns events for a user that start within the given window.
func (r *CalendarRepo) UpcomingEvents(userID string, from, to time.Time) ([]model.CalendarEvent, error) {
	rows, err := r.db.Query(`
		SELECT id, user_id, title, start_time, end_time, location, notes, synced_at
		FROM calendar_events
		WHERE user_id = $1 AND start_time >= $2 AND start_time <= $3
		ORDER BY start_time`,
		userID, from, to)
	if err != nil {
		return nil, fmt.Errorf("query upcoming events: %w", err)
	}
	defer rows.Close()

	var events []model.CalendarEvent
	for rows.Next() {
		var e model.CalendarEvent
		var location, notes sql.NullString
		var endTime sql.NullTime
		if err := rows.Scan(&e.ID, &e.UserID, &e.Title, &e.StartTime, &endTime, &location, &notes, &e.SyncedAt); err != nil {
			return nil, fmt.Errorf("scan event: %w", err)
		}
		if endTime.Valid {
			e.EndTime = &endTime.Time
		}
		e.Location = location.String
		e.Notes = notes.String
		events = append(events, e)
	}
	return events, rows.Err()
}
