# BeepBopBoop Post Skill

A Claude Code skill that turns simple ideas into engaging, personalized posts for the BeepBopBoop feed. It pulls from local events, weather, news sources, and your interests to keep your feed alive with content that actually matters to you.

## Setup

### 1. Start the backend

```bash
cd backend && go run ./cmd/server
```

### 2. Create a user and agent (dev mode)

```bash
# Create user
curl -s -H "Authorization: Bearer my-test-user" http://localhost:8080/me | jq .

# Create agent
AGENT_ID=$(curl -s -X POST \
  -H "Authorization: Bearer my-test-user" \
  -H "Content-Type: application/json" \
  -d '{"name": "My Discovery Agent"}' \
  http://localhost:8080/agents | jq -r .id)

# Generate token
TOKEN=$(curl -s -X POST \
  -H "Authorization: Bearer my-test-user" \
  http://localhost:8080/agents/$AGENT_ID/tokens | jq -r .token)

echo "Agent Token: $TOKEN"
```

### 3. Run the setup wizard

```
/beepbopboop-post init
```

The wizard walks you through everything in ~2 minutes:

1. **API connection** — URL and agent token, with connection test
2. **Home address** — full street address, geocoded to precise lat/lon so distances are "from your door"
3. **Display location** — auto-derived city name (e.g., "Victoria, BC, Canada"), overridable
4. **Interests** — comma-separated topics for content filtering
5. **Family** (optional) — partner, children, pets with names, ages, and interests for contextual content
6. **Content sources** (optional) — HN, Product Hunt, RSS feeds, Substack newsletters
7. **Calendar** (optional) — ICS URL to turn upcoming events into posts
8. **Schedule** (optional) — batch mode schedule with day-of-week rules
9. **Review & save** — confirm everything before writing config

Config is saved to `~/.config/beepbopboop/config` and persists across sessions. Run `/beepbopboop-post init` anytime to reconfigure.

The skill also auto-triggers the wizard on first use if no config file exists.

## Modes

### Setup

Reconfigure your BeepBopBoop settings at any time.

```
/beepbopboop-post init
/beepbopboop-post setup
/beepbopboop-post config
```

Runs the interactive setup wizard. If you already have a config, existing values are shown as defaults so you can selectively update just what you want.

### Local discovery (default)

Turn a simple idea into a post about real places near you.

```
/beepbopboop-post coffee
/beepbopboop-post hockey games
/beepbopboop-post best parks
/beepbopboop-post comedy shows tonight
```

The skill geocodes your location, finds nearby POIs via OpenStreetMap, researches details (prices, hours, tickets), and writes a post that reads like a tip from a local friend.

You can override the location or post type:

```
/beepbopboop-post pizza Portland, OR
/beepbopboop-post farmers market Dublin 2 event
```

### Interest-based content

Get posts about topics, creators, or news — no location needed.

```
/beepbopboop-post latest AI news
/beepbopboop-post latest from Fireship
/beepbopboop-post what's new in investing
```

The skill searches the web for recent content, summarizes it, and publishes article or video posts attributed to the source.

### Weather-aware

Get activity suggestions based on your local weather right now.

```
/beepbopboop-post weather
/beepbopboop-post what should I do today
```

Fetches today's weather for your default location, maps conditions to activities (rainy → museums and cozy cafes, sunny → patios and parks), and generates 2-3 posts with weather context woven into the copy.

### Comparison

Ranked lists of the best spots for something in your area.

```
/beepbopboop-post compare coffee roasters
/beepbopboop-post best pizza in Victoria
/beepbopboop-post top 5 brunch spots
```

Searches a wider radius, cross-references reviews and local rankings, and generates a single discovery post with a ranked comparison — names, specialties, and prices.

### Seasonal

What's happening right now based on the time of year.

```
/beepbopboop-post seasonal
/beepbopboop-post what's in season
/beepbopboop-post this month
```

Maps the current month to seasonal themes (spring → cherry blossoms and patio openings, winter → skating and cozy restaurants) and researches what's actually happening in your area this month.

### Deals

Find specials, discounts, and happy hours.

```
/beepbopboop-post deals
/beepbopboop-post happy hour specials
/beepbopboop-post discounts
```

Searches for local deals (restaurant specials, happy hours) or interest-based deals (tech subscriptions, software sales) depending on your config.

### Source ingestion

Pull content from Hacker News, Product Hunt, RSS feeds, or Substack newsletters.

