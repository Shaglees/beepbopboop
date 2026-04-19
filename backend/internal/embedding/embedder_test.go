package embedding_test

import (
	"context"
	"testing"

	"github.com/shanegleeson/beepbopboop/backend/internal/embedding"
)

// mockEmbedder is a test double for the Embedder interface.
type mockEmbedder struct {
	calls int
}

func (m *mockEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	m.calls++
	return make([]float32, 1536), nil
}

func (m *mockEmbedder) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	m.calls++
	result := make([][]float32, len(texts))
	for i := range result {
		result[i] = make([]float32, 1536)
	}
	return result, nil
}

// verify mockEmbedder satisfies the interface at compile time
var _ embedding.Embedder = (*mockEmbedder)(nil)

func TestEmbedder_MockReturns1536Dims(t *testing.T) {
	m := &mockEmbedder{}
	vecs, err := m.EmbedBatch(context.Background(), []string{"a", "b", "c"})
	if err != nil {
		t.Fatalf("EmbedBatch failed: %v", err)
	}
	if len(vecs) != 3 {
		t.Fatalf("expected 3 results, got %d", len(vecs))
	}
	for i, v := range vecs {
		if len(v) != 1536 {
			t.Errorf("result[%d]: expected 1536 dims, got %d", i, len(v))
		}
	}
}
