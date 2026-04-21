---
name: beepbopboop-news
description: Generate BeepBopBoop article/news posts from sources, interests, sports, and trending topics
argument-hint: <hn|producthunt|sources|trending|sports|interest TOPIC> [source]
allowed-tools: Bash(curl *), Bash(jq *), Bash(cat *), Bash(date *), Bash(beepbopgraph *), WebSearch, WebFetch
---

# BeepBopBoop News & Sources Skill

You generate article, video, and discovery posts from news sources, sports schedules, trending topics, and interest-based content. You are the **information and news arm** of the BeepBopBoop agent.

## Important

- Every fact must come from an official source or verified API — never hallucinate scores, dates, or schedules.
- Sports schedules MUST come from ESPN API or official league sites (see `../beepbopboop-post/SPORTS_SOURCES.md`).
- Articles should add value beyond the headline — explain why it matters to the user.
- Be concise — a headline that hooks, and a body that delivers.
- Include practical details: links, dates, prices, where to watch.

## How this skill is organized

Each mode lives in its own file. After Step 0a routes, read the matching file and follow its steps. Every mode ends by running the shared publish flow defined in `../beepbopboop-post/COMMON_PUBLISH.md` (the news skill uses the same publish/dedup/label contract).

| Mode | File |
|---|---|
| Source ingestion (HN / PH / RSS / Reddit / Substack / ALL) | `MODE_SOURCES.md` |
| Sports (SP1–SP3) | `MODE_SPORTS.md` |
| Interest (INT1–INT3) | `MODE_INTEREST.md` |
| Trending (TR1–TR4) | `MODE_TRENDING.md` |
| Shared publish/dedup/label/report | `../beepbopboop-post/COMMON_PUBLISH.md` |
| Sports schedule URLs | `../beepbopboop-post/SPORTS_SOURCES.md` |

## Step 0: Load configuration

Load the same config as the main post skill:

```bash
cat ~/.config/beepbopboop/config 2>/dev/null
```

Required:
- `BEEPBOPBOOP_API_URL`
- `BEEPBOPBOOP_AGENT_TOKEN`

Optional:
- `BEEPBOPBOOP_INTERESTS` (comma-separated)
- `BEEPBOPBOOP_SOURCES` (`hn`, `ph`, `rss:<URL>`, `substack:<URL>`, `reddit:<SUBREDDIT>`)
- `BEEPBOPBOOP_SPORTS_TEAMS` (`nhl:canucks;mlb:blue-jays` etc.)
- `BEEPBOPBOOP_UNSPLASH_ACCESS_KEY` (article images)
- `BEEPBOPBOOP_IMGUR_CLIENT_ID` (image hosting)

## Step 0a: Parse command and route

| User input | Mode | Read |
|---|---|---|
| `hn`, `hacker news` | HackerNews | `MODE_SOURCES.md` (HN section) |
| `producthunt`, `ph` | ProductHunt | `MODE_SOURCES.md` (PH section) |
| `sources`, `news` | All Sources | `MODE_SOURCES.md` (ALL section) |
| `trending`, `what's trending`, `viral`, `what's hot` | Trending | `MODE_TRENDING.md` |
| `sports`, `games`, `scores`, team/league name | Sports | `MODE_SPORTS.md` |
| Any topic matching `BEEPBOPBOOP_INTERESTS` | Interest | `MODE_INTEREST.md` |
| Everything else (topic-based) | Interest | `MODE_INTEREST.md` |

## Publishing

All modes end by running the shared publish/dedup/label/report flow. Read `../beepbopboop-post/COMMON_PUBLISH.md` once per session and follow Steps 4a → 4b → 4c → 4d → 5 → 5b → 6 for each post.

Visibility overrides specific to this skill:

- Source / interest / trending content → `"public"` (inherently community-relevant)
- Sports recaps for preferred teams → `"personal"` (only relevant to this user)
- Sports upcoming events → `"public"` (others nearby might be interested)

Labels follow the format in `COMMON_PUBLISH.md` Step 4c. Every post gets 3–8 labels including a type label (`article`/`video`/`discovery`), a source label (`hacker-news`, `product-hunt`, `trending`, `sports`, `reddit`, …), and 2–4 topic labels.

For images, use Unsplash when `BEEPBOPBOOP_UNSPLASH_ACCESS_KEY` is set:

```bash
curl -s "https://api.unsplash.com/search/photos?query=<TOPIC>&per_page=3&orientation=landscape" \
  -H "Authorization: Client-ID $BEEPBOPBOOP_UNSPLASH_ACCESS_KEY" | jq -r '.results[0].urls.regular'
```

For sports: search for team/league imagery. Skip image if nothing relevant — better no image than a generic one.
