package wimp

import (
	"bytes"
	"regexp"
	"strings"
	"time"

	"golang.org/x/net/html"
)

// noiseLabels are label strings that carry no filtering value across the
// catalog because they describe the medium, not the topic. Shared by
// buildLiveLabels (page keywords) and RSSLister.List (feed categories) so
// both sources agree on what to drop.
var noiseLabels = map[string]bool{
	"":       true,
	"video":  true,
	"videos": true,
	"clip":   true,
	"clips":  true,
}

// Metadata is the subset of a Wimp page's HTML we care about for catalog rows.
type Metadata struct {
	Title        string
	Description  string
	ThumbnailURL string
	CanonicalURL string
	Keywords     []string
	// PublishedAt is parsed from <meta property="article:published_time">.
	// Nil when the meta tag is absent or the value can't be parsed as RFC3339.
	PublishedAt *time.Time
}

// Embed is the first third-party video reference found in a Wimp page.
type Embed struct {
	Provider string // "youtube" or "vimeo"
	VideoID  string
	WatchURL string
	EmbedURL string
}

// ExtractMetadata pulls title / description / thumbnail / canonical URL /
// keywords from the given HTML. It never panics and returns zero values
// for fields it cannot locate.
func ExtractMetadata(htmlBytes []byte) Metadata {
	var md Metadata

	doc, err := html.Parse(bytes.NewReader(htmlBytes))
	if err != nil || doc == nil {
		return md
	}

	var inTitle bool
	var titleBuf strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		switch n.Type {
		case html.ElementNode:
			switch n.Data {
			case "title":
				inTitle = true
				defer func() { inTitle = false }()
			case "meta":
				applyMetaTag(&md, n)
			case "link":
				if md.CanonicalURL == "" && attr(n, "rel") == "canonical" {
					md.CanonicalURL = attr(n, "href")
				}
			}
		case html.TextNode:
			if inTitle && md.Title == "" {
				titleBuf.WriteString(n.Data)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	if md.Title == "" {
		md.Title = strings.TrimSpace(titleBuf.String())
	}
	return md
}

func applyMetaTag(md *Metadata, n *html.Node) {
	name := strings.ToLower(attr(n, "name"))
	prop := strings.ToLower(attr(n, "property"))
	content := attr(n, "content")
	if content == "" {
		return
	}
	switch {
	case prop == "og:title" && md.Title == "":
		md.Title = content
	case prop == "og:description" && md.Description == "":
		md.Description = content
	case prop == "og:image" && md.ThumbnailURL == "":
		md.ThumbnailURL = content
	case prop == "og:url" && md.CanonicalURL == "":
		md.CanonicalURL = content
	case prop == "article:published_time" && md.PublishedAt == nil:
		if t, err := time.Parse(time.RFC3339, content); err == nil {
			md.PublishedAt = &t
		}
	case name == "description" && md.Description == "":
		md.Description = content
	case name == "keywords" && len(md.Keywords) == 0:
		md.Keywords = splitCSV(content)
	}
}

// ExtractEmbed scans the HTML in document order and returns the first
// YouTube or Vimeo reference it finds. Candidates considered, in order of
// appearance: <iframe src>, <a href>, <embed src>, <source src>, and raw
// URLs in on-page script bodies.
//
// The "first-in-document" rule is deterministic and documented so callers
// and tests have stable expectations when a page references multiple
// providers.
func ExtractEmbed(htmlBytes []byte) (Embed, bool) {
	doc, err := html.Parse(bytes.NewReader(htmlBytes))
	if err != nil || doc == nil {
		return Embed{}, false
	}

	var found Embed
	var ok bool
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if ok {
			return
		}
		if n.Type == html.ElementNode {
			switch n.Data {
			case "iframe", "embed", "source", "video":
				if e, hit := parseEmbedURL(attr(n, "src")); hit {
					found, ok = e, true
					return
				}
			case "a":
				if e, hit := parseEmbedURL(attr(n, "href")); hit {
					found, ok = e, true
					return
				}
			case "script":
				for c := n.FirstChild; c != nil && !ok; c = c.NextSibling {
					if c.Type == html.TextNode {
						if e, hit := scanScriptForEmbed(c.Data); hit {
							found, ok = e, true
							return
						}
					}
				}
			}
		}
		for c := n.FirstChild; c != nil && !ok; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return found, ok
}

// --- URL parsing --------------------------------------------------------------

// Provider matchers.
//
// Each matcher extracts (provider, id) and reconstructs canonical watch/embed
// URLs. This is the deliberate complement to oEmbed lookup: once we have a
// canonical URL we can ask the provider for enriched metadata, and once we
// have (provider, video_id) we have a stable catalog key.
//
// We intentionally do NOT try to parse every social embed wimp.com ever used
// (Facebook, Instagram, TikTok, Twitter/X). Those either:
//   - don't expose a stable public oEmbed endpoint (FB/IG: require app tokens),
//   - or change embed URL shape frequently (TikTok, X).
// For now they're a follow-up. The tests below exercise the providers we DO
// recognize; when a new one needs support, add a regex + constructor here and
// extend the switch in scanScriptForEmbed + parseEmbedURL.
var (
	reYouTubeEmbed = regexp.MustCompile(`(?:https?:)?//(?:www\.)?youtube(?:-nocookie)?\.com/embed/([A-Za-z0-9_-]{6,20})`)
	reYouTubeWatch = regexp.MustCompile(`(?:https?:)?//(?:www\.)?youtube\.com/watch\?[^"'\s]*?v=([A-Za-z0-9_-]{6,20})`)
	reYouTubeShort = regexp.MustCompile(`(?:https?:)?//youtu\.be/([A-Za-z0-9_-]{6,20})`)
	reVimeoPlayer  = regexp.MustCompile(`(?:https?:)?//player\.vimeo\.com/video/([0-9]{4,})`)
	reVimeoCanon   = regexp.MustCompile(`(?:https?:)?//(?:www\.)?vimeo\.com/([0-9]{4,})(?:[/?#]|$)`)
	// Dailymotion: both /video/<id> and dai.ly/<id> shapes.
	// Dailymotion ids are at least 5 chars of base62 and terminated by a
	// path/query/fragment boundary. The length floor avoids matching navigation
	// slugs like /video/hot or /video/channel.
	reDailymotion      = regexp.MustCompile(`(?:https?:)?//(?:www\.)?dailymotion\.com/(?:embed/)?video/([A-Za-z0-9]{5,})(?:[/?#_]|$)`)
	reDailymotionShort = regexp.MustCompile(`(?:https?:)?//dai\.ly/([A-Za-z0-9]{5,})(?:[/?#]|$)`)
	// Twitch clips and videos.
	reTwitchClip   = regexp.MustCompile(`(?:https?:)?//clips\.twitch\.tv/([A-Za-z0-9_-]+)`)
	reTwitchPlayer = regexp.MustCompile(`(?:https?:)?//player\.twitch\.tv/\?(?:[^"'\s]*?(?:video|clip)=([A-Za-z0-9_-]+))`)
	// Streamable: streamable.com/<id> where <id> is 4+ chars of base62. The
	// length floor and terminator keep us from matching nav paths like /about,
	// /login, /terms which would otherwise produce broken catalog rows.
	reStreamable = regexp.MustCompile(`(?:https?:)?//(?:www\.)?streamable\.com/(?:e/)?([A-Za-z0-9]{4,})(?:[/?#]|$)`)
	// Generic JWPlayer-style setup({file: "..."}) is parsed as-is; we look for
	// a direct .mp4 URL as the last resort and mark provider as "mp4" so the
	// catalog can still serve it (iOS VideoEmbedCard supports raw mp4).
	//
	// Important: we REQUIRE .mp4 to be in the URL *path*, not a query string.
	// Forbidding ? and # before the .mp4 literal prevents pages like
	// https://player.example.com/watch?file=foo.mp4 — which are HTML watch
	// pages, not raw video — from being miscategorized as mp4 embeds.
	reDirectMP4 = regexp.MustCompile(`https?://[^"'\s<>?#]+\.mp4(?:\?[^"'\s<>]*)?`)
)

// embedMatcher pairs a URL-matching regex with a constructor for its Embed.
// Order matters: earlier entries win, so the most specific regexes are first.
// reDirectMP4 is the last-resort fallback because raw mp4s lack provider
// metadata and can't be enriched via oEmbed.
var embedMatchers = []struct {
	re    *regexp.Regexp
	build func(id string) Embed
}{
	{reYouTubeEmbed, youTubeEmbed},
	{reYouTubeWatch, youTubeEmbed},
	{reYouTubeShort, youTubeEmbed},
	{reVimeoPlayer, vimeoEmbed},
	{reVimeoCanon, vimeoEmbed},
	{reDailymotion, dailymotionEmbed},
	{reDailymotionShort, dailymotionEmbed},
	{reTwitchClip, twitchClipEmbed},
	{reTwitchPlayer, twitchClipEmbed},
	{reStreamable, streamableEmbed},
	{reDirectMP4, directMP4Embed},
}

func parseEmbedURL(u string) (Embed, bool) {
	if u == "" {
		return Embed{}, false
	}
	for _, m := range embedMatchers {
		if match := m.re.FindStringSubmatch(u); match != nil {
			// reDirectMP4 has no capture group for an id; the whole URL is the id.
			var id string
			if len(match) < 2 {
				id = match[0]
			} else {
				id = match[1]
			}
			built := m.build(id)
			if !isPlausibleEmbed(built) {
				continue
			}
			return built, true
		}
	}
	return Embed{}, false
}

func scanScriptForEmbed(body string) (Embed, bool) {
	// Scripts may register player URLs via setup({ file: "..." }) or plain
	// inline JSON blobs. We scan for any known provider pattern.
	for _, m := range embedMatchers {
		if match := m.re.FindStringSubmatch(body); match != nil {
			var id string
			if len(match) < 2 {
				id = match[0]
			} else {
				id = match[1]
			}
			built := m.build(id)
			if !isPlausibleEmbed(built) {
				continue
			}
			return built, true
		}
	}
	return Embed{}, false
}

// isPlausibleEmbed filters out matches whose "id" is actually a navigation
// slug on the provider's domain. Without this check, /about or /login pages
// linked from a wimp post would produce permanently-broken catalog rows.
//
// We only deny-list for providers whose id shape is loose enough to
// collide with English words (Streamable, Dailymotion). YouTube/Vimeo ids
// are already well-constrained by their respective regexes.
func isPlausibleEmbed(e Embed) bool {
	switch e.Provider {
	case "streamable":
		return !streamableNavSlugs[strings.ToLower(e.VideoID)]
	case "dailymotion":
		return !dailymotionNavSlugs[strings.ToLower(e.VideoID)]
	}
	return true
}

// streamableNavSlugs are real streamable.com URL paths that are NOT videos.
// Compiled from the site's top-level nav. If Streamable ships a video whose
// id collides with one of these words, we'll miss it — but that's vastly
// preferable to ingesting a broken embed every time a wimp post links to
// streamable.com/about.
var streamableNavSlugs = map[string]bool{
	"about": true, "login": true, "signup": true, "terms": true, "privacy": true,
	"pricing": true, "help": true, "contact": true, "upgrade": true, "upload": true,
	"explore": true, "trending": true, "account": true, "settings": true,
	"dashboard": true, "search": true, "logout": true,
}

var dailymotionNavSlugs = map[string]bool{
	"hot": true, "new": true, "trending": true, "featured": true, "channels": true,
	"channel": true, "topics": true, "feed": true, "live": true, "search": true,
	"about": true, "login": true, "signup": true, "upload": true,
}

func youTubeEmbed(id string) Embed {
	return Embed{
		Provider: "youtube",
		VideoID:  id,
		WatchURL: "https://www.youtube.com/watch?v=" + id,
		EmbedURL: "https://www.youtube.com/embed/" + id,
	}
}

func vimeoEmbed(id string) Embed {
	return Embed{
		Provider: "vimeo",
		VideoID:  id,
		WatchURL: "https://vimeo.com/" + id,
		EmbedURL: "https://player.vimeo.com/video/" + id,
	}
}

func dailymotionEmbed(id string) Embed {
	return Embed{
		Provider: "dailymotion",
		VideoID:  id,
		WatchURL: "https://www.dailymotion.com/video/" + id,
		EmbedURL: "https://www.dailymotion.com/embed/video/" + id,
	}
}

func twitchClipEmbed(id string) Embed {
	return Embed{
		Provider: "twitch",
		VideoID:  id,
		WatchURL: "https://clips.twitch.tv/" + id,
		EmbedURL: "https://clips.twitch.tv/embed?clip=" + id,
	}
}

func streamableEmbed(id string) Embed {
	return Embed{
		Provider: "streamable",
		VideoID:  id,
		WatchURL: "https://streamable.com/" + id,
		EmbedURL: "https://streamable.com/e/" + id,
	}
}

// directMP4Embed treats the raw URL as both the watch and embed URL, and uses
// the URL itself as the natural id. Upstream code that cares about oEmbed
// enrichment should skip provider="mp4" since there is no oEmbed endpoint.
func directMP4Embed(url string) Embed {
	return Embed{
		Provider: "mp4",
		VideoID:  url,
		WatchURL: url,
		EmbedURL: url,
	}
}

// --- html helpers -------------------------------------------------------------

func attr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if strings.EqualFold(a.Key, key) {
			return a.Val
		}
	}
	return ""
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}
