package repository

import (
	"database/sql"
	"fmt"

	"github.com/shanegleeson/beepbopboop/backend/internal/model"
)

type CreatePostParams struct {
	AgentID     string
	UserID      string
	Title       string
	Body        string
	ImageURL    string
	ExternalURL string
	Locality    string
	PostType    string
}

type PostRepo struct {
	db *sql.DB
}

func NewPostRepo(db *sql.DB) *PostRepo {
	return &PostRepo{db: db}
}

func (r *PostRepo) Create(p CreatePostParams) (*model.Post, error) {
	id, err := generateID()
	if err != nil {
		return nil, fmt.Errorf("generate id: %w", err)
	}

	_, err = r.db.Exec(`
		INSERT INTO posts (id, agent_id, user_id, title, body, image_url, external_url, locality, post_type)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, p.AgentID, p.UserID, p.Title, p.Body,
		nullString(p.ImageURL), nullString(p.ExternalURL),
		nullString(p.Locality), nullString(p.PostType),
	)
	if err != nil {
		return nil, fmt.Errorf("insert post: %w", err)
	}

	return r.GetByID(id)
}

func (r *PostRepo) GetByID(id string) (*model.Post, error) {
	var post model.Post
	var imageURL, externalURL, locality, postType sql.NullString
	err := r.db.QueryRow(`
		SELECT p.id, p.agent_id, a.name, p.user_id, p.title, p.body,
		       p.image_url, p.external_url, p.locality, p.post_type, p.created_at
		FROM posts p
		JOIN agents a ON a.id = p.agent_id
		WHERE p.id = ?`, id,
	).Scan(&post.ID, &post.AgentID, &post.AgentName, &post.UserID,
		&post.Title, &post.Body,
		&imageURL, &externalURL, &locality, &postType, &post.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("query post: %w", err)
	}
	post.ImageURL = imageURL.String
	post.ExternalURL = externalURL.String
	post.Locality = locality.String
	post.PostType = postType.String
	return &post, nil
}

func (r *PostRepo) ListByUserID(userID string, limit int) ([]model.Post, error) {
	rows, err := r.db.Query(`
		SELECT p.id, p.agent_id, a.name, p.user_id, p.title, p.body,
		       p.image_url, p.external_url, p.locality, p.post_type, p.created_at
		FROM posts p
		JOIN agents a ON a.id = p.agent_id
		WHERE p.user_id = ?
		ORDER BY p.created_at DESC, p.rowid DESC
		LIMIT ?`, userID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("query posts: %w", err)
	}
	defer rows.Close()

	posts := make([]model.Post, 0)
	for rows.Next() {
		var p model.Post
		var imageURL, externalURL, locality, postType sql.NullString
		if err := rows.Scan(&p.ID, &p.AgentID, &p.AgentName, &p.UserID,
			&p.Title, &p.Body,
			&imageURL, &externalURL, &locality, &postType, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan post: %w", err)
		}
		p.ImageURL = imageURL.String
		p.ExternalURL = externalURL.String
		p.Locality = locality.String
		p.PostType = postType.String
		posts = append(posts, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate posts: %w", err)
	}
	return posts, nil
}

func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}
