# BeepBopBoop Init Wizard

Steps IN1–IN10 for initial setup and configuration.

**Trigger**: `init`, `setup`, `configure`, `config`, or auto-triggered when config file is missing/incomplete.

**Skip this section unless Step 0a detected init mode or Step 0 found missing config.**

Interactive wizard using `AskUserQuestion` at each step. If re-running with an existing config, show current values as defaults.

## IN1: Welcome

Tell the user:
> "Welcome to BeepBopBoop setup! This takes about 2 minutes and only needs to happen once. I'll walk you through connecting to the API, setting your home location, interests, and optional extras like family context and calendar integration. You can re-run this anytime with `/beepbopboop-post init` to update your config."

## IN2: API Connection

Ask for:
- **API URL** — suggest `http://localhost:8080` as default
- **Agent token** — the `bbp_` token from their agent setup

Test the connection:
```bash
curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer <TOKEN>" <API_URL>/feed
```

If the response is not `200`, warn the user: "Could not connect to the API. Check the URL and token. Continue anyway?" If they say no, stop the wizard.

## IN2b: Ensure dependencies

Check that beepbopgraph is available (used for post deduplication):

```bash
which beepbopgraph >/dev/null 2>&1 && echo "OK" || echo "beepbopgraph not found — build with: cd ~/beepbopboop/backend && make beepbopgraph && make install-beepbopgraph"
```

If beepbopgraph is not installed, warn the user but continue — dedup will be skipped at runtime.

## IN3: Home Address

Ask for their full street address (e.g., "1234 Oak Bay Ave, Victoria, BC").

Geocode it:
```bash
osm geocode "ADDRESS" | jq '.[0] | {lat, lon, display_name}'
```

Show the resolved result and ask for confirmation: "Is this correct? [resolved display_name] (lat, lon)". If not, let them try again or enter lat/lon manually.

Store: `BEEPBOPBOOP_HOME_ADDRESS`, `BEEPBOPBOOP_HOME_LAT`, `BEEPBOPBOOP_HOME_LON`.

This step is optional — the user can skip it and fall back to city-level location.

## IN4: Display Location

Auto-derive a city-level display name from the IN3 geocoded result (e.g., "Victoria, BC, Canada"). Show the derived name and let the user override it.

If IN3 was skipped, ask the user for their city/area directly.

Store as `BEEPBOPBOOP_DEFAULT_LOCATION`.

## IN5: Interests

Ask for comma-separated topics. Suggest examples based on common categories:
> "What topics interest you? Examples: AI, startups, investing, cooking, fitness, gaming, music, travel, parenting"

Store as `BEEPBOPBOOP_INTERESTS`.

This step is optional — the user can skip it.

## IN6: Family

Ask: "Would you like to add family members? This helps personalize content — e.g., suggesting kid-friendly venues or date-night spots."

If yes, loop for each member:
1. **Relationship**: partner, child, or pet
2. **Name**: first name
3. **Age**: number (children only — skip for partner/pet, store as `na`)
4. **Interests**: comma-separated (optional)

Continue asking "Add another family member?" until they say no.

Format into `BEEPBOPBOOP_FAMILY` string: `role:name:age_or_na:interests` per member, separated by `;`.

Example: `partner:Sarah:na:hiking,wine;child:Max:5:dinosaurs,lego;pet:Luna:na:walks`

This step is optional — the user can skip it entirely.

## IN7: Content Sources

Explain source types:
> "You can add content sources that batch mode pulls from automatically:
> - `hn` — Hacker News top stories filtered by your interests
> - `ph` — Product Hunt daily launches
> - `rss:<URL>` — any RSS/Atom feed (e.g., `rss:https://simonwillison.net/atom/everything`)
> - `substack:<URL>` — a Substack newsletter"

Ask for a comma-separated list. This step is optional.

Store as `BEEPBOPBOOP_SOURCES`.

## IN8: Calendar

Ask: "Do you have a calendar URL (ICS format) you'd like to connect? This lets BeepBopBoop turn your upcoming events into posts with travel time, weather, and practical details."

Explain how to get it:
> - **Google Calendar**: Settings → calendar → "Secret address in iCal format"
> - **Apple Calendar**: Share calendar → copy the webcal:// URL
> - **Outlook**: Settings → Shared calendars → Publish a calendar → ICS link

If they provide a URL, test-fetch it:
```bash
curl -s -o /dev/null -w "%{http_code}" "<CALENDAR_URL>"
```

If the fetch fails, warn and let them skip or retry.

Store as `BEEPBOPBOOP_CALENDAR_URL`. This step is optional.

## IN8b: Image Services

Ask: "Posts look much better with images. Three optional services are supported — Unsplash for stock photos, imgur for hosting AI-generated and re-uploaded images, and Google Places for reliable venue photos. Set up any or all?"

**Unsplash** (for real stock photos):
> "Sign up at https://unsplash.com/developers, create an app, and copy the Access Key. Free tier: 50 requests/hour."

**imgur** (for hosting AI-generated images and Google Places re-uploads):
> "Register an app at https://api.imgur.com/oauth2/addclient (choose 'Anonymous usage without user authorization'). Copy the Client-ID. Free: 1250 uploads/day."

**Google Places** (for reliable venue photos as a fallback):
> "Enable the Places API (New) in Google Cloud Console, create an API key, and restrict it to Places API only. The free $200/month credit covers approximately 5000 place lookups — more than enough for typical usage."

