package main

import (
	"bufio"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/shanegleeson/beepbopboop/backend/internal/dedup"
)

var (
	dbPath string
	ttl    int
)

func main() {
	flag.StringVar(&dbPath, "db", defaultDBPath(), "path to SQLite database")
	flag.IntVar(&ttl, "ttl", 30, "TTL in days for post history")
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: beepbopgraph [--db PATH] [--ttl DAYS] <check|save|history|prune|stats> [flags]")
		os.Exit(1)
	}

	// stats doesn't need the local SQLite DB
	if args[0] == "stats" {
		runStats(args[1:])
		return
	}

	db, err := dedup.Open(dbPath)
	if err != nil {
		fatal("open db: %v", err)
	}
	defer db.Close()

	switch args[0] {
	case "check":
		runCheck(db, args[1:])
	case "save":
		runSave(db, args[1:])
	case "history":
		runHistory(db, args[1:])
	case "prune":
		runPrune(db, args[1:])
	default:
		fatal("unknown command: %s", args[0])
	}
}

func runCheck(db *sql.DB, args []string) {
	inputs, err := parseInputFlags("check", args)
	if err != nil {
		fatal("check: %v", err)
	}

	// Auto-prune expired entries
	dedup.Prune(db, ttl)

	existing, err := dedup.ListRecent(db, ttl)
	if err != nil {
		fatal("list recent: %v", err)
	}

	results := dedup.CheckBatch(existing, inputs, ttl)
	jsonOut(map[string]any{"results": results})
}

func runSave(db *sql.DB, args []string) {
	inputs, err := parseInputFlags("save", args)
	if err != nil {
		fatal("save: %v", err)
	}

	entries := make([]dedup.PostEntry, len(inputs))
	for i, input := range inputs {
		entries[i] = inputToEntry(input)
	}

	if err := dedup.SavePosts(db, entries); err != nil {
		fatal("save: %v", err)
	}

	jsonOut(map[string]int{"saved": len(entries)})
}

func runHistory(db *sql.DB, args []string) {
	fs := flag.NewFlagSet("history", flag.ExitOnError)
	limit := fs.Int("limit", 20, "max entries to show")
	days := fs.Int("days", 0, "filter to entries within N days (0 = use TTL)")
	filterType := fs.String("type", "", "filter by post type")
	filterLabel := fs.String("label", "", "filter by label")
	filterTag := fs.String("tag", "", "filter by tag")
	fs.Parse(args)

	lookback := ttl
	if *days > 0 {
		lookback = *days
	}

	posts, err := dedup.ListRecent(db, lookback)
	if err != nil {
		fatal("list: %v", err)
	}

	var filtered []dedup.PostEntry
	for _, p := range posts {
		if *filterType != "" && !strings.EqualFold(p.PostType, *filterType) {
			continue
		}
		if *filterLabel != "" && !hasLabel(p.Labels, *filterLabel) {
			continue
		}
		if *filterTag != "" && !strings.EqualFold(p.Tag, *filterTag) {
			continue
		}
		filtered = append(filtered, p)
		if len(filtered) >= *limit {
			break
		}
	}

	type row struct {
		ID       int64    `json:"id"`
		Title    string   `json:"title"`
		Type     string   `json:"type"`
		Labels   []string `json:"labels"`
		DaysAgo  int      `json:"days_ago"`
		Locality string   `json:"locality,omitempty"`
		Tag      string   `json:"tag,omitempty"`
	}
	rows := make([]row, len(filtered))
	for i, p := range filtered {
		rows[i] = row{
			ID:       p.ID,
			Title:    p.Title,
			Type:     p.PostType,
			Labels:   p.Labels,
			DaysAgo:  dedup.DaysAgo(p.CreatedAt),
			Locality: p.Locality,
			Tag:      p.Tag,
		}
	}
	jsonOut(map[string]any{"posts": rows, "total": len(rows)})
}

func runPrune(db *sql.DB, args []string) {
	fs := flag.NewFlagSet("prune", flag.ExitOnError)
	days := fs.Int("days", ttl, "prune entries older than N days")
	fs.Parse(args)

	n, err := dedup.Prune(db, *days)
	if err != nil {
		fatal("prune: %v", err)
	}
	jsonOut(map[string]int{"pruned": n})
}

