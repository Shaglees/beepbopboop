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

// UpsertVerticalForFrequency adds or updates a vertical's weight from a
// posts-per-month frequency (1-30, where 30 ≈ daily). Other verticals are
// scaled so total weights still sum to 1.0. Pinned status on the target
// vertical is preserved.
//
// Mapping: targetWeight = freq/30 * 0.1 (so daily = 0.1, monthly ≈ 0.003).
// The 0.1 cap keeps a single user-skill from dominating a multi-vertical
// spread.
func (r *SpreadRepo) UpsertVerticalForFrequency(userID, name string, postsPerMonth int) error {
	if name == "" {
		return fmt.Errorf("vertical name required")
	}
	if postsPerMonth < 1 {
		postsPerMonth = 1
	}
	if postsPerMonth > 30 {
		postsPerMonth = 30
	}

	targets, err := r.GetTargets(userID)
	if err != nil {
		return fmt.Errorf("load existing targets: %w", err)
	}
	if targets == nil {
		targets = DefaultTargets()
	}
	if targets.Verticals == nil {
		targets.Verticals = map[string]model.SpreadVertical{}
	}

	targetWeight := float64(postsPerMonth) / 30.0 * 0.1

	// Sum every vertical except the one we're upserting.
	otherSum := 0.0
	for k, v := range targets.Verticals {
		if k != name {
			otherSum += v.Weight
		}
	}

	available := 1.0 - targetWeight
	if otherSum > 0 && available > 0 {
		scale := available / otherSum
		for k, v := range targets.Verticals {
			if k == name {
				continue
			}
			v.Weight *= scale
			targets.Verticals[k] = v
		}
	}

	pinned := false
	if existing, ok := targets.Verticals[name]; ok {
		pinned = existing.Pinned
	}
	targets.Verticals[name] = model.SpreadVertical{Weight: targetWeight, Pinned: pinned}

	return r.UpsertTargets(userID, targets)
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
