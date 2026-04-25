package repository

import (
	"database/sql"
	"fmt"
)

type UserPhotoRepo struct {
	db *sql.DB
}

func NewUserPhotoRepo(db *sql.DB) *UserPhotoRepo {
	return &UserPhotoRepo{db: db}
}

// SaveHeadshot stores headshot image data and content type for the given user.
func (r *UserPhotoRepo) SaveHeadshot(userID string, data []byte, contentType string) error {
	_, err := r.db.Exec(
		`UPDATE users SET headshot_data=$1, headshot_type=$2 WHERE id=$3`,
		data, contentType, userID,
	)
	if err != nil {
		return fmt.Errorf("save headshot: %w", err)
	}
	return nil
}

// SaveBodyshot stores bodyshot image data and content type for the given user.
func (r *UserPhotoRepo) SaveBodyshot(userID string, data []byte, contentType string) error {
	_, err := r.db.Exec(
		`UPDATE users SET bodyshot_data=$1, bodyshot_type=$2 WHERE id=$3`,
		data, contentType, userID,
	)
	if err != nil {
		return fmt.Errorf("save bodyshot: %w", err)
	}
	return nil
}

// GetHeadshot returns the headshot image data and content type for the given user.
// Returns (nil, "", nil) if the user exists but has no headshot, or if the user is not found.
func (r *UserPhotoRepo) GetHeadshot(userID string) ([]byte, string, error) {
	var data []byte
	var ct sql.NullString
	err := r.db.QueryRow(
		`SELECT headshot_data, headshot_type FROM users WHERE id=$1`,
		userID,
	).Scan(&data, &ct)
	if err == sql.ErrNoRows {
		return nil, "", nil
	}
	if err != nil {
		return nil, "", fmt.Errorf("get headshot: %w", err)
	}
	if data == nil {
		return nil, "", nil
	}
	return data, ct.String, nil
}

// GetBodyshot returns the bodyshot image data and content type for the given user.
// Returns (nil, "", nil) if the user exists but has no bodyshot, or if the user is not found.
func (r *UserPhotoRepo) GetBodyshot(userID string) ([]byte, string, error) {
	var data []byte
	var ct sql.NullString
	err := r.db.QueryRow(
		`SELECT bodyshot_data, bodyshot_type FROM users WHERE id=$1`,
		userID,
	).Scan(&data, &ct)
	if err == sql.ErrNoRows {
		return nil, "", nil
	}
	if err != nil {
		return nil, "", fmt.Errorf("get bodyshot: %w", err)
	}
	if data == nil {
		return nil, "", nil
	}
	return data, ct.String, nil
}

// DeletePhoto NULLs the data and type columns for the given photo type ("headshot" or "bodyshot").
func (r *UserPhotoRepo) DeletePhoto(userID, photoType string) error {
	dataCol := fmt.Sprintf("%s_data", photoType)
	typeCol := fmt.Sprintf("%s_type", photoType)
	query := fmt.Sprintf(`UPDATE users SET %s=NULL, %s=NULL WHERE id=$1`, dataCol, typeCol)
	_, err := r.db.Exec(query, userID)
	if err != nil {
		return fmt.Errorf("delete %s: %w", photoType, err)
	}
	return nil
}
