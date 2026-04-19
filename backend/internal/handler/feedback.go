package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/shanegleeson/beepbopboop/backend/internal/middleware"
	"github.com/shanegleeson/beepbopboop/backend/internal/model"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

// FeedbackHandler handles feedback submission and retrieval for feedback posts.
type FeedbackHandler struct {
	userRepo     *repository.UserRepo
	feedbackRepo *repository.FeedbackRepo
}

func NewFeedbackHandler(userRepo *repository.UserRepo, feedbackRepo *repository.FeedbackRepo) *FeedbackHandler {
	return &FeedbackHandler{userRepo: userRepo, feedbackRepo: feedbackRepo}
}

// SubmitResponse handles POST /posts/{postID}/response
// Body: { "type": "poll", "selected": ["nba","nhl"] }
//       { "type": "freeform", "text": "I prefer local food" }
//       { "type": "rating", "value": 4 }
//       { "type": "survey", "answers": [...] }
func (h *FeedbackHandler) SubmitResponse(w http.ResponseWriter, r *http.Request) {
	uid := middleware.FirebaseUIDFromContext(r.Context())
	user, err := h.userRepo.FindOrCreateByFirebaseUID(uid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to resolve user"})
		return
	}

	postID := chi.URLParam(r, "postID")

	var body model.FeedbackResponseBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	// Basic validation
	switch body.Type {
	case "poll", "survey":
		if len(body.Selected) == 0 && len(body.Answers) == 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "poll/survey response requires selected or answers"})
			return
		}
	case "freeform":
		if body.Text == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "freeform response requires text"})
			return
		}
	case "rating":
		if body.Value == nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "rating response requires value"})
			return
		}
	default:
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "type must be one of: poll, survey, freeform, rating"})
		return
	}

	// Marshal response for storage
	responseJSON, err := json.Marshal(body)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to marshal response"})
		return
	}

	fb, err := h.feedbackRepo.Upsert(postID, user.ID, responseJSON)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save response"})
		return
	}

	writeJSON(w, http.StatusCreated, fb)
}

// GetResponses handles GET /posts/{postID}/responses
// Returns aggregated results for a feedback post.
func (h *FeedbackHandler) GetResponses(w http.ResponseWriter, r *http.Request) {
	uid := middleware.FirebaseUIDFromContext(r.Context())
	user, err := h.userRepo.FindOrCreateByFirebaseUID(uid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to resolve user"})
		return
	}

	postID := chi.URLParam(r, "postID")

	summary, err := h.feedbackRepo.GetSummary(postID, user.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to get responses"})
		return
	}

	writeJSON(w, http.StatusOK, summary)
}
