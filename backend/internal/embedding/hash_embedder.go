package embedding

import (
	"context"
	"hash/fnv"
	"strings"
)

// HashEmbedder is a deterministic local fallback embedder.
// It is not semantic SOTA, but guarantees availability when external APIs fail.
type HashEmbedder struct {
	dim          int
	modelVersion string
}

func NewHashEmbedder(dim int, modelVersion string) *HashEmbedder {
	if dim <= 0 {
		dim = 1536
	}
	if strings.TrimSpace(modelVersion) == "" {
		modelVersion = "hash-v1"
	}
	return &HashEmbedder{dim: dim, modelVersion: modelVersion}
}

func (h *HashEmbedder) ModelVersion() string { return h.modelVersion }

func (h *HashEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	return h.EmbedInput(ctx, EmbeddingInput{Text: text})
}

func (h *HashEmbedder) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	inputs := make([]EmbeddingInput, len(texts))
	for i := range texts {
		inputs[i] = EmbeddingInput{Text: texts[i]}
	}
	return h.EmbedBatchInputs(ctx, inputs)
}

func (h *HashEmbedder) EmbedInputWithVersion(ctx context.Context, input EmbeddingInput) ([]float32, string, error) {
	vec, err := h.EmbedInput(ctx, input)
	if err != nil {
		return nil, "", err
	}
	return vec, h.ModelVersion(), nil
}

func (h *HashEmbedder) EmbedBatchInputsWithVersion(ctx context.Context, inputs []EmbeddingInput) ([][]float32, string, error) {
	vecs, err := h.EmbedBatchInputs(ctx, inputs)
	if err != nil {
		return nil, "", err
	}
	return vecs, h.ModelVersion(), nil
}

func (h *HashEmbedder) EmbedInput(ctx context.Context, input EmbeddingInput) ([]float32, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	v := make([]float32, h.dim)
	text := strings.ToLower(strings.TrimSpace(input.Text))
	if text == "" {
		return v, nil
	}

	tokens := strings.Fields(text)
	for i, tok := range tokens {
		hasher := fnv.New64a()
		_, _ = hasher.Write([]byte(tok))
		idx := int(hasher.Sum64() % uint64(h.dim))
		v[idx] += 1.0 / float32(1+i)
	}
	for i, u := range input.ImageURLs {
		hasher := fnv.New64a()
		_, _ = hasher.Write([]byte("img:" + u))
		idx := int(hasher.Sum64() % uint64(h.dim))
		v[idx] += 0.25 / float32(1+i)
	}

	l2Normalize(v)
	return v, nil
}

func (h *HashEmbedder) EmbedBatchInputs(ctx context.Context, inputs []EmbeddingInput) ([][]float32, error) {
	out := make([][]float32, len(inputs))
	for i := range inputs {
		vec, err := h.EmbedInput(ctx, inputs[i])
		if err != nil {
			return nil, err
		}
		out[i] = vec
	}
	return out, nil
}

func l2Normalize(v []float32) {
	var sum float64
	for _, x := range v {
		sum += float64(x * x)
	}
	if sum == 0 {
		return
	}
	inv := float32(1.0 / sqrt(sum))
	for i := range v {
		v[i] *= inv
	}
}

func sqrt(x float64) float64 {
	if x == 0 {
		return 0
	}
	z := x
	for i := 0; i < 8; i++ {
		z -= (z*z - x) / (2 * z)
	}
	return z
}
