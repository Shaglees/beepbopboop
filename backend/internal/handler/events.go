package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/shanegleeson/beepbopboop/backend/internal/middleware"
	"github.com/shanegleeson/beepbopboop/backend/internal/model"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

var validEventTypes = map[string]bool{
	"view":   true,
	"save":   true,
	"unsave": true,
	"click":  true,
	"share":  true,
}

// EventsHandler handles engagement event endpoints.
type EventsHandler struct {
	userRepo  *repository.UserRepo
	agentRepo *repository.AgentRepo
	eventRepo *repository.EventRepo
}

func NewEventsHandler(userRepo *repository.UserRepo, agentRepo *repository.AgentRepo, eventRepo *repository.EventRepo) *EventsHandler {
	return &EventsHandler{
		userRepo:  userRepo,
		agentRepo: agentRepo,
		eventRepo: eventRepo,
	}
}

// TrackEvent records a single engagement event (Firebase-auth, from iOS app).
func (h *EventsHandler) TrackEvent(w http.ResponseWriter, r *http.Request) {
	uid := middleware.FirebaseUIDFromContext(r.Context())
	user, err := h.userRepo.FindOrCreateByFirebaseUID(uid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to resolve user"})
		return
	}

	postID := chi.URLParam(r, "postID")
	if postID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing post ID"})
		return
	}

	var req model.EventInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if !validEventTypes[req.EventType] {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid event_type"})
		return
	}

	if err := h.eventRepo.Create(postID, user.ID, req.EventType, req.DwellMs); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to record event"})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// BatchTrack records multiple engagement events at once (Firebase-auth, from iOS app).
func (h *EventsHandler) BatchTrack(w http.ResponseWriter, r *http.Request) {
	uid := middleware.FirebaseUIDFromContext(r.Context())
	user, err := h.userRepo.FindOrCreateByFirebaseUID(uid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to resolve user"})
		return
	}

	var req model.EventBatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if len(req.Events) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "no events provided"})
		return
	}
	if len(req.Events) > 100 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "max 100 events per batch"})
		return
	}

	for _, e := range req.Events {
		if !validEventTypes[e.EventType] {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid event_type: " + e.EventType})
			return
		}
		if e.PostID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing post_id in event"})
			return
		}
	}

	if err := h.eventRepo.BatchCreate(user.ID, req.Events); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to record events"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]int{"recorded": len(req.Events)})
}

// Summary returns aggregated engagement stats (agent-auth, for Lobs to read).
func (h *EventsHandler) Summary(w http.ResponseWriter, r *http.Request) {
	agentID := middleware.AgentIDFromContext(r.Context())
	agent, err := h.agentRepo.GetByID(agentID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to resolve agent"})
		return
	}

	summary, err := h.eventRepo.Summary(agent.UserID, 30)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to compute summary"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}
