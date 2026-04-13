package handler

import (
	"encoding/json"
	"net/http"

	"github.com/shanegleeson/beepbopboop/backend/internal/middleware"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

var validPostTypes = map[string]bool{
	"event":     true,
	"place":     true,
	"discovery": true,
	"article":   true,
	"video":     true,
}

var validVisibility = map[string]bool{
	"public":   true,
	"personal": true,
	"private":  true,
}

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
	Title       string   `json:"title"`
	Body        string   `json:"body"`
	ImageURL    string   `json:"image_url,omitempty"`
	ExternalURL string   `json:"external_url,omitempty"`
	Locality    string   `json:"locality,omitempty"`
	Latitude    *float64 `json:"latitude,omitempty"`
	Longitude   *float64 `json:"longitude,omitempty"`
	PostType    string   `json:"post_type,omitempty"`
	Visibility  string   `json:"visibility,omitempty"`
	Labels      []string `json:"labels,omitempty"`
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

	if req.PostType == "" {
		req.PostType = "discovery"
	}
	if !validPostTypes[req.PostType] {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid post_type: must be event, place, discovery, article, or video"})
		return
	}

	if req.Visibility == "" {
		req.Visibility = "public"
	}
	if !validVisibility[req.Visibility] {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid visibility: must be public, personal, or private"})
		return
	}

	if len(req.Labels) > 20 {
		req.Labels = req.Labels[:20]
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
		Latitude:    req.Latitude,
		Longitude:   req.Longitude,
		PostType:    req.PostType,
		Visibility:  req.Visibility,
		Labels:      req.Labels,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create post"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(post)
}
