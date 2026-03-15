package repository

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"

	"github.com/shanegleeson/beepbopboop/backend/internal/model"
)

type UserRepo struct {
	db *sql.DB
}

func NewUserRepo(db *sql.DB) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) FindOrCreateByFirebaseUID(firebaseUID string) (*model.User, error) {
	var user model.User
	err := r.db.QueryRow(
		"SELECT id, firebase_uid, created_at FROM users WHERE firebase_uid = ?",
		firebaseUID,
	).Scan(&user.ID, &user.FirebaseUID, &user.CreatedAt)

	if err == sql.ErrNoRows {
		id, err := generateID()
		if err != nil {
			return nil, fmt.Errorf("generate id: %w", err)
		}
		_, err = r.db.Exec(
			"INSERT INTO users (id, firebase_uid) VALUES (?, ?)",
			id, firebaseUID,
		)
		if err != nil {
			return nil, fmt.Errorf("insert user: %w", err)
		}
		return r.FindOrCreateByFirebaseUID(firebaseUID)
	}
	if err != nil {
		return nil, fmt.Errorf("query user: %w", err)
	}
	return &user, nil
}

func generateID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
