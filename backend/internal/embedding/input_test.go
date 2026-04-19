package embedding_test

import (
	"strings"
	"testing"

	"github.com/shanegleeson/beepbopboop/backend/internal/embedding"
	"github.com/shanegleeson/beepbopboop/backend/internal/model"
)

func TestBuildEmbeddingInput_IncludesAllFields(t *testing.T) {
	p := model.Post{
		Title:    "Game 3",
		Body:     "Lakers lead series",
		Labels:   []string{"sports", "nba"},
		PostType: "discovery",
		Locality: "Los Angeles",
	}
	out := embedding.BuildEmbeddingInput(p)
	if out == "" {
		t.Fatal("expected non-empty output")
	}
	for _, want := range []string{"Game 3", "Lakers", "sports", "nba", "Los Angeles"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in output, got: %q", want, out)
		}
	}
}

func TestBuildEmbeddingInput_StructuredHints(t *testing.T) {
	t.Run("scoreboard_natural_language", func(t *testing.T) {
		p := model.Post{
			Title:       "Lakers vs Celtics",
			Body:        "Game 3 preview",
			DisplayHint: "scoreboard",
			ExternalURL: `{"sport":"NBA","home":{"name":"Lakers","score":110},"away":{"name":"Celtics","score":105},"status":"Final"}`,
		}
		out := embedding.BuildEmbeddingInput(p)
		if strings.Contains(out, "{") || strings.Contains(out, "}") {
			t.Errorf("scoreboard: expected natural language, got JSON in output: %q", out)
		}
		if !strings.Contains(strings.ToLower(out), "lakers") {
			t.Errorf("scoreboard: expected team name in output, got: %q", out)
		}
	})

	t.Run("weather_natural_language", func(t *testing.T) {
		p := model.Post{
			Title:       "Today's Weather",
			Body:        "Check the forecast",
			DisplayHint: "weather",
			ExternalURL: `{"current":{"temp_c":20,"condition":"Sunny"}}`,
		}
		out := embedding.BuildEmbeddingInput(p)
		if strings.Contains(out, "{") || strings.Contains(out, "}") {
			t.Errorf("weather: expected natural language, got JSON in output: %q", out)
		}
		if !strings.Contains(strings.ToLower(out), "weather") {
			t.Errorf("weather: expected 'weather' in output, got: %q", out)
		}
	})

	t.Run("card_uses_title_and_body", func(t *testing.T) {
		p := model.Post{
			Title:       "My Post",
			Body:        "Post body text",
			DisplayHint: "card",
			ExternalURL: `{"some":"data"}`,
		}
		out := embedding.BuildEmbeddingInput(p)
		if !strings.Contains(out, "My Post") {
			t.Errorf("card: expected title in output, got: %q", out)
		}
		if !strings.Contains(out, "Post body text") {
			t.Errorf("card: expected body in output, got: %q", out)
		}
	})
}

func TestBuildEmbeddingInput_EmptyPost(t *testing.T) {
	p := model.Post{}
	out := embedding.BuildEmbeddingInput(p)
	_ = out // should not panic; empty string is acceptable
}

func TestSummariseForEmbedding_Scoreboard(t *testing.T) {
	raw := `{"sport":"NBA","home":{"name":"Lakers","score":110},"away":{"name":"Celtics","score":105},"status":"Final"}`
	out := embedding.SummariseForEmbedding("scoreboard", raw)
	if strings.Contains(out, "{") {
		t.Errorf("expected no JSON braces, got: %q", out)
	}
	if !strings.Contains(strings.ToLower(out), "lakers") {
		t.Errorf("expected team name, got: %q", out)
	}
}

func TestSummariseForEmbedding_Weather(t *testing.T) {
	raw := `{"current":{"temp_c":20,"condition":"Sunny"}}`
	out := embedding.SummariseForEmbedding("weather", raw)
	if strings.Contains(out, "{") {
		t.Errorf("expected no JSON braces, got: %q", out)
	}
	if !strings.Contains(strings.ToLower(out), "weather") {
		t.Errorf("expected 'weather' in output, got: %q", out)
	}
}

func TestSummariseForEmbedding_Weather_ZeroCelsius(t *testing.T) {
	// 0°C is a valid temperature and must not be treated as "missing"
	raw := `{"current":{"temp_c":0,"temp_f":32,"condition":"Clear"}}`
	out := embedding.SummariseForEmbedding("weather", raw)
	if !strings.Contains(out, "0°C") {
		t.Errorf("expected 0°C in output (not falling back to Fahrenheit), got: %q", out)
	}
}
