package sports_test

import (
	"testing"

	"github.com/shanegleeson/beepbopboop/backend/internal/sports"
)

// TestHintForState verifies that ESPN game states map to the correct iOS display hint.
// "pre" → "matchup" (pre-game card with countdown/venue/broadcast)
// "in"  → "scoreboard" (live scores card)
// "post" → "scoreboard" (final scores card)
// ""    → "scoreboard" (safe default for unknown/empty state)
func TestHintForState(t *testing.T) {
	tests := []struct {
		state string
		want  string
	}{
		{state: "pre", want: "matchup"},
		{state: "in", want: "scoreboard"},
		{state: "post", want: "scoreboard"},
		{state: "", want: "scoreboard"},
		{state: "unknown-future-state", want: "scoreboard"},
	}

	for _, tc := range tests {
		got := sports.HintForState(tc.state)
		if got != tc.want {
			t.Errorf("HintForState(%q) = %q, want %q", tc.state, got, tc.want)
		}
	}
}
