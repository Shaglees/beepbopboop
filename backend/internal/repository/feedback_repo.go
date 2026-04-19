package repository

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/shanegleeson/beepbopboop/backend/internal/model"
)

// FeedbackRepo handles storage and retrieval of user feedback responses.
type FeedbackRepo struct {
	db *sql.DB
}

func NewFeedbackRepo(db *sql.DB) *FeedbackRepo {
	return &FeedbackRepo{db: db}
}

// Upsert inserts or replaces a user's response to a feedback post.
// One response per (post_id, user_id) — later answer replaces the earlier one.
func (r *FeedbackRepo) Upsert(postID, userID string, response json.RawMessage) (*model.UserFeedback, error) {
	_, err := r.db.Exec(`
		INSERT INTO user_feedback (post_id, user_id, response)
		VALUES ($1, $2, $3)
		ON CONFLICT (post_id, user_id) DO UPDATE SET
			response = excluded.response`,
		postID, userID, response,
	)
	if err != nil {
		return nil, fmt.Errorf("upsert feedback: %w", err)
	}
	return r.GetByPostAndUser(postID, userID)
}

// GetByPostAndUser returns a specific user's response to a post, or nil if none.
func (r *FeedbackRepo) GetByPostAndUser(postID, userID string) (*model.UserFeedback, error) {
	var fb model.UserFeedback
	var responseStr string
	err := r.db.QueryRow(`
		SELECT id, post_id, user_id, response, created_at
		FROM user_feedback
		WHERE post_id = $1 AND user_id = $2`,
		postID, userID,
	).Scan(&fb.ID, &fb.PostID, &fb.UserID, &responseStr, &fb.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query feedback: %w", err)
	}
	fb.Response = json.RawMessage(responseStr)
	return &fb, nil
}

// GetSummary returns aggregated results for a post.
func (r *FeedbackRepo) GetSummary(postID, currentUserID string) (*model.FeedbackSummary, error) {
	summary := &model.FeedbackSummary{PostID: postID}

	// Count total responses
	err := r.db.QueryRow(`SELECT COUNT(*) FROM user_feedback WHERE post_id = $1`, postID).
		Scan(&summary.TotalResponses)
	if err != nil {
		return nil, fmt.Errorf("count feedback: %w", err)
	}

	// Fetch current user's own response
	if currentUserID != "" {
		var responseStr sql.NullString
		err = r.db.QueryRow(`
			SELECT response FROM user_feedback WHERE post_id = $1 AND user_id = $2`,
			postID, currentUserID,
		).Scan(&responseStr)
		if err != nil && err != sql.ErrNoRows {
			return nil, fmt.Errorf("query my feedback: %w", err)
		}
		if responseStr.Valid {
			summary.MyResponse = json.RawMessage(responseStr.String)
		}
	}

	// Aggregate poll tallies — only works when response type is "poll"
	rows, err := r.db.Query(`
		SELECT response FROM user_feedback WHERE post_id = $1`, postID)
	if err != nil {
		return nil, fmt.Errorf("query feedback responses: %w", err)
	}
	defer rows.Close()

	tally := make(map[string]int)
	var totalRating float64
	var ratingCount int

	for rows.Next() {
		var responseStr string
		if err := rows.Scan(&responseStr); err != nil {
			continue
		}
		var resp model.FeedbackResponseBody
		if err := json.Unmarshal([]byte(responseStr), &resp); err != nil {
			continue
		}
		switch resp.Type {
		case "poll", "survey":
			for _, sel := range resp.Selected {
				tally[sel]++
			}
		case "rating":
			if resp.Value != nil {
				totalRating += *resp.Value
				ratingCount++
			}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate feedback: %w", err)
	}

	if len(tally) > 0 {
		summary.Tally = tally
	}
	if ratingCount > 0 {
		avg := totalRating / float64(ratingCount)
		summary.AvgRating = &avg
	}

	return summary, nil
}
