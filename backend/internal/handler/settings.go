package handler

import (
	"encoding/json"
	"net/http"

	"github.com/shanegleeson/beepbopboop/backend/internal/middleware"
	"github.com/shanegleeson/beepbopboop/backend/internal/model"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

type SettingsHandler struct {
	userRepo         *repository.UserRepo
	userSettingsRepo *repository.UserSettingsRepo
}

func NewSettingsHandler(userRepo *repository.UserRepo, userSettingsRepo *repository.UserSettingsRepo) *SettingsHandler {
	return &SettingsHandler{
		userRepo:         userRepo,
		userSettingsRepo: userSettingsRepo,
	}
}

func (h *SettingsHandler) GetSettings(w http.ResponseWriter, r *http.Request) {
	uid := middleware.FirebaseUIDFromContext(r.Context())

	user, err := h.userRepo.FindOrCreateByFirebaseUID(uid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to resolve user"})
		return
	}

	settings, err := h.userSettingsRepo.Get(user.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load settings"})
		return
	}

	if settings == nil {
		settings = &model.UserSettings{
			UserID:   user.ID,
			RadiusKm: 25,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(settings)
}

type updateSettingsRequest struct {
	LocationName         string   `json:"location_name"`
	Latitude             *float64 `json:"latitude"`
	Longitude            *float64 `json:"longitude"`
	RadiusKm             float64  `json:"radius_km"`
	FollowedTeams        []string `json:"followed_teams"`
	NotificationsEnabled *bool    `json:"notifications_enabled"`
	DigestHour           *int     `json:"digest_hour"`
	CalendarEnabled      *bool    `json:"calendar_enabled"`
}

func (h *SettingsHandler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	uid := middleware.FirebaseUIDFromContext(r.Context())

	user, err := h.userRepo.FindOrCreateByFirebaseUID(uid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to resolve user"})
		return
	}

	var req updateSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.RadiusKm <= 0 {
		req.RadiusKm = 25
	}
	if req.RadiusKm > 100 {
		req.RadiusKm = 100
	}

	notificationsEnabled := true
	if req.NotificationsEnabled != nil {
		notificationsEnabled = *req.NotificationsEnabled
	}

	digestHour := 8
	if req.DigestHour != nil {
		digestHour = *req.DigestHour
		if digestHour < 0 || digestHour > 23 {
			digestHour = 8
		}
	}

	settings, err := h.userSettingsRepo.Upsert(user.ID, req.LocationName, req.Latitude, req.Longitude, req.RadiusKm, req.FollowedTeams, notificationsEnabled, digestHour, req.CalendarEnabled)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save settings"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(settings)
}

func (h *SettingsHandler) UpdateLocation(w http.ResponseWriter, r *http.Request) {
	uid := middleware.FirebaseUIDFromContext(r.Context())

	user, err := h.userRepo.FindOrCreateByFirebaseUID(uid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to resolve user"})
		return
	}

	var req struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
		Name      string  `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}

	lat, lon := req.Latitude, req.Longitude
	if err := h.userSettingsRepo.SetLocation(user.ID, req.Name, &lat, &lon); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to update location"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
