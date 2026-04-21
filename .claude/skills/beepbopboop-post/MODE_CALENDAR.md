# Calendar mode (CL1–CL3)

**Trigger:** `calendar`, `my calendar`, `upcoming events from calendar`, or auto-included in batch mode.

**Requires** `BEEPBOPBOOP_CALENDAR_URL`. If not set, tell the user: "No calendar URL configured. Run `/beepbopboop-post init` to add one."

## CL1: Fetch and parse ICS

```bash
curl -s "<CALENDAR_URL>"
```

Parse `VEVENT` blocks. For each event extract:
- `SUMMARY` — event title
- `DTSTART` / `DTEND`
- `LOCATION`
- `DESCRIPTION`
- `URL`

**Date format handling:**
- `DTSTART;TZID=America/Los_Angeles:20260318T183000` → with timezone
- `DTSTART:20260318T183000Z` → UTC
- `DTSTART;VALUE=DATE:20260318` → all-day

Filter to the next 7 days:
```bash
date +%Y%m%d
```
Compare each event's `DTSTART` against today through today+7.

Max **5 events**. Skip events with complex recurrence rules (`RRULE`) — only single-instance and simple recurring.

## CL2: Research and enrich

For each upcoming event:

1. If the event has a `LOCATION`:
   - Geocode: `osm geocode "<LOCATION>" | jq '.[0] | {lat, lon, display_name}'`
   - Compute distance from `HOME_LAT`/`HOME_LON` if available
   - `WebSearch "<VENUE_NAME> <LOCATION>"` for details (parking, what to bring)

2. Research the event:
   - `WebSearch "<EVENT_NAME> <LOCATION> <DATE>"` for context, dress code, parking tips
   - If event has a `URL`, `WebFetch` it

3. Weather check for the event day:
   - `WebSearch "<DISPLAY_LOCATION> weather <EVENT_DATE>"`

## CL3: Generate calendar posts

For each event:

- **Post type:** `event`
- **Title:** Timing + actionable framing. Lead with when, not what. Examples:
  - "Team dinner at Il Terrazzo is Thursday at 6:30pm"
  - "Max's soccer practice moved to the indoor field Saturday morning"
  - "Victoria Tech Meetup is tomorrow at 6pm — there's still parking on Fisgard after 5"
- **Body:** Practical context a calendar alert wouldn't give you:
  - Travel time from home
  - Weather for that day
  - What to bring or prepare
  - Parking or transit tips
  - Family context when relevant
- **Tone:** Helpful friend reminder, not a notification.
- **locality:** Event location or venue name
- **latitude/longitude:** from geocoded location, or `null`
- **external_url:** event URL if available

Visibility is typically `private` (see `COMMON_PUBLISH.md` Step 4a).

Then proceed to `COMMON_PUBLISH.md`.
