package embedding

import (
	"context"
	"fmt"
	"log/slog"
)

// ResilientEmbedder wraps a primary and fallback provider.
// On primary failure, it logs and retries using fallback.
type ResilientEmbedder struct {
	primary  Embedder
	fallback Embedder
}

func NewResilientEmbedder(primary, fallback Embedder) *ResilientEmbedder {
	return &ResilientEmbedder{primary: primary, fallback: fallback}
}

func (r *ResilientEmbedder) ModelVersion() string {
	if mv, ok := r.primary.(ModelVersioner); ok {
		return mv.ModelVersion()
	}
	if mv, ok := r.fallback.(ModelVersioner); ok {
		return mv.ModelVersion()
	}
	return "unknown"
}

func (r *ResilientEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	vec, _, err := r.EmbedInputWithVersion(ctx, EmbeddingInput{Text: text})
	return vec, err
}

func (r *ResilientEmbedder) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	inputs := make([]EmbeddingInput, len(texts))
	for i := range texts {
		inputs[i] = EmbeddingInput{Text: texts[i]}
	}
	vecs, _, err := r.EmbedBatchInputsWithVersion(ctx, inputs)
	return vecs, err
}

func (r *ResilientEmbedder) EmbedInput(ctx context.Context, input EmbeddingInput) ([]float32, error) {
	vec, _, err := r.EmbedInputWithVersion(ctx, input)
	return vec, err
}

func (r *ResilientEmbedder) EmbedBatchInputs(ctx context.Context, inputs []EmbeddingInput) ([][]float32, error) {
	vecs, _, err := r.EmbedBatchInputsWithVersion(ctx, inputs)
	return vecs, err
}

func (r *ResilientEmbedder) EmbedInputWithVersion(ctx context.Context, input EmbeddingInput) ([]float32, string, error) {
	if r.primary != nil {
		if va, ok := r.primary.(VersionAwareEmbedder); ok {
			if v, ver, err := va.EmbedInputWithVersion(ctx, input); err == nil {
				return v, ver, nil
			} else {
				slog.Warn("embedding primary multimodal failed; falling back", "error", err)
			}
		} else if m, ok := r.primary.(MultimodalEmbedder); ok {
			if v, err := m.EmbedInput(ctx, input); err == nil {
				return v, modelVersionOf(r.primary), nil
			} else {
				slog.Warn("embedding primary multimodal failed; falling back", "error", err)
			}
		} else if v, err := r.primary.Embed(ctx, input.Text); err == nil {
			return v, modelVersionOf(r.primary), nil
		} else {
			slog.Warn("embedding primary failed; falling back", "error", err)
		}
	}

	if r.fallback == nil {
		if r.primary == nil {
			return nil, "", fmt.Errorf("embedding: no providers configured")
		}
		v, err := r.primary.Embed(ctx, input.Text)
		return v, modelVersionOf(r.primary), err
	}
	if va, ok := r.fallback.(VersionAwareEmbedder); ok {
		return va.EmbedInputWithVersion(ctx, input)
	}
	if m, ok := r.fallback.(MultimodalEmbedder); ok {
		if v, err := m.EmbedInput(ctx, input); err == nil {
			return v, modelVersionOf(r.fallback), nil
		}
	}
	v, err := r.fallback.Embed(ctx, input.Text)
	return v, modelVersionOf(r.fallback), err
}

func (r *ResilientEmbedder) EmbedBatchInputsWithVersion(ctx context.Context, inputs []EmbeddingInput) ([][]float32, string, error) {
	if r.primary != nil {
		if va, ok := r.primary.(VersionAwareEmbedder); ok {
			if v, ver, err := va.EmbedBatchInputsWithVersion(ctx, inputs); err == nil {
				return v, ver, nil
			} else {
				slog.Warn("embedding primary multimodal batch failed; falling back", "error", err, "count", len(inputs))
			}
		} else if m, ok := r.primary.(MultimodalEmbedder); ok {
			if v, err := m.EmbedBatchInputs(ctx, inputs); err == nil {
				return v, modelVersionOf(r.primary), nil
			} else {
				slog.Warn("embedding primary multimodal batch failed; falling back", "error", err, "count", len(inputs))
			}
		} else {
			texts := make([]string, len(inputs))
			for i := range inputs {
				texts[i] = inputs[i].Text
			}
			if v, err := r.primary.EmbedBatch(ctx, texts); err == nil {
				return v, modelVersionOf(r.primary), nil
			}
		}
	}

	texts := make([]string, len(inputs))
	for i := range inputs {
		texts[i] = inputs[i].Text
	}
	if r.fallback == nil {
		if r.primary == nil {
			return nil, "", fmt.Errorf("embedding: no providers configured")
		}
		v, err := r.primary.EmbedBatch(ctx, texts)
		return v, modelVersionOf(r.primary), err
	}
	if va, ok := r.fallback.(VersionAwareEmbedder); ok {
		return va.EmbedBatchInputsWithVersion(ctx, inputs)
	}
	if m, ok := r.fallback.(MultimodalEmbedder); ok {
		if v, err := m.EmbedBatchInputs(ctx, inputs); err == nil {
			return v, modelVersionOf(r.fallback), nil
		}
	}
	v, err := r.fallback.EmbedBatch(ctx, texts)
	return v, modelVersionOf(r.fallback), err
}

func modelVersionOf(e Embedder) string {
	if mv, ok := e.(ModelVersioner); ok {
		return mv.ModelVersion()
	}
	return ""
}
