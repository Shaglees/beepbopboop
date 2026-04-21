package handler_test

// Tests for GET /posts/hints.
//
// The hints endpoint is the discoverability contract between backend
// handlers/validators and every agent/skill that publishes posts. Skills call
// it at Step 0 to learn:
//
//   - the full catalog of display_hints the backend accepts
//   - for each hint: whether external_url carries structured JSON, what the
//     required JSON shape is, and one canonical example that is guaranteed
//     to pass `validatePost`
//   - the display_hint/post_type/visibility enumerations, so skills never
//     guess at string values
//
// The critical invariant these tests enforce: every example returned by the
// hints endpoint must lint-clean through the same `validatePost` path a real
// POST /posts would hit. If a validator is tightened and an example in the
// catalog stops passing, this test fails loudly — keeping the public contract
// in sync with server-side behavior by construction.

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/shanegleeson/beepbopboop/backend/internal/database"
	"github.com/shanegleeson/beepbopboop/backend/internal/handler"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

// hintsResponse mirrors the expected JSON shape. Kept local so the test
// describes the intended contract without depending on the handler's
// internal types.
type hintsResponse struct {
	Version      int                    `json:"version"`
	DisplayHints []hintEntry            `json:"display_hints"`
	Enums        map[string][]string    `json:"enums"`
	Endpoints    map[string]endpointDoc `json:"endpoints"`
}

type hintEntry struct {
	Hint           string          `json:"hint"`
	Description    string          `json:"description"`
	PostType       string          `json:"post_type"`
	StructuredJSON bool            `json:"structured_json"`
	RequiredFields []string        `json:"required_fields"`
	Example        json.RawMessage `json:"example"`
}

type endpointDoc struct {
	Method      string `json:"method"`
	Path        string `json:"path"`
	Description string `json:"description"`
}

func newHintsHandler(t *testing.T) *handler.PostHandler {
	t.Helper()
	db := database.OpenTestDB(t)
	return handler.NewPostHandler(repository.NewAgentRepo(db), repository.NewPostRepo(db))
}

func fetchHints(t *testing.T, h *handler.PostHandler) hintsResponse {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/posts/hints", nil)
	rec := httptest.NewRecorder()
	h.GetPostHints(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /posts/hints status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	var hr hintsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &hr); err != nil {
		t.Fatalf("decode hints body: %v; body=%s", err, rec.Body.String())
	}
	return hr
}

// TestHints_CatalogMatchesValidDisplayHints guarantees the hints catalog
// covers every hint the server actually accepts. If someone adds a hint to
// ValidDisplayHints without updating the catalog, skills would silently miss
// it — this test prevents that.
func TestHints_CatalogMatchesValidDisplayHints(t *testing.T) {
	h := newHintsHandler(t)
	hr := fetchHints(t, h)

	got := map[string]bool{}
	for _, e := range hr.DisplayHints {
		got[e.Hint] = true
	}

	for hint := range handler.ValidDisplayHints {
		if !got[hint] {
			t.Errorf("hints catalog missing entry for display_hint %q (registered in ValidDisplayHints)", hint)
		}
	}
	for hint := range got {
		if !handler.ValidDisplayHints[hint] {
			t.Errorf("hints catalog returned display_hint %q that is not in ValidDisplayHints", hint)
		}
	}
}

// TestHints_ExamplesLintClean is the load-bearing correctness check.
// Every example in the catalog is submitted to POST /posts/lint and must
// come back valid. This prevents documentation drift: if a validator is
// tightened and an example stops passing, the test fails and forces the
// catalog to be updated in the same PR.
func TestHints_ExamplesLintClean(t *testing.T) {
	h := newHintsHandler(t)
	hr := fetchHints(t, h)

	for _, entry := range hr.DisplayHints {
		entry := entry
		t.Run(entry.Hint, func(t *testing.T) {
			if len(entry.Example) == 0 {
				t.Fatalf("hint %q has empty example", entry.Hint)
			}

			req := httptest.NewRequest(http.MethodPost, "/posts/lint", bytes.NewReader(entry.Example))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			h.LintPost(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("lint status = %d, want 200; body=%s", rec.Code, rec.Body.String())
			}

			var result struct {
				Valid  bool `json:"valid"`
				Errors []struct {
					Field   string `json:"field"`
					Code    string `json:"code"`
					Message string `json:"message"`
				} `json:"errors"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
				t.Fatalf("decode lint body: %v; body=%s", err, rec.Body.String())
			}
			if !result.Valid {
				t.Fatalf("example for hint %q failed lint: %+v", entry.Hint, result.Errors)
			}
		})
	}
}

// TestHints_StructuredHintsMarkedAsSuch keeps the structured_json flag in
// the catalog honest. Every hint that requires a JSON payload in
// external_url must be flagged true; otherwise skills will ship a plain URL
// and create invalid posts.
func TestHints_StructuredHintsMarkedAsSuch(t *testing.T) {
	h := newHintsHandler(t)
	hr := fetchHints(t, h)

	structured := map[string]bool{
		"weather": true, "scoreboard": true, "matchup": true, "standings": true,
		"entertainment": true, "album": true, "concert": true,
		"game_release": true, "game_review": true,
		"restaurant": true, "destination": true, "pet_spotlight": true,
		"fitness": true, "science": true, "movie": true, "show": true,
		"player_spotlight": true, "box_score": true,
		"feedback": true, "creator_spotlight": true, "video_embed": true,
	}
	for _, e := range hr.DisplayHints {
		want := structured[e.Hint]
		if e.StructuredJSON != want {
			t.Errorf("hint %q structured_json = %v, want %v", e.Hint, e.StructuredJSON, want)
		}
	}
}

// TestHints_ExposesEnums gives skills a single source of truth for
// post_type / visibility / image_role enumerations so they never hard-code
// values that drift from ValidPostTypes etc.
func TestHints_ExposesEnums(t *testing.T) {
	h := newHintsHandler(t)
	hr := fetchHints(t, h)

	for _, key := range []string{"post_type", "visibility", "image_role"} {
		if len(hr.Enums[key]) == 0 {
			t.Errorf("enums[%q] is empty", key)
		}
	}
}

// TestHints_DocumentsKeyEndpoints gives skills the full picture of what's
// callable without needing to read server route tables. This is the
// "capabilities" half of the context-bootstrap contract.
func TestHints_DocumentsKeyEndpoints(t *testing.T) {
	h := newHintsHandler(t)
	hr := fetchHints(t, h)

	wantEndpoints := []string{
		"create_post", "lint_post", "list_posts", "post_stats",
		"events_summary", "reactions_summary",
	}
	for _, key := range wantEndpoints {
		if _, ok := hr.Endpoints[key]; !ok {
			t.Errorf("endpoints[%q] missing from hints response", key)
		}
	}
}
