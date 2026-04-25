package repository

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/shanegleeson/beepbopboop/backend/internal/model"
)

type SpreadRepo struct {
	db *sql.DB
}

func NewSpreadRepo(db *sql.DB) *SpreadRepo {
	return &SpreadRepo{db: db}
}

// DefaultTargets returns an even distribution across all known verticals.
func DefaultTargets() *model.SpreadTargets {
	verticals := []string{"sports", "food", "music", "travel", "science", "gaming", "creators", "fashion", "movies", "pets", "news"}
	weight := 1.0 / float64(len(verticals))
	m := make(map[string]model.SpreadVertical, len(verticals))
	for _, v := range verticals {
		m[v] = model.SpreadVertical{Weight: weight, Pinned: false}
	}
	return &model.SpreadTargets{
		Verticals:  m,
		Omega:      "sports",
		AutoAdjust: true,
		UpdatedAt:  time.Now(),
	}
}

// GetTargets returns the user's spread targets, or nil when none are stored.
func (r *SpreadRepo) GetTargets(userID string) (*model.SpreadTargets, error) {
	var raw sql.NullString
	err := r.db.QueryRow(
		`SELECT spread_targets FROM user_settings WHERE user_id = $1`, userID,
	).Scan(&raw)
	if err == sql.ErrNoRows || !raw.Valid {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query spread_targets: %w", err)
	}
	var st model.SpreadTargets
	if err := json.Unmarshal([]byte(raw.String), &st); err != nil {
		return nil, fmt.Errorf("unmarshal spread_targets: %w", err)
	}
	return &st, nil
}

// UpsertTargets stores the user's spread targets.
func (r *SpreadRepo) UpsertTargets(userID string, st *model.SpreadTargets) error {
	st.UpdatedAt = time.Now()
	b, err := json.Marshal(st)
	if err != nil {
		return fmt.Errorf("marshal spread_targets: %w", err)
	}
	_, err = r.db.Exec(`
		INSERT INTO user_settings (user_id, spread_targets, updated_at)
		VALUES ($1, $2, CURRENT_TIMESTAMP)
		ON CONFLICT(user_id) DO UPDATE SET
			spread_targets = excluded.spread_targets,
			updated_at     = CURRENT_TIMESTAMP`,
		userID, string(b))
	if err != nil {
		return fmt.Errorf("upsert spread_targets: %w", err)
	}
	return nil
}

// Actual30d computes the actual allocation over the last 30 days from posts.
// It groups posts by their first label and returns the fraction for each label.
func (r *SpreadRepo) Actual30d(userID string) (map[string]float64, error) {
	rows, err := r.db.Query(`
		SELECT labels FROM posts
		WHERE agent_id IN (SELECT id FROM agents WHERE user_id = $1)
		  AND created_at > NOW() - INTERVAL '30 days'
		  AND visibility = 'public'`,
		userID)
	if err != nil {
		return nil, fmt.Errorf("query posts for actual_30d: %w", err)
	}
	defer rows.Close()

	counts := make(map[string]int)
	total := 0
	for rows.Next() {
		var labelsRaw sql.NullString
		if err := rows.Scan(&labelsRaw); err != nil {
			return nil, fmt.Errorf("scan labels: %w", err)
		}
		if !labelsRaw.Valid || labelsRaw.String == "" {
			continue
		}
		// Labels are stored as a JSON array; first element is the vertical.
		primary := firstLabel(labelsRaw.String)
		if primary != "" {
			counts[primary]++
			total++
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	result := make(map[string]float64, len(counts))
	if total == 0 {
		return result, nil
	}
	for k, v := range counts {
		result[k] = float64(v) / float64(total)
	}
	return result, nil
}

// firstLabel extracts the first label from a JSON-array labels column.
func firstLabel(labels string) string {
	var arr []string
	if err := json.Unmarshal([]byte(labels), &arr); err != nil || len(arr) == 0 {
		return ""
	}
	return arr[0]
}

// InsertHistory writes a daily snapshot row.
func (r *SpreadRepo) InsertHistory(userID string, date string, targets, actuals map[string]float64) error {
	targetsJSON, _ := json.Marshal(targets)
	actualsJSON, _ := json.Marshal(actuals)
	_, err := r.db.Exec(`
		INSERT INTO spread_history (user_id, date, targets, actuals)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id, date) DO UPDATE SET
			targets = excluded.targets,
			actuals = excluded.actuals`,
		userID, date, string(targetsJSON), string(actualsJSON))
	if err != nil {
		return fmt.Errorf("insert spread_history: %w", err)
	}
	return nil
}

// GetHistory returns the last N days of spread history.
func (r *SpreadRepo) GetHistory(userID string, days int) ([]model.SpreadHistoryDay, error) {
	rows, err := r.db.Query(`
		SELECT date, targets, actuals FROM spread_history
		WHERE user_id = $1 AND date > CURRENT_DATE - $2::INT
		ORDER BY date ASC`, userID, days)
	if err != nil {
		return nil, fmt.Errorf("query spread_history: %w", err)
	}
	defer rows.Close()

	var result []model.SpreadHistoryDay
	for rows.Next() {
		var d model.SpreadHistoryDay
		var dateVal time.Time
		var targetsRaw, actualsRaw string
		if err := rows.Scan(&dateVal, &targetsRaw, &actualsRaw); err != nil {
			return nil, fmt.Errorf("scan spread_history: %w", err)
		}
		d.Date = dateVal.Format("2006-01-02")
		json.Unmarshal([]byte(targetsRaw), &d.Target)
		json.Unmarshal([]byte(actualsRaw), &d.Actual)
		result = append(result, d)
	}
	return result, rows.Err()
}
