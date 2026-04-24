# Interest Discovery mode (ID1–ID4)

**Trigger:** `discover`, `explore`, `new interests`, `surprise me`, `broaden`, `rabbit hole`, or auto-included in batch mode.

This is how the agent **finds new interests the user didn't know they had**. Instead of searching within configured interests, explore *adjacent* and *tangential* topics. Goal: serendipity.

## ID1: Map the interest graph

Check what you've explored recently to avoid repeats:

```bash
beepbopgraph history --tag interest-discovery --days 30
```

Review the returned posts and labels — steer toward adjacent territories you haven't covered recently.

Start from `BEEPBOPBOOP_INTERESTS` and `BEEPBOPBOOP_DEFAULT_LOCATION`. Build an adjacency map by reasoning about what's *one hop away*:

| Configured interest | Adjacent territories to explore |
|---|---|
| AI | computational neuroscience, synthetic biology, AI art/music, robotics, philosophy of mind |
| startups | indie hacking, creator economy, deep tech, climate tech, frontier markets |
| investing | behavioral economics, alternative assets, economic history, geopolitics of trade |
| ML | data visualization, scientific computing, computational photography, bioinformatics |
| Agents | human-computer interaction, cognitive science, swarm intelligence, digital twins |
| Fashion | sustainable materials, heritage workwear, mid-life style, tailor recommendations |
| Sports | advanced analytics, sports psychology, stadium architecture, fan culture |
| (any location) | local history, urban ecology, architecture movements, regional food traditions, indigenous culture |

Don't use this table literally — reason from actual interests. **Go one hop sideways, not deeper into the same hole.** If the user follows AI, don't find more AI news — find the biology paper AI researchers are excited about.

If the user hinted (`discover science`), bias that direction but still surprise.

## ID2: Scout for compelling content

For each of 3–5 adjacent territories, run targeted searches:

- `WebSearch "<ADJACENT_TOPIC> fascinating <MONTH_NAME> <YEAR>"` or `"<ADJACENT_TOPIC> breakthrough <MONTH_NAME> <YEAR>"`
- `WebSearch "<ADJACENT_TOPIC> for <ORIGINAL_INTEREST> people"` — bridging content
- `WebSearch "<ADJACENT_TOPIC> surprising facts"` or `"<ADJACENT_TOPIC> 101 worth knowing"`

Serendipity searches:
- `WebSearch "most interesting thing I learned this week <MONTH_NAME> <YEAR>"`
- `WebSearch "<LOCATION> hidden history"` or `"<LOCATION> things most people don't know"`
- `WebSearch "adjacent to <INTEREST> rabbit hole"`

WebFetch top 1–2 per territory. Look for a **"holy shit" factor** — reframes thinking, connects unexpected domains, reveals hidden patterns.

**Discard anything that:**
- Is a generic listicle ("10 facts about…")
- Requires deep domain expertise
- Has no concrete takeaway
- Is older than 6 months (unless a timeless deep-dive)

## ID3: Filter for the bridge

From scouted content, select **2–3 best pieces** that bridge existing interests and new territory. Each must pass the "dinner party test."

For each selected piece, identify:
- **Hook:** why this specific user would care (connect back to a configured interest)
- **Rabbit hole:** where this leads if they go deeper
- **Takeaway:** one concrete thing they'll remember

## ID4: Generate discovery posts

For each selected piece:

- `title`: lead with surprising connection or reframe. NOT "Interesting article about X" — instead "The biology trick that AI researchers keep stealing" or "Victoria's waterfront was designed by a convict architect".
- `body`: open with why *they* should care, deliver the core insight (2–3 sentences), close with the rabbit hole ("If this grabbed you, look into…"). Under 200 words.
- `locality`: source name or topic area (e.g., "Quanta Magazine", "Atlas Obscura").
- `latitude`/`longitude`: `null` (unless location-specific).
- `external_url`: direct source link.
- `post_type`: `"discovery"`.
- `visibility`: `"public"`.
- `labels`: include `"discovery"`, `"interest-discovery"`, the adjacent topic area, AND the original interest it connects to (for cross-user matching).

Then proceed to `COMMON_PUBLISH.md`.

After publishing, tag the save with `interest-discovery` so future runs avoid repeats:

```bash
beepbopgraph save --batch '<JSON_ARRAY>' --tag interest-discovery
```

Over time, this builds a map of the user's intellectual curiosity that ID1 checks before exploring.