The full image pipeline for place/location posts: Wikimedia Commons and Panoramax are tried first (free, no config needed), then Google Places if configured, then Unsplash for non-place content, then Pollinations AI as a last resort. For non-geographic posts, Unsplash is tried first followed by Pollinations.

Store as `BEEPBOPBOOP_UNSPLASH_ACCESS_KEY`, `BEEPBOPBOOP_IMGUR_CLIENT_ID`, and `BEEPBOPBOOP_GOOGLE_PLACES_KEY`. All are optional — if none are set, Wikimedia Commons and Panoramax (which need no keys) are still tried for geographic posts.

## IN9: Schedule

Ask: "Would you like to set up a batch schedule? This tells batch mode what content to generate on which days."

Explain the format: `DAY|MODE|ARGS` triplets. Suggest a starter schedule based on their interests from IN5:
> "Here's a suggested starter schedule based on your interests:
> `daily|weather|monday|interest|<FIRST_INTEREST> roundup|daily|source|hn`
> Want to use this, customize it, or skip?"

Also ask for batch range (default 8-15):
> "How many posts should batch mode target? Default is 8-15."

Store as `BEEPBOPBOOP_SCHEDULE`, `BEEPBOPBOOP_BATCH_MIN`, `BEEPBOPBOOP_BATCH_MAX`.

This step is optional.

## IN10: Confirm & Save

Show a full config summary:
> ```
> API URL:        http://localhost:8080
> Agent Token:    bbp_xxxxx...
> Home Address:   1234 Oak Bay Ave, Victoria, BC
> Home Coords:    48.4284, -123.3248
> Display Location: Victoria, BC, Canada
> Interests:      AI, startups, investing
> Family:         Sarah (partner), Max (child, 5), Luna (pet)
> Sources:        hn, ph, rss:https://simonwillison.net/atom/everything
> Calendar:       https://calendar.google.com/.../basic.ics
> Schedule:       daily|weather|monday|interest|AI roundup|daily|source|hn
> Batch Range:    8-15
> Unsplash Key:   (configured)
> imgur Client-ID: (configured)
> ```

Ask: "Save this config? (confirm / edit / cancel)"

- **confirm**: Write the config file (see below)
- **edit**: Ask which section to change, jump back to that step
- **cancel**: Abort without saving

Write the config file:
```bash
mkdir -p ~/.config/beepbopboop && cat > ~/.config/beepbopboop/config << 'ENDOFCONFIG'
BEEPBOPBOOP_API_URL=<URL>
BEEPBOPBOOP_AGENT_TOKEN=<TOKEN>
BEEPBOPBOOP_DEFAULT_LOCATION=<LOCATION>
BEEPBOPBOOP_INTERESTS=<INTERESTS>
BEEPBOPBOOP_SOURCES=<SOURCES>
BEEPBOPBOOP_SCHEDULE=<SCHEDULE>
BEEPBOPBOOP_BATCH_MIN=<MIN>
BEEPBOPBOOP_BATCH_MAX=<MAX>
BEEPBOPBOOP_HOME_ADDRESS=<ADDRESS>
BEEPBOPBOOP_HOME_LAT=<LAT>
BEEPBOPBOOP_HOME_LON=<LON>
BEEPBOPBOOP_FAMILY=<FAMILY>
BEEPBOPBOOP_CALENDAR_URL=<CALENDAR_URL>
BEEPBOPBOOP_UNSPLASH_ACCESS_KEY=<UNSPLASH_KEY>
BEEPBOPBOOP_IMGUR_CLIENT_ID=<IMGUR_CLIENT_ID>
ENDOFCONFIG
```

For optional keys that the user skipped, write them as comments:
```bash
# BEEPBOPBOOP_FAMILY=
# BEEPBOPBOOP_CALENDAR_URL=
```

Confirm: "Config saved to `~/.config/beepbopboop/config`. You're all set! Run `/beepbopboop-post init` anytime to reconfigure."

If the wizard was auto-triggered (missing config), return to the main skill and continue with Step 0a to execute the user's original command. If it was triggered by `init`/`setup`, stop here.

---

## Family Context Rules

**Parse once after Step 0 loads config.** Only applies when `BEEPBOPBOOP_FAMILY` is set.

Parse the family string and derive these flags:
- `has_children` — at least one member with role `child`
- `has_young_children` — at least one child with age ≤ 6
- `has_school_age_children` — at least one child with age 7–17
- `has_partner` — at least one member with role `partner`
- `has_pets` — at least one member with role `pet`
- `children_interests` — combined interests from all children
- `partner_interests` — interests from partner

**How family flags modify existing modes:**

- **Weather (W2)**: When `has_children`, include kid-friendly activities in the suggestions (playgrounds, family-friendly venues). When `has_pets`, include dog-friendly venues/walks. When `has_partner`, frame ~20% of suggestions as date-night options.
- **Local (Step 2)**: When the idea is "activities"/"things to do" and `has_children`, include playgrounds and kid-friendly venues in POI discovery.
- **Batch (BT3 Phase 2)**: When `has_children`, add 1-2 family-relevant posts (kid-friendly events, activities matching `children_interests`). When `has_partner`, occasionally include a date-spot suggestion.
- **Post body texture**: Naturally mention family where relevant — e.g., "bring the kids — playground next to the patio", or use children's names and interests sparingly: "Max would love this — dinosaur exhibit until April". Never forced, never the primary angle.

**Key rule**: Family context is **never** the primary driver of a post. It adds texture to already-relevant content. An AI news article never mentions family. A coffee shop post might mention "kid-friendly" if it has a play area, but the coffee is still the lead.
