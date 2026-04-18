package sports

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// GameData matches the Swift GameData struct in SportsData.swift.
type GameData struct {
	Sport     string   `json:"sport"`
	League    string   `json:"league"`
	Status    string   `json:"status"`
	GameTime  *string  `json:"gameTime,omitempty"`
	Home      TeamInfo `json:"home"`
	Away      TeamInfo `json:"away"`
	Headline  *string  `json:"headline,omitempty"`
	Venue     *string  `json:"venue,omitempty"`
	Broadcast *string  `json:"broadcast,omitempty"`
	Series    *string  `json:"series,omitempty"`
}

// TeamInfo matches the Swift TeamInfo struct.
type TeamInfo struct {
	Name   string  `json:"name"`
	Abbr   string  `json:"abbr"`
	Score  *int    `json:"score,omitempty"`
	Record *string `json:"record,omitempty"`
	Color  *string `json:"color,omitempty"`
}

// FetchedGame wraps GameData with the ESPN event ID for internal tracking.
type FetchedGame struct {
	EventID string
	League  string
	Data    GameData
}

type leagueDef struct {
	sport    string // GameData.sport value matched by iOS icon logic
	espnPath string // ESPN API path: {sport}/{league}
	name     string // short league name
}

var leagueDefs = []leagueDef{
	{"hockey", "hockey/nhl", "nhl"},
	{"basketball", "basketball/nba", "nba"},
	{"baseball", "baseball/mlb", "mlb"},
	{"football", "american-football/nfl", "nfl"},
}

const (
	espnCacheTTL = 5 * time.Minute
	espnBaseURL  = "https://site.api.espn.com/apis/site/v2/sports"
)

type cacheEntry struct {
	games     []FetchedGame
	expiresAt time.Time
}

// Service fetches live scores from the ESPN public scoreboard API.
type Service struct {
	client *http.Client
	mu     sync.RWMutex
	cache  map[string]cacheEntry
}

func NewService() *Service {
	return &Service{
		client: &http.Client{Timeout: 10 * time.Second},
		cache:  make(map[string]cacheEntry),
	}
}

// FetchAll returns games from all supported leagues.
func (s *Service) FetchAll() ([]FetchedGame, error) {
	var all []FetchedGame
	for _, lg := range leagueDefs {
		games, err := s.fetchLeagueCached(lg)
		if err != nil {
			// Log is handled by the caller; skip failing leagues.
			continue
		}
		all = append(all, games...)
	}
	return all, nil
}

func (s *Service) fetchLeagueCached(lg leagueDef) ([]FetchedGame, error) {
	s.mu.RLock()
	if e, ok := s.cache[lg.name]; ok && time.Now().Before(e.expiresAt) {
		s.mu.RUnlock()
		return e.games, nil
	}
	s.mu.RUnlock()

	games, err := s.fetchLeague(lg)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	s.cache[lg.name] = cacheEntry{games: games, expiresAt: time.Now().Add(espnCacheTTL)}
	s.mu.Unlock()

	return games, nil
}

func (s *Service) fetchLeague(lg leagueDef) ([]FetchedGame, error) {
	url := fmt.Sprintf("%s/%s/scoreboard", espnBaseURL, lg.espnPath)
	resp, err := s.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("espn %s request: %w", lg.name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("espn %s returned %d", lg.name, resp.StatusCode)
	}

	var raw espnScoreboard
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("espn %s decode: %w", lg.name, err)
	}

	return transformEvents(raw.Events, lg), nil
}

func transformEvents(events []espnEvent, lg leagueDef) []FetchedGame {
	games := make([]FetchedGame, 0, len(events))
	for _, ev := range events {
		if len(ev.Competitions) == 0 {
			continue
		}
		gd := transformGame(ev.Date, ev.Competitions[0], lg)
		games = append(games, FetchedGame{
			EventID: ev.ID,
			League:  lg.name,
			Data:    gd,
		})
	}
	return games
}

func transformGame(date string, comp espnCompetition, lg leagueDef) GameData {
	var home, away espnCompetitor
	for _, c := range comp.Competitors {
		if c.HomeAway == "home" {
			home = c
		} else {
			away = c
		}
	}

	status := comp.Status.Type.ShortDetail
	// Prefix live games so iOS isLive check (s.hasPrefix("live")) works reliably.
	if comp.Status.Type.State == "in" && !strings.HasPrefix(strings.ToLower(status), "live") {
		status = "Live " + status
	}

	gd := GameData{
		Sport:  lg.sport,
		League: lg.name,
		Status: status,
	}

	if comp.Status.Type.State == "pre" {
		gd.GameTime = &date
	}

	gd.Home = toTeamInfo(home)
	gd.Away = toTeamInfo(away)

	if comp.Venue != nil && comp.Venue.FullName != "" {
		gd.Venue = &comp.Venue.FullName
	}

	if len(comp.Broadcasts) > 0 && len(comp.Broadcasts[0].Names) > 0 {
		b := strings.Join(comp.Broadcasts[0].Names, ", ")
		gd.Broadcast = &b
	}

	if len(comp.Headlines) > 0 && comp.Headlines[0].ShortLinkText != "" {
		gd.Headline = &comp.Headlines[0].ShortLinkText
	}

	if comp.Series != nil && comp.Series.Summary != "" {
		gd.Series = &comp.Series.Summary
	}

	return gd
}

func toTeamInfo(c espnCompetitor) TeamInfo {
	ti := TeamInfo{
		Name: c.Team.DisplayName,
		Abbr: c.Team.Abbreviation,
	}
	if s, err := strconv.Atoi(c.Score); err == nil {
		ti.Score = &s
	}
	for _, rec := range c.Records {
		if rec.Name == "overall" {
			r := rec.Summary
			ti.Record = &r
			break
		}
	}
	if c.Team.Color != "" {
		col := "#" + c.Team.Color
		ti.Color = &col
	}
	return ti
}

// --- ESPN raw response types ---

type espnScoreboard struct {
	Events []espnEvent `json:"events"`
}

type espnEvent struct {
	ID           string           `json:"id"`
	Date         string           `json:"date"`
	Competitions []espnCompetition `json:"competitions"`
}

type espnCompetition struct {
	Competitors []espnCompetitor `json:"competitors"`
	Status      espnStatus       `json:"status"`
	Venue       *espnVenue       `json:"venue"`
	Broadcasts  []espnBroadcast  `json:"broadcasts"`
	Headlines   []espnHeadline   `json:"headlines"`
	Series      *espnSeries      `json:"series"`
}

type espnStatus struct {
	Type espnStatusType `json:"type"`
}

type espnStatusType struct {
	State       string `json:"state"`       // "pre", "in", "post"
	ShortDetail string `json:"shortDetail"` // "7:30 PM ET", "Live Q4 2:30", "Final"
}

type espnCompetitor struct {
	HomeAway string      `json:"homeAway"`
	Team     espnTeam    `json:"team"`
	Score    string      `json:"score"`
	Records  []espnRecord `json:"records"`
}

type espnTeam struct {
	DisplayName  string `json:"displayName"`
	Abbreviation string `json:"abbreviation"`
	Color        string `json:"color"`
}

type espnRecord struct {
	Name    string `json:"name"`
	Summary string `json:"summary"`
}

type espnVenue struct {
	FullName string `json:"fullName"`
}

type espnBroadcast struct {
	Names []string `json:"names"`
}

type espnHeadline struct {
	ShortLinkText string `json:"shortLinkText"`
}

type espnSeries struct {
	Summary string `json:"summary"`
}
