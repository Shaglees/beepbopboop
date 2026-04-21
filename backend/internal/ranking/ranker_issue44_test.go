package ranking

import (
	"testing"
	"time"
)

func randomVec(n int) []float32 {
	v := make([]float32, n)
	for i := range v {
		v[i] = float32(i%10) / 10.0
	}
	return v
}

func makeCandidates(n, dim int) [][]float32 {
	out := make([][]float32, n)
	for i := range out {
		out[i] = randomVec(dim)
	}
	return out
}

func TestScoreBatch_LatencyUnder20ms(t *testing.T) {
	r := NewStubRanker(128)
	userVec := randomVec(128)
	candidates := makeCandidates(250, 128)

	start := time.Now()
	scores, err := r.ScoreBatch(userVec, candidates)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("ScoreBatch: %v", err)
	}
	if elapsed > 20*time.Millisecond {
		t.Errorf("ScoreBatch too slow: %v (limit 20ms)", elapsed)
	}
	if len(scores) != 250 {
		t.Errorf("expected 250 scores, got %d", len(scores))
	}
}

func TestScoreBatch_OutputInUnitInterval(t *testing.T) {
	r := NewStubRanker(128)
	scores, err := r.ScoreBatch(randomVec(128), makeCandidates(10, 128))
	if err != nil {
		t.Fatal(err)
	}
	for i, s := range scores {
		if s < 0 || s > 1 {
			t.Errorf("score[%d] = %f out of [0,1]", i, s)
		}
	}
}

func TestNewRanker_MissingModelPathDoesNotPanic(t *testing.T) {
	r, err := NewRanker("")
	if err != nil {
		t.Fatalf("unexpected error with empty path: %v", err)
	}
	if r != nil {
		t.Errorf("expected nil Ranker when no model path configured")
	}
}

func TestNewRanker_InvalidPathReturnsError(t *testing.T) {
	_, err := NewRanker("/nonexistent/ranker.onnx")
	if err == nil {
		t.Error("expected error for missing model file")
	}
}

func TestScoreBatch_EmptyCandidateList(t *testing.T) {
	r := NewStubRanker(4)
	v, err := r.ScoreBatch([]float32{1, 0, 0, 0}, [][]float32{})
	if err != nil {
		t.Fatal(err)
	}
	if len(v) != 0 {
		t.Fatalf("expected empty scores, got len %d", len(v))
	}
}