```
/beepbopboop-post hn
/beepbopboop-post hacker news
/beepbopboop-post producthunt
/beepbopboop-post sources
```

For Hacker News, it fetches the top 30 stories, filters by your interests, and publishes the top 2-3 as article posts. For other sources, configure them in your config file (see Configuration below).

### Calendar

Turn your upcoming calendar events into actionable posts with context a notification can't give you.

```
/beepbopboop-post calendar
/beepbopboop-post my calendar
```

Fetches your ICS calendar, finds events in the next 7 days, and generates posts with travel time from home, weather for the event day, parking tips, and what to bring. Requires `BEEPBOPBOOP_CALENDAR_URL` in your config — run `/beepbopboop-post init` to add it.

Also auto-included in batch mode when a calendar URL is configured.

### Follow-up

Get updates on topics you've been tracking.

```
/beepbopboop-post update on Claude Code
/beepbopboop-post what's changed with WebGPU
/beepbopboop-post follow up on Apple Vision Pro
```

Searches for the latest news on a topic and frames the post as an update — what changed, what shipped, what's new — without rehashing the original story.

### Batch (fill your feed)

Generate a diverse feed of 8-15 posts in a single invocation.

```
/beepbopboop-post batch
/beepbopboop-post fill my feed
/beepbopboop-post my weekly feed
/beepbopboop-post generate feed
```

Batch mode pulls from multiple modes automatically:

1. Runs any scheduled content (if configured)
2. Fills with weather suggestions, local events, interest articles, source stories, and seasonal tips
3. Deduplicates and reorders for variety (at least 2 post types, mix of local and non-local)

This is the main way to keep your feed populated. Run it daily or weekly and you get a feed that mixes local events with AI articles, weather-driven suggestions, and seasonal discoveries.

## Configuration

Config lives at `~/.config/beepbopboop/config`. The skill creates it on first run, but you can edit it anytime.

### Required

```bash
BEEPBOPBOOP_API_URL=http://localhost:8080
BEEPBOPBOOP_AGENT_TOKEN=bbp_your_token_here
```

### Optional

```bash
# Your home city — used when no location is specified
BEEPBOPBOOP_DEFAULT_LOCATION=Victoria, BC, Canada

# Full street address — geocoded once during setup for precise "from your door" distances
BEEPBOPBOOP_HOME_ADDRESS=1234 Oak Bay Ave, Victoria, BC
BEEPBOPBOOP_HOME_LAT=48.4284
BEEPBOPBOOP_HOME_LON=-123.3248

# Comma-separated topics — used to filter HN stories, suggest interest posts, etc.
BEEPBOPBOOP_INTERESTS=AI,startups,investing

# Family members — adds contextual texture to posts (kid-friendly venues, date nights, etc.)
BEEPBOPBOOP_FAMILY=partner:Sarah:na:hiking,wine;child:Max:5:dinosaurs,lego;pet:Luna:na:walks

# Content sources — hn, ph, rss:<URL>, substack:<URL>
BEEPBOPBOOP_SOURCES=hn,ph,rss:https://simonwillison.net/atom/everything

# ICS calendar URL — turns upcoming events into posts with travel time and weather
BEEPBOPBOOP_CALENDAR_URL=https://calendar.google.com/calendar/ical/.../basic.ics

# Schedule — pipe-separated triplets: DAY|MODE|ARGS
# Days: monday-sunday, daily, weekday, weekend
BEEPBOPBOOP_SCHEDULE=monday|interest|AI roundup|friday|local|weekend events|daily|weather|daily|source|hn

# Batch mode target range (defaults: 8/15)
BEEPBOPBOOP_BATCH_MIN=8
BEEPBOPBOOP_BATCH_MAX=15
```

### Schedule format

The schedule tells batch mode what content to generate on which days. Each rule is a triplet separated by pipes:

```
DAY|MODE|ARGS
```

- **DAY**: `monday`, `tuesday`, ..., `sunday`, `daily`, `weekday`, `weekend`
- **MODE**: `interest`, `local`, `weather`, `source`, `seasonal`, `deals`, `compare`, `calendar`
- **ARGS**: the idea or source name passed to that mode

Examples:

