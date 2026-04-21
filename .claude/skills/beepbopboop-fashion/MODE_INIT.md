# Fashion onboarding (INIT1–INIT4)

**Trigger:** `init`, `setup`, `onboard`.

## INIT1: Collect physical attributes

If `FASHION_PROFILE` is not set, ask for or confirm:
- Height (e.g., 5'11")
- Build (slim, normal, athletic, heavy)
- Hair color
- Age
- Gender

Format as `height:5-11;build:normal;hair:brown;age:44;gender:male` and save to config.

## INIT2: Collect style preferences

Present style archetypes; ask the user to pick 2–3:

- minimalist, smart-casual, streetwear, classic, contemporary, athleisure, avant-garde, americana

Also ask for:
- Budget tier: `budget` / `moderate` / `premium` / `luxury`
- 3–5 brands they like or aspire to wear

Save to config as `FASHION_STYLE`, `FASHION_BUDGET`, `FASHION_BRANDS`.

## INIT3: Collect headshots (optional)

Ask for 2–3 photos:
- Front-facing, well-lit
- 3/4 angle
- Full body (optional, helps with proportion rendering)

Store paths in `FASHION_HEADSHOTS`. These are used as reference images for Flex.1 to generate "you wearing it" renders. If the user declines, prompt-based generation still works using physical description.

## INIT4: Validation post

Generate a single test fashion post to validate the full pipeline:

1. Quick trend scan (1 source)
2. Find 1–2 matching products
3. Generate an outfit image
4. Post it
5. Ask user: "Does this feel right? Want to adjust anything?"
