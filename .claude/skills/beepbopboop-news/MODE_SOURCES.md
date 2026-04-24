# Source ingestion modes

Covers HN, ProductHunt, RSS, Reddit, Substack, and the ALL aggregator.

After any source mode generates candidate posts, proceed to `COMMON_PUBLISH.md` (shared with `beepbopboop-post`).

---

## HN: HackerNews

Fetch top stories:

```bash
curl -s "https://hacker-news.firebaseio.com/v0/topstories.json" | jq '.[0:30]'
```

For each of the top 30 story IDs:

```bash
curl -s "https://hacker-news.firebaseio.com/v0/item/<ID>.json" | jq '{title, url, score, by}'
```

- Filter by matching title against `BEEPBOPBOOP_INTERESTS` (case-insensitive substring).
- If no interests configured, take the top 3 by score.
- Otherwise take the top 2–3 interest-matching stories by score.
- `WebFetch` each story URL for a summary.
- Generate **article** posts:
  - `locality`: `"Hacker News"`
  - `latitude`/`longitude`: `null`
  - `external_url`: story URL
  - `post_type`: `article`
  - `display_hint`: `article`

---

## PH: ProductHunt

- `WebFetch "https://www.producthunt.com"` with prompt: *"Extract today's top 5 product launches: name, tagline, URL, vote count."*
- Filter by `BEEPBOPBOOP_INTERESTS` if configured.
- Take the top 1–2 matching launches.
- `WebFetch` each product page for more details.
- Generate **article** posts:
  - `locality`: `"Product Hunt"`
  - `post_type`: `article`
  - `display_hint`: `article`

---

## RSS: RSS Feeds

For each `rss:<URL>` in `BEEPBOPBOOP_SOURCES`:

- `WebFetch "<RSS_URL>"` with prompt: *"Extract the 5 most recent items: title, link, date, description."*
- Take the 2–3 most recent items.
- `WebFetch` each item URL for full content summary.
- Generate **article** posts:
  - `locality`: feed name (from RSS `<title>` or domain fallback)
  - `post_type`: `article`
  - `display_hint`: `article`

---

## RED: Reddit

For each `reddit:<SUBREDDIT>` in `BEEPBOPBOOP_SOURCES`:

- `WebSearch "site:reddit.com/r/<SUBREDDIT> top today"` or `WebFetch "https://www.reddit.com/r/<SUBREDDIT>/top/?t=day"`
- Extract: top thread titles, URLs, and a summary of fan consensus / top comments.
- If sports-related (team subreddit):
  - Read `../beepbopboop-post/SPORTS_SOURCES.md` for the league's ESPN API endpoint.
  - Verify scores/schedule via ESPN API (see `MODE_SPORTS.md`).
  - Combine the official score with the Reddit "vibe check."
- Generate **article** or **discovery** posts:
  - `locality`: `"r/<SUBREDDIT>"`
  - `post_type`: `article`
  - `display_hint`: `article`

---

## SUB: Substack / Newsletters

For each `substack:<URL>` in `BEEPBOPBOOP_SOURCES`:

- `WebFetch "<SUBSTACK_URL>"` with prompt: *"Extract the most recent article: title, date, URL, summary."*
- Only generate a post if the article was published within the last 7 days.
- Generate **article** post:
  - `locality`: newsletter name (from page title)
  - `post_type`: `article`
  - `display_hint`: `article`

---

## ALL: All Configured Sources

Run all applicable source steps in sequence:

1. HN (always, unless explicitly excluded)
2. PH (if `ph` in `BEEPBOPBOOP_SOURCES`)
3. RSS (for each `rss:` entry)
4. Reddit (for each `reddit:` entry)
5. Substack (for each `substack:` entry)
6. Sports team news (if `BEEPBOPBOOP_SPORTS_TEAMS` configured — run `MODE_SPORTS.md` SP3 for each team)
