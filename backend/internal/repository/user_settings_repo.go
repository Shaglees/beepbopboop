package repository

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"

	"github.com/shanegleeson/beepbopboop/backend/internal/model"
)

// GridCell represents a geographic grid cell with at least one user.
type GridCell struct {
	Lat       float64
	Lon       float64
	UserCount int
}

type UserSettingsRepo struct {
	db *sql.DB
}

func NewUserSettingsRepo(db *sql.DB) *UserSettingsRepo {
	return &UserSettingsRepo{db: db}
}

// Get returns the user's settings, or nil (not error) when none exist.
func (r *UserSettingsRepo) Get(userID string) (*model.UserSettings, error) {
	var s model.UserSettings
	var locationName sql.NullString
	var latitude, longitude sql.NullFloat64
	var followedTeamsJSON sql.NullString

	err := r.db.QueryRow(`
		SELECT user_id, location_name, latitude, longitude, radius_km,
		       followed_teams, notifications_enabled, digest_hour,
		       COALESCE(calendar_enabled, FALSE), updated_at
		FROM user_settings WHERE user_id = $1`, userID,
	).Scan(&s.UserID, &locationName, &latitude, &longitude, &s.RadiusKm,
		&followedTeamsJSON, &s.NotificationsEnabled, &s.DigestHour, &s.CalendarEnabled, &s.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query user_settings: %w", err)
	}
	s.LocationName = locationName.String
	if latitude.Valid {
		s.Latitude = &latitude.Float64
	}
	if longitude.Valid {
		s.Longitude = &longitude.Float64
	}
	if followedTeamsJSON.Valid && followedTeamsJSON.String != "" && followedTeamsJSON.String != "null" {
		if err := json.Unmarshal([]byte(followedTeamsJSON.String), &s.FollowedTeams); err != nil {
			return nil, fmt.Errorf("unmarshal followed_teams: %w", err)
		}
	}
	return &s, nil
}

// Upsert inserts or updates the user's settings.
func (r *UserSettingsRepo) Upsert(userID, locationName string, lat, lon *float64, radiusKm float64, followedTeams []string, notificationsEnabled bool, digestHour int, calendarEnabled bool) (*model.UserSettings, error) {
	var followedTeamsJSON sql.NullString
	if len(followedTeams) > 0 {
		b, err := json.Marshal(followedTeams)
		if err != nil {
			return nil, fmt.Errorf("marshal followed_teams: %w", err)
		}
		followedTeamsJSON = sql.NullString{String: string(b), Valid: true}
	}

	_, err := r.db.Exec(`
		INSERT INTO user_settings (user_id, location_name, latitude, longitude, radius_km, followed_teams, notifications_enabled, digest_hour, calendar_enabled, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, CURRENT_TIMESTAMP)
		ON CONFLICT(user_id) DO UPDATE SET
			location_name         = excluded.location_name,
			latitude              = excluded.latitude,
			longitude             = excluded.longitude,
			radius_km             = excluded.radius_km,
			followed_teams        = excluded.followed_teams,
			notifications_enabled = excluded.notifications_enabled,
			digest_hour           = excluded.digest_hour,
			calendar_enabled      = excluded.calendar_enabled,
			updated_at            = CURRENT_TIMESTAMP`,
		userID, nullString(locationName), nullFloat64(lat), nullFloat64(lon), radiusKm,
		followedTeamsJSON, notificationsEnabled, digestHour, calendarEnabled,
	)
	if err != nil {
		return nil, fmt.Errorf("upsert user_settings: %w", err)
	}
	return r.Get(userID)
}

// SetCalendarEnabled updates only the calendar_enabled flag for a user.
func (r *UserSettingsRepo) SetCalendarEnabled(userID string, enabled bool) error {
	_, err := r.db.Exec(`
		INSERT INTO user_settings (user_id, calendar_enabled, updated_at)
		VALUES ($1, $2, CURRENT_TIMESTAMP)
		ON CONFLICT(user_id) DO UPDATE SET
			calendar_enabled = excluded.calendar_enabled,
			updated_at       = CURRENT_TIMESTAMP`,
		userID, enabled)
	if err != nil {
		return fmt.Errorf("set calendar_enabled: %w", err)
	}
	return nil
}

// UsersWithCalendarEnabled returns all user IDs that have calendar integration enabled.
func (r *UserSettingsRepo) UsersWithCalendarEnabled() ([]string, error) {
	rows, err := r.db.Query(`SELECT user_id FROM user_settings WHERE calendar_enabled = TRUE`)
	if err != nil {
		return nil, fmt.Errorf("query calendar users: %w", err)
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan user_id: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// DistinctLocationCells returns unique geographic grid cells that have at least
// one user with a set location. Grid cells are ~gridSize degrees (~11 km at 0.1).
func (r *UserSettingsRepo) DistinctLocationCells(gridSize float64) ([]GridCell, error) {
	rows, err := r.db.Query(`
		SELECT
			ROUND(latitude / $1) * $1 AS grid_lat,
			ROUND(longitude / $1) * $1 AS grid_lon,
			COUNT(*) AS user_count
		FROM user_settings
		WHERE latitude IS NOT NULL AND longitude IS NOT NULL
		GROUP BY grid_lat, grid_lon`, gridSize)
	if err != nil {
		return nil, fmt.Errorf("query location cells: %w", err)
	}
	defer rows.Close()

	var cells []GridCell
	for rows.Next() {
		var c GridCell
		if err := rows.Scan(&c.Lat, &c.Lon, &c.UserCount); err != nil {
			return nil, fmt.Errorf("scan location cell: %w", err)
		}
		// Round to avoid floating point drift.
		c.Lat = math.Round(c.Lat/gridSize) * gridSize
		c.Lon = math.Round(c.Lon/gridSize) * gridSize
		cells = append(cells, c)
	}
	return cells, rows.Err()
}
