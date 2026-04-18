package repository

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/shanegleeson/beepbopboop/backend/internal/model"
)

type ReactionRepo struct {
	db *sql.DB
}

func NewReactionRepo(db *sql.DB) *ReactionRepo {
	return &ReactionRepo{db: db}
}

// Upsert sets or replaces a user's reaction on a post (last one wins).
func (r *ReactionRepo) Upsert(postID, userID, reaction string) (*model.PostReaction, error) {
	var pr model.PostReaction
	err := r.db.QueryRow(`
		INSERT INTO post_reactions (post_id, user_id, reaction)
		VALUES ($1, $2, $3)
		ON CONFLICT (post_id, user_id)
		DO UPDATE SET reaction = $3, updated_at = CURRENT_TIMESTAMP
		RETURNING post_id, user_id, reaction, created_at, updated_at`,
		postID, userID, reaction,
	).Scan(&pr.PostID, &pr.UserID, &pr.Reaction, &pr.CreatedAt, &pr.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("upsert reaction: %w", err)
	}
	// Keep denormalized reaction_count in sync (count only positive "more" reactions).
	if err := r.syncReactionCount(postID); err != nil {
		return nil, fmt.Errorf("sync reaction_count: %w", err)
	}
	return &pr, nil
}

// Delete removes a user's reaction from a post.
func (r *ReactionRepo) Delete(postID, userID string) error {
	_, err := r.db.Exec(`DELETE FROM post_reactions WHERE post_id = $1 AND user_id = $2`, postID, userID)
	if err != nil {
		return fmt.Errorf("delete reaction: %w", err)
	}
	if err := r.syncReactionCount(postID); err != nil {
		return fmt.Errorf("sync reaction_count: %w", err)
	}
	return nil
}

// syncReactionCount updates the denormalized reaction_count on the post.
func (r *ReactionRepo) syncReactionCount(postID string) error {
	_, err := r.db.Exec(`
		UPDATE posts SET reaction_count = (
			SELECT COUNT(*) FROM post_reactions WHERE post_id = $1 AND reaction = 'more'
		) WHERE id = $1`, postID)
	return err
}

// GetForPost returns a user's reaction on a single post (empty string if none).
func (r *ReactionRepo) GetForPost(postID, userID string) (string, error) {
	var reaction string
	err := r.db.QueryRow(`
		SELECT reaction FROM post_reactions WHERE post_id = $1 AND user_id = $2`,
		postID, userID,
	).Scan(&reaction)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("get reaction: %w", err)
	}
	return reaction, nil
}

// GetForPosts returns a map of postID → reaction for a batch of posts.
func (r *ReactionRepo) GetForPosts(postIDs []string, userID string) (map[string]string, error) {
	if len(postIDs) == 0 {
		return nil, nil
	}

	var b strings.Builder
	b.WriteString("SELECT post_id, reaction FROM post_reactions WHERE user_id = ")
	b.WriteString(fmt.Sprintf("$1 AND post_id IN ("))

	args := []any{userID}
	for i, id := range postIDs {
		if i > 0 {
			b.WriteString(", ")
		}
		fmt.Fprintf(&b, "$%d", i+2)
		args = append(args, id)
	}
	b.WriteString(")")

	rows, err := r.db.Query(b.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("query reactions for posts: %w", err)
	}
	defer rows.Close()

	reactions := make(map[string]string)
	for rows.Next() {
		var postID, reaction string
		if err := rows.Scan(&postID, &reaction); err != nil {
			return nil, fmt.Errorf("scan reaction: %w", err)
		}
		reactions[postID] = reaction
	}
	return reactions, rows.Err()
}

// NegativeReactions is the set of reactions that should hide a post from the feed.
var NegativeReactions = map[string]bool{
	"less":       true,
	"stale":      true,
	"not_for_me": true,
}

// Summary returns aggregated reaction counts by label and post_type over the last N days.
func (r *ReactionRepo) Summary(userID string, days int) (*model.ReactionSummary, error) {
	summary := &model.ReactionSummary{PeriodDays: days}

	// Total reactions
	err := r.db.QueryRow(`
		SELECT COUNT(*) FROM post_reactions
		WHERE user_id = $1 AND updated_at > NOW() - INTERVAL '1 day' * $2`,
		userID, days,
	).Scan(&summary.TotalReactions)
	if err != nil {
		return nil, fmt.Errorf("count reactions: %w", err)
	}

	// Label reactions
	rows, err := r.db.Query(`
		SELECT label,
			COUNT(*) FILTER (WHERE pr.reaction = 'more') AS more,
			COUNT(*) FILTER (WHERE pr.reaction = 'less') AS less,
			COUNT(*) FILTER (WHERE pr.reaction = 'stale') AS stale,
			COUNT(*) FILTER (WHERE pr.reaction = 'not_for_me') AS not_for_me
		FROM post_reactions pr
		JOIN posts p ON p.id = pr.post_id,
		LATERAL jsonb_array_elements_text(p.labels::jsonb) AS label
		WHERE pr.user_id = $1 AND pr.updated_at > NOW() - INTERVAL '1 day' * $2
		GROUP BY label
		ORDER BY (COUNT(*) FILTER (WHERE pr.reaction = 'more'))::float /
			GREATEST(COUNT(*), 1) DESC`,
		userID, days,
	)
	if err != nil {
		return nil, fmt.Errorf("query label reactions: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var lr model.LabelReaction
		if err := rows.Scan(&lr.Label, &lr.More, &lr.Less, &lr.Stale, &lr.NotForMe); err != nil {
			return nil, fmt.Errorf("scan label reaction: %w", err)
		}
		summary.LabelReactions = append(summary.LabelReactions, lr)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate label reactions: %w", err)
	}

	// Type reactions
	rows2, err := r.db.Query(`
		SELECT p.post_type,
			COUNT(*) FILTER (WHERE pr.reaction = 'more') AS more,
			COUNT(*) FILTER (WHERE pr.reaction = 'less') AS less,
			COUNT(*) FILTER (WHERE pr.reaction = 'stale') AS stale,
			COUNT(*) FILTER (WHERE pr.reaction = 'not_for_me') AS not_for_me
		FROM post_reactions pr
		JOIN posts p ON p.id = pr.post_id
		WHERE pr.user_id = $1 AND pr.updated_at > NOW() - INTERVAL '1 day' * $2
		GROUP BY p.post_type
		ORDER BY more DESC`,
		userID, days,
	)
	if err != nil {
		return nil, fmt.Errorf("query type reactions: %w", err)
	}
	defer rows2.Close()

	for rows2.Next() {
		var tr model.TypeReaction
		if err := rows2.Scan(&tr.PostType, &tr.More, &tr.Less, &tr.Stale, &tr.NotForMe); err != nil {
			return nil, fmt.Errorf("scan type reaction: %w", err)
		}
		summary.TypeReactions = append(summary.TypeReactions, tr)
	}
	if err := rows2.Err(); err != nil {
		return nil, fmt.Errorf("iterate type reactions: %w", err)
	}

	return summary, nil
}
