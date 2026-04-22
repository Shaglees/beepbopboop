package embedding

import "context"

// EmbeddingInput is a provider-agnostic embedding payload.
// Text is always present; ImageURLs are optional multimodal context.
type EmbeddingInput struct {
	Text      string
	ImageURLs []string
}

// Embedder converts text into dense vector representations.
type Embedder interface {
	Embed(ctx context.Context, text string) ([]float32, error)
	EmbedBatch(ctx context.Context, texts []string) ([][]float32, error)
}

// MultimodalEmbedder is an optional extension for providers that can embed
// richer payloads (text + media context).
type MultimodalEmbedder interface {
	EmbedInput(ctx context.Context, input EmbeddingInput) ([]float32, error)
	EmbedBatchInputs(ctx context.Context, inputs []EmbeddingInput) ([][]float32, error)
}

// ModelVersioner allows embedding providers to surface model/version metadata
// for storage, logging, and safe migrations.
type ModelVersioner interface {
	ModelVersion() string
}

// VersionAwareEmbedder can return the exact model version used for a given
// embedding call (important when resilient/fallback providers are in play).
type VersionAwareEmbedder interface {
	EmbedInputWithVersion(ctx context.Context, input EmbeddingInput) ([]float32, string, error)
	EmbedBatchInputsWithVersion(ctx context.Context, inputs []EmbeddingInput) ([][]float32, string, error)
}
