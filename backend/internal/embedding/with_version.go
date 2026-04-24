package embedding

import "context"

// EmbedInputResolved embeds a single payload and returns vector + actual model version used.
func EmbedInputResolved(ctx context.Context, embedder Embedder, input EmbeddingInput) ([]float32, string, error) {
	if va, ok := embedder.(VersionAwareEmbedder); ok {
		return va.EmbedInputWithVersion(ctx, input)
	}

	var (
		vec []float32
		err error
	)
	if mm, ok := embedder.(MultimodalEmbedder); ok {
		vec, err = mm.EmbedInput(ctx, input)
	} else {
		vec, err = embedder.Embed(ctx, input.Text)
	}
	if err != nil {
		return nil, "", err
	}
	modelVersion := ""
	if mv, ok := embedder.(ModelVersioner); ok {
		modelVersion = mv.ModelVersion()
	}
	return vec, modelVersion, nil
}

// EmbedBatchResolved embeds a batch payload and returns vectors + actual model version used.
func EmbedBatchResolved(ctx context.Context, embedder Embedder, inputs []EmbeddingInput) ([][]float32, string, error) {
	if va, ok := embedder.(VersionAwareEmbedder); ok {
		return va.EmbedBatchInputsWithVersion(ctx, inputs)
	}

	var (
		vecs [][]float32
		err  error
	)
	if mm, ok := embedder.(MultimodalEmbedder); ok {
		vecs, err = mm.EmbedBatchInputs(ctx, inputs)
	} else {
		texts := make([]string, len(inputs))
		for i := range inputs {
			texts[i] = inputs[i].Text
		}
		vecs, err = embedder.EmbedBatch(ctx, texts)
	}
	if err != nil {
		return nil, "", err
	}
	modelVersion := ""
	if mv, ok := embedder.(ModelVersioner); ok {
		modelVersion = mv.ModelVersion()
	}
	return vecs, modelVersion, nil
}
