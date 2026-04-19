package repository

import (
	"database/sql"
	"fmt"

	"github.com/shanegleeson/beepbopboop/backend/internal/model"
)

type TrainingRepo struct {
	db *sql.DB
}

func NewTrainingRepo(db *sql.DB) *TrainingRepo {
	return &TrainingRepo{db: db}
}

// ComputeLabel converts raw engagement signals into a training label.
// Hard negatives (less/not_for_me) take priority over all positive signals.
// "stale" is excluded: it signals content age, not dislike, so it's not a training negative.
func ComputeLabel(saved, clicked bool, maxDwellMs int, reaction string) float64 {
	if reaction == "less" || reaction == "not_for_me" {
		return 0.0
	}
	if saved || reaction == "more" {
		return 1.0
	}
	if maxDwellMs >= 10000 {
		return 1.0
	}
	if clicked {
		return 0.8
	}
	if maxDwellMs >= 3000 {
		return 0.6
	}
	if maxDwellMs >= 1000 {
		return 0.3
	}
	return 0.0
}

// ExportPairs queries engagement data for the last `days` days and returns
// deduplicated, labeled (user, post) training pairs.
// Each (user, post) pair appears at most once; signals are aggregated across all events.
func (r *TrainingRepo) ExportPairs(days int) ([]model.TrainingPair, error) {
	rows, err := r.db.Query(`
		SELECT
			pe.user_id,
			pe.post_id,
			BOOL_OR(pe.event_type = 'save') AS saved,
			BOOL_OR(pe.event_type = 'click') AS clicked,
			COALESCE(MAX(pe.dwell_ms), 0) AS max_dwell_ms,
			COALESCE((
				SELECT reaction FROM post_reactions
				WHERE post_id = pe.post_id AND user_id = pe.user_id
			), '') AS reaction
		FROM post_events pe
		WHERE pe.created_at > NOW() - INTERVAL '1 day' * $1
		GROUP BY pe.user_id, pe.post_id
		ORDER BY pe.user_id, pe.post_id`,
		days,
	)
	if err != nil {
		return nil, fmt.Errorf("query training pairs: %w", err)
	}
	defer rows.Close()

	var pairs []model.TrainingPair
	for rows.Next() {
		var p model.TrainingPair
		if err := rows.Scan(&p.UserID, &p.PostID, &p.Saved, &p.Clicked, &p.MaxDwellMs, &p.Reaction); err != nil {
			return nil, fmt.Errorf("scan training pair: %w", err)
		}
		p.Label = ComputeLabel(p.Saved, p.Clicked, p.MaxDwellMs, p.Reaction)
		pairs = append(pairs, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate training pairs: %w", err)
	}

	return pairs, nil
}

// ValidateLabelDistribution returns an error if the fraction of pairs with label >= 0.5
// falls outside [minPositive, maxPositive].
func ValidateLabelDistribution(pairs []model.TrainingPair, minPositive, maxPositive float64) error {
	if len(pairs) == 0 {
		return fmt.Errorf("no training pairs")
	}
	var positives int
	for _, p := range pairs {
		if p.Label >= 0.5 {
			positives++
		}
	}
	rate := float64(positives) / float64(len(pairs))
	if rate < minPositive || rate > maxPositive {
		return fmt.Errorf("positive rate %.2f outside [%.2f, %.2f]", rate, minPositive, maxPositive)
	}
	return nil
}

// SplitByUser partitions pairs into train/val/test splits with no user leakage.
// All pairs for a given user go entirely into one split.
// Users are assigned to splits in the order they first appear in pairs.
func SplitByUser(pairs []model.TrainingPair, trainRatio, valRatio float64) (train, val, test []model.TrainingPair) {
	userPairs := make(map[string][]model.TrainingPair)
	var users []string
	for _, p := range pairs {
		if len(userPairs[p.UserID]) == 0 {
			users = append(users, p.UserID)
		}
		userPairs[p.UserID] = append(userPairs[p.UserID], p)
	}

	nTrain := int(float64(len(users)) * trainRatio)
	nVal := int(float64(len(users)) * valRatio)

	for i, uid := range users {
		switch {
		case i < nTrain:
			train = append(train, userPairs[uid]...)
		case i < nTrain+nVal:
			val = append(val, userPairs[uid]...)
		default:
			test = append(test, userPairs[uid]...)
		}
	}
	return
}
