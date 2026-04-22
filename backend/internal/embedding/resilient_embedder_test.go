package embedding

import (
	"context"
	"errors"
	"testing"
)

type fakeVersionedEmbedder struct {
	version string
	fail    bool
}

func (f *fakeVersionedEmbedder) ModelVersion() string { return f.version }

func (f *fakeVersionedEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	return f.EmbedInput(ctx, EmbeddingInput{Text: text})
}

func (f *fakeVersionedEmbedder) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	inputs := make([]EmbeddingInput, len(texts))
	for i := range texts {
		inputs[i] = EmbeddingInput{Text: texts[i]}
	}
	return f.EmbedBatchInputs(ctx, inputs)
}

func (f *fakeVersionedEmbedder) EmbedInput(ctx context.Context, input EmbeddingInput) ([]float32, error) {
	if f.fail {
		return nil, errors.New("boom")
	}
	return []float32{1, 2, 3}, nil
}

func (f *fakeVersionedEmbedder) EmbedBatchInputs(ctx context.Context, inputs []EmbeddingInput) ([][]float32, error) {
	if f.fail {
		return nil, errors.New("boom")
	}
	out := make([][]float32, len(inputs))
	for i := range inputs {
		out[i] = []float32{1, 2, 3}
	}
	return out, nil
}

func (f *fakeVersionedEmbedder) EmbedInputWithVersion(ctx context.Context, input EmbeddingInput) ([]float32, string, error) {
	vec, err := f.EmbedInput(ctx, input)
	if err != nil {
		return nil, "", err
	}
	return vec, f.version, nil
}

func (f *fakeVersionedEmbedder) EmbedBatchInputsWithVersion(ctx context.Context, inputs []EmbeddingInput) ([][]float32, string, error) {
	vecs, err := f.EmbedBatchInputs(ctx, inputs)
	if err != nil {
		return nil, "", err
	}
	return vecs, f.version, nil
}

func TestEmbedInputResolved_UsesFallbackModelVersion(t *testing.T) {
	primary := &fakeVersionedEmbedder{version: "google/gemini-embedding-002:dim1536", fail: true}
	fallback := &fakeVersionedEmbedder{version: "hash-v1", fail: false}
	r := NewResilientEmbedder(primary, fallback)

	vec, version, err := EmbedInputResolved(context.Background(), r, EmbeddingInput{Text: "hello"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(vec) == 0 {
		t.Fatal("expected vector")
	}
	if version != "hash-v1" {
		t.Fatalf("expected fallback version hash-v1, got %q", version)
	}
}

func TestEmbedInputResolved_UsesPrimaryModelVersionOnSuccess(t *testing.T) {
	primary := &fakeVersionedEmbedder{version: "google/gemini-embedding-002:dim1536", fail: false}
	fallback := &fakeVersionedEmbedder{version: "hash-v1", fail: false}
	r := NewResilientEmbedder(primary, fallback)

	_, version, err := EmbedInputResolved(context.Background(), r, EmbeddingInput{Text: "hello"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if version != "google/gemini-embedding-002:dim1536" {
		t.Fatalf("expected primary version, got %q", version)
	}
}
