package wimp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

// defaultCDXBaseURL is the production Wayback endpoint. Tests override it.
const defaultCDXBaseURL = "https://web.archive.org"

// Capture is a single Wayback CDX row reduced to the fields we use.
type Capture struct {
	URLKey     string
	Timestamp  string // 14-char YYYYMMDDhhmmss
	Original   string
	MimeType   string
	StatusCode string
}

// IDURL returns the `web/{ts}id_/{original}` Wayback permalink, which
// serves the raw archived bytes without Wayback's toolbar rewrite.
func (c Capture) IDURL() string {
	return defaultCDXBaseURL + "/web/" + c.Timestamp + "id_/" + c.Original
}

// CaptureYear returns the 4-digit year from the 14-char timestamp, or "" if
// the timestamp is malformed.
func (c Capture) CaptureYear() string {
	if len(c.Timestamp) < 4 {
		return ""
	}
	return c.Timestamp[:4]
}

// CaptureTime parses the timestamp into UTC time. Returns zero time on failure.
func (c Capture) CaptureTime() time.Time {
	t, err := time.Parse("20060102150405", c.Timestamp)
	if err != nil {
		return time.Time{}
	}
	return t
}

// CDXClient wraps the Wayback CDX Search API.
type CDXClient struct {
	baseURL string
	http    *http.Client
}

// NewCDXClient returns a client pointed at baseURL (typically https://web.archive.org).
// httpClient is optional; passing nil uses http.DefaultClient.
func NewCDXClient(baseURL string, httpClient *http.Client) *CDXClient {
	if baseURL == "" {
		baseURL = defaultCDXBaseURL
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &CDXClient{baseURL: strings.TrimRight(baseURL, "/"), http: httpClient}
}

// LatestCapture returns the newest HTTP 200 text/html capture for rawURL.
// Returns ErrNoCapture when the archive has none.
func (c *CDXClient) LatestCapture(ctx context.Context, rawURL string) (Capture, error) {
	q := url.Values{}
	q.Set("url", rawURL)
	q.Set("output", "json")
	q.Set("filter", "statuscode:200")
	q.Add("filter", "mimetype:text/html")
	q.Set("limit", "25")

	reqURL := c.baseURL + "/cdx/search/cdx?" + q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return Capture{}, fmt.Errorf("cdx: build request: %w", err)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return Capture{}, fmt.Errorf("cdx: get: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return Capture{}, fmt.Errorf("cdx: upstream status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Capture{}, fmt.Errorf("cdx: read body: %w", err)
	}
	caps, err := parseCDXJSON(body)
	if err != nil {
		return Capture{}, fmt.Errorf("cdx: parse: %w", err)
	}
	if len(caps) == 0 {
		return Capture{}, ErrNoCapture
	}
	sort.Slice(caps, func(i, j int) bool { return caps[i].Timestamp > caps[j].Timestamp })
	return caps[0], nil
}

// parseCDXJSON turns the CDX Search JSON array response into captures.
// The first row is always a header; data rows are positional.
func parseCDXJSON(body []byte) ([]Capture, error) {
	var raw [][]string
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}
	if len(raw) < 2 {
		return nil, nil
	}
	header := raw[0]
	idx := map[string]int{}
	for i, h := range header {
		idx[h] = i
	}
	get := func(row []string, key string) string {
		i, ok := idx[key]
		if !ok || i >= len(row) {
			return ""
		}
		return row[i]
	}
	out := make([]Capture, 0, len(raw)-1)
	for _, row := range raw[1:] {
		out = append(out, Capture{
			URLKey:     get(row, "urlkey"),
			Timestamp:  get(row, "timestamp"),
			Original:   get(row, "original"),
			MimeType:   get(row, "mimetype"),
			StatusCode: get(row, "statuscode"),
		})
	}
	return out, nil
}
