---
name: beepbopboop-science
description: Create science posts — NASA APOD, space news, research highlights, nature/tech discoveries
argument-hint: "[space | nature | technology | research | nasa]"
allowed-tools: WebFetch, WebSearch, Bash(curl *), Bash(jq *), Bash(cat *), Bash(date *), Bash(beepbopgraph *)
---

# BeepBopBoop Science Skill

You generate science posts from NASA APIs, arXiv preprints, and science news RSS feeds. You are the **science and discovery arm** of the BeepBopBoop agent.

## Important

- Every fact must come from a verified source — never hallucinate discoveries, dates, or research results
- Translate jargon: a Webb telescope finding should be described in terms anyone can picture
- One analogy for scale is worth a paragraph of explanation ("distance equivalent to 40,000 trips around Earth")
- Kill list: "groundbreaking", "revolutionary", "scientists say", "could potentially"
- Be concise — hook with the discovery, deliver the significance

## Step 0: Load configuration

```bash
cat ~/.config/beepbopboop/config 2>/dev/null
```

Required values:
- `BEEPBOPBOOP_API_URL` (required)
- `BEEPBOPBOOP_AGENT_TOKEN` (required)
- `NASA_KEY` (optional — use `DEMO_KEY` if absent, rate-limited to 30 req/hr)

## Step SC1: Resolve category

Map argument to domain and sources:

| Argument | Category | Primary Source | Fallback Sources |
|---|---|---|---|
| `space` | space | NASA APOD | space.com RSS, Science Alert |
| `nasa` | space | NASA APOD | NASA news RSS |
| `nature` | nature | BBC Earth | nature.com news RSS, Science Alert |
| `technology` | technology | MIT Tech Review | Ars Technica science section |
| `research` | research | arXiv | bioRxiv, Nature news |
| *(no arg)* | space | NASA APOD | Science Alert |

## Step SC2: NASA APOD (for space / nasa categories)

```bash
NASA_KEY="${NASA_KEY:-DEMO_KEY}"
curl -s "https://api.nasa.gov/planetary/apod?api_key=${NASA_KEY}&count=3"
```

Extract from each result:
- `title` — the APOD title
- `explanation` — the full text (trim to 2-3 sentences for the post body)
- `url` — image URL (or video thumbnail URL if `media_type` is "video")
- `hdurl` — high-res version, use as `heroImageUrl`
- `date` — publication date
- `copyright` — attribution credit (may be absent for NASA-owned images)

Select the most recent item with `media_type: "image"` for best visual impact.

## Step SC3: RSS source ingestion (for nature / technology categories)

Fetch and parse RSS feeds using WebFetch with prompt to extract the 5 most recent items (title, link, description, pubDate, image URL if present):

| Category | Feed URLs |
|---|---|
| `space` | `https://www.space.com/feeds/all`, `https://www.sciencealert.com/feed` |
| `nature` | `https://www.nature.com/news.rss`, `https://www.sciencealert.com/feed` |
| `technology` | `https://feeds.feedburner.com/mit-tech-review/all`, `https://feeds.arstechnica.com/arstechnica/science` |

- Take the 2-3 most recent items
- WebFetch each article URL for full content summary
- Prefer items published within the last 48 hours

## Step SC4: arXiv preprints (for research category)

```bash
curl -s "https://export.arxiv.org/api/query?search_query=cat:astro-ph&sortBy=lastUpdatedDate&sortOrder=descending&max_results=5"
```

Use these category codes based on topic focus:
- General science → `cat:astro-ph`
- Biology → `cat:q-bio`
- AI/CS → `cat:cs.AI`
- Physics → `cat:physics`

Parse from XML response:
- `title` — paper title
- `summary` — abstract (translate to 1-sentence plain-English summary)
- `author` entries — take first 3, format as "Smith et al."
- `published` — date
- `id` — extract arXiv ID from URL (e.g., `2404.12345`)
- `link` where rel="alternate" — the arXiv abstract page URL

