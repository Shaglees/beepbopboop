package handler

import (
	"encoding/json"
	"net/http"

	"github.com/shanegleeson/beepbopboop/backend/internal/middleware"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
	"github.com/shanegleeson/beepbopboop/backend/internal/weather"
)

type WeatherHandler struct {
	userRepo         *repository.UserRepo
	userSettingsRepo *repository.UserSettingsRepo
	weatherSvc       *weather.Service
}

func NewWeatherHandler(userRepo *repository.UserRepo, userSettingsRepo *repository.UserSettingsRepo, weatherSvc *weather.Service) *WeatherHandler {
	return &WeatherHandler{
		userRepo:         userRepo,
		userSettingsRepo: userSettingsRepo,
		weatherSvc:       weatherSvc,
	}
}

func (h *WeatherHandler) GetWeather(w http.ResponseWriter, r *http.Request) {
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
	if settings == nil || settings.Latitude == nil || settings.Longitude == nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "location_required"})
		return
	}

	data, err := h.weatherSvc.Fetch(*settings.Latitude, *settings.Longitude)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "weather_unavailable"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}
