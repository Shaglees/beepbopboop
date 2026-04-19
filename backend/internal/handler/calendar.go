package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/shanegleeson/beepbopboop/backend/internal/middleware"
	"github.com/shanegleeson/beepbopboop/backend/internal/model"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

type CalendarHandler struct {
	userRepo         *repository.UserRepo
	calendarRepo     *repository.CalendarRepo
	userSettingsRepo *repository.UserSettingsRepo
}

func NewCalendarHandler(userRepo *repository.UserRepo, calendarRepo *repository.CalendarRepo, userSettingsRepo *repository.UserSettingsRepo) *CalendarHandler {
	return &CalendarHandler{
		userRepo:         userRepo,
		calendarRepo:     calendarRepo,
		userSettingsRepo: userSettingsRepo,
	}
}

type calendarEventInput struct {
	ID        string  `json:"id"`
	Title     string  `json:"title"`
	StartTime string  `json:"start_time"`
	EndTime   *string `json:"end_time,omitempty"`
	Location  string  `json:"location,omitempty"`
	Notes     string  `json:"notes,omitempty"`
}

type syncCalendarEventsRequest struct {
	Events []calendarEventInput `json:"events"`
}

func (h *CalendarHandler) SyncCalendarEvents(w http.ResponseWriter, r *http.Request) {
	uid := middleware.FirebaseUIDFromContext(r.Context())

	user, err := h.userRepo.FindOrCreateByFirebaseUID(uid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to resolve user"})
		return
	}

	var req syncCalendarEventsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if len(req.Events) > 500 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "too many events (max 500)"})
		return
	}

	events := make([]model.CalendarEvent, 0, len(req.Events))
	for _, e := range req.Events {
		if e.ID == "" || e.Title == "" || e.StartTime == "" {
			continue
		}
		startTime, err := time.Parse(time.RFC3339, e.StartTime)
		if err != nil {
			continue
		}
		ev := model.CalendarEvent{
			ID:        e.ID,
			UserID:    user.ID,
			Title:     e.Title,
			StartTime: startTime,
			Location:  e.Location,
			Notes:     e.Notes,
		}
		if e.EndTime != nil {
			if t, err := time.Parse(time.RFC3339, *e.EndTime); err == nil {
				ev.EndTime = &t
			}
		}
		events = append(events, ev)
	}

	if err := h.calendarRepo.UpsertEvents(user.ID, events); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to store events"})
		return
	}

	// Enable calendar integration for this user automatically on first sync.
	_ = h.userSettingsRepo.SetCalendarEnabled(user.ID, true)

	writeJSON(w, http.StatusOK, map[string]any{"synced": len(events)})
}
