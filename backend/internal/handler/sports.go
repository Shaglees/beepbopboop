package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/shanegleeson/beepbopboop/backend/internal/sports"
)

type SportsHandler struct {
	svc *sports.Service
}

func NewSportsHandler(svc *sports.Service) *SportsHandler {
	return &SportsHandler{svc: svc}
}

// GetScores returns live scores for all supported leagues (NHL, NBA, MLB, NFL).
func (h *SportsHandler) GetScores(w http.ResponseWriter, r *http.Request) {
	games, err := h.svc.FetchAll()
	if err != nil {
		slog.Error("sports handler: fetch failed", "error", err)
		http.Error(w, "failed to fetch scores", http.StatusInternalServerError)
		return
	}

	data := make([]sports.GameData, len(games))
	for i, g := range games {
		data[i] = g.Data
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}
