package repository

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/shanegleeson/beepbopboop/backend/internal/model"
)

// IntentRepo persists and retrieves user intent signals extracted from calendar events.
type IntentRepo struct {
	db *sql.DB
}

func NewIntentRepo(db *sql.DB) *IntentRepo {
	return &IntentRepo{db: db}
}

// UpsertIntents inserts or replaces a batch of intents for a user.
// Uses the intent ID as the conflict target so re-syncing calendar data is idempotent.
func (r *IntentRepo) UpsertIntents(intents []model.UserIntent) error {
	if len(intents) == 0 {
		return nil
	}

	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO user_intents (id, user_id, signal_type, intent_type, payload, active_from, active_until, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT(id) DO UPDATE SET
			payload      = excluded.payload,
			active_from  = excluded.active_from,
			active_until = excluded.active_until`)
	if err != nil {
		return fmt.Errorf("prepare stmt: %w", err)
	}
	defer stmt.Close()

	for _, intent := range intents {
		payloadBytes, err := json.Marshal(intent.Payload)
		if err != nil {
			return fmt.Errorf("marshal payload: %w", err)
		}
		if _, err := stmt.Exec(
			intent.ID, intent.UserID, intent.SignalType, intent.IntentType,
			payloadBytes, intent.ActiveFrom, intent.ActiveUntil, intent.CreatedAt,
		); err != nil {
			return fmt.Errorf("upsert intent %s: %w", intent.ID, err)
		}
	}

	return tx.Commit()
}

// GetActive returns all intents that are currently active (now falls within
// active_from .. active_until) for the given user.
func (r *IntentRepo) GetActive(userID string) ([]model.UserIntent, error) {
	now := time.Now()

	rows, err := r.db.Query(`
		SELECT id, user_id, signal_type, intent_type, payload, active_from, active_until, created_at
		FROM user_intents
		WHERE user_id = $1
		  AND active_from  <= $2
		  AND active_until >= $2`,
		userID, now,
	)
	if err != nil {
		return nil, fmt.Errorf("query active intents: %w", err)
	}
	defer rows.Close()

	var intents []model.UserIntent
	for rows.Next() {
		var i model.UserIntent
		var payloadRaw []byte
		if err := rows.Scan(
			&i.ID, &i.UserID, &i.SignalType, &i.IntentType,
			&payloadRaw, &i.ActiveFrom, &i.ActiveUntil, &i.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan intent: %w", err)
		}
		i.Payload = json.RawMessage(payloadRaw)
		intents = append(intents, i)
	}
	return intents, rows.Err()
}

// DeleteExpired removes intent rows whose active_until is in the past.
// Called periodically to keep the table tidy.
func (r *IntentRepo) DeleteExpired() (int64, error) {
	result, err := r.db.Exec(
		"DELETE FROM user_intents WHERE active_until < $1", time.Now(),
	)
	if err != nil {
		return 0, fmt.Errorf("delete expired intents: %w", err)
	}
	return result.RowsAffected()
}
