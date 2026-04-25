package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/lib/pq"
	"github.com/shanegleeson/beepbopboop/backend/internal/model"
)

type CalendarEventRepo struct {
	db *sql.DB
}

func NewCalendarEventRepo(db *sql.DB) *CalendarEventRepo {
	return &CalendarEventRepo{db: db}
}

// Upsert inserts or updates an interest calendar event keyed by event_key.
func (r *CalendarEventRepo) Upsert(e model.InterestCalendarEvent) error {
	_, err := r.db.Exec(`
		INSERT INTO interest_calendar_events
			(event_key, domain, title, start_time, end_time, timezone, status, entity_type, entity_ids, interest_tags, payload)
		VALUES
			($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (event_key) DO UPDATE SET
			title         = EXCLUDED.title,
			start_time    = EXCLUDED.start_time,
			end_time      = EXCLUDED.end_time,
			status        = EXCLUDED.status,
			entity_ids    = EXCLUDED.entity_ids,
			interest_tags = EXCLUDED.interest_tags,
			payload       = EXCLUDED.payload,
			updated_at    = CURRENT_TIMESTAMP`,
		e.EventKey, e.Domain, e.Title, e.StartTime, nullTime(e.EndTime),
		e.Timezone, e.Status, e.EntityType,
		nullableJSON(e.EntityIDs), pq.Array(e.InterestTags), nullableJSON(e.Payload),
	)
	if err != nil {
		return fmt.Errorf("upsert calendar event: %w", err)
	}
	return nil
}

// Upcoming returns all interest calendar events for the given domain in [from, to].
func (r *CalendarEventRepo) Upcoming(domain string, from, to time.Time) ([]model.InterestCalendarEvent, error) {
	rows, err := r.db.Query(`
		SELECT id, event_key, domain, title, start_time, end_time, timezone, status,
		       entity_type, entity_ids, interest_tags, payload, created_at, updated_at
		FROM interest_calendar_events
		WHERE domain = $1
		  AND start_time >= $2
		  AND start_time <= $3
		ORDER BY start_time ASC`,
		domain, from, to,
	)
	if err != nil {
		return nil, fmt.Errorf("upcoming calendar events: %w", err)
	}
	defer rows.Close()

	return scanCalendarEvents(rows)
}

// ForUser returns interest calendar events matching the user's interests in [from, to].
// Returns nil if interests is empty.
func (r *CalendarEventRepo) ForUser(userID string, interests []string, from, to time.Time) ([]model.InterestCalendarEvent, error) {
	if len(interests) == 0 {
		return nil, nil
	}

	rows, err := r.db.Query(`
		SELECT id, event_key, domain, title, start_time, end_time, timezone, status,
		       entity_type, entity_ids, interest_tags, payload, created_at, updated_at
		FROM interest_calendar_events
		WHERE interest_tags && $1
		  AND start_time >= $2
		  AND start_time <= $3
		  AND status IN ('scheduled', 'live')
		ORDER BY start_time ASC`,
		pq.Array(interests), from, to,
	)
	if err != nil {
		return nil, fmt.Errorf("for user calendar events: %w", err)
	}
	defer rows.Close()

	return scanCalendarEvents(rows)
}

// LogPost records that a post has been published for an event/user/window combo.
func (r *CalendarEventRepo) LogPost(eventKey, userID, window, postID string) error {
	_, err := r.db.Exec(`
		INSERT INTO calendar_post_log (event_key, user_id, "window", post_id)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT DO NOTHING`,
		eventKey, userID, window, postID,
	)
	if err != nil {
		return fmt.Errorf("log calendar post: %w", err)
	}
	return nil
}

// IsPublished reports whether a post has already been published for event/user/window.
func (r *CalendarEventRepo) IsPublished(eventKey, userID, window string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM calendar_post_log
			WHERE event_key = $1 AND user_id = $2 AND "window" = $3
		)`,
		eventKey, userID, window,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check calendar post published: %w", err)
	}
	return exists, nil
}

// scanCalendarEvents scans a result set of interest calendar event rows.
func scanCalendarEvents(rows *sql.Rows) ([]model.InterestCalendarEvent, error) {
	var result []model.InterestCalendarEvent
	for rows.Next() {
		var e model.InterestCalendarEvent
		var endTime sql.NullTime
		var entityIDs, payload []byte
		if err := rows.Scan(
			&e.ID, &e.EventKey, &e.Domain, &e.Title,
			&e.StartTime, &endTime,
			&e.Timezone, &e.Status, &e.EntityType,
			&entityIDs, pq.Array(&e.InterestTags), &payload,
			&e.CreatedAt, &e.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan calendar event: %w", err)
		}
		if endTime.Valid {
			t := endTime.Time
			e.EndTime = &t
		}
		e.EntityIDs = entityIDs
		e.Payload = payload
		result = append(result, e)
	}
	return result, rows.Err()
}

// nullableJSON returns a JSON value or nil if it's empty/null.
func nullableJSON(b []byte) interface{} {
	if len(b) == 0 {
		return nil
	}
	return b
}
