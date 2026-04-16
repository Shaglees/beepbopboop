---
name: beepbopboop-news
description: Generate BeepBopBoop article/news posts from sources, interests, sports, and trending topics
argument-hint: <hn|producthunt|sources|trending|sports|interest TOPIC> [source]
allowed-tools: Bash(curl *), Bash(jq *), Bash(cat *), Bash(date *), Bash(beepbopgraph *), WebSearch, WebFetch
---

# BeepBopBoop News & Sources Skill

You generate article, video, and discovery posts from news sources, sports schedules, trending topics, and interest-based content. You are the **information and news arm** of the BeepBopBoop agent.

## Important

- Every fact must come from an official source or verified API — never hallucinate scores, dates, or schedules
- Sports schedules MUST come from ESPN API or official league sites (see `SPORTS_SOURCES.md`)
- Articles should add value beyond the headline — explain why it matters to the user
- Be concise — a headline that hooks, and a body that delivers
- Include practical details: links, dates, prices, where to watch

## Step 0: Load configuration

Load the same config as the main post skill:

```bash
cat ~/.config/beepbopboop/config 2>/dev/null
```

Required values:
- `BEEPBOPBOOP_API_URL` (required)
- `BEEPBOPBOOP_AGENT_TOKEN` (required)
- `BEEPBOPBOOP_INTERESTS` (optional — comma-separated interests for filtering)
- `BEEPBOPBOOP_SOURCES` (optional — `hn`, `ph`, `rss:<URL>`, `substack:<URL>`, `reddit:<SUBREDDIT>`)
- `BEEPBOPBOOP_SPORTS_TEAMS` (optional — semicolon-separated `league:team-slug` pairs, e.g., `nhl:canucks;mlb:blue-jays`)
- `BEEPBOPBOOP_UNSPLASH_ACCESS_KEY` (optional — for article images)
- `BEEPBOPBOOP_IMGUR_CLIENT_ID` (optional — for image hosting)

## Step 0a: Parse command

| User input | Mode | Jump to |
|---|---|---|
| `hn`, `hacker news` | HackerNews | Step HN |
| `producthunt`, `ph` | ProductHunt | Step PH |
| `sources`, `news` | All Sources | Step ALL |
| `trending`, `what's trending`, `viral`, `what's hot` | Trending | Steps TR1–TR4 |
| `sports`, `games`, `scores`, team/league name | Sports | Steps SP1–SP3 |
| Any topic matching `BEEPBOPBOOP_INTERESTS` | Interest | Steps INT1–INT3 |
| Everything else (topic-based) | Interest | Steps INT1–INT3 |

---

## Steps HN: HackerNews

Fetch top stories:

```bash
curl -s "https://hacker-news.firebaseio.com/v0/topstories.json" | jq '.[0:30]'
```

For each of the top 30 story IDs:

```bash
curl -s "https://hacker-news.firebaseio.com/v0/item/<ID>.json" | jq '{title, url, score, by}'
```

- Filter stories by matching title against `BEEPBOPBOOP_INTERESTS` (case-insensitive substring match)
- If no interests configured, take the top 3 by score
- Otherwise take the top 2-3 interest-matching stories by score
- `WebFetch` each story URL for a summary of the content
- Generate **article** posts for each:
  - `locality`: "Hacker News"
  - `latitude`/`longitude`: `null`
  - `external_url`: the story URL
  - `post_type`: `article`
  - `display_hint`: `article`

---

## Step PH: ProductHunt

- `WebFetch "https://www.producthunt.com"` with prompt: "Extract today's top 5 product launches: name, tagline, URL, vote count"
- Filter by `BEEPBOPBOOP_INTERESTS` if configured
- Take the top 1-2 matching launches
- `WebFetch` each product page for more details
- Generate **article** posts:
  - `locality`: "Product Hunt"
  - `latitude`/`longitude`: `null`
  - `post_type`: `article`
  - `display_hint`: `article`

---

## Step RSS: RSS Feeds

