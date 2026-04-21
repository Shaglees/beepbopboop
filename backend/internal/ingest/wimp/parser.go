package wimp

import (
	"bytes"
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

// Metadata is the subset of a Wimp page's HTML we care about for catalog rows.
type Metadata struct {
	Title        string
	Description  string
	ThumbnailURL string
	CanonicalURL string
	Keywords     []string
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
			case "iframe", "embed", "source":
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

var (
	reYouTubeEmbed = regexp.MustCompile(`(?:https?:)?//(?:www\.)?youtube(?:-nocookie)?\.com/embed/([A-Za-z0-9_-]{6,20})`)
	reYouTubeWatch = regexp.MustCompile(`(?:https?:)?//(?:www\.)?youtube\.com/watch\?[^"'\s]*?v=([A-Za-z0-9_-]{6,20})`)
	reYouTubeShort = regexp.MustCompile(`(?:https?:)?//youtu\.be/([A-Za-z0-9_-]{6,20})`)
	reVimeoPlayer  = regexp.MustCompile(`(?:https?:)?//player\.vimeo\.com/video/([0-9]{4,})`)
	reVimeoCanon   = regexp.MustCompile(`(?:https?:)?//(?:www\.)?vimeo\.com/([0-9]{4,})(?:[/?#]|$)`)
)

func parseEmbedURL(u string) (Embed, bool) {
	if u == "" {
		return Embed{}, false
	}
	if m := reYouTubeEmbed.FindStringSubmatch(u); m != nil {
		return youTubeEmbed(m[1]), true
	}
	if m := reYouTubeWatch.FindStringSubmatch(u); m != nil {
		return youTubeEmbed(m[1]), true
	}
	if m := reYouTubeShort.FindStringSubmatch(u); m != nil {
		return youTubeEmbed(m[1]), true
	}
	if m := reVimeoPlayer.FindStringSubmatch(u); m != nil {
		return vimeoEmbed(m[1]), true
	}
	if m := reVimeoCanon.FindStringSubmatch(u); m != nil {
		return vimeoEmbed(m[1]), true
	}
	return Embed{}, false
}

func scanScriptForEmbed(body string) (Embed, bool) {
	// Scripts may register player URLs via setup({ file: "..." }) etc.;
	// we only care about YouTube / Vimeo references by regex match.
	for _, re := range []*regexp.Regexp{reYouTubeEmbed, reYouTubeWatch, reYouTubeShort, reVimeoPlayer, reVimeoCanon} {
		if m := re.FindStringSubmatch(body); m != nil {
			switch re {
			case reVimeoPlayer, reVimeoCanon:
				return vimeoEmbed(m[1]), true
			default:
				return youTubeEmbed(m[1]), true
			}
		}
	}
	return Embed{}, false
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
