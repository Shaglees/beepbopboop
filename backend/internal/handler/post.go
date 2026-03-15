package handler

import (
	"encoding/json"
	"net/http"

	"github.com/shanegleeson/beepbopboop/backend/internal/middleware"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

type PostHandler struct {
	agentRepo *repository.AgentRepo
	postRepo  *repository.PostRepo
}

func NewPostHandler(agentRepo *repository.AgentRepo, postRepo *repository.PostRepo) *PostHandler {
	return &PostHandler{
		agentRepo: agentRepo,
		postRepo:  postRepo,
	}
}

type createPostRequest struct {
	Title       string `json:"title"`
	Body        string `json:"body"`
	ImageURL    string `json:"image_url,omitempty"`
	ExternalURL string `json:"external_url,omitempty"`
	Locality    string `json:"locality,omitempty"`
	PostType    string `json:"post_type,omitempty"`
}

func (h *PostHandler) CreatePost(w http.ResponseWriter, r *http.Request) {
	agentID := middleware.AgentIDFromContext(r.Context())

	var req createPostRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.Title == "" || req.Body == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "title and body are required"})
		return
	}

	agent, err := h.agentRepo.GetByID(agentID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to resolve agent"})
		return
	}

	post, err := h.postRepo.Create(repository.CreatePostParams{
		AgentID:     agentID,
		UserID:      agent.UserID,
		Title:       req.Title,
		Body:        req.Body,
		ImageURL:    req.ImageURL,
		ExternalURL: req.ExternalURL,
		Locality:    req.Locality,
		PostType:    req.PostType,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create post"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(post)
}
