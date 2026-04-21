package wimp_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shanegleeson/beepbopboop/backend/internal/ingest/wimp"
)

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return b
}

// --- ExtractEmbed -------------------------------------------------------------

func TestExtractEmbed_FlashEraNoLiveEmbed(t *testing.T) {
	html := readFixture(t, "flyingbike_2014.html")
	_, ok := wimp.ExtractEmbed(html)
	if ok {
		t.Fatalf("expected no live third-party embed for Flash-era page")
	}
}

func TestExtractEmbed_YouTubeIframe(t *testing.T) {
	html := readFixture(t, "beatles_bloopers_2019_youtube.html")
	got, ok := wimp.ExtractEmbed(html)
	if !ok {
		t.Fatalf("expected YouTube embed on real 2019 fixture")
	}
	if got.Provider != "youtube" {
		t.Errorf("provider: got %q want youtube", got.Provider)
	}
	if got.VideoID != "NZd3R2iw4cA" {
		t.Errorf("video_id: got %q want NZd3R2iw4cA", got.VideoID)
	}
	if want := "https://www.youtube.com/watch?v=NZd3R2iw4cA"; got.WatchURL != want {
		t.Errorf("watch_url: got %q want %q", got.WatchURL, want)
	}
	if want := "https://www.youtube.com/embed/NZd3R2iw4cA"; got.EmbedURL != want {
		t.Errorf("embed_url: got %q want %q", got.EmbedURL, want)
	}
}

func TestExtractEmbed_YouTubeWatchLink(t *testing.T) {
	html := readFixture(t, "youtube_watch_link.html")
	got, ok := wimp.ExtractEmbed(html)
	if !ok {
		t.Fatalf("expected embed from youtube.com/watch link")
	}
	if got.Provider != "youtube" || got.VideoID != "jNQXAC9IVRw" {
		t.Errorf("unexpected extraction: %+v", got)
	}
}

func TestExtractEmbed_VimeoPlayerIframe(t *testing.T) {
	html := readFixture(t, "vimeo_player_iframe.html")
	got, ok := wimp.ExtractEmbed(html)
	if !ok {
		t.Fatalf("expected vimeo embed")
	}
	if got.Provider != "vimeo" || got.VideoID != "76979871" {
		t.Errorf("unexpected extraction: %+v", got)
	}
	if want := "https://vimeo.com/76979871"; got.WatchURL != want {
		t.Errorf("watch_url: got %q want %q", got.WatchURL, want)
	}
	if want := "https://player.vimeo.com/video/76979871"; got.EmbedURL != want {
		t.Errorf("embed_url: got %q want %q", got.EmbedURL, want)
	}
}

func TestExtractEmbed_MultipleEmbeds_FirstInDocumentWins(t *testing.T) {
	// Documented tiebreaker: first candidate in document order wins.
	// In this fixture the YouTube iframe appears before the Vimeo iframe.
	html := readFixture(t, "multiple_embeds.html")
	got, ok := wimp.ExtractEmbed(html)
	if !ok {
		t.Fatalf("expected an embed")
	}
	if got.Provider != "youtube" || got.VideoID != "dQw4w9WgXcQ" {
		t.Errorf("expected YouTube dQw4w9WgXcQ (first in document), got %+v", got)
	}
}

func TestExtractEmbed_Malformed_NoEmbed(t *testing.T) {
	html := readFixture(t, "malformed.html")
	_, ok := wimp.ExtractEmbed(html)
	if ok {
		t.Fatalf("expected no embed in malformed fixture")
	}
}

// --- ExtractMetadata ----------------------------------------------------------

func TestExtractMetadata_BeatlesFixture(t *testing.T) {
	html := readFixture(t, "beatles_bloopers_2019_youtube.html")
	md := wimp.ExtractMetadata(html)

	if md.Title == "" {
		t.Errorf("expected non-empty title")
	}
	if !strings.Contains(strings.ToLower(md.Title), "beatles") {
		t.Errorf("expected title to mention beatles, got %q", md.Title)
	}
	if md.Description == "" {
		t.Errorf("expected non-empty description")
	}
	if md.ThumbnailURL == "" {
		t.Errorf("expected non-empty thumbnail (og:image)")
	}
	if !strings.HasPrefix(md.ThumbnailURL, "http") {
		t.Errorf("thumbnail should be absolute URL, got %q", md.ThumbnailURL)
	}
	if md.CanonicalURL == "" || !strings.Contains(md.CanonicalURL, "wimp.com") {
		t.Errorf("expected wimp.com canonical url, got %q", md.CanonicalURL)
	}
}

func TestExtractMetadata_VimeoFixture(t *testing.T) {
	html := readFixture(t, "vimeo_player_iframe.html")
	md := wimp.ExtractMetadata(html)

	if md.Title != "A short film worth watching" {
		t.Errorf("title: got %q", md.Title)
	}
	if !strings.Contains(md.Description, "independent director") {
		t.Errorf("description: got %q", md.Description)
	}
	if md.ThumbnailURL != "https://www.wimp.com/images/thumbs/abc123_shortfilm_800_450.jpg" {
		t.Errorf("thumbnail: got %q", md.ThumbnailURL)
	}
	if md.CanonicalURL != "https://www.wimp.com/a-short-film-worth-watching/" {
		t.Errorf("canonical: got %q", md.CanonicalURL)
	}
	if len(md.Keywords) != 3 || md.Keywords[0] != "short film" {
		t.Errorf("keywords: got %+v", md.Keywords)
	}
}

func TestExtractMetadata_MalformedFixture_ReturnsWhatItCan(t *testing.T) {
	// Parser must not panic on broken markup; returns the description it
	// can find via og:description and leaves the rest empty.
	html := readFixture(t, "malformed.html")
	md := wimp.ExtractMetadata(html)
	if !strings.Contains(md.Description, "Broken markup") {
		t.Errorf("expected og:description to be recovered, got %q", md.Description)
	}
}
