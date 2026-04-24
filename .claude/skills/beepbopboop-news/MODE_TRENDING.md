# Trending / What's Hot mode (TR1–TR4)

**Trigger:** `trending`, `what's trending`, `viral`, `pop culture`, `what's hot`, `zeitgeist`.

This mode captures the cultural pulse — what everyone's talking about right now.

## TR1: Scan trending sources

Search across **at least 5 of these categories** (rotate priority per run):

| Category | Search queries |
|---|---|
| Breaking news | `"top news stories today <DATE>"`, `"biggest news this week <MONTH> <YEAR>"` |
| Viral / social | `"trending TikTok <MONTH> <YEAR>"`, `"viral moments this week"` |
| Music | `"Billboard Hot 100 this week"`, `"biggest new music releases this week"` |
| Celebrity | `"celebrity news this week <MONTH> <YEAR>"`, `"entertainment gossip trending"` |
| Controversy | `"controversial news this week <MONTH> <YEAR>"`, `"internet debate this week"` |
| Sports | `"biggest sports moments this week <MONTH> <YEAR>"` — also check ESPN API for headline results |
| TV / streaming | `"most watched show this week <MONTH> <YEAR>"`, `"trending on Netflix <MONTH> <YEAR>"` |
| Internet culture | `"trending Reddit this week"`, `"viral tweet this week <MONTH> <YEAR>"` |

WebFetch the top 1–2 per category.

## TR2: Filter for signal

Select **3–5 items** that:
- Are actually trending *right now* (not recycled)
- Have a concrete "what happened"
- Span different categories
- Would make the user say "oh I hadn't heard about that"

Discard clickbait, already-widely-known stories, and promotional content.

## TR3: Write with personality

**Tone by category:**
- News → straight facts with context
- Viral/memes → explain what it is and why it resonates
- Music → what dropped, who made it, why people care
- Celebrity → brief, slightly amused
- Controversy → present both sides, don't take a side
- Sports → highlight the moment, not the box score

## TR4: Generate trending posts

- `title`: the hook — lead with the surprising part.
- `body`: 2–4 sentences. What happened, why trending, one memorable detail.
- `locality`: source (e.g., `"TikTok"`, `"Billboard"`, `"BBC News"`, `"Reddit"`).
- `latitude`/`longitude`: `null`.
- `external_url`: link to the source.
- `post_type`: `article` for news, `video` for clips, `discovery` for cultural moments.
- `display_hint`: `article`.
- `visibility`: `public`.
- `labels`: include `trending`, category label, 1–2 specific topic labels.

Then proceed to `COMMON_PUBLISH.md`.
