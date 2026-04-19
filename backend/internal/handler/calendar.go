package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/shanegleeson/beepbopboop/backend/internal/calendar"
	"github.com/shanegleeson/beepbopboop/backend/internal/middleware"
	"github.com/shanegleeson/beepbopboop/backend/internal/model"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

// CalendarHandler handles calendar-context endpoints.
type CalendarHandler struct {
	userRepo   *repository.UserRepo
	intentRepo *repository.IntentRepo
}

func NewCalendarHandler(userRepo *repository.UserRepo, intentRepo *repository.IntentRepo) *CalendarHandler {
	return &CalendarHandler{userRepo: userRepo, intentRepo: intentRepo}
}

// PostCalendarContext receives upcoming calendar events from the iOS client,
// extracts structured intent signals, and persists them for feed personalisation.
//
// POST /user/calendar-context
func (h *CalendarHandler) PostCalendarContext(w http.ResponseWriter, r *http.Request) {
	uid := middleware.FirebaseUIDFromContext(r.Context())

	user, err := h.userRepo.FindOrCreateByFirebaseUID(uid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to resolve user"})
		return
	}

	var req model.CalendarContextRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if len(req.Events) == 0 {
		writeJSON(w, http.StatusOK, model.CalendarContextResponse{IntentsExtracted: 0})
		return
	}

	// Cap at 100 events per sync to avoid abuse.
	if len(req.Events) > 100 {
		req.Events = req.Events[:100]
	}

	intents := calendar.ExtractIntents(user.ID, req.Events)

	if err := h.intentRepo.UpsertIntents(intents); err != nil {
		slog.Error("failed to upsert calendar intents", "user_id", user.ID, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save calendar context"})
		return
	}

	slog.Info("calendar context synced", "user_id", user.ID,
		"events", len(req.Events), "intents", len(intents))

	writeJSON(w, http.StatusOK, model.CalendarContextResponse{IntentsExtracted: len(intents)})
}

// GetCalendarContext returns the user's currently-active intent signals.
// Useful for debugging and for the future "Context panel" in iOS settings.
//
// GET /user/calendar-context
func (h *CalendarHandler) GetCalendarContext(w http.ResponseWriter, r *http.Request) {
	uid := middleware.FirebaseUIDFromContext(r.Context())

	user, err := h.userRepo.FindOrCreateByFirebaseUID(uid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to resolve user"})
		return
	}

	intents, err := h.intentRepo.GetActive(user.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load calendar context"})
		return
	}

	if intents == nil {
		intents = []model.UserIntent{}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"intents": intents})
}
