package wimp_test

import (
	"testing"

	"github.com/shanegleeson/beepbopboop/backend/internal/ingest/wimp"
)

// TestExtractEmbed_ExtendedProviders covers the new provider regexes added to
// the parser: Dailymotion, Twitch clips, Streamable, direct mp4.
//
// Each sub-case builds a minimal HTML snippet with exactly one provider URL so
// we verify both detection and the canonical watch/embed URLs produced.
func TestExtractEmbed_ExtendedProviders(t *testing.T) {
	cases := []struct {
		name, html     string
		provider       string
		videoID        string
		expectWatch    string
		expectEmbed    string
	}{
		{
			name:        "dailymotion canonical",
			html:        `<html><body><iframe src="https://www.dailymotion.com/video/x9abc12"></iframe></body></html>`,
			provider:    "dailymotion",
			videoID:     "x9abc12",
			expectWatch: "https://www.dailymotion.com/video/x9abc12",
			expectEmbed: "https://www.dailymotion.com/embed/video/x9abc12",
		},
		{
			name:        "dailymotion short",
			html:        `<html><body><a href="https://dai.ly/x9abc12">watch</a></body></html>`,
			provider:    "dailymotion",
			videoID:     "x9abc12",
			expectWatch: "https://www.dailymotion.com/video/x9abc12",
			expectEmbed: "https://www.dailymotion.com/embed/video/x9abc12",
		},
		{
			name:        "twitch clip",
			html:        `<html><body><iframe src="https://clips.twitch.tv/HilariousPandaClip-abc"></iframe></body></html>`,
			provider:    "twitch",
			videoID:     "HilariousPandaClip-abc",
			expectWatch: "https://clips.twitch.tv/HilariousPandaClip-abc",
			expectEmbed: "https://clips.twitch.tv/embed?clip=HilariousPandaClip-abc",
		},
		{
			name:        "streamable",
			html:        `<html><body><iframe src="https://streamable.com/e/abc123"></iframe></body></html>`,
			provider:    "streamable",
			videoID:     "abc123",
			expectWatch: "https://streamable.com/abc123",
			expectEmbed: "https://streamable.com/e/abc123",
		},
		{
			name:        "direct mp4 fallback",
			html:        `<html><body><video src="https://cdn.wimp.com/videos/2026/04/clip.mp4?v=1"></video></body></html>`,
			provider:    "mp4",
			videoID:     "https://cdn.wimp.com/videos/2026/04/clip.mp4?v=1",
			expectWatch: "https://cdn.wimp.com/videos/2026/04/clip.mp4?v=1",
			expectEmbed: "https://cdn.wimp.com/videos/2026/04/clip.mp4?v=1",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := wimp.ExtractEmbed([]byte(tc.html))
			if !ok {
				t.Fatalf("expected embed detection for %s", tc.name)
			}
			if got.Provider != tc.provider {
				t.Errorf("provider: got %q want %q", got.Provider, tc.provider)
			}
			if got.VideoID != tc.videoID {
				t.Errorf("video_id: got %q want %q", got.VideoID, tc.videoID)
			}
			if got.WatchURL != tc.expectWatch {
				t.Errorf("watch_url: got %q want %q", got.WatchURL, tc.expectWatch)
			}
			if got.EmbedURL != tc.expectEmbed {
				t.Errorf("embed_url: got %q want %q", got.EmbedURL, tc.expectEmbed)
			}
		})
	}
}

// TestExtractEmbed_YouTubeStillWinsOverMP4 confirms the ordering invariant:
// when a page has BOTH a YouTube iframe and a raw mp4 (common on modern
// wimp posts with a preview reel), the YouTube URL is preferred because it's
// earlier in the embedMatchers list AND first-in-document.
func TestExtractEmbed_YouTubeStillWinsOverMP4(t *testing.T) {
	html := `<html><body>
		<iframe src="https://www.youtube.com/embed/FIRST_YTXYZ"></iframe>
		<video src="https://cdn.example.com/backup.mp4"></video>
	</body></html>`
	got, ok := wimp.ExtractEmbed([]byte(html))
	if !ok {
		t.Fatal("expected embed detection")
	}
	if got.Provider != "youtube" {
		t.Errorf("expected YouTube to win over mp4, got provider=%q", got.Provider)
	}
}
