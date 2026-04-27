package handler

import (
	"encoding/json"
	"fmt"
	"strings"
)

// wrongKeyResponse is returned as HTTP 422 when external_url has wrong keys
// for a structured display_hint. The corrected JSON is included so the caller
// can retry immediately without guessing the right schema.
type wrongKeyResponse struct {
	Error                string   `json:"error"`
	Message              string   `json:"message"`
	CorrectedExternalURL string   `json:"corrected_external_url"`
	FixesApplied         []string `json:"fixes_applied"`
}

// remapExternalURL detects wrong keys in external_url JSON for structured hints
// and returns the corrected JSON plus a description of each fix applied.
// Returns ("", nil, nil) if no remapping is needed.
// Returns an error if the JSON is unparseable.
func remapExternalURL(hint, externalURL string) (corrected string, fixes []string, err error) {
	if externalURL == "" {
		return "", nil, nil
	}

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(externalURL), &data); err != nil {
		return "", nil, fmt.Errorf("invalid JSON: %w", err)
	}

	switch hint {
	case "matchup":
		fixes = remapMatchup(data)
	case "destination":
		fixes = remapDestination(data)
	case "entertainment":
		fixes = remapEntertainment(data)
	case "fitness":
		fixes = remapFitness(data)
	case "comparison":
		fixes = remapComparison(data)
	default:
		return "", nil, nil
	}

	if len(fixes) == 0 {
		return "", nil, nil
	}

	b, err := json.Marshal(data)
	if err != nil {
		return "", nil, fmt.Errorf("failed to re-encode corrected JSON: %w", err)
	}
	return string(b), fixes, nil
}

// remapMatchup fixes matchup external_url keys.
// - gameTime → date
// - sport value that is a league abbreviation → moves it to league, sets sport to sport name
// - league missing → adds it if inferable
func remapMatchup(data map[string]interface{}) []string {
	var fixes []string

	// gameTime → date
	if gt, ok := data["gameTime"]; ok {
		if _, hasDate := data["date"]; !hasDate {
			data["date"] = gt
			fixes = append(fixes, "gameTime → date")
		} else {
			fixes = append(fixes, "removed duplicate gameTime (date already present)")
		}
		delete(data, "gameTime")
	}

	// sport/league: if "sport" contains a league abbreviation, normalize
	leagueAbbrevToSport := map[string]string{
		"NBA": "basketball", "WNBA": "basketball",
		"NFL": "american football", "CFL": "american football",
		"MLB": "baseball", "NHL": "hockey",
		"MLS": "soccer", "NWSL": "soccer",
		"NBA G League": "basketball",
	}
	if sportVal, ok := data["sport"].(string); ok {
		if sportName, isLeague := leagueAbbrevToSport[strings.ToUpper(sportVal)]; isLeague {
			if _, hasLeague := data["league"]; !hasLeague {
				data["league"] = strings.ToUpper(sportVal)
				fixes = append(fixes, fmt.Sprintf("moved %q from sport to league", sportVal))
			}
			data["sport"] = sportName
			fixes = append(fixes, fmt.Sprintf("sport %q → %q (sport name, not league abbr)", sportVal, sportName))
		}
	}

	// league missing entirely — try to infer from context
	if _, hasLeague := data["league"]; !hasLeague {
		// Check if sport gives us a hint
		if sportVal, ok := data["sport"].(string); ok {
			switch strings.ToLower(sportVal) {
			case "basketball":
				data["league"] = "NBA"
				fixes = append(fixes, "added missing league (inferred NBA from sport=basketball)")
			case "american football", "football":
				data["league"] = "NFL"
				fixes = append(fixes, "added missing league (inferred NFL from sport=football)")
			case "baseball":
				data["league"] = "MLB"
				fixes = append(fixes, "added missing league (inferred MLB from sport=baseball)")
			case "hockey":
				data["league"] = "NHL"
				fixes = append(fixes, "added missing league (inferred NHL from sport=hockey)")
			case "soccer":
				data["league"] = "MLS"
				fixes = append(fixes, "added missing league (inferred MLS from sport=soccer)")
			}
		}
	}

	return fixes
}

// remapDestination fixes destination external_url keys.
// - city → name
func remapDestination(data map[string]interface{}) []string {
	var fixes []string
	if cityVal, ok := data["city"]; ok {
		if _, hasName := data["name"]; !hasName {
			data["name"] = cityVal
			fixes = append(fixes, "city → name")
		}
		delete(data, "city")
	}
	// Also fix "location" and "place" which are other common wrong keys
	for _, wrongKey := range []string{"location", "place"} {
		if val, ok := data[wrongKey]; ok {
			if _, hasName := data["name"]; !hasName {
				data["name"] = val
				fixes = append(fixes, fmt.Sprintf("%s → name", wrongKey))
			}
			delete(data, wrongKey)
		}
	}
	return fixes
}

