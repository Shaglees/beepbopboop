package embedding

import "strings"

type ProviderConfig struct {
	Provider             string
	FallbackProvider     string
	GoogleAPIKey         string
	Model                string
	OutputDimensionality int
	AllowImageURLParts   bool
}

func NewEmbedderFromConfig(cfg ProviderConfig) Embedder {
	primary := buildProvider(cfg.Provider, cfg)
	fallback := buildProvider(cfg.FallbackProvider, cfg)
	if fallback == nil {
		fallback = NewHashEmbedder(cfg.OutputDimensionality, "hash-v1")
	}
	if primary == nil {
		return fallback
	}
	return NewResilientEmbedder(primary, fallback)
}

func buildProvider(name string, cfg ProviderConfig) Embedder {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "", "hash":
		return NewHashEmbedder(cfg.OutputDimensionality, "hash-v1")
	case "gemini", "google":
		if strings.TrimSpace(cfg.GoogleAPIKey) == "" {
			return nil
		}
		return NewGeminiEmbedder(cfg.GoogleAPIKey, cfg.Model, cfg.OutputDimensionality, cfg.AllowImageURLParts)
	default:
		return nil
	}
}