func parseInputFlags(cmd string, args []string) ([]dedup.CheckInput, error) {
	fs := flag.NewFlagSet(cmd, flag.ExitOnError)
	title := fs.String("title", "", "post title")
	labels := fs.String("labels", "", "comma-separated labels")
	postType := fs.String("type", "", "post type")
	locality := fs.String("locality", "", "locality name")
	lat := fs.Float64("lat", 0, "latitude")
	lon := fs.Float64("lon", 0, "longitude")
	url := fs.String("url", "", "external URL")
	body := fs.String("body", "", "post body (first 200 chars used for body hash)")
	tag := fs.String("tag", "", "tag for categorizing entries (e.g. interest-discovery)")
	batch := fs.String("batch", "", "JSON array of posts")
	fs.Parse(args)

	if *batch != "" {
		var inputs []dedup.CheckInput
		if err := json.Unmarshal([]byte(*batch), &inputs); err != nil {
			return nil, fmt.Errorf("parse batch JSON: %w", err)
		}
		// Compute body hashes for batch inputs that have body text
		for i := range inputs {
			if inputs[i].Body != "" {
				inputs[i].Body = bodyHash(inputs[i].Body)
			}
		}
		return inputs, nil
	}

	if *title == "" {
		return nil, fmt.Errorf("--title is required (or use --batch)")
	}

	input := dedup.CheckInput{
		Title:    *title,
		PostType: *postType,
		Locality: *locality,
		URL:      *url,
		Tag:      *tag,
	}
	if *body != "" {
		input.Body = bodyHash(*body)
	}
	if *labels != "" {
		input.Labels = strings.Split(*labels, ",")
	}
	if *lat != 0 || *lon != 0 {
		input.Lat = lat
		input.Lon = lon
	}
	return []dedup.CheckInput{input}, nil
}

// bodyHash computes a SHA-256 hash of the first 200 characters of the body text.
func bodyHash(body string) string {
	// Truncate to first 200 chars
	text := body
	if len(text) > 200 {
		text = text[:200]
	}
	text = strings.ToLower(strings.TrimSpace(text))
	h := sha256.Sum256([]byte(text))
	return hex.EncodeToString(h[:])
}

func inputToEntry(input dedup.CheckInput) dedup.PostEntry {
	return dedup.PostEntry{
		Title:       input.Title,
		ExternalURL: input.URL,
		PostType:    input.PostType,
		Locality:    input.Locality,
		Latitude:    input.Lat,
		Longitude:   input.Lon,
		Labels:      input.Labels,
		BodyHash:    input.Body, // Already hashed by parseInputFlags
		Tag:         input.Tag,
	}
}

func hasLabel(labels []string, target string) bool {
	for _, l := range labels {
		if strings.EqualFold(l, target) {
			return true
		}
	}
	return false
}

func runStats(args []string) {
	fs := flag.NewFlagSet("stats", flag.ExitOnError)
	showWeights := fs.Bool("weights", false, "also show current user weights")
	noHistory := fs.Bool("no-history", false, "skip post history stats")
	fs.Parse(args)

	apiURL, token := loadConfig()

	// Fetch engagement summary
	summary := apiGet(apiURL+"/events/summary", token)

	output := map[string]any{"engagement": summary}

	// Fetch post history stats (default: included)
	if !*noHistory {
		history := apiGet(apiURL+"/posts/stats", token)
		output["post_history"] = history
	}

	if *showWeights {
		weights := apiGet(apiURL+"/user/weights", token)
		output["weights"] = weights
	}

	jsonOut(output)
}

func loadConfig() (apiURL, token string) {
	home, err := os.UserHomeDir()
	if err != nil {
		fatal("home dir: %v", err)
	}
	configPath := filepath.Join(home, ".config", "beepbopboop", "config")
	f, err := os.Open(configPath)
	if err != nil {
		fatal("open config: %v", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if k, v, ok := strings.Cut(line, "="); ok {
			switch k {
			case "BEEPBOPBOOP_API_URL":
				apiURL = v
			case "BEEPBOPBOOP_AGENT_TOKEN":
				token = v
			}
		}
	}
	if apiURL == "" || token == "" {
		fatal("config missing BEEPBOPBOOP_API_URL or BEEPBOPBOOP_AGENT_TOKEN")
	}
	return apiURL, token
}

func apiGet(url, token string) any {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fatal("build request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fatal("request %s: %v", url, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fatal("read response: %v", err)
	}
	if resp.StatusCode != 200 {
		fatal("GET %s: %d %s", url, resp.StatusCode, string(body))
	}

	var result any
	if err := json.Unmarshal(body, &result); err != nil {
		fatal("parse response: %v", err)
	}
	return result
}

func defaultDBPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "beepbopgraph.db"
	}
	return filepath.Join(home, ".config", "beepbopboop", "beepbopgraph.db")
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "beepbopgraph: "+format+"\n", args...)
	os.Exit(1)
}

func jsonOut(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		fatal("json encode: %v", err)
	}
}
