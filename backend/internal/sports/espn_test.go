package sports_test

// Test G: when every ESPN league endpoint fails, FetchAll must return an error
// (not a nil slice or a panic).  Guards against regressions that silently swallow
// all league errors and return empty results as if the fetch succeeded.

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/shanegleeson/beepbopboop/backend/internal/sports"
)

func TestESPNFetchAll_ReturnsErrorOnAllFailures(t *testing.T) {
	// Mock server that returns 500 for every request, simulating total ESPN outage.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	svc := sports.NewServiceWithBaseURL(srv.URL, srv.Client())
	games, err := svc.FetchAll()

	if err == nil {
		t.Errorf("expected error when all ESPN leagues fail, got nil (games=%v)", games)
	}
	if len(games) != 0 {
		t.Errorf("expected no games when all leagues fail, got %d", len(games))
	}
}

func TestESPNFetchAll_PartialFailureReturnsGames(t *testing.T) {
	// One league returns valid JSON, others return 500.
	// FetchAll must return games from the successful league (no error).
	const validNBAScoreboard = `{"events":[]}`

	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			// First league (nhl) returns valid empty scoreboard.
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(validNBAScoreboard))
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer srv.Close()

	svc := sports.NewServiceWithBaseURL(srv.URL, srv.Client())
	_, err := svc.FetchAll()

	if err != nil {
		t.Errorf("expected no error when at least one league succeeds, got: %v", err)
	}
}
