package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/shanegleeson/beepbopboop/backend/internal/database"
	"github.com/shanegleeson/beepbopboop/backend/internal/handler"
	"github.com/shanegleeson/beepbopboop/backend/internal/model"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

func newNewsSourceHandler(t *testing.T) *handler.NewsSourceHandler {
	t.Helper()
	db := database.OpenTestDB(t)
	repo := repository.NewNewsSourceRepo(db)
	return handler.NewNewsSourceHandler(repo)
}

func TestNewsSourceHandler_Create(t *testing.T) {
	h := newNewsSourceHandler(t)

	body := `{
		"name": "Dublin Live",
		"url": "https://www.dublinlive.ie",
		"area_label": "Dublin, Ireland",
		"latitude": 53.35,
		"longitude": -6.26,
		"radius_km": 30,
		"topics": ["local", "news"],
		"trust_score": 80,
		"fetch_method": "rss",
		"active": true
	}`

	req := httptest.NewRequest(http.MethodPost, "/news-sources", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Create(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 got %d: %s", w.Code, w.Body.String())
	}
}

func TestNewsSourceHandler_List(t *testing.T) {
	h := newNewsSourceHandler(t)

	// Create a source first.
	createBody := `{
		"name": "Dublin Live",
		"url": "https://www.dublinlive.ie/list-test",
		"area_label": "Dublin, Ireland",
		"latitude": 53.35,
		"longitude": -6.26,
		"radius_km": 30,
		"topics": ["local", "news"],
		"trust_score": 80,
		"fetch_method": "rss",
		"active": true
	}`

	createReq := httptest.NewRequest(http.MethodPost, "/news-sources", bytes.NewBufferString(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()
	h.Create(createW, createReq)

	if createW.Code != http.StatusCreated {
		t.Fatalf("setup: create expected 201 got %d: %s", createW.Code, createW.Body.String())
	}

	// Now list near Dublin.
	listReq := httptest.NewRequest(http.MethodGet, "/news-sources?lat=53.35&lon=-6.26&radius_km=50", nil)
	listW := httptest.NewRecorder()
	h.List(listW, listReq)

	if listW.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d: %s", listW.Code, listW.Body.String())
	}

	var sources []model.NewsSource
	if err := json.NewDecoder(listW.Body).Decode(&sources); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(sources) < 1 {
		t.Fatalf("expected at least 1 source, got %d", len(sources))
	}
}
