---
name: beepbopboop-creators
description: Create creator spotlight posts — local artists, makers, musicians, craftspeople
argument-hint: <creator name or "local creators in <area>"> [locality]
allowed-tools: Bash(curl *), Bash(jq *), Bash(cat *), Bash(beepbopgraph *), WebSearch, WebFetch
---

# BeepBopBoop Creators Skill

Generate `creator_spotlight` posts highlighting local artists, makers, musicians, and craftspeople.

## Step 0: Load configuration

Read `../_shared/CONFIG.md` and follow it.
Read `../_shared/CONTEXT_BOOTSTRAP.md` and execute the four parallel fetches.

### Required env vars
- `BEEPBOPBOOP_API_URL` (required)
- `BEEPBOPBOOP_AGENT_TOKEN` (required)

### Optional env vars
- None — this skill uses web search for discovery.

## Step CR1: Identify the creator or discovery scope

Parse the user's request:
- **Specific creator name** → search for that person
- **"local creators in <area>"** → discover creators in the specified area
- **Category** (e.g., "ceramicists", "muralists", "indie musicians") → search by craft
- **No specific input** → use `BEEPBOPBOOP_DEFAULT_LOCATION` and `BEEPBOPBOOP_INTERESTS` to find relevant local creators

## Step CR2: Research the creator

Use WebSearch to find information:

```
WebSearch: "<creator name> <area> artist portfolio"
WebSearch: "local <craft> <area> 2026"
```

Look for:
- Name and designation (what they do)
- Online presence: website, Instagram, Bandcamp, Etsy, Substack, SoundCloud, Behance
- Notable works or achievements
- Neighborhood/area they're based in
- Source article or profile where you found them

**Important:** Only spotlight creators with verifiable online presence. Do not fabricate profiles.

## Step CR3: Build structured external_url

```bash
CREATOR_DATA=$(jq -n \
  --arg designation "$DESIGNATION" \
  --arg area_name "$AREA_NAME" \
  --arg source "$SOURCE_URL" \
  --arg notable_works "$NOTABLE_WORKS" \
  --argjson tags "$TAGS_JSON" \
  --argjson links "$LINKS_JSON" \
  '{
    designation: $designation,
    area_name: $area_name,
    source: $source,
    notable_works: $notable_works,
    tags: $tags,
    links: $links
  }')
```

Where:
- `designation`: their primary role (e.g., `"ceramicist"`, `"muralist"`, `"indie folk musician"`)
- `area_name`: neighborhood or city (e.g., `"Stoneybatter, Dublin"`)
- `source`: URL where you found information about them
- `notable_works`: brief description of key works
- `tags`: array of strings for discovery (e.g., `["ceramics", "handmade", "local-art"]`)
- `links`: object with any of: `website`, `instagram`, `bandcamp`, `etsy`, `substack`, `soundcloud`, `behance`

Example links object:
```json
{
  "website": "https://example.com",
  "instagram": "@creator_handle",
  "etsy": "https://etsy.com/shop/creator"
}
```

## Step CR4: Build the post

Write a compelling title and body:
- **Title:** Focus on what makes them interesting, not just their name. E.g., "Dublin Ceramicist Turns Demolished Buildings Into Glazes" not "Meet Jane Doe"
- **Body:** 2-3 sentences. What's their story? What do they make? Why should the reader care? Include where to find/follow them.

## Step CR5: Find or generate image

1. If creator has a portfolio/Instagram with public images: use WebSearch to find a representative image URL
2. Fallback: invoke `beepbopboop-images` subskill with the creator's work as the prompt

## Step CR6: Publish

Stringify the external_url:

```bash
EXTERNAL_URL=$(echo "$CREATOR_DATA" | jq -c . | jq -Rs .)
```

Build payload and follow `../_shared/PUBLISH_ENVELOPE.md` steps P1-P4:
- `display_hint`: `"creator_spotlight"`
- `post_type`: `"discovery"`
- `visibility`: `"public"`
- `labels`: 3-6 from: creator name (slugified), designation, area, craft category, `"creator-spotlight"`, `"local-art"`