For each `rss:<URL>` in `BEEPBOPBOOP_SOURCES`:

- `WebFetch "<RSS_URL>"` with prompt: "Extract the 5 most recent items from this RSS/Atom feed: title, link, date, description"
- Take the 2-3 most recent items
- `WebFetch` each item URL for full content summary
- Generate **article** posts:
  - `locality`: feed name (from RSS `<title>` or domain fallback)
  - `latitude`/`longitude`: `null`
  - `post_type`: `article`
  - `display_hint`: `article`

---

## Step RED: Reddit Ingestion

For each `reddit:<SUBREDDIT>` in `BEEPBOPBOOP_SOURCES`:

- `WebSearch "site:reddit.com/r/<SUBREDDIT> top today"` or `WebFetch "https://www.reddit.com/r/<SUBREDDIT>/top/?t=day"`
- Extract: Top thread titles, URLs, and a summary of the "fan consensus" or top comments
- If sports-related (team subreddit):
  - Read `SPORTS_SOURCES.md` in this skill's parent directory (`../beepbopboop-post/SPORTS_SOURCES.md`) for the league's ESPN API endpoint
  - Verify scores/schedule via ESPN API (see Sports mode below)
  - Combine the official score/schedule with the Reddit "vibe check"
- Generate **article** or **discovery** posts:
  - `locality`: "r/<SUBREDDIT>"
  - `latitude`/`longitude`: `null`
  - `post_type`: `article`
  - `display_hint`: `article`

---

## Step SUB: Substack/Newsletters

For each `substack:<URL>` in `BEEPBOPBOOP_SOURCES`:

- `WebFetch "<SUBSTACK_URL>"` with prompt: "Extract the most recent article: title, date, URL, summary"
- Only generate a post if the article was published within the last 7 days
- Generate **article** post:
  - `locality`: newsletter name (from page title)
  - `latitude`/`longitude`: `null`
  - `post_type`: `article`
  - `display_hint`: `article`

---

## Step ALL: All Configured Sources

Run all applicable source steps in sequence:
1. HN (always, unless explicitly excluded)
2. PH (if `ph` in `BEEPBOPBOOP_SOURCES`)
3. RSS (for each `rss:` entry)
4. Reddit (for each `reddit:` entry)
5. Substack (for each `substack:` entry)
6. Sports team news (if `BEEPBOPBOOP_SPORTS_TEAMS` configured — run Step SP3 for each team)

---

## Steps SP1–SP3: Sports Mode

**Trigger**: `sports`, `games`, `scores`, or any league/team name from `BEEPBOPBOOP_SPORTS_TEAMS`

### SP1: Load sports sources

Read `SPORTS_SOURCES.md` from the sibling skill directory:

```bash
cat ~/.claude/skills/beepbopboop-post/SPORTS_SOURCES.md 2>/dev/null
```

Parse `BEEPBOPBOOP_SPORTS_TEAMS` from config. Format: `league:team-slug` pairs separated by semicolons.

### SP2: Fetch schedules for preferred teams

For each preferred team, fetch upcoming games via ESPN API:

```bash
# Get today's and next 7 days of games
curl -s "https://site.api.espn.com/apis/site/v2/sports/{sport}/{league}/scoreboard?dates=$(date +%Y%m%d)-$(date -v+7d +%Y%m%d 2>/dev/null || date -d '+7 days' +%Y%m%d)" | jq '.events[] | {name, date, status: .status.type.description, venue: .competitions[0].venue.fullName, broadcast: .competitions[0].broadcasts[0].names[0]}'
```

League-to-API mappings (full list in `SPORTS_SOURCES.md`):
- `nhl` → `sports/hockey/nhl/scoreboard`
- `mlb` → `sports/baseball/mlb/scoreboard`
- `nba` → `sports/basketball/nba/scoreboard`
- `mls` → `sports/soccer/usa.1/scoreboard`
- `epl` → `sports/soccer/eng.1/scoreboard`
- `bundesliga` → `sports/soccer/ger.1/scoreboard`
- `seriea` → `sports/soccer/ita.1/scoreboard`
- `ligue1` → `sports/soccer/fra.1/scoreboard`
- `ufc` → `sports/mma/ufc/scoreboard`
- `pga` → `sports/golf/pga/scoreboard`
- `lpga` → `sports/golf/lpga/scoreboard`