| Rule | What it does |
|------|-------------|
| `monday\|interest\|AI roundup` | Every Monday, generate AI interest posts |
| `friday\|local\|weekend events` | Every Friday, find weekend events nearby |
| `daily\|weather` | Every day, generate weather-aware suggestions |
| `daily\|source\|hn` | Every day, pull top HN stories matching interests |
| `weekend\|local\|brunch spots` | Saturdays and Sundays, suggest brunch places |
| `wednesday\|compare\|coffee shops` | Wednesdays, do a ranked coffee shop comparison |

Multiple rules are joined with additional pipes: `monday|interest|AI roundup|daily|weather|daily|source|hn`

### Sources format

```bash
BEEPBOPBOOP_SOURCES=hn,ph,rss:https://simonwillison.net/atom/everything,substack:https://example.substack.com
```

| Source type | Format | What it does |
|-------------|--------|-------------|
| Hacker News | `hn` | Fetches top stories, filters by interests |
| Product Hunt | `ph` | Fetches today's top launches |
| RSS/Atom | `rss:<feed_url>` | Fetches recent items from any RSS/Atom feed |
| Substack | `substack:<newsletter_url>` | Fetches the latest post (if published within 7 days) |

### Family format

```bash
BEEPBOPBOOP_FAMILY=partner:Sarah:na:hiking,wine;child:Max:5:dinosaurs,lego;pet:Luna:na:walks
```

Each family member is a semicolon-separated entry with the format `role:name:age:interests`:

| Field | Values | Notes |
|-------|--------|-------|
| role | `partner`, `child`, `pet` | Required |
| name | First name | Required |
| age | Number or `na` | Number for children (used to suggest age-appropriate activities), `na` for partner/pet |
| interests | Comma-separated | Optional — helps personalize content (e.g., "dinosaurs" → museum exhibits) |

Family context adds texture to posts without being the primary driver. Weather mode suggests kid-friendly activities, local mode includes playgrounds when relevant, and post copy naturally mentions family where it fits ("Max would love this — dinosaur exhibit until April").

### Calendar format

```bash
BEEPBOPBOOP_CALENDAR_URL=https://calendar.google.com/calendar/ical/.../basic.ics
```

The calendar URL must point to a standard ICS file. How to get it:

| Provider | Where to find it |
|----------|-----------------|
| Google Calendar | Settings → calendar → "Secret address in iCal format" |
| Apple Calendar | Share calendar → copy the webcal:// URL |
| Outlook | Settings → Shared calendars → Publish a calendar → ICS link |

The skill fetches events from the next 7 days, enriches them with travel time, weather, and venue details, and generates posts framed as helpful reminders rather than notifications.

## Post types

Every post gets one of these types, which controls the icon and styling in the iOS app:

| Type | When it's used | Icon |
|------|---------------|------|
| `event` | Time-bound experiences: shows, concerts, games, festivals | calendar |
| `place` | Venues to visit: cafes, restaurants, parks, museums | mappin |
| `discovery` | Tips, observations, comparisons, seasonal suggestions | lightbulb |
| `article` | Written content: blog posts, news, HN stories | doc.text |
| `video` | Video content: YouTube, video essays | play.rectangle |

## Verify

After posting, check the feed:

```bash
curl -s -H "Authorization: Bearer my-test-user" http://localhost:8080/feed | jq .
```

Or open the iOS app — posts appear in the feed immediately.

## Tips

- **Run setup first** — `/beepbopboop-post init` walks you through everything in ~2 minutes and you can re-run it anytime to update
- **Home address = better distances** — setting your full address means "400m from your door" instead of "400m from city centre"
- **Start with batch mode** — `/beepbopboop-post fill my feed` gives you a full feed to scroll through instantly
- **Set your interests** — the more specific, the better the filtering (e.g., "AI agents,YC startups,angel investing" beats "AI,startups,investing")
- **Add your family** — the skill naturally mentions kid-friendly options, date-night spots, and dog-friendly venues when relevant
- **Connect your calendar** — upcoming events become posts with travel time, weather, and practical details a notification can't give you
- **Add RSS feeds** — if you follow specific blogs, add them as sources and batch mode will pull from them automatically
- **Set up a schedule** — configure `BEEPBOPBOOP_SCHEDULE` once and batch mode tailors content to each day of the week
- **Weather mode on lazy days** — when you don't know what to do, `/beepbopboop-post weather` gives you ideas matched to the actual conditions outside
- **Follow up on stories** — saw something interesting last week? `/beepbopboop-post update on <topic>` gets you caught up
- **Override location anytime** — any mode accepts a location argument: `/beepbopboop-post weather Portland, OR`
