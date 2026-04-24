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

const userColumns = `id, firebase_uid, display_name, avatar_url, timezone,
	home_location, home_lat, home_lon, profile_updated_at, created_at`

func scanUser(row interface{ Scan(...any) error }) (*model.User, error) {
	var u model.User
	err := row.Scan(
		&u.ID, &u.FirebaseUID, &u.DisplayName, &u.AvatarURL, &u.Timezone,
		&u.HomeLocation, &u.HomeLat, &u.HomeLon, &u.ProfileUpdatedAt, &u.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepo) FindOrCreateByFirebaseUID(firebaseUID string) (*model.User, error) {
	row := r.db.QueryRow(
		"SELECT "+userColumns+" FROM users WHERE firebase_uid = $1",
		firebaseUID,
	)
	u, err := scanUser(row)
	if err == nil {
		return u, nil
	}
	if err != sql.ErrNoRows {
		return nil, fmt.Errorf("query user by firebase_uid: %w", err)
	}

	id, err := generateID()
	if err != nil {
		return nil, fmt.Errorf("generate id: %w", err)
	}
	_, err = r.db.Exec(
		"INSERT INTO users (id, firebase_uid) VALUES ($1, $2)",
		id, firebaseUID,
	)
	if err != nil {
		return nil, fmt.Errorf("insert user: %w", err)
	}
	return r.FindOrCreateByFirebaseUID(firebaseUID)
}

func (r *UserRepo) GetByID(id string) (*model.User, error) {
	row := r.db.QueryRow(
		"SELECT "+userColumns+" FROM users WHERE id = $1", id,
	)
	return scanUser(row)
}

func (r *UserRepo) UpdateProfile(userID, displayName, avatarURL, timezone, homeLocation string, homeLat, homeLon *float64) error {
	_, err := r.db.Exec(`
		UPDATE users SET
			display_name = $2,
			avatar_url = $3,
			timezone = $4,
			home_location = $5,
			home_lat = $6,
			home_lon = $7,
			profile_updated_at = CURRENT_TIMESTAMP
		WHERE id = $1`,
		userID, displayName, avatarURL, timezone, homeLocation, homeLat, homeLon,
	)
	if err != nil {
		return fmt.Errorf("update profile: %w", err)
	}
	return nil
}

func generateID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