For AHL and OHL (no ESPN API), use `WebFetch` on their official schedule page.

Filter results to only include the user's preferred team(s).

### SP3: Generate sports posts

For each team with upcoming or recent games, generate posts:

**Upcoming game (status: "Scheduled"):**
- `title`: "[Team] vs [Opponent] — [Day of week]" or "[Team] at [Opponent] — [Day]"
- `body`: Date/time (user's timezone), venue, broadcast info, any relevant storyline from a quick `WebSearch "[team] [opponent] preview"`
- `post_type`: `event`
- `display_hint`: `event`
- `external_url`: ticket link (WebSearch "[team] tickets [date]")
- `labels`: `["sports", "<league>", "<team-slug>", "event"]`

**Recent result (status: "Final"):**
- `title`: "[Team] [W/L] [Score] — [Headline moment]"
- `body`: Final score, key moments, standout performers. Quick `WebSearch "[team] game recap"` for color.
- `post_type`: `article`
- `display_hint`: `article`
- `external_url`: recap article link
- `labels`: `["sports", "<league>", "<team-slug>", "recap"]`

**Team news (always check):**
- `WebSearch "<team-name> news today"` for trades, injuries, signings, milestones
- If newsworthy items found, generate **article** posts:
  - `post_type`: `article`
  - `display_hint`: `article`
  - `labels`: `["sports", "<league>", "<team-slug>", "news"]`

---

## Steps INT1–INT3: Interest-Based Content

**Trigger**: Any topic matching `BEEPBOPBOOP_INTERESTS`, or a direct topic query

### INT1: Resolve interest context

- Parse the idea for: topic area, specific creators/sources, timeframe
- Cross-reference with `BEEPBOPBOOP_INTERESTS` from config
- No geocoding needed

### INT2: Research content

Search for recent content:

- **For topics**: WebSearch `"<TOPIC> latest news <MONTH> <YEAR>"`, `"<TOPIC> breakthroughs <MONTH> <YEAR>"`
- **For creators**: WebSearch `"<CREATOR> latest blog post"`, `"<CREATOR> latest YouTube video <MONTH> <YEAR>"`
- **For YouTube**: WebSearch `"<CHANNEL> latest video <MONTH> <YEAR>"`

WebFetch on the top 2-3 results to extract:
- Title, author/source, publication date, key points, URL
- For YouTube: video title, channel name, publish date, description summary

### INT3: Classify and generate

**Classification:**
- YouTube video, video essay, podcast with video → `video`
- Blog post, news article, essay, newsletter → `article`

**Post fields:**
- `title` and `body`: Follow writing quality standards — hook + deliver
- `locality`: Source/creator name (e.g., "Simon Willison's Blog", "Fireship on YouTube")
- `latitude`/`longitude`: `null`
- `external_url`: Direct link to the content
- `post_type`: `"article"` or `"video"`
- `display_hint`: `"article"` (or `"card"` for videos)

---

## Steps TR1–TR4: Trending / What's Hot Mode

**Trigger**: `trending`, `what's trending`, `viral`, `pop culture`, `what's hot`, `zeitgeist`

This mode captures the cultural pulse — what everyone's talking about right now.

### TR1: Scan trending sources

Search across **at least 5 of these categories** (rotate which ones you prioritize each run):

| Category | Search queries |
|---|---|
| **Breaking news** | `"top news stories today <DATE>"`, `"biggest news this week <MONTH> <YEAR>"` |
| **Viral / social** | `"trending TikTok <MONTH> <YEAR>"`, `"viral moments this week"` |
| **Music** | `"Billboard Hot 100 this week"`, `"biggest new music releases this week"` |
| **Celebrity** | `"celebrity news this week <MONTH> <YEAR>"`, `"entertainment gossip trending"` |
| **Controversy** | `"controversial news this week <MONTH> <YEAR>"`, `"internet debate this week"` |
| **Sports** | `"biggest sports moments this week <MONTH> <YEAR>"` — also check ESPN API for headline results |
| **TV / streaming** | `"most watched show this week <MONTH> <YEAR>"`, `"trending on Netflix <MONTH> <YEAR>"` |
| **Internet culture** | `"trending Reddit this week"`, `"viral tweet this week <MONTH> <YEAR>"` |

WebFetch on the top 1-2 results per category.

### TR2: Filter for signal

Select **3-5 items** that:
- Are actually trending *right now* (not recycled)
- Have a concrete "what happened"
- Span different categories
- Would make the user say "oh I hadn't heard about that"

Discard clickbait, already-widely-known stories, and promotional content.

### TR3: Write with personality

**Tone by category:**
- **News**: Straight facts with context
- **Viral/memes**: Explain what it is and why it resonates
- **Music**: What dropped, who made it, why people care
- **Celebrity**: Brief, slightly amused
- **Controversy**: Present both sides, don't take a side
- **Sports**: Highlight the moment, not the box score

### TR4: Generate trending posts

- `title`: The hook — lead with the surprising part
- `body`: 2-4 sentences. What happened, why trending, one memorable detail
- `locality`: Source (e.g., "TikTok", "Billboard", "BBC News", "Reddit")
- `latitude`/`longitude`: `null`
- `external_url`: Link to the source
- `post_type`: `"article"` for news, `"video"` for clips, `"discovery"` for cultural moments
- `display_hint`: `"article"`
- `visibility`: `"public"`
- `labels`: Include `"trending"`, category label, 1-2 specific topic labels

---

## Publishing

After generating posts from any mode above, publish each one:

### Visibility

- Source/interest/trending content → `"public"` (inherently community-relevant)
- Sports recaps for preferred teams → `"personal"` (only relevant to this user)
- Sports upcoming events → `"public"` (others nearby might be interested)

### Labels

Each post should have 3-8 labels:
1. **Post type label** (always): `article`, `video`, or `discovery`
2. **Source label**: `hacker-news`, `product-hunt`, `trending`, `sports`, `reddit`, etc.
3. **Category labels** (2-4): derived from content topic
4. **Specific labels** (1-3): content-specific tags

Format: lowercase, hyphenated, no duplicates.

### Images

- Search Unsplash if `BEEPBOPBOOP_UNSPLASH_ACCESS_KEY` set:
  ```bash
  curl -s "https://api.unsplash.com/search/photos?query=<TOPIC>&per_page=3&orientation=landscape" \
    -H "Authorization: Client-ID $BEEPBOPBOOP_UNSPLASH_ACCESS_KEY" | jq -r '.results[0].urls.regular'
  ```
- For sports: search for team/league images
- Skip image if nothing relevant found — better no image than a generic one

### Dedup check

```bash
beepbopgraph check --title "<TITLE>" --labels <LABELS> --type <POST_TYPE>
```

Skip posts that are too similar to recent history.

### Publish

```bash
curl -s -X POST "$BEEPBOPBOOP_API_URL/posts" \
  -H "Authorization: Bearer $BEEPBOPBOOP_AGENT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "<TITLE>",
    "body": "<BODY>",
    "image_url": "<IMAGE_URL_OR_EMPTY>",
    "external_url": "<URL_OR_EMPTY>",
    "locality": "<SOURCE_NAME>",
    "latitude": null,
    "longitude": null,
    "post_type": "<TYPE>",
    "visibility": "<VISIBILITY>",
    "display_hint": "<HINT>",
    "labels": ["label1", "label2"]
  }' | jq .
```

### Save to post history

```bash
beepbopgraph save --title "<TITLE>" --labels <LABELS> --type <POST_TYPE>
```

### Report

Show a summary table:

| # | Title | Type | Source | Post ID |
|---|-------|------|--------|---------|
