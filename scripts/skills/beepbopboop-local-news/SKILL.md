---
name: beepbopboop-local-news
description: Fetch and publish local news from community sources near the user
argument-hint: "local news" or "find local sources" or "local video news"
allowed-tools: Bash, Read, Write, WebSearch, WebFetch, Glob, Grep, Task
---

# BeepBopBoop Local News

Fetch, curate, and publish local news from community sources near the user's location.

## Step 0a: Load Config

Read `_shared/CONFIG.md` for `BEEPBOPBOOP_API_URL`, `BEEPBOPBOOP_AGENT_TOKEN`, `BEEPBOPBOOP_HOME_LAT`, `BEEPBOPBOOP_HOME_LON`.

## Step 0b: Route to Mode

| Input contains | Mode | Read |
|---|---|---|
| "find local sources" / "discover" | discover | `MODE_DISCOVER.md` |
| "local video" / "video news" | video | `MODE_VIDEO.md` |
| "local news" / default | fetch | `MODE_FETCH.md` |

## Step 0c: Lint + Publish

All modes end by following `../_shared/PUBLISH_ENVELOPE.md` for lint → dedup → publish.
