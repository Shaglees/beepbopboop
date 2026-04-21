package video

import "strings"

type ProviderPolicy struct {
	Provider           string `json:"provider"`
	SupportsPreviewCap bool   `json:"supports_preview_cap"`
	FallbackBehavior   string `json:"fallback_behavior"`
}

type ViabilitySample struct {
	Provider string `json:"provider"`
	Health   string `json:"health"`
}

type ProviderStats struct {
	OK      int            `json:"ok"`
	Blocked int            `json:"blocked"`
	Gone    int            `json:"gone"`
	Unknown int            `json:"unknown"`
	Policy  ProviderPolicy `json:"policy"`
}

type ViabilityReport struct {
	SampleSize       int                      `json:"sample_size"`
	NoLiveEmbedCount int                      `json:"no_live_embed_count"`
	Providers        map[string]ProviderStats `json:"providers"`
	Recommendation   string                   `json:"recommendation"`
}

func PolicyForProvider(provider string) ProviderPolicy {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "youtube":
		return ProviderPolicy{Provider: "youtube", SupportsPreviewCap: true, FallbackBehavior: "drop"}
	case "vimeo":
		return ProviderPolicy{Provider: "vimeo", SupportsPreviewCap: true, FallbackBehavior: "drop"}
	default:
		return ProviderPolicy{Provider: strings.ToLower(strings.TrimSpace(provider)), SupportsPreviewCap: false, FallbackBehavior: "drop"}
	}
}

func BuildViabilityReport(samples []ViabilitySample) ViabilityReport {
	report := ViabilityReport{
		SampleSize:     len(samples),
		Providers:      map[string]ProviderStats{},
		Recommendation: "Drop blocked/gone embeds from feed; do not fall back to article links.",
	}
	for _, sample := range samples {
		provider := strings.ToLower(strings.TrimSpace(sample.Provider))
		health := strings.ToLower(strings.TrimSpace(sample.Health))
		if provider == "" {
			if health == "no_live_embed" {
				report.NoLiveEmbedCount++
			}
			continue
		}
		stats := report.Providers[provider]
		stats.Policy = PolicyForProvider(provider)
		switch health {
		case "ok":
			stats.OK++
		case "blocked":
			stats.Blocked++
		case "gone":
			stats.Gone++
		default:
			stats.Unknown++
		}
		report.Providers[provider] = stats
	}
	return report
}
