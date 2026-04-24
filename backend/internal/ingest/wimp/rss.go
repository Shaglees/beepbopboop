package wimp

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Wimp.com exposes a standard WordPress RSS 2.0 feed at /feed/ with the most
// recent posts (usually 10). Each item carries:
//
//   - <title>           — wimp's post title (generic summary of the video)
//   - <link>            — canonical post permalink
//   - <pubDate>         — RFC1123Z timestamp
//   - <category>×N      — editorial categories ("Dogs", "Funny", "Technology")
//   - <description>     — short caption (HTML-escaped)
//   - <dc:creator>      — wimp curator name
//
// The RSS feed itself does NOT contain the embedded YouTube/Vimeo URL — that
// lives on the post page. The lister's job is just to enumerate candidate
// permalinks; the adapter pulls the actual embed from the page HTML.

// rssMaxBodyBytes caps the RSS body read. Wimp's feed is ~20KB today; a 2MB
// ceiling is orders of magnitude beyond any plausible growth and protects
// against hostile upstreams.
const rssMaxBodyBytes = 2 * 1024 * 1024

// RSSItem is the subset of RSS fields we care about for ingest.
type RSSItem struct {
	Title       string    `json:"title"`
	Link        string    `json:"link"`
	PubDate     time.Time `json:"pub_date"`
	Categories  []string  `json:"categories"`
	Description string    `json:"description"`
	Author      string    `json:"author"`
}

// defaultWimpUserAgent is the single UA string used by every component that
// talks to a wimp.com origin (RSS feed, live post pages, oEmbed). Keeping it
// unified avoids a footgun where the adapter and lister appear as two
// different clients to upstream rate-limiters.
const defaultWimpUserAgent = "beepbopboop-wimp-ingest/1.0 (+https://github.com/Shaglees/beepbopboop)"

// RSSLister fetches wimp.com's RSS feed and returns structured items.
//
// It is intentionally NOT a full Atom/RSS 2.0 parser — we parse only the
// fields wimp uses. If wimp migrates to a different CMS with a different
// feed shape, this is the one file to rewrite.
type RSSLister struct {
	feedURL   string
	http      *http.Client
	userAgent string
}

// NewRSSLister returns a lister for the given feed URL. If feedURL is empty,
// the default https://www.wimp.com/feed/ is used. The UA is shared with the
// adapter via defaultWimpUserAgent.
func NewRSSLister(feedURL string, httpClient *http.Client) *RSSLister {
	if feedURL == "" {
		feedURL = "https://www.wimp.com/feed/"
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}
	return &RSSLister{
		feedURL:   feedURL,
		http:      httpClient,
		userAgent: defaultWimpUserAgent,
	}
}

// rssFeed / rssItem are the private XML unmarshaling shapes. They're kept
// narrow on purpose so the JSON-facing RSSItem is stable even if the upstream
// feed adds noise fields later.
type rssFeed struct {
	XMLName xml.Name  `xml:"rss"`
	Channel rssChannel `xml:"channel"`
}
type rssChannel struct {
	Items []rssItem `xml:"item"`
}
type rssItem struct {
	Title       string   `xml:"title"`
	Link        string   `xml:"link"`
	PubDate     string   `xml:"pubDate"`
	Categories  []string `xml:"category"`
	Description string   `xml:"description"`
	Creator     string   `xml:"http://purl.org/dc/elements/1.1/ creator"`
}

// List fetches the feed and returns items, newest first. limit<=0 returns all.
func (l *RSSLister) List(ctx context.Context, limit int) ([]RSSItem, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, l.feedURL, nil)
	if err != nil {
		return nil, fmt.Errorf("wimp rss: build request: %w", err)
	}
	req.Header.Set("User-Agent", l.userAgent)
	req.Header.Set("Accept", "application/rss+xml, application/xml, text/xml")

	resp, err := l.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("wimp rss: fetch: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("wimp rss: upstream status %d", resp.StatusCode)
	}
	// Wimp's feed is ~20KB; 2MB is a very generous cap that bounds memory
	// without breaking legitimate growth.
	body, err := io.ReadAll(io.LimitReader(resp.Body, rssMaxBodyBytes))
	if err != nil {
		return nil, fmt.Errorf("wimp rss: read body: %w", err)
	}

	var feed rssFeed
	if err := xml.Unmarshal(body, &feed); err != nil {
		return nil, fmt.Errorf("wimp rss: parse xml: %w", err)
	}

	out := make([]RSSItem, 0, len(feed.Channel.Items))
	for _, it := range feed.Channel.Items {
		item := RSSItem{
			Title:       strings.TrimSpace(it.Title),
			Link:        strings.TrimSpace(it.Link),
			Description: strings.TrimSpace(stripCDATA(it.Description)),
			Author:      strings.TrimSpace(it.Creator),
		}
		for _, c := range it.Categories {
			c = strings.ToLower(strings.TrimSpace(c))
			if c == "" || noiseLabels[c] {
				continue
			}
			item.Categories = append(item.Categories, c)
		}
		if t, err := parseRSSDate(it.PubDate); err == nil {
			item.PubDate = t
		}
		if item.Link == "" {
			continue
		}
		out = append(out, item)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, nil
}

// stripCDATA removes the leading/trailing CDATA wrapper WordPress likes to
// emit on <description>.
func stripCDATA(s string) string {
	s = strings.TrimPrefix(s, "<![CDATA[")
	s = strings.TrimSuffix(s, "]]>")
	return s
}

// parseRSSDate accepts the two formats we see in wimp's feed:
//   - RFC1123Z ("Tue, 21 Apr 2026 14:00:07 +0000") — primary WordPress form
//   - RFC1123  (without timezone offset) — older feeds
func parseRSSDate(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if t, err := time.Parse(time.RFC1123Z, s); err == nil {
		return t, nil
	}
	return time.Parse(time.RFC1123, s)
}
