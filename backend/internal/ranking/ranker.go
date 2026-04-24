package ranking

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"os"
	"strings"
	"sync/atomic"
	"time"
)

// checkpoint is the JSON serialisation of a trained two-tower model.
type checkpoint struct {
	InputDim  int `json:"input_dim"`
	HiddenDim int `json:"hidden_dim"`
	ReprDim   int `json:"repr_dim"`

	UserW1 [][]float32 `json:"user_weights_1"`
	UserB1 []float32   `json:"user_bias_1"`
	UserW2 [][]float32 `json:"user_weights_2"`
	UserB2 []float32   `json:"user_bias_2"`

	PostW1 [][]float32 `json:"post_weights_1"`
	PostB1 []float32   `json:"post_bias_1"`
	PostW2 [][]float32 `json:"post_weights_2"`
	PostB2 []float32   `json:"post_bias_2"`
}

// rankerWeights holds all immutable weight data for one model version.
// It is swapped atomically so in-flight Score calls always see a consistent set.
type rankerWeights struct {
	userW1, userW2 [][]float32
	userB1, userB2 []float32
	postW1, postW2 [][]float32
	postB1, postB2 []float32
	inputDim       int
	hiddenDim      int
	reprDim        int
}

// Ranker scores (user, post) pairs using a two-tower model.
// Weights are held behind an atomic pointer so Reload never blocks in-flight calls.
type Ranker struct {
	weights atomic.Pointer[rankerWeights]
}

// NewRanker loads a checkpoint from the JSON file at path and returns a ready Ranker.
// Returns (nil, nil) when path is empty so callers can disable ML without error.
func NewRanker(path string) (*Ranker, error) {
	if strings.TrimSpace(path) == "" {
		return nil, nil
	}
	w, err := loadWeights(path)
	if err != nil {
		return nil, err
	}
	r := &Ranker{}
	r.weights.Store(w)
	return r, nil
}

// Reload atomically replaces the model weights with the checkpoint at path.
// Returns an error and leaves the current weights intact if loading fails.
func (r *Ranker) Reload(path string) error {
	w, err := loadWeights(path)
	if err != nil {
		return err
	}
	r.weights.Store(w)
	return nil
}

// StartWatcher polls path for mtime changes at interval and calls Reload on change.
func (r *Ranker) StartWatcher(ctx context.Context, path string, interval time.Duration) {
	go func() {
		var lastMod time.Time
		if fi, err := os.Stat(path); err == nil {
			lastMod = fi.ModTime()
		}
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				fi, err := os.Stat(path)
				if err != nil {
					continue
				}
				if fi.ModTime().After(lastMod) {
					lastMod = fi.ModTime()
					if err := r.Reload(path); err != nil {
						slog.Warn("ranker: hot-reload failed", "path", path, "error", err)
					} else {
						slog.Info("ranker: hot-reloaded checkpoint", "path", path)
					}
				}
			}
		}
	}()
}

// InputDim returns the expected dimension of user and post input vectors.
func (r *Ranker) InputDim() int { return r.weights.Load().inputDim }

// ReprDim returns the dimension of the L2-normalised representation vectors.
func (r *Ranker) ReprDim() int { return r.weights.Load().reprDim }

// Score returns a relevance score in [0, 1] for a single (user, post) pair.
func (r *Ranker) Score(userVec, postVec []float32) (float32, error) {
	w := r.weights.Load()
	if containsNaN(userVec) || containsNaN(postVec) {
		slog.Warn("ranker: NaN in input vector, returning 0.0")
		return 0.0, nil
	}
	if err := checkDims(w, len(userVec), len(postVec)); err != nil {
		return 0, err
	}
	uRepr := towerForward(w.userW1, w.userB1, w.userW2, w.userB2, userVec)
	pRepr := towerForward(w.postW1, w.postB1, w.postW2, w.postB2, postVec)
	return clamp01((dot(uRepr, pRepr) + 1) / 2), nil
}

