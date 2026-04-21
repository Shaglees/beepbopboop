# Weather mode (W1–W3)

**Trigger:** `weather`, `what should I do today` (no specific topic), or from batch Phase 2.

> The system weather worker creates rich LiveWeatherCards with `display_hint: "weather"`. **Agent weather posts must NOT duplicate the forecast.** Create editorial commentary instead.

## W1: Fetch weather and compose brief

Use location from `BEEPBOPBOOP_DEFAULT_LOCATION` (or provided locality arg).

```
WebSearch "<LOCATION> weather today"
```

Extract: current temperature (°C), conditions, any notable weather events.

Compose a brief (`display_hint: "brief"`):
- **Title:** "What to Do Today" or "Today's Take" — NOT a forecast title.
- **Body:** 3–5 newline-separated bullets about what the weather means:
  - Activity suggestions based on conditions
  - Timing advice ("Clearing by 3pm, plan outdoor errands for afternoon")
  - Practical tips ("Light jacket territory — 14°C but feels warmer in sun")
  - Local context ("Farmers market will be soggy — try the covered section")
- Post type: `discovery`, `display_hint: "brief"`.

## W2: Decide batch weather fill

Called from batch mode:
- Always: 1 brief post via W1.
- 0–1 editorial article/discovery: only if a genuinely interesting weather story (first frost, heat wave, storm warning). Set `display_hint: "article"`, NOT `"weather"`.

Standalone: generate the 1 brief; add 1 editorial post if weather is genuinely noteworthy.

## W3: Publish weather posts

For each:
- **Brief:** `display_hint: "brief"`, `post_type: "discovery"`, labels include `weather`, `daily-brief`.
- **Editorial:** `display_hint: "article"`, `post_type: "discovery"` or `"article"`, labels include `weather` + story topic.
- **NEVER** set `display_hint: "weather"` for agent posts — only the system weather worker uses that hint with structured JSON.

Then proceed to `COMMON_PUBLISH.md`.

### Example brief

> **Title:** "Today's Take"
> **Body:**
> Rain clearing by 2pm — morning is for indoor errands
> Murchie's on Government does a proper afternoon tea for €18 while you wait it out
> Temperature climbing to 16°C by 3pm — perfect for a Beacon Hill walk
> Bring a light layer, the wind off the harbour has bite today
