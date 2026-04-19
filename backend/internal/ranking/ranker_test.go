package ranking_test

import (
	"math"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/shanegleeson/beepbopboop/backend/internal/ranking"
)

// randomUnitVec creates a random float32 slice of the given length.
func randomUnitVec(dim int) []float32 {
	v := make([]float32, dim)
	var mag float64
	for i := range v {
		f := rand.Float32()*2 - 1
		v[i] = f
		mag += float64(f) * float64(f)
	}
	mag = math.Sqrt(mag)
	if mag > 1e-10 {
		for i := range v {
			v[i] /= float32(mag)
		}
	}
	return v
}

// --- load ---

// TestRanker_Load_Valid verifies NewRanker loads a well-formed checkpoint
// without error and returns a non-nil Ranker.
func TestRanker_Load_Valid(t *testing.T) {
	r, err := ranking.NewRanker("testdata/ranker.json")
	if err != nil {
		t.Fatalf("NewRanker: %v", err)
	}
	if r == nil {
		t.Fatal("expected non-nil Ranker")
	}
}

// TestRanker_LoadMissingFile_ReturnsError verifies NewRanker returns a non-nil
// error when the checkpoint file does not exist, and does not panic.
func TestRanker_LoadMissingFile_ReturnsError(t *testing.T) {
	_, err := ranking.NewRanker("/nonexistent/path.json")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

// TestRanker_LoadCorrupt_ReturnsError verifies malformed JSON returns an error.
func TestRanker_LoadCorrupt_ReturnsError(t *testing.T) {
	_, err := ranking.NewRanker("testdata/corrupt.json")
	if err == nil {
		t.Fatal("expected error for corrupt checkpoint, got nil")
	}
}

// --- Score ---

// TestRanker_Score_ReturnsInRange verifies Score returns a value in [0, 1].
func TestRanker_Score_ReturnsInRange(t *testing.T) {
	r, _ := ranking.NewRanker("testdata/ranker.json")
	dim := r.InputDim()
	for i := 0; i < 20; i++ {
		score, err := r.Score(randomUnitVec(dim), randomUnitVec(dim))
		if err != nil {
			t.Fatalf("Score: %v", err)
		}
		if score < 0 || score > 1 {
			t.Errorf("score %f not in [0,1]", score)
		}
	}
}

// TestRanker_Score_NaN_ReturnsZero verifies NaN inputs produce score=0.0
// without propagating NaN into the feed.
func TestRanker_Score_NaN_ReturnsZero(t *testing.T) {
	r, _ := ranking.NewRanker("testdata/ranker.json")
	dim := r.InputDim()

	nanVec := randomUnitVec(dim)
	nanVec[0] = float32(math.NaN())

	score, err := r.Score(nanVec, randomUnitVec(dim))
	if err != nil {
		t.Fatalf("Score with NaN user vec: %v", err)
	}
	if score != 0.0 {
		t.Errorf("expected 0.0 for NaN user input, got %f", score)
	}

	score, err = r.Score(randomUnitVec(dim), nanVec)
	if err != nil {
		t.Fatalf("Score with NaN post vec: %v", err)
	}
	if score != 0.0 {
		t.Errorf("expected 0.0 for NaN post input, got %f", score)
	}
}

// TestRanker_Score_WrongDim_ReturnsError verifies Score returns an error when
// input vectors have the wrong dimension.
func TestRanker_Score_WrongDim_ReturnsError(t *testing.T) {
	r, _ := ranking.NewRanker("testdata/ranker.json")
	_, err := r.Score([]float32{1, 2}, []float32{3, 4, 5})
	if err == nil {
		t.Fatal("expected error for wrong input dimension")
	}
}

// TestRanker_Score_IdenticalVecs_ScoresHigh verifies that scoring a user
// against an identical post vector produces the maximum possible score (1.0)
// when using identity-like projection weights.
func TestRanker_Score_IdenticalVecs_ScoresHigh(t *testing.T) {
	r, _ := ranking.NewRanker("testdata/ranker.json")
	dim := r.InputDim()
	v := randomUnitVec(dim)
	score, err := r.Score(v, v)
	if err != nil {
		t.Fatalf("Score: %v", err)
	}
	// With identity weights, identical vectors project to the same repr → dot=1 → score=1.
	if score < 0.9 {
		t.Errorf("identical vecs scored %f, expected >= 0.9 with identity weights", score)
	}
}

// --- ScoreBatch ---

// TestRanker_ScoreBatch_CountMatchesInput verifies ScoreBatch returns exactly
// one score per post candidate.
func TestRanker_ScoreBatch_CountMatchesInput(t *testing.T) {
	r, _ := ranking.NewRanker("testdata/ranker.json")
	dim := r.InputDim()
	postVecs := make([][]float32, 50)
	for i := range postVecs {
		postVecs[i] = randomUnitVec(dim)
	}
	scores, err := r.ScoreBatch(randomUnitVec(dim), postVecs)
	if err != nil {
		t.Fatalf("ScoreBatch: %v", err)
	}
	if len(scores) != 50 {
		t.Errorf("expected 50 scores, got %d", len(scores))
	}
}

// TestRanker_ScoreBatch_AllScoresInRange verifies every score in a batch is
// within [0, 1].
func TestRanker_ScoreBatch_AllScoresInRange(t *testing.T) {
	r, _ := ranking.NewRanker("testdata/ranker.json")
	dim := r.InputDim()
	postVecs := make([][]float32, 32)
	for i := range postVecs {
		postVecs[i] = randomUnitVec(dim)
	}
	scores, err := r.ScoreBatch(randomUnitVec(dim), postVecs)
	if err != nil {
		t.Fatalf("ScoreBatch: %v", err)
	}
	for i, s := range scores {
		if s < 0 || s > 1 {
			t.Errorf("scores[%d]=%f not in [0,1]", i, s)
		}
	}
}

// TestRanker_ScoreBatch_EmptyInput_ReturnsEmpty verifies ScoreBatch with no
// candidates returns an empty (not nil) slice without error.
func TestRanker_ScoreBatch_EmptyInput_ReturnsEmpty(t *testing.T) {
	r, _ := ranking.NewRanker("testdata/ranker.json")
	scores, err := r.ScoreBatch(randomUnitVec(r.InputDim()), nil)
	if err != nil {
		t.Fatalf("ScoreBatch(nil): %v", err)
	}
	if scores == nil {
		t.Error("expected non-nil empty slice, got nil")
	}
	if len(scores) != 0 {
		t.Errorf("expected 0 scores for nil input, got %d", len(scores))
	}
}

// TestRanker_ScoreBatch_NaNUserVec_AllZero verifies ScoreBatch returns all-zero
// scores when the user vector contains NaN.
func TestRanker_ScoreBatch_NaNUserVec_AllZero(t *testing.T) {
	r, _ := ranking.NewRanker("testdata/ranker.json")
	dim := r.InputDim()
	nanUser := randomUnitVec(dim)
	nanUser[0] = float32(math.NaN())
	postVecs := make([][]float32, 5)
	for i := range postVecs {
		postVecs[i] = randomUnitVec(dim)
	}
	scores, err := r.ScoreBatch(nanUser, postVecs)
	if err != nil {
		t.Fatalf("ScoreBatch with NaN user: %v", err)
	}
	for i, s := range scores {
		if s != 0.0 {
			t.Errorf("scores[%d]=%f, expected 0.0 for NaN user vec", i, s)
		}
	}
}

// TestRanker_ScoreBatch_Latency verifies 250 candidates are scored in < 20ms.
// NOTE: uses the test checkpoint (small dims); production dims (1536) should
// be benchmarked separately with pre-projected post reprs.
func TestRanker_ScoreBatch_Latency(t *testing.T) {
	r, _ := ranking.NewRanker("testdata/ranker.json")
	dim := r.InputDim()
	postVecs := make([][]float32, 250)
	for i := range postVecs {
		postVecs[i] = randomUnitVec(dim)
	}
	userVec := randomUnitVec(dim)

	start := time.Now()
	_, err := r.ScoreBatch(userVec, postVecs)
	if err != nil {
		t.Fatalf("ScoreBatch: %v", err)
	}
	if elapsed := time.Since(start); elapsed > 20*time.Millisecond {
		t.Errorf("ScoreBatch(250) took %v, want < 20ms", elapsed)
	}
}