// remapEntertainment fixes entertainment external_url keys.
// - subject → title
// - category → type
// - headline, tags, source, director, starring, etc. removed (not valid keys)
func remapEntertainment(data map[string]interface{}) []string {
	var fixes []string

	if subj, ok := data["subject"]; ok {
		if _, hasTitle := data["title"]; !hasTitle {
			data["title"] = subj
			fixes = append(fixes, "subject → title")
		}
		delete(data, "subject")
	}
	if headline, ok := data["headline"]; ok {
		if _, hasTitle := data["title"]; !hasTitle {
			data["title"] = headline
			fixes = append(fixes, "headline → title")
		}
		delete(data, "headline")
	}
	if cat, ok := data["category"]; ok {
		if _, hasType := data["type"]; !hasType {
			data["type"] = cat
			fixes = append(fixes, "category → type")
		}
		delete(data, "category")
	}

	// Remove invalid keys — entertainment only uses title and type
	invalidKeys := []string{"source", "tags", "director", "starring", "whereToWatch", "rtScore",
		"publishedDate", "doi", "readMoreUrl"}
	for _, k := range invalidKeys {
		if _, ok := data[k]; ok {
			delete(data, k)
			fixes = append(fixes, fmt.Sprintf("removed invalid key %q", k))
		}
	}

	// Normalize type value to valid options
	validTypes := map[string]string{
		"film": "film", "movie": "film", "cinema": "film",
		"tv": "tv", "television": "tv", "series": "tv", "show": "tv",
		"music": "music", "album": "music", "song": "music",
		"podcast": "podcast",
		"event": "event",
	}
	if typeVal, ok := data["type"].(string); ok {
		if normalized, known := validTypes[strings.ToLower(typeVal)]; known && normalized != typeVal {
			data["type"] = normalized
			fixes = append(fixes, fmt.Sprintf("type %q → %q", typeVal, normalized))
		}
	}

	return fixes
}

// remapFitness fixes fitness external_url keys.
// - activity → title
// - intensity → type
// - duration_min, notes removed (not valid keys)
func remapFitness(data map[string]interface{}) []string {
	var fixes []string

	if act, ok := data["activity"]; ok {
		if _, hasTitle := data["title"]; !hasTitle {
			data["title"] = act
			fixes = append(fixes, "activity → title")
		}
		delete(data, "activity")
	}
	if intens, ok := data["intensity"]; ok {
		if _, hasType := data["type"]; !hasType {
			data["type"] = intens
			fixes = append(fixes, "intensity → type")
		}
		delete(data, "intensity")
	}

	// Remove invalid keys
	for _, k := range []string{"duration_min", "notes", "calories", "equipment"} {
		if _, ok := data[k]; ok {
			delete(data, k)
			fixes = append(fixes, fmt.Sprintf("removed invalid key %q", k))
		}
	}

	// Normalize type value
	validFitnessTypes := map[string]string{
		"run": "run", "running": "run", "jog": "run",
		"workout": "workout", "gym": "workout", "hiit": "workout", "strength": "workout",
		"yoga": "yoga",
		"cycling": "cycling", "bike": "cycling", "bicycle": "cycling",
		"swim": "swim", "swimming": "swim",
	}
	if typeVal, ok := data["type"].(string); ok {
		if normalized, known := validFitnessTypes[strings.ToLower(typeVal)]; known && normalized != typeVal {
			data["type"] = normalized
			fixes = append(fixes, fmt.Sprintf("type %q → %q", typeVal, normalized))
		}
	}

	return fixes
}

// remapComparison validates comparison external_url has required keys.
// Returns fixes if the structure needs correction.
func remapComparison(data map[string]interface{}) []string {
	var fixes []string

	// comparison requires "title" and "items" array — no key renaming, just check
	// If title is missing but there's a heading/name field, promote it
	if _, hasTitle := data["title"]; !hasTitle {
		for _, alt := range []string{"heading", "name", "subject"} {
			if val, ok := data[alt]; ok {
				data["title"] = val
				delete(data, alt)
				fixes = append(fixes, fmt.Sprintf("%s → title", alt))
				break
			}
		}
	}

	return fixes
}

// missingRequiredKeysForHint returns which required keys are absent from the
// external_url JSON for a given hint. Used to decide whether to return 422
// even after remapping.
func missingRequiredKeysForHint(hint string, data map[string]interface{}) []string {
	required := map[string][]string{
		"matchup":       {"sport", "league", "date", "home", "away"},
		"destination":   {"name"},
		"entertainment": {"title", "type"},
		"fitness":       {"title", "type"},
		"comparison":    {"title", "items"},
		"concert":       {"artist", "venue", "date"},
		"restaurant":    {"name", "cuisine"},
	}
	keys, ok := required[hint]
	if !ok {
		return nil
	}
	var missing []string
	for _, k := range keys {
		if _, exists := data[k]; !exists {
			missing = append(missing, k)
		}
	}
	return missing
}

// templateForHint returns a minimal valid external_url JSON template for a hint.
func templateForHint(hint string) string {
	templates := map[string]string{
		"matchup":       `{"sport":"basketball","league":"NBA","date":"YYYY-MM-DD","home":{"name":"Home Team","abbr":"HOM"},"away":{"name":"Away Team","abbr":"AWY"}}`,
		"destination":   `{"name":"Place Name","country":"Country"}`,
		"entertainment": `{"title":"Post Title","type":"film"}`,
		"fitness":       `{"title":"Post Title","type":"run"}`,
		"comparison":    `{"title":"Ranking Title","items":[{"name":"Item 1","verdict":"Best for X"},{"name":"Item 2","verdict":"Best for Y"},{"name":"Item 3","verdict":"Best for Z"}]}`,
		"concert":       `{"artist":"Artist Name","venue":"Venue Name","date":"YYYY-MM-DD"}`,
		"restaurant":    `{"name":"Restaurant Name","cuisine":"Cuisine Type"}`,
	}
	if t, ok := templates[hint]; ok {
		return t
	}
	return ""
}
