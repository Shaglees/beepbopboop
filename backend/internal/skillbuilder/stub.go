// Package skillbuilder produces SKILL.md and reference files from a user's
// intent submitted via POST /skills/user. The current implementation is a
// stub that emits a fixed-shape example skill so the rest of the pipeline
// (storage, manifest, openclaw sync) can be exercised end-to-end. A
// follow-up will replace it with a real Claude-API-backed builder.
package skillbuilder

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/shanegleeson/beepbopboop/backend/internal/model"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

// Result is what the builder hands back to the handler: the resolved skill
// name plus the files to persist.
type Result struct {
	SkillName string
	Files     []repository.FileInput
}

// Build returns a stub skill for the given request. It validates the
// request shape (kind/extends), sanitizes the skill name, and produces a
// minimal but valid skill folder. Errors here surface as 4xx in the handler.
func Build(req model.CreateUserSkillRequest) (*Result, error) {
	intent := strings.TrimSpace(req.Intent)
	if intent == "" {
		return nil, errors.New("intent is required")
	}

	kind := req.Kind
	if kind == "" {
		kind = model.UserSkillKindStandalone
	}

	switch kind {
	case model.UserSkillKindStandalone:
		name := slugify(intent)
		if name == "" {
			return nil, errors.New("could not derive a skill name from intent")
		}
		return &Result{
			SkillName: name,
			Files:     standaloneFiles(name, intent),
		}, nil

	case model.UserSkillKindExtension:
		if req.Extends == "" {
			return nil, errors.New("extends is required when kind is extension")
		}
		return &Result{
			SkillName: req.Extends,
			Files:     extensionFiles(req.Extends, intent),
		}, nil

	default:
		return nil, fmt.Errorf("unknown kind %q", kind)
	}
}

func standaloneFiles(name, intent string) []repository.FileInput {
	desc := summarize(intent)
	skillMD := fmt.Sprintf(`---
name: %s
description: %s
---

# %s

This skill was generated from a user intent submitted via the iOS app.

## Intent

> %s

## Notes

This is a placeholder produced by the skill-builder stub. A future
revision will replace this body with a real, sourced skill.
`, name, desc, name, intent)

	modeBrief := fmt.Sprintf(`# Brief mode

Produce one short post matching the intent below.

## Intent

> %s

## Output

A single post under 80 words.
`, intent)

	return []repository.FileInput{
		{Path: "SKILL.md", Content: []byte(skillMD)},
		{Path: "MODE_brief.md", Content: []byte(modeBrief)},
	}
}

func extensionFiles(extends, intent string) []repository.FileInput {
	prefs := fmt.Sprintf(`# User preferences for %s

These preferences are layered on top of the shipped %s skill at
post-composition time. They were captured from a user submission.

- %s
`, extends, extends, intent)

	return []repository.FileInput{
		{Path: "preferences.md", Content: []byte(prefs)},
	}
}

// summarize produces a short description suitable for SKILL.md frontmatter.
// Truncated to a single line; trailing whitespace stripped.
func summarize(intent string) string {
	line := strings.SplitN(intent, "\n", 2)[0]
	line = strings.TrimSpace(line)
	const maxLen = 120
	if len(line) > maxLen {
		line = line[:maxLen] + "..."
	}
	return line
}

var slugRe = regexp.MustCompile(`[^a-z0-9]+`)

// slugify converts free text into a kebab-case skill name. The result is
// lowercase, dash-separated, and capped at 48 characters to keep filesystem
// paths reasonable. Returns "" for inputs with no alphanumeric characters.
func slugify(s string) string {
	s = strings.ToLower(s)
	s = slugRe.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	const maxLen = 48
	if len(s) > maxLen {
		s = s[:maxLen]
		s = strings.TrimRight(s, "-")
	}
	return s
}
