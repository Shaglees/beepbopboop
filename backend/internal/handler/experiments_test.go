package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/shanegleeson/beepbopboop/backend/internal/ab"
	"github.com/shanegleeson/beepbopboop/backend/internal/database"
	"github.com/shanegleeson/beepbopboop/backend/internal/handler"
	"github.com/shanegleeson/beepbopboop/backend/internal/middleware"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

func chiCtx(r *http.Request, key, val string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, val)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

// TestGetVariant_ReturnControlWhenPaused verifies that a paused experiment
// always returns "control" without writing a new assignment row.
func TestGetVariant_ReturnControlWhenPaused(t *testing.T) {
	db := database.OpenTestDB(t)
	userRepo := repository.NewUserRepo(db)
	expRepo := repository.NewExperimentRepo(db)
	assigner := ab.NewAssigner(db)
	h := handler.NewExperimentsHandler(assigner, userRepo, expRepo)

	// Insert a paused experiment directly.
	db.Exec(`INSERT INTO ab_experiments (name, treatment_pct, status, paused_at)
		VALUES ('paused-exp', 50, 'paused', NOW())`)

	req := httptest.NewRequest(http.MethodGet, "/experiments/paused-exp/variant", nil)
	req = req.WithContext(middleware.WithFirebaseUID(req.Context(), "firebase-paused-variant-user"))
	req = chiCtx(req, "name", "paused-exp")
	rec := httptest.NewRecorder()

	h.GetVariant(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]string
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["variant"] != "control" {
		t.Errorf("paused experiment: expected variant='control', got %q", resp["variant"])
	}
	if resp["experiment"] != "paused-exp" {
		t.Errorf("expected experiment='paused-exp', got %q", resp["experiment"])
	}
}

// TestGetVariant_RunningExperimentAssignsNormally confirms the happy path is
// unaffected by the paused-status gate.
func TestGetVariant_RunningExperimentAssignsNormally(t *testing.T) {
	db := database.OpenTestDB(t)
	userRepo := repository.NewUserRepo(db)
	expRepo := repository.NewExperimentRepo(db)
	assigner := ab.NewAssigner(db)
	h := handler.NewExperimentsHandler(assigner, userRepo, expRepo)

	db.Exec(`INSERT INTO ab_experiments (name, treatment_pct, status)
		VALUES ('running-exp', 100, 'running')`)

	req := httptest.NewRequest(http.MethodGet, "/experiments/running-exp/variant", nil)
	req = req.WithContext(middleware.WithFirebaseUID(req.Context(), "firebase-running-variant-user"))
	req = chiCtx(req, "name", "running-exp")
	rec := httptest.NewRecorder()

	h.GetVariant(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]string
	json.NewDecoder(rec.Body).Decode(&resp)
	// 100% treatment, so any user must be in treatment.
	if resp["variant"] != "treatment" {
		t.Errorf("running experiment (100%% treatment): expected 'treatment', got %q", resp["variant"])
	}
}
