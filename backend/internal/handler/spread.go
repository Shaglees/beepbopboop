package handler

import (
	"encoding/json"
	"math"
	"net/http"

	"github.com/shanegleeson/beepbopboop/backend/internal/middleware"
	"github.com/shanegleeson/beepbopboop/backend/internal/model"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

type SpreadHandler struct {
	userRepo   *repository.UserRepo
	spreadRepo *repository.SpreadRepo
}

func NewSpreadHandler(userRepo *repository.UserRepo, spreadRepo *repository.SpreadRepo) *SpreadHandler {
	return &SpreadHandler{userRepo: userRepo, spreadRepo: spreadRepo}
}

func (h *SpreadHandler) GetSpread(w http.ResponseWriter, r *http.Request) {
	uid := middleware.FirebaseUIDFromContext(r.Context())
	user, err := h.userRepo.FindOrCreateByFirebaseUID(uid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to resolve user"})
		return
	}

	st, err := h.spreadRepo.GetTargets(user.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load spread targets"})
		return
	}
	if st == nil {
		st = repository.DefaultTargets()
	}

	actual, err := h.spreadRepo.Actual30d(user.ID)
	if err != nil {
		actual = make(map[string]float64)
	}

	resp := model.SpreadResponse{
		Targets:    make(map[string]float64, len(st.Verticals)),
		Omega:      st.Omega,
		AutoAdjust: st.AutoAdjust,
		Actual30d:  actual,
		Status:     make(map[string]string, len(st.Verticals)),
	}

	for k, v := range st.Verticals {
		resp.Targets[k] = v.Weight
		if v.Pinned {
			resp.Pinned = append(resp.Pinned, k)
		}
		a := actual[k]
		diff := a - v.Weight
		if math.Abs(diff) <= 0.03 {
			resp.Status[k] = "on_target"
		} else if diff < 0 {
			resp.Status[k] = "below_target"
		} else {
			resp.Status[k] = "above_target"
		}
	}

	writeJSON(w, http.StatusOK, resp)
}

type putSpreadRequest struct {
	Targets    map[string]float64 `json:"targets"`
	Omega      string             `json:"omega"`
	Pinned     []string           `json:"pinned"`
	AutoAdjust bool               `json:"auto_adjust"`
}

func (h *SpreadHandler) PutSpread(w http.ResponseWriter, r *http.Request) {
	uid := middleware.FirebaseUIDFromContext(r.Context())
	user, err := h.userRepo.FindOrCreateByFirebaseUID(uid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to resolve user"})
		return
	}

	var req putSpreadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	// Validate weights sum to 1.0 (±0.01).
	sum := 0.0
	for _, wt := range req.Targets {
		if wt < 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "weights must be >= 0"})
			return
		}
		sum += wt
	}
	if math.Abs(sum-1.0) > 0.01 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "weights must sum to 1.0"})
		return
	}

	// Validate omega exists in targets.
	if _, ok := req.Targets[req.Omega]; !ok {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "omega must be a key in targets"})
		return
	}

	// Validate not all pinned when auto_adjust is on.
	pinnedSet := make(map[string]bool, len(req.Pinned))
	for _, p := range req.Pinned {
		pinnedSet[p] = true
	}
	if req.AutoAdjust {
		allPinned := true
		for k := range req.Targets {
			if !pinnedSet[k] {
				allPinned = false
				break
			}
		}
		if allPinned {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "at least one vertical must be unpinned when auto_adjust is enabled"})
			return
		}
	}

	// Build SpreadTargets.
	st := &model.SpreadTargets{
		Verticals:  make(map[string]model.SpreadVertical, len(req.Targets)),
		Omega:      req.Omega,
		AutoAdjust: req.AutoAdjust,
	}
	for k, wt := range req.Targets {
		st.Verticals[k] = model.SpreadVertical{Weight: wt, Pinned: pinnedSet[k]}
	}

	if err := h.spreadRepo.UpsertTargets(user.ID, st); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save spread targets"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *SpreadHandler) GetHistory(w http.ResponseWriter, r *http.Request) {
	uid := middleware.FirebaseUIDFromContext(r.Context())
	user, err := h.userRepo.FindOrCreateByFirebaseUID(uid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to resolve user"})
		return
	}

	days, err := h.spreadRepo.GetHistory(user.ID, 30)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load history"})
		return
	}
	if days == nil {
		days = []model.SpreadHistoryDay{}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"days": days})
}
