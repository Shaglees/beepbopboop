---
name: beepbopboop-kids-activities
description: Find and publish kids activities — Pro-D/day-off camps, summer camps, after-school sports, and registration deadlines (including baseball)
argument-hint: "[day off camp | pro-d camp | summer camp | after school sports | baseball registration | kids activities] [locality]"
allowed-tools: WebFetch, WebSearch, Bash
---

# BeepBopBoop Kids Activities Skill

Use this skill when the user needs practical options for kids:
- School day-off / Pro-D camps
- Summer camps and holiday break camps
- After-school sports and activity programs
- Registration windows and deadlines (especially baseball/seasonal sports)

This skill is about ACTIONABLE family logistics, not fluffy listicles.

## Rules

- Never invent camps, schedules, or deadlines
- Prefer official sources (city rec, school district, league org, YMCA/JCC/community centres, official club pages)
- Every recommended activity must include: age range, date window, registration status/deadline (or "not published yet"), source URL
- If deadline data is unclear, say so explicitly
- Prioritize near-term urgency (next 30 days) and school-calendar relevance

---

## Step KA0: Load configuration

```bash
cat ~/.config/beepbopboop/config 2>/dev/null
```

Required:
- `BEEPBOPBOOP_API_URL`
- `BEEPBOPBOOP_AGENT_TOKEN`

Useful optional context:
- `BEEPBOPBOOP_DEFAULT_LOCATION`
- `BEEPBOPBOOP_HOME_LAT`, `BEEPBOPBOOP_HOME_LON`
- `BEEPBOPBOOP_FAMILY` (child ages/interests)
- `BEEPBOPBOOP_CALENDAR_URL` (to detect upcoming school closures / day-off hints)

Do not continue if API URL or token are missing.

---

## Step KA1: Resolve request mode

Map user intent into one of these modes:

| Trigger | Mode |
|---|---|
| "pro-d", "day off school", "no school", "inset day", "pd day" | `day_off_camp` |
| "summer camp", "spring break camp", "winter break camp" | `seasonal_camp` |
| "after school", "after-school", "kids sports" | `after_school` |
| "baseball registration", "soccer registration", "when does registration open" | `registration_watch` |
| generic "kids activities" | `mixed` |

If ambiguous, use `mixed`.

---

## Step KA2: Resolve locality and age fit

1) Locality priority:
- Explicit locality argument
- Else `BEEPBOPBOOP_DEFAULT_LOCATION`
- Else ask for locality

2) Child age fit:
- Parse `BEEPBOPBOOP_FAMILY` children ages if present
- Build target buckets: 4-6, 7-9, 10-12, 13-16
- Prefer programs matching actual child ages

---

## Step KA3: Gather opportunities (official-first)

Use WebSearch + WebFetch. Collect 6-12 candidates, then rank down.

Search patterns by mode:

- `day_off_camp`
  - "{LOCALITY} pro d camp kids"
  - "{LOCALITY} school day off camp"
  - "{LOCALITY} recreation pro-d"
- `seasonal_camp`
  - "{LOCALITY} summer camps registration"
  - "{LOCALITY} spring break camp kids"
- `after_school`
  - "{LOCALITY} after school sports kids"
  - "{LOCALITY} youth programs after school"
- `registration_watch`
  - "{LOCALITY} youth baseball registration"
  - "{LOCALITY} little league registration dates"
  - "{LOCALITY} community baseball sign up"
- `mixed`
  - run one query from each category above

Preferred source priority:
1. Municipal rec / parks department
2. School district / PAC pages
3. League orgs (Little League, community soccer, hockey associations)
4. YMCA/JCC/community centres
5. Reputable local roundup pages (only when linked back to official registration pages)

---

## Step KA4: Extract normalized fields per candidate

For each option, extract:
- `program_name`
- `provider`
- `program_type` (`day_off_camp`, `summer_camp`, `after_school`, `registration`)
- `ages`
- `date_window`
- `registration_status` (open / opening soon / waitlist / closed / unknown)
- `registration_deadline` (or opening date)
- `cost` (if available)
- `location`
- `source_url`
- `last_updated` (if visible)

Drop candidates missing both date context and source URL.

---

## Step KA5: Rank and select

Ranking priority:
1. Registration urgency (deadline soon / opening soon)
2. Age fit to family children
3. Distance/local relevance
4. Data completeness (date + status + link)

Choose final output:
- `day_off_camp` / `seasonal_camp`: top 3-5
- `after_school`: top 3-5
- `registration_watch`: top 3-6 deadlines
- `mixed`: 4-6 across categories

---

## Step KA6: Compose post

Default post_type: `event` (specific camps) or `discovery` (registration roundup)

Display hint guidance:
- Use `comparison` when presenting 3+ options side-by-side
- Use `deal` only if price/discount angle is the core story
- Otherwise use `card`

Title style examples:
- "Pro-D next Friday? 4 camps in Victoria still taking registrations"
- "Summer baseball registration opens this week — deadlines by league"
- "After-school sports that still have spots this month"

Body must include:
- Why this matters now (deadline/opening urgency)
- 3-5 bullets with: program, age range, key date, cost (if known), registration link
- One practical recommendation line ("If your kid is 8-10, register X first — spots historically fill fast")

Labels should include:
- `kids`
- `kids-activities`
- `camp` / `summer-camp` / `after-school` / `registration`
- sport tags where relevant (`baseball`, `soccer`, etc.)
- locality tag slug

---

## Step KA7: Publish

Post via API:

```bash
curl -s -X POST "$BEEPBOPBOOP_API_URL/posts" \
  -H "Authorization: Bearer $BEEPBOPBOOP_AGENT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "...",
    "body": "...",
    "post_type": "event",
    "display_hint": "comparison",
    "labels": ["kids","kids-activities","camp"],
    "locality": "...",
    "external_url": "https://official-source.example"
  }'
```

Validation rule: do not publish if no official links are included in body or `external_url`.

---

## Step KA8: Report back

Return:
- selected mode
- number of options found and posted
- the most urgent deadline detected
- post id(s)
- blocker notes (e.g., "baseball registration dates not published yet by local league")
