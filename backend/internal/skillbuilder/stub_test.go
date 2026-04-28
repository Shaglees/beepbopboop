package skillbuilder

import (
	"strings"
	"testing"

	"github.com/shanegleeson/beepbopboop/backend/internal/model"
)

func TestBuild_Standalone(t *testing.T) {
	res, err := Build(model.CreateUserSkillRequest{
		Intent: "local high school football for Springfield, IL — score recaps and previews",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.SkillName != "local-high-school-football-for-springfield-il-sc" {
		t.Errorf("unexpected skill name: %q", res.SkillName)
	}
	if len(res.SkillName) > 48 {
		t.Errorf("skill name exceeds 48-char cap: %d", len(res.SkillName))
	}
	if len(res.Files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(res.Files))
	}
	skillMD := string(res.Files[0].Content)
	if res.Files[0].Path != "SKILL.md" {
		t.Errorf("expected SKILL.md, got %s", res.Files[0].Path)
	}
	if !strings.Contains(skillMD, "name: "+res.SkillName) {
		t.Errorf("SKILL.md missing frontmatter name: %s", skillMD)
	}
	if !strings.Contains(skillMD, "Springfield") {
		t.Errorf("SKILL.md should embed user intent")
	}
}

func TestBuild_Extension(t *testing.T) {
	res, err := Build(model.CreateUserSkillRequest{
		Kind:    model.UserSkillKindExtension,
		Extends: "beepbopboop-local-news",
		Intent:  "stop using paywalled sources, prefer my neighborhood",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.SkillName != "beepbopboop-local-news" {
		t.Errorf("extension skill name should equal extends, got %s", res.SkillName)
	}
	if len(res.Files) != 1 || res.Files[0].Path != "preferences.md" {
		t.Fatalf("expected single preferences.md, got %+v", res.Files)
	}
	if !strings.Contains(string(res.Files[0].Content), "paywalled") {
		t.Errorf("preferences.md should embed user intent")
	}
}

func TestBuild_Errors(t *testing.T) {
	cases := []struct {
		name string
		req  model.CreateUserSkillRequest
	}{
		{"empty intent", model.CreateUserSkillRequest{}},
		{"whitespace intent", model.CreateUserSkillRequest{Intent: "   "}},
		{"extension without extends", model.CreateUserSkillRequest{Kind: model.UserSkillKindExtension, Intent: "x"}},
		{"unknown kind", model.CreateUserSkillRequest{Kind: "weird", Intent: "x"}},
		{"intent with no alphanumeric", model.CreateUserSkillRequest{Intent: "!!!"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := Build(tc.req); err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"Hello World":                      "hello-world",
		"Local HS Football!":               "local-hs-football",
		"   spaces   ":                     "spaces",
		"emoji 🎉 stripped":                 "emoji-stripped",
		"":                                 "",
	}
	for in, want := range cases {
		if got := slugify(in); got != want {
			t.Errorf("slugify(%q) = %q, want %q", in, got, want)
		}
	}
}
