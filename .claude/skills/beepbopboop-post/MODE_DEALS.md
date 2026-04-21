# Deal mode (DL1–DL3)

**Trigger:** `deals`, `sales`, `specials`, `discounts`, or from batch Phase 2.

## DL1: Parse deal context

- **Local deals:** restaurants/shops/services near `BEEPBOPBOOP_DEFAULT_LOCATION` (default).
- **Interest deals:** tech subscriptions, software sales, courses — matched against `BEEPBOPBOOP_INTERESTS`.

## DL2: Research deals

**Local:**
- `WebSearch "<LOCATION> deals this week"`, `"<LOCATION> happy hour specials"`, `"<LOCATION> restaurant specials"`
- `WebFetch` top results for prices, dates, conditions.

**Interest:**
- `WebSearch "<INTEREST> deals <MONTH_NAME> <YEAR>"`, `"<INTEREST> discounts"`
- `WebFetch` top results.

## DL3: Generate deal posts

Generate **1–2 discovery posts**:

- Title leads with the value proposition: specific price, percent off, or "free"
- Body: what the deal is, where/how to get it, expiry, conditions
- `post_type: "discovery"`
- **`display_hint: "deal"`** — iOS renders a deal-focused card with price emphasis.

Then proceed to `COMMON_PUBLISH.md`.