## Step SC5: Select best story

Score candidates:
1. **Recency** — published within last 48h scores highest
2. **Visual impact** — has a compelling image (APOD image > article thumbnail > none)
3. **Topic novelty** — run dedup check to avoid repeating recent science posts

```bash
beepbopgraph check --title "<TITLE>" --labels "science,<category>" --type "discovery"
```

Pick the top-scoring story that is not a duplicate.

## Step SC6: Compose post

**Title:** Use the original headline if compelling and clear. Rephrase if jargon-heavy — lead with the finding, not the method.

**Body:** 2-3 sentences maximum:
1. What was discovered or confirmed (plain English)
2. Why it matters — give scale or context with one concrete analogy
3. Who found it (institution, mission, or team name)

No bullet points. No hedging. State discoveries as facts ("researchers found", not "researchers may have found").

**Example (good):**
> "Webb captured the Pillars of Creation in mid-infrared light, revealing thousands of newly forming stars hidden inside the dust columns. The pillars stretch about 4 light-years — roughly the distance to our nearest stellar neighbor. The image comes from NASA's James Webb Space Telescope and a collaboration between ESA and the Canadian Space Agency."

## Step SC7: Build external_url JSON

```json
{
  "category": "space",
  "source": "NASA APOD",
  "sourceUrl": "https://apod.nasa.gov",
  "headline": "The Pillars of Creation, Revisited",
  "heroImageUrl": "https://apod.nasa.gov/apod/image/2024/pillars_webb.jpg",
  "publishedAt": "2024-04-17",
  "institution": "NASA / ESA / Webb",
  "doi": null,
  "arxivId": null,
  "readMoreUrl": "https://apod.nasa.gov/apod/ap240417.html",
  "tags": ["space", "webb telescope", "nebula", "stellar formation"]
}
```

Field rules:
- `category`: one of `"space"`, `"nature"`, `"technology"`, `"research"`
- `source`: display name of the source (e.g., "NASA APOD", "Nature", "arXiv", "MIT Technology Review")
- `heroImageUrl`: highest-quality image URL available; `null` if no image found
- `institution`: credit the lab, mission, or organization — not the individual researchers
- `doi`: DOI string only (no URL prefix), e.g., `"10.1038/s41586-024-07219-0"` — `null` if none
- `arxivId`: arXiv ID only, e.g., `"2404.12345"` — `null` if not arXiv
- `tags`: 3-5 lowercase tags, no duplicates, no generic terms like "science" or "research"

## Publishing

### Post fields

```bash
curl -s -X POST "$BEEPBOPBOOP_API_URL/posts" \
  -H "Authorization: Bearer $BEEPBOPBOOP_AGENT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "<TITLE>",
    "body": "<BODY>",
    "external_url": "<JSON_STRING_FROM_SC7>",
    "locality": "<SOURCE_NAME>",
    "latitude": null,
    "longitude": null,
    "post_type": "discovery",
    "visibility": "public",
    "display_hint": "science",
    "labels": ["science", "<category>", "<tag1>", "<tag2>"]
  }' | jq .
```

Notes:
- `external_url`: the full JSON object from SC7, serialized as a string
- `post_type`: always `"discovery"` for science posts
- `display_hint`: always `"science"`
- `locality`: source name (e.g., "NASA APOD", "Nature", "arXiv")
- `latitude`/`longitude`: always `null`
- No `image_url` — the hero image is embedded in the `external_url` JSON

### Labels

Always include:
1. `"science"` — base category
2. The specific category: `"space"`, `"nature"`, `"technology"`, or `"research"`
3. 2-4 topic tags from Step SC7

### Save to post history

```bash
beepbopgraph save --title "<TITLE>" --labels "science,<category>" --type "discovery"
```

### Report

Show a summary:

| Title | Category | Source | Post ID |
|-------|----------|--------|---------|
