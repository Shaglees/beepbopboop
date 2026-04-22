package embedding

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type GeminiEmbedder struct {
	apiKey               string
	model                string
	outputDimensionality int
	allowImageURLParts   bool
	httpClient           *http.Client
}

func NewGeminiEmbedder(apiKey, model string, outputDimensionality int, allowImageURLParts bool) *GeminiEmbedder {
	if strings.TrimSpace(model) == "" {
		model = "gemini-embedding-002"
	}
	if outputDimensionality <= 0 {
		outputDimensionality = 1536
	}
	return &GeminiEmbedder{
		apiKey:               strings.TrimSpace(apiKey),
		model:                model,
		outputDimensionality: outputDimensionality,
		allowImageURLParts:   allowImageURLParts,
		httpClient:           &http.Client{Timeout: 30 * time.Second},
	}
}

func (g *GeminiEmbedder) ModelVersion() string {
	return "google/" + g.model + ":dim" + fmt.Sprintf("%d", g.outputDimensionality)
}

func (g *GeminiEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	vecs, err := g.EmbedBatchInputs(ctx, []EmbeddingInput{{Text: text}})
	if err != nil {
		return nil, err
	}
	if len(vecs) != 1 {
		return nil, fmt.Errorf("gemini: expected 1 vector, got %d", len(vecs))
	}
	return vecs[0], nil
}

func (g *GeminiEmbedder) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	inputs := make([]EmbeddingInput, len(texts))
	for i := range texts {
		inputs[i] = EmbeddingInput{Text: texts[i]}
	}
	return g.EmbedBatchInputs(ctx, inputs)
}

func (g *GeminiEmbedder) EmbedInput(ctx context.Context, input EmbeddingInput) ([]float32, error) {
	vecs, err := g.EmbedBatchInputs(ctx, []EmbeddingInput{input})
	if err != nil {
		return nil, err
	}
	if len(vecs) != 1 {
		return nil, fmt.Errorf("gemini: expected 1 vector, got %d", len(vecs))
	}
	return vecs[0], nil
}

func (g *GeminiEmbedder) EmbedInputWithVersion(ctx context.Context, input EmbeddingInput) ([]float32, string, error) {
	vec, err := g.EmbedInput(ctx, input)
	if err != nil {
		return nil, "", err
	}
	return vec, g.ModelVersion(), nil
}

func (g *GeminiEmbedder) EmbedBatchInputsWithVersion(ctx context.Context, inputs []EmbeddingInput) ([][]float32, string, error) {
	vecs, err := g.EmbedBatchInputs(ctx, inputs)
	if err != nil {
		return nil, "", err
	}
	return vecs, g.ModelVersion(), nil
}

func (g *GeminiEmbedder) EmbedBatchInputs(ctx context.Context, inputs []EmbeddingInput) ([][]float32, error) {
	if strings.TrimSpace(g.apiKey) == "" {
		return nil, fmt.Errorf("gemini: missing API key")
	}
	if len(inputs) == 0 {
		return [][]float32{}, nil
	}

	reqs := make([]map[string]any, 0, len(inputs))
	for _, in := range inputs {
		reqs = append(reqs, map[string]any{
			"model": fmt.Sprintf("models/%s", g.model),
			"content": map[string]any{
				"parts": g.partsForInput(in),
			},
			"outputDimensionality": g.outputDimensionality,
		})
	}

	payload := map[string]any{"requests": reqs}
	body, _ := json.Marshal(payload)

	u := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:batchEmbedContents?key=%s", url.PathEscape(g.model), url.QueryEscape(g.apiKey))
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := g.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("gemini: status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var decoded struct {
		Embeddings []struct {
			Values []float32 `json:"values"`
		} `json:"embeddings"`
	}
	if err := json.Unmarshal(respBody, &decoded); err != nil {
		return nil, fmt.Errorf("gemini: decode response: %w", err)
	}
	if len(decoded.Embeddings) != len(inputs) {
		return nil, fmt.Errorf("gemini: expected %d embeddings, got %d", len(inputs), len(decoded.Embeddings))
	}

	out := make([][]float32, len(decoded.Embeddings))
	for i := range decoded.Embeddings {
		out[i] = decoded.Embeddings[i].Values
	}
	return out, nil
}

func (g *GeminiEmbedder) partsForInput(in EmbeddingInput) []map[string]any {
	parts := []map[string]any{{"text": in.Text}}
	if !g.allowImageURLParts {
		return parts
	}
	for _, u := range in.ImageURLs {
		u = strings.TrimSpace(u)
		if strings.HasPrefix(u, "http://") || strings.HasPrefix(u, "https://") {
			parts = append(parts, map[string]any{
				"fileData": map[string]any{
					"mimeType": "image/*",
					"fileUri":  u,
				},
			})
		}
	}
	return parts
}
