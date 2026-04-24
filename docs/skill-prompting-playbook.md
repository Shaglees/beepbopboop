# BeepBopBoop skill prompting playbook

This is not a scheduler. It's a reference for how to talk to an AI agent (Claude Code, openclaw, any Cursor agent) so the BeepBopBoop skills produce a balanced, on-strategy feed on a daily / weekly / monthly cadence.

All recipes assume:

- the config at `~/.config/beepbopboop/config` is populated (see `.claude/skills/_shared/CONFIG.md`)
- the backend is running and `/posts/hints`, `/posts/stats`, `/reactions/summary`, `/events/summary` are reachable with the agent token
- the agent has access to the `beepbopboop-post`, `beepbopboop-news`, `beepbopboop-fashion`, and `beepbopboop-images` skills

> **Why prompts matter:** every skill now opens by pulling `GET /posts/hints` and `GET /posts/stats` into context. Bad prompts skip the context step, skip the hint catalog, and produce unbalanced or invalid posts. The recipes below encode the right context signal.

---

## Daily prompts

Each of these is a single-turn ask designed to produce ~2–6 posts. Paste, let it run, review.

### Morning brief (weekday, before 9am local)

```
Use beepbopboop-post/brief. Before composing, pull /posts/stats and /reactions/summary; avoid any label in the top 3 over-represented this week, and skip anything under not_for_me. Produce one brief post for today. If sports are in-season, add a matchup post using /sports/scores for tonight's game.
```

### Evening rollup (end of day)

```
Use beepbopboop-news/trending. Pull trending + check /posts/stats — the goal is to cover a topic I haven't seen in 7 days. Cap at 2 posts. If the result would re-label with the same top-3 labels I already posted today, pivot to a different interest from BEEPBOPBOOP_INTERESTS.
```

### Local discovery (lunch slot, geographic)

```
Use beepbopboop-post with the default local flow for "new this week near <LOCALITY>". Start with /posts/hints and /posts/stats. Target 1 place post + 1 event post. Every post goes through beepbopboop-images; no bare gradient placeholders.
```

### Fashion daily

```
Use beepbopboop-fashion/outfit for today's context (check weather). Before rendering, confirm the image_role enum from /posts/hints and use Flex.1 or Nanobanana per BEEPBOPBOOP_FASHION_IMGGEN. Publish 1 outfit post.
```

---

## Weekly prompts

Saturday or Sunday morning is the usual slot. One session replaces a week's worth of curation.

### Weekly feed balance

```
Use beepbopboop-post/batch for a full week plan. Hard requirements:
  - pull /posts/stats (30-day window); pick 8–12 posts that shift distribution toward underweight labels
  - pull /reactions/summary; exclude not_for_me labels, prefer more labels
  - mix post_types: at least 2 event, 2 place, 2 article, 1 discovery
  - include at least 1 video_embed and 1 scoreboard/matchup if sports are in-season
  - every post runs through /posts/lint before /posts
Report the before/after label distribution at the end.
```

### Weekly news digest

```
Use beepbopboop-news/digest for the past 7 days of BEEPBOPBOOP_SOURCES. Pick at most 5 items. For each, add why-this-matters in the body. Cross-check with /reactions/summary — if a source has been marked stale twice, rotate it out this week.
```

### Weekly kids/family plan

```
Use beepbopboop-kids-activities. Scope: next 7 days within 30 minutes of home. Check for Pro-D days / school closures. Output as calendar-hint posts (visibility=private or personal per FAMILY_CONTEXT rules). Goal: 3–6 calendar posts covering the week.
```

---

## Monthly prompts

Run at the start of the month. These populate calendar + long-horizon discovery.

### Monthly calendar seed

```
Use beepbopboop-post/calendar. Scope: the whole upcoming month. Pull from BEEPBOPBOOP_CALENDAR_URL + local event APIs. Target 10–20 calendar-hint posts. Every post is visibility=private or personal.
```

### Monthly trend audit

```
Open /posts/stats (90-day window) and compare label distribution to BEEPBOPBOOP_INTERESTS. Report:
  - labels in my interests that are under-posted (< 3 in 90 days)
  - labels over-posted that I have NOT marked "more" on
  - labels I have marked not_for_me but still appear in history
Then propose a 10-post beepbopboop-post/batch that closes the gaps. Do not publish until I confirm.
```

### Monthly sport season check

```
For each team in BEEPBOPBOOP_SPORTS_TEAMS, pull /sports/scores for the next 14 days and /posts/stats for sports labels. If a team has a home game in that window and the matchup label is under-posted, add it. Output the plan; I'll approve before batch.
```

---

## Continuous / "site-aware" prompts

These pull from the site itself, not external sources.

### "Why did you show me this?"

```
For my most recent 5 posts, fetch /events/summary and /reactions/summary. Explain which posts actually got dwell/saves vs ignored. Propose 2 changes to my batch schedule based on that. Do not publish.
```

### "Catch up on my feed"

```
Using beepbopboop-news/interest with topic=<X>, build a 1-post recap of what happened in the last 72 hours. Cite sources. No hallucinated facts.
```

---

## Patterns every prompt should follow

Copy these phrasings to avoid common misses:

| Do | Why |
|---|---|
| "Before composing, pull /posts/hints and /posts/stats." | Forces the context-bootstrap step — otherwise the agent may skip it. |
| "Lint every payload via /posts/lint before /posts." | Dry-run catches schema drift before writes. |
| "Every post runs through the beepbopboop-images subskill." | Ensures image pipeline isn't silently skipped. |
| "Exclude labels in /reactions/summary where not_for_me > 0." | Respect negative feedback. |
| "Report the before/after distribution." | Closes the feedback loop and makes the run auditable. |
| "If /posts/hints is unreachable, abort with a diagnostic, don't guess." | Keeps drift fatal during incidents. |

---

## Anti-patterns

Avoid these — they produce the problems this playbook exists to fix:

- "Just publish 10 posts about X." → no spread awareness, no hint discovery, no feedback respect.
- "Skip the lint step." → creates invalid posts; validators will reject.
- "Inline the Unsplash curl yourself." → bypasses the image pipeline and misses AI / Wikimedia / Google Places tiers.
- Hard-coded `display_hint` values without consulting `/posts/hints`. → breaks on any hint addition.

---

## Observability loop

After any run, the final report should include:

- How many posts were proposed vs published
- Which lints failed and why
- Which dedup hits were observed
- Which tier of the image pipeline produced each `image_url`
- The label distribution delta vs the start of the run

Keep this surface in muscle memory — it's the human-readable complement to `/posts/stats`.