// ScoreBatch scores one user against multiple post candidates.
func (r *Ranker) ScoreBatch(userVec []float32, postVecs [][]float32) ([]float32, error) {
	w := r.weights.Load()
	scores := make([]float32, len(postVecs))
	if len(postVecs) == 0 {
		return scores, nil
	}
	if containsNaN(userVec) {
		slog.Warn("ranker: NaN in user vector, returning all-zero batch scores")
		return scores, nil
	}
	if len(userVec) != w.inputDim {
		return nil, fmt.Errorf("ranker: user vec dim %d != input_dim %d", len(userVec), w.inputDim)
	}
	uRepr := towerForward(w.userW1, w.userB1, w.userW2, w.userB2, userVec)
	for i, pv := range postVecs {
		if containsNaN(pv) || len(pv) != w.inputDim {
			scores[i] = 0.0
			continue
		}
		pRepr := towerForward(w.postW1, w.postB1, w.postW2, w.postB2, pv)
		scores[i] = clamp01((dot(uRepr, pRepr) + 1) / 2)
	}
	return scores, nil
}

// ProjectUser returns the L2-normalised representation for a user vector.
func (r *Ranker) ProjectUser(userVec []float32) ([]float32, error) {
	w := r.weights.Load()
	if containsNaN(userVec) {
		return make([]float32, w.reprDim), nil
	}
	if len(userVec) != w.inputDim {
		return nil, fmt.Errorf("ranker: user vec dim %d != input_dim %d", len(userVec), w.inputDim)
	}
	return towerForward(w.userW1, w.userB1, w.userW2, w.userB2, userVec), nil
}

// ProjectPost returns the L2-normalised representation for a post vector.
func (r *Ranker) ProjectPost(postVec []float32) ([]float32, error) {
	w := r.weights.Load()
	if containsNaN(postVec) {
		return make([]float32, w.reprDim), nil
	}
	if len(postVec) != w.inputDim {
		return nil, fmt.Errorf("ranker: post vec dim %d != input_dim %d", len(postVec), w.inputDim)
	}
	return towerForward(w.postW1, w.postB1, w.postW2, w.postB2, postVec), nil
}

// ScoreBatchFromReprs scores a pre-projected user repr against pre-projected post reprs.
func (r *Ranker) ScoreBatchFromReprs(userRepr []float32, postReprs [][]float32) ([]float32, error) {
	w := r.weights.Load()
	if len(userRepr) != w.reprDim {
		return nil, fmt.Errorf("ranker: user repr dim %d != repr_dim %d", len(userRepr), w.reprDim)
	}
	scores := make([]float32, len(postReprs))
	for i, pr := range postReprs {
		if len(pr) != w.reprDim {
			return nil, fmt.Errorf("ranker: post repr[%d] dim %d != repr_dim %d", i, len(pr), w.reprDim)
		}
		scores[i] = clamp01((dot(userRepr, pr) + 1) / 2)
	}
	return scores, nil
}

// --- internal helpers ---

func loadWeights(path string) (*rankerWeights, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("ranker: read %s: %w", path, err)
	}
	var cp checkpoint
	if err := json.Unmarshal(data, &cp); err != nil {
		return nil, fmt.Errorf("ranker: unmarshal checkpoint: %w", err)
	}
	if err := validateCheckpoint(&cp); err != nil {
		return nil, fmt.Errorf("ranker: invalid checkpoint: %w", err)
	}
	return &rankerWeights{
		userW1:    cp.UserW1,
		userB1:    cp.UserB1,
		userW2:    cp.UserW2,
		userB2:    cp.UserB2,
		postW1:    cp.PostW1,
		postB1:    cp.PostB1,
		postW2:    cp.PostW2,
		postB2:    cp.PostB2,
		inputDim:  cp.InputDim,
		hiddenDim: cp.HiddenDim,
		reprDim:   cp.ReprDim,
	}, nil
}

