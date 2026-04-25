package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/shanegleeson/beepbopboop/backend/internal/model"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

type NewsSourceHandler struct {
	repo *repository.NewsSourceRepo
}

func NewNewsSourceHandler(repo *repository.NewsSourceRepo) *NewsSourceHandler {
	return &NewsSourceHandler{repo: repo}
}

// List handles GET /news-sources — returns news sources near a location.
// Query params: lat, lon, radius_km (default 50.0), topics (comma-separated).
func (h *NewsSourceHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	lat, err := strconv.ParseFloat(q.Get("lat"), 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid lat"})
		return
	}

	lon, err := strconv.ParseFloat(q.Get("lon"), 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid lon"})
		return
	}

	radiusKm := 50.0
	if raw := q.Get("radius_km"); raw != "" {
		radiusKm, err = strconv.ParseFloat(raw, 64)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid radius_km"})
			return
		}
	}

	var topics []string
	if raw := q.Get("topics"); raw != "" {
		for _, t := range strings.Split(raw, ",") {
			if trimmed := strings.TrimSpace(t); trimmed != "" {
				topics = append(topics, trimmed)
			}
		}
	}

	sources, err := h.repo.List(lat, lon, radiusKm, topics)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list news sources"})
		return
	}
	if sources == nil {
		sources = []model.NewsSource{}
	}

	writeJSON(w, http.StatusOK, sources)
}

// Create handles POST /news-sources — creates a new news source.
func (h *NewsSourceHandler) Create(w http.ResponseWriter, r *http.Request) {
	var src model.NewsSource
	if err := json.NewDecoder(r.Body).Decode(&src); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if src.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
		return
	}
	if src.URL == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "url is required"})
		return
	}
	if src.AreaLabel == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "area_label is required"})
		return
	}

	if err := h.repo.Create(src); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create news source"})
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{"status": "created"})
}

// Get handles GET /news-sources/{id} — returns a single news source by ID.
func (h *NewsSourceHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing id"})
		return
	}

	src, err := h.repo.Get(id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to get news source"})
		return
	}
	if src == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "news source not found"})
		return
	}

	writeJSON(w, http.StatusOK, src)
}
