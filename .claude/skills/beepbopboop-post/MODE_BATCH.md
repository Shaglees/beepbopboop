# Batch Orchestration mode (BT1–BT9)

**Trigger:** `batch`, `my weekly feed`, `fill my feed`, `generate feed`.

Batch mode composes multiple modes into one diverse feed. It's the most complex flow — read the whole file before executing.

## BT1: Load schedule

Today's day of the week:

```bash
date +%A | tr '[:upper:]' '[:lower:]'
```

If `BEEPBOPBOOP_SCHEDULE` is configured, parse it into rules. Format: pipe-separated triplets `DAY|MODE|ARGS`. Match today against each rule:
- Exact day name match
- `daily` matches every day
- `weekday` matches Monday–Friday
- `weekend` matches Saturday–Sunday

Collect matching rules into "today's agenda."

## BT1b: Check engagement stats

```bash
curl -s -H "Authorization: Bearer $BEEPBOPBOOP_AGENT_TOKEN" "$BEEPBOPBOOP_API_URL/events/summary" | jq .
```

If `total_events > 0`, use as **soft guidance**:
- High save-rate labels (saves/views > 0.3): generate more with these labels
- High dwell-time types: favor these post types in the mix
- Low-engagement labels: reduce. If you include one, angle MUST differ from your last 5 posts with this label
- Guidance, not a hard constraint.

Empty data / errors: skip silently.

## BT1b2: Check user reactions

```bash
curl -s -H "Authorization: Bearer $BEEPBOPBOOP_AGENT_TOKEN" "$BEEPBOPBOOP_API_URL/reactions/summary" | jq .
```

30-day aggregated reaction counts by label and post type. Reactions are **explicit signals** — they carry more weight than implicit engagement.

- `more`: prioritize this label/type
- `less`: reduce unless you have a compelling new angle
- `stale`: find fresh perspectives / new sources / different angles. Don't just reduce — innovate.
- `not_for_me`: strongly avoid.

**Reactions override engagement** when they conflict. Mixed signals:
- High `more` + low engagement → user likes the idea, execution needs work.
- Low `more` + high engagement → fine for browsing, not noteworthy.
- `stale` on a popular label → vary the angle, don't drop the topic.
- `not_for_me` is strongest = almost never generate.

Empty data / errors: skip silently.

## BT1c: Check posting history

```bash
curl -s -H "Authorization: Bearer $BEEPBOPBOOP_AGENT_TOKEN" "$BEEPBOPBOOP_API_URL/posts/stats" | jq .
```

7/30/90-day stats with post counts by type (`last_days_ago` shows recency) and top labels.

- **Type cadence:** if a type hasn't appeared in 5+ days, strongly prefer including it.
- **Hint cadence:** same — if `outfit`, `scoreboard`, `matchup`, or `standings` hints haven't appeared in 3+ days, strongly prefer them.
- **Type saturation:** if a type is >40% of 7-day posts, reduce. If you include one, angle MUST differ from last 5.
- **Label diversity:** if top 3 labels > 60% of 30-day posts, MUST include at least 1 post with a label unused in 14+ days.
- **Volume tracking:** compare `avg_per_day` against `BATCH_MIN`. If consistently under target, boost today's count.

Especially important for "every so often" modes (comparison, deals, seasonal, discovery).

Empty data / errors: skip silently.

## BT2: Set target post count

Random integer between `BATCH_MIN` and `BATCH_MAX` (defaults: 8 and 15).

## BT2.5: Load spread targets

Fetch the user's content-mix preferences:

```bash
SPREAD=$(curl -s -H "$AUTH" "$API/settings/spread")
```

If the endpoint returns an error or is unavailable, fall back to even distribution across all verticals.

Use the `targets` map to allocate BT2's post count across verticals:

1. **Omega** vertical (`echo "$SPREAD" | jq -r '.omega'`) always gets at least 1 slot.
2. Remaining slots are distributed proportionally by weight: `slots[v] = round(weight[v] * (total - 1))`.
3. If a vertical's weight is 0, skip it entirely.
4. Validate: if the sum of allocated slots exceeds the total, trim the lowest-weight verticals first.

When building the content plan in BT3, use these per-vertical slot counts instead of the hardcoded category defaults. The `status` field shows which verticals are under/over-represented — prioritize `below_target` verticals when filling remaining slots.

## BT3: Build content plan

**Phase 1 — Scheduled content.** Execute each matching schedule rule from BT1. Map schedule modes:

| Schedule mode | Route to |
|---|---|
| `interest` | Interest mode → delegate to `beepbopboop-news` |
| `local` | `BASE_LOCAL.md` flow with ARGS as the idea |
| `weather` | `MODE_WEATHER.md` |
| `source` | Delegate to `beepbopboop-news` with ARGS specifying the source |
| `seasonal` | `MODE_SEASONAL.md` |
| `deals` | `MODE_DEALS.md` |
| `compare` | `MODE_COMPARISON.md` with ARGS as subject |
| `calendar` | `MODE_CALENDAR.md` |
| `discover` | `MODE_DISCOVERY.md` |
| `trending` | Delegate to `beepbopboop-news` with `trending` |
| `digest` | `MODE_DIGEST.md` |
| `brief` | `MODE_BRIEF.md` |
| `sports` | Delegate to `beepbopboop-news` with `sports` |
| `fashion` | Delegate to `beepbopboop-fashion` with ARGS (default: rotating focus) |