func towerForward(w1 [][]float32, b1 []float32, w2 [][]float32, b2 []float32, x []float32) []float32 {
	hidden := addBias(project(w1, x), b1)
	activated := relu(hidden)
	out := addBias(project(w2, activated), b2)
	return l2norm(out)
}

func checkDims(w *rankerWeights, userLen, postLen int) error {
	if userLen != w.inputDim || postLen != w.inputDim {
		return fmt.Errorf("ranker: expected input_dim=%d, got user=%d post=%d", w.inputDim, userLen, postLen)
	}
	return nil
}

func validateCheckpoint(cp *checkpoint) error {
	if cp.InputDim <= 0 || cp.HiddenDim <= 0 || cp.ReprDim <= 0 {
		return fmt.Errorf("input_dim, hidden_dim and repr_dim must be positive")
	}
	if err := validateMatrix("user_weights_1", cp.UserW1, cp.HiddenDim, cp.InputDim); err != nil {
		return err
	}
	if err := validateBias("user_bias_1", cp.UserB1, cp.HiddenDim); err != nil {
		return err
	}
	if err := validateMatrix("user_weights_2", cp.UserW2, cp.ReprDim, cp.HiddenDim); err != nil {
		return err
	}
	if err := validateBias("user_bias_2", cp.UserB2, cp.ReprDim); err != nil {
		return err
	}
	if err := validateMatrix("post_weights_1", cp.PostW1, cp.HiddenDim, cp.InputDim); err != nil {
		return err
	}
	if err := validateBias("post_bias_1", cp.PostB1, cp.HiddenDim); err != nil {
		return err
	}
	if err := validateMatrix("post_weights_2", cp.PostW2, cp.ReprDim, cp.HiddenDim); err != nil {
		return err
	}
	if err := validateBias("post_bias_2", cp.PostB2, cp.ReprDim); err != nil {
		return err
	}
	return nil
}

func validateMatrix(name string, w [][]float32, wantRows, wantCols int) error {
	if len(w) != wantRows {
		return fmt.Errorf("%s rows %d != %d", name, len(w), wantRows)
	}
	for i, row := range w {
		if len(row) != wantCols {
			return fmt.Errorf("%s[%d] has %d cols, want %d", name, i, len(row), wantCols)
		}
	}
	return nil
}

func validateBias(name string, b []float32, wantLen int) error {
	if len(b) != wantLen {
		return fmt.Errorf("%s len %d != %d", name, len(b), wantLen)
	}
	return nil
}

// project computes w @ v. If v is shorter than a row, only the covered columns
// contribute (defense-in-depth; validateCheckpoint prevents this in practice).
func project(w [][]float32, v []float32) []float32 {
	out := make([]float32, len(w))
	for i, row := range w {
		var s float32
		for j, wij := range row {
			if j >= len(v) {
				break
			}
			s += wij * v[j]
		}
		out[i] = s
	}
	return out
}

func addBias(v []float32, b []float32) []float32 {
	out := make([]float32, len(v))
	for i, x := range v {
		if i < len(b) {
			out[i] = x + b[i]
		} else {
			out[i] = x
		}
	}
	return out
}

func relu(v []float32) []float32 {
	out := make([]float32, len(v))
	for i, x := range v {
		if x > 0 {
			out[i] = x
		}
	}
	return out
}

func l2norm(v []float32) []float32 {
	var mag float64
	for _, x := range v {
		mag += float64(x) * float64(x)
	}
	mag = math.Sqrt(mag)
	out := make([]float32, len(v))
	if mag < 1e-10 {
		return out
	}
	for i, x := range v {
		out[i] = float32(float64(x) / mag)
	}
	return out
}

func dot(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}
	var s float32
	for i := range a {
		s += a[i] * b[i]
	}
	return s
}

func clamp01(x float32) float32 {
	if x < 0 {
		return 0
	}
	if x > 1 {
		return 1
	}
	return x
}

func containsNaN(v []float32) bool {
	for _, x := range v {
		if math.IsNaN(float64(x)) {
			return true
		}
	}
	return false
}
