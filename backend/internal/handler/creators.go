package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/shanegleeson/beepbopboop/backend/internal/middleware"
	"github.com/shanegleeson/beepbopboop/backend/internal/model"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

// CreatorsHandler serves local creator discovery endpoints.
type CreatorsHandler struct {
	userRepo         *repository.UserRepo
	userSettingsRepo *repository.UserSettingsRepo
	postRepo         *repository.PostRepo
}

func NewCreatorsHandler(
	userRepo *repository.UserRepo,
	userSettingsRepo *repository.UserSettingsRepo,
	postRepo *repository.PostRepo,
) *CreatorsHandler {
	return &CreatorsHandler{
		userRepo:         userRepo,
		userSettingsRepo: userSettingsRepo,
		postRepo:         postRepo,
	}
}

// GetLocalCreators handles GET /discovery/local-creators?lat=X&lon=Y&radius=Z
// It returns creator_spotlight posts near the requested coordinates.
func (h *CreatorsHandler) GetLocalCreators(w http.ResponseWriter, r *http.Request) {
	latStr := r.URL.Query().Get("lat")
	lonStr := r.URL.Query().Get("lon")
	radiusStr := r.URL.Query().Get("radius")

	// Resolve coordinates: prefer explicit query params, fall back to user settings.
	var lat, lon, radius float64

	if latStr != "" && lonStr != "" {
		var err error
		lat, err = strconv.ParseFloat(latStr, 64)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid lat"})
			return
		}
		lon, err = strconv.ParseFloat(lonStr, 64)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid lon"})
			return
		}
	} else {
		// Fall back to user's stored location.
		uid := middleware.FirebaseUIDFromContext(r.Context())
		user, err := h.userRepo.FindOrCreateByFirebaseUID(uid)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to resolve user"})
			return
		}
		settings, err := h.userSettingsRepo.Get(user.ID)
		if err != nil || settings == nil || settings.Latitude == nil || settings.Longitude == nil {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "location required"})
			return
		}
		lat = *settings.Latitude
		lon = *settings.Longitude
		if settings.RadiusKm > 0 {
			radius = settings.RadiusKm
		}
	}

	if radius <= 0 {
		if radiusStr != "" {
			var err error
			radius, err = strconv.ParseFloat(radiusStr, 64)
			if err != nil || radius <= 0 {
				radius = 25
			}
		} else {
			radius = 25
		}
	}
	if radius > 150 {
		radius = 150
	}

	posts, err := h.postRepo.ListCreatorsByRegion(lat, lon, radius, 50)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load creators"})
		return
	}

	if posts == nil {
		posts = []model.Post{}
	}

	resp := model.FeedResponse{Posts: posts}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// UpdateUserLocation handles POST /user/location
// It stores the user's current coordinates for background creator discovery.
type updateLocationRequest struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Accuracy  float64 `json:"accuracy_m,omitempty"`
}

func (h *CreatorsHandler) UpdateUserLocation(w http.ResponseWriter, r *http.Request) {
	uid := middleware.FirebaseUIDFromContext(r.Context())

	user, err := h.userRepo.FindOrCreateByFirebaseUID(uid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to resolve user"})
		return
	}

	var req updateLocationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.Latitude == 0 && req.Longitude == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "latitude and longitude required"})
		return
	}

	// Load existing settings so we don't overwrite other fields.
	existing, err := h.userSettingsRepo.Get(user.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load settings"})
		return
	}

	locationName := ""
	radiusKm := 25.0
	var followedTeams []string
	notificationsEnabled := true
	digestHour := 8

	if existing != nil {
		locationName = existing.LocationName
		if existing.RadiusKm > 0 {
			radiusKm = existing.RadiusKm
		}
		followedTeams = existing.FollowedTeams
		notificationsEnabled = existing.NotificationsEnabled
		digestHour = existing.DigestHour
	}

	lat := req.Latitude
	lon := req.Longitude
	settings, err := h.userSettingsRepo.Upsert(user.ID, locationName, &lat, &lon, radiusKm, followedTeams, notificationsEnabled, digestHour, nil)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to update location"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(settings)
}
