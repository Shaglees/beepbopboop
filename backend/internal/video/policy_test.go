package video_test

import (
	"testing"

	videokit "github.com/shanegleeson/beepbopboop/backend/internal/video"
)

func TestPolicyForProvider(t *testing.T) {
	youtube := videokit.PolicyForProvider("youtube")
	if !youtube.SupportsPreviewCap || youtube.FallbackBehavior != "drop" {
		t.Fatalf("unexpected youtube policy: %+v", youtube)
	}

	vimeo := videokit.PolicyForProvider("vimeo")
	if !vimeo.SupportsPreviewCap || vimeo.FallbackBehavior != "drop" {
		t.Fatalf("unexpected vimeo policy: %+v", vimeo)
	}

	unknown := videokit.PolicyForProvider("unknown")
	if unknown.SupportsPreviewCap || unknown.FallbackBehavior != "drop" {
		t.Fatalf("unexpected unknown policy: %+v", unknown)
	}
}

func TestBuildViabilityReport(t *testing.T) {
	report := videokit.BuildViabilityReport([]videokit.ViabilitySample{
		{Provider: "youtube", Health: "ok"},
		{Provider: "youtube", Health: "blocked"},
		{Provider: "vimeo", Health: "ok"},
		{Provider: "vimeo", Health: "gone"},
		{Provider: "", Health: "no_live_embed"},
	})

	if report.SampleSize != 5 {
		t.Fatalf("sample size: got %d", report.SampleSize)
	}
	if report.NoLiveEmbedCount != 1 {
		t.Fatalf("no_live_embed count: got %d", report.NoLiveEmbedCount)
	}
	if report.Providers["youtube"].Blocked != 1 || report.Providers["youtube"].OK != 1 {
		t.Fatalf("unexpected youtube stats: %+v", report.Providers["youtube"])
	}
	if report.Providers["vimeo"].Gone != 1 || report.Providers["vimeo"].OK != 1 {
		t.Fatalf("unexpected vimeo stats: %+v", report.Providers["vimeo"])
	}
	if report.Recommendation != "Drop blocked/gone embeds from feed; do not fall back to article links." {
		t.Fatalf("recommendation: %q", report.Recommendation)
	}
}
