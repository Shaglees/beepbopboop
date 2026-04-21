package wimp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// CDXArchiveLister pages through Wayback CDX results for wimp.com HTML pages
// and returns the original source URLs for the crawler to inspect.
type CDXArchiveLister struct {
	baseURL string
	http    *http.Client
}

func NewCDXArchiveLister(baseURL string, httpClient *http.Client) *CDXArchiveLister {
	if baseURL == "" {
		baseURL = defaultCDXBaseURL
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &CDXArchiveLister{
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    httpClient,
	}
}

func (l *CDXArchiveLister) ListPageURLs(ctx context.Context, offset, limit int) ([]string, error) {
	if limit <= 0 {
		limit = 25
	}
	q := url.Values{}
	q.Set("url", "wimp.com")
	q.Set("matchType", "domain")
	q.Set("output", "json")
	q.Set("fl", "original,timestamp")
	q.Set("filter", "statuscode:200")
	q.Add("filter", "mimetype:text/html")
	q.Set("limit", fmt.Sprintf("%d", limit))
	q.Set("offset", fmt.Sprintf("%d", offset))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, l.baseURL+"/cdx/search/cdx?"+q.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("wimp list: build request: %w", err)
	}
	resp, err := l.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("wimp list: request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("wimp list: upstream status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("wimp list: read body: %w", err)
	}
	var raw [][]string
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("wimp list: parse body: %w", err)
	}
	if len(raw) < 2 {
		return nil, nil
	}
	seen := map[string]bool{}
	out := make([]string, 0, len(raw)-1)
	for _, row := range raw[1:] {
		if len(row) == 0 || row[0] == "" {
			continue
		}
		normalized, err := NormalizeWimpURL(row[0])
		if err != nil {
			continue
		}
		if normalized == "https://www.wimp.com/" || strings.Contains(normalized, "/search/") {
			continue
		}
		if !seen[normalized] {
			seen[normalized] = true
			out = append(out, normalized)
		}
	}
	return out, nil
}
