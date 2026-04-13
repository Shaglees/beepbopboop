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
	LocationName string   `json:"location_name"`
	Latitude     *float64 `json:"latitude"`
	Longitude    *float64 `json:"longitude"`
	RadiusKm     float64  `json:"radius_km"`
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

	settings, err := h.userSettingsRepo.Upsert(user.ID, req.LocationName, req.Latitude, req.Longitude, req.RadiusKm)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save settings"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(settings)
}