**Phase 2 — Fill with defaults** (if post count still under target):

1. Always: weather brief → 1 brief post (via `MODE_BRIEF.md` adapted with weather focus).
2. Weather editorial → 0–1 article post (only if a genuinely interesting story).
3. Always: digest → 1 digest post (via `MODE_DIGEST.md`).
4. Always: local mode with idea "events this week" → 2–4 posts.
5. If `BEEPBOPBOOP_SPORTS_TEAMS` configured: **MUST delegate to `beepbopboop-news` with `sports`** → 1–3 posts. Not optional. Expect scoreboard/matchup/standings hints.
6. Always: **delegate to `beepbopboop-fashion`** → 1–2 posts. Rotate focus among trend/outfit/seasonal based on recency. Expect outfit hints.
7. If `BEEPBOPBOOP_INTERESTS` configured: pick 1–2 interests → delegate to `beepbopboop-news` → 2–4 posts.
8. If `BEEPBOPBOOP_SOURCES` configured: pick 1–2 sources → delegate to `beepbopboop-news` → 1–3 posts.
9. If `BEEPBOPBOOP_CALENDAR_URL` configured: calendar mode → 1–3 posts.
10. If seasonal month is notable (Dec, Mar, Jun, Sep, Oct): seasonal mode → 1 post.
11. Always: interest discovery mode → 1–2 posts.
12. Always: delegate to `beepbopboop-news` with `trending` → 2–3 posts.
13. Occasionally: comparison mode → 1 post (roughly 30% of runs).
14. Occasionally: deal mode → 1 post (roughly 20% of runs).

**Phase 3 — Trim** if total > `BATCH_MAX`. Drop least essential (deals, comparison, extra local/interest duplicates first). **Preserve at least 1 sports post (if teams configured) and 1 fashion post** — these are high-value specialty content; trim generic fills before cutting them.

## BT4: Execute scheduled content

Run each Phase 1 rule. After each mode: "Generated N posts from [mode] (running_total/target)".

## BT5: Execute default fill

Run Phase 2 modes as needed to reach target. Report progress after each.

## BT6: Deduplicate

Run `COMMON_PUBLISH.md` Step 4d (beepbopgraph dedup) across the entire batch. Additionally remove:
- Duplicate venues within this batch (same name + same coords)
- Duplicate articles within this batch (same URL or title)
- Keep the richer version.

## BT7: Diversity check

Verify the final set:
- At least 2 different `post_type` values
- At least 1 local (with coords) and 1 non-local (without coords)
- No more than 3 consecutive same-type posts — reorder if needed
- If `BEEPBOPBOOP_SPORTS_TEAMS` configured: at least 1 sports-hint post (scoreboard/matchup/standings)
- At least 1 fashion-hint post (outfit)

If any check fails, reorder or swap. For missing sports/fashion posts, delegate to the appropriate sibling skill.

## BT7.5: Diversity scorecard

Before publishing, produce a scorecard:

```
DIVERSITY SCORECARD
- Types: 3/5 used (place, article, discovery) ✓
- Hints: 5/14 used (place, article, brief, digest, outfit) ✓
- Labels: 8 unique, top 3 = 42% ✓
- Local vs non-local: 5/8 (63%) ✓
- Consecutive same-type: max 2 ✓
- Weather posts: 1 (brief) ✓
- Sports posts: 2 (matchup, standings) ✓
- Fashion posts: 1 (outfit) ✓
```

**Flag thresholds — fix the batch before publishing if any fail:**
- Type count < 2 → swap a post for a different type
- Hint count < 3 → use digest/brief/place for suitable posts
- Top 3 labels > 60% → swap one post for an unexplored label
- Consecutive same-type > 3 → reorder
- Weather posts > 2 → cut to 1 brief + 0–1 editorial
- Sports posts = 0 (with teams configured) → must delegate to `beepbopboop-news sports`
- Fashion posts = 0 → must delegate to `beepbopboop-fashion`

Print the scorecard before BT8.

## BT8: Publish all posts

Run `COMMON_PUBLISH.md` Step 5 for each post. Publish in parallel where possible.

## BT9: Report with mode attribution

`COMMON_PUBLISH.md` Step 6 with extra `Vis`, `Labels`, and `Source` columns:

| # | Title | Type | Vis | Labels | Source | Post ID |
|---|-------|------|-----|--------|--------|---------|
| 1 | It's 19°C and clear — three patios open today | place | public | place, patio, outdoor, sunny-day | weather | abc123 |
| 2 | Claude 4 scores 94% on ARC-AGI | article | public | article, ai, machine-learning, research | HN | def456 |
| 3 | Royals host three games this week | event | public | event, hockey, sports, live-events | local | ghi789 |
