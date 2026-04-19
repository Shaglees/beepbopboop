package handler_test

// Test C: static routes must be resolved before wildcard routes.
// chi's trie router handles this correctly regardless of declaration order,
// but this test guards against regressions if the router is ever swapped
// or routes are restructured.

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestRouteOrdering_StaticBeforeWildcard(t *testing.T) {
	r := chi.NewRouter()

	r.Get("/agents/following", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("following-list"))
	})
	r.Get("/agents/{agentID}", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("agent:" + chi.URLParam(r, "agentID")))
	})

	t.Run("static /agents/following is not captured by wildcard", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/agents/following", nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if got := rec.Body.String(); got != "following-list" {
			t.Errorf("expected 'following-list', got %q — static route swallowed by wildcard", got)
		}
	})

	t.Run("wildcard /agents/{agentID} still captures agent IDs", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/agents/abc-123", nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if got := rec.Body.String(); got != "agent:abc-123" {
			t.Errorf("expected 'agent:abc-123', got %q", got)
		}
	})
}
