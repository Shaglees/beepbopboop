package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/shanegleeson/beepbopboop/backend/internal/database"
	"github.com/shanegleeson/beepbopboop/backend/internal/handler"
	"github.com/shanegleeson/beepbopboop/backend/internal/middleware"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

func TestGetWeightsFirebase_NoWeights(t *testing.T) {
	db := database.OpenTestDB(t)
	userRepo := repository.NewUserRepo(db)
	agentRepo := repository.NewAgentRepo(db)
	weightsRepo := repository.NewWeightsRepo(db)
	h := handler.NewWeightsHandler(agentRepo, userRepo, weightsRepo)

	req := httptest.NewRequest("GET", "/user/weights", nil)
	req = req.WithContext(middleware.WithFirebaseUID(req.Context(), "firebase-new-user"))
	rec := httptest.NewRecorder()

	h.GetWeightsFirebase(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["weights"] != nil {
		t.Errorf("expected nil weights for new user, got %v", resp["weights"])
	}
}

func TestUpdateWeightsFirebase_ThenGet(t *testing.T) {
	db := database.OpenTestDB(t)
	userRepo := repository.NewUserRepo(db)
	agentRepo := repository.NewAgentRepo(db)
	weightsRepo := repository.NewWeightsRepo(db)
	h := handler.NewWeightsHandler(agentRepo, userRepo, weightsRepo)

	body := `{"freshness_bias":0.65,"geo_bias":0.8}`

	putReq := httptest.NewRequest("PUT", "/user/weights", bytes.NewBufferString(body))
	putReq = putReq.WithContext(middleware.WithFirebaseUID(putReq.Context(), "firebase-tuner"))
	putRec := httptest.NewRecorder()

	h.UpdateWeightsFirebase(putRec, putReq)

	if putRec.Code != http.StatusOK {
		t.Fatalf("PUT expected 200, got %d: %s", putRec.Code, putRec.Body.String())
	}

	// Now GET and verify the weights persisted
	getReq := httptest.NewRequest("GET", "/user/weights", nil)
	getReq = getReq.WithContext(middleware.WithFirebaseUID(getReq.Context(), "firebase-tuner"))
	getRec := httptest.NewRecorder()

	h.GetWeightsFirebase(getRec, getReq)

	if getRec.Code != http.StatusOK {
		t.Fatalf("GET expected 200, got %d: %s", getRec.Code, getRec.Body.String())
	}

	var resp map[string]any
	json.NewDecoder(getRec.Body).Decode(&resp)
	if resp["weights"] == nil {
		t.Error("expected weights to be set after PUT")
	}

	weightsMap, ok := resp["weights"].(map[string]any)
	if !ok {
		t.Fatalf("expected weights to be an object, got %T", resp["weights"])
	}
	if weightsMap["geo_bias"] != 0.8 {
		t.Errorf("expected geo_bias 0.8, got %v", weightsMap["geo_bias"])
	}
	if weightsMap["freshness_bias"] != 0.65 {
		t.Errorf("expected freshness_bias 0.65, got %v", weightsMap["freshness_bias"])
	}
}

func TestUpdateWeightsFirebase_InvalidJSON(t *testing.T) {
	db := database.OpenTestDB(t)
	userRepo := repository.NewUserRepo(db)
	agentRepo := repository.NewAgentRepo(db)
	weightsRepo := repository.NewWeightsRepo(db)
	h := handler.NewWeightsHandler(agentRepo, userRepo, weightsRepo)

	putReq := httptest.NewRequest("PUT", "/user/weights", bytes.NewBufferString("not-json"))
	putReq = putReq.WithContext(middleware.WithFirebaseUID(putReq.Context(), "firebase-bad"))
	putRec := httptest.NewRecorder()

	h.UpdateWeightsFirebase(putRec, putReq)

	if putRec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", putRec.Code)
	}
}
