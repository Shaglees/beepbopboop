package repository

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/shanegleeson/beepbopboop/backend/internal/model"
)

type CalendarRepo struct {
	db *sql.DB
}

func NewCalendarRepo(db *sql.DB) *CalendarRepo {
	return &CalendarRepo{db: db}
}

// UpsertEvents syncs calendar events for a user. Each incoming event is upserted,
// then any stored events not present in the new set are pruned. This avoids the
// DELETE+INSERT race where a concurrent reader sees an empty table mid-transaction.
func (r *CalendarRepo) UpsertEvents(userID string, events []model.CalendarEvent) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	ids := make([]string, 0, len(events))
	for _, e := range events {
		ids = append(ids, e.ID)
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

	// Prune events that were removed or cancelled on device.
	if len(ids) > 0 {
		placeholders := make([]string, len(ids))
		args := make([]any, len(ids)+1)
		args[0] = userID
		for i, id := range ids {
			placeholders[i] = fmt.Sprintf("$%d", i+2)
			args[i+1] = id
		}
		query := fmt.Sprintf(
			`DELETE FROM calendar_events WHERE user_id = $1 AND id NOT IN (%s)`,
			strings.Join(placeholders, ","),
		)
		if _, err := tx.Exec(query, args...); err != nil {
			return fmt.Errorf("prune stale events: %w", err)
		}
	} else {
		if _, err := tx.Exec(`DELETE FROM calendar_events WHERE user_id = $1`, userID); err != nil {
			return fmt.Errorf("delete all events: %w", err)
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
