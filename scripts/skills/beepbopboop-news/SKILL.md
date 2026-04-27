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
| Load config (cross-skill) | `../_shared/CONFIG.md` |
| Bootstrap server context (cross-skill) | `../_shared/CONTEXT_BOOTSTRAP.md` |
| Image pipeline quick reference | `../_shared/IMAGES.md` |
| Publish envelope (lint → dedup → POST) | `../_shared/PUBLISH_ENVELOPE.md` |
| Full image pipeline (invokable subskill) | `../beepbopboop-images/SKILL.md` |

## Step 0: Load configuration

Read `../_shared/CONFIG.md` and load the config file. The news skill relies on `BEEPBOPBOOP_INTERESTS`, `BEEPBOPBOOP_SOURCES`, and `BEEPBOPBOOP_SPORTS_TEAMS` in addition to the universal required keys.

## Step 0d: Bootstrap server context

Read `../_shared/CONTEXT_BOOTSTRAP.md` and run the four parallel fetches. The `/posts/stats` response is especially important for news mode — use it to avoid re-posting the same labels the user already saw this week. The `/reactions/summary` response tells you which topics the user has explicitly muted.

## Step 0e: Image pipeline awareness

Read `../_shared/IMAGES.md`. Article and sports posts should still have hero images — do not skip this step.

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

For images, invoke the `beepbopboop-images` subskill (see `../_shared/IMAGES.md`) with `mode=auto` and the post's topic/keywords. It handles Unsplash, AI fallback, poster rehost, etc. — never call Unsplash directly from a news mode, or the pipeline stays "invisible" and easy to skip.

For sports: pass team / league identifiers as keywords. If nothing relevant is found, prefer an empty `image_url` over a generic one.
