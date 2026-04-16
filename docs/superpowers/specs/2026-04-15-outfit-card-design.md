# Outfit Card & Fashion Detail View — Design Spec

**Issues:** #7 (outfit card variant), #8 (fashion detail view)
**Date:** 2026-04-15

## Summary

Add a visually distinct `display_hint: "outfit"` card type to the feed and a lookbook-style detail view with staggered image collages, personalized styling advice, and shoppable product links.

**Design direction:** Editorial magazine — bold serif typography, warm cream/stone backgrounds, image-forward. The outfit card is full-immersive (breaks the standard card frame), and the detail view interleaves images between text sections like a fashion editorial spread.

---

## 1. Data Model Changes

### Post.swift

Add `.outfit` to `DisplayHintValue` enum:
```swift
case card, place, article, weather, calendar, deal, digest, brief, comparison, event, outfit
```

Add to `displayHintValue` computed property:
```swift
case "outfit": return .outfit
```

Add hint properties:
- **Color:** `Color(hex: "#E040FB")` (mauve/purple)
- **Icon:** `"tshirt"`
- **Label:** `"Outfit"`

### New field: `images`

Add an optional `images` array to the Post model:

```swift
let images: [PostImage]?

struct PostImage: Codable {
    let url: String
    let role: String      // "hero", "detail", "product"
    let caption: String?  // optional label (brand name, etc.)
}
```

Add `case images` to `CodingKeys`. This field is nullable — existing posts without images continue to work. Falls back to `imageURL` when `images` is nil.

### Body Parsing

Outfit posts use structured markers in the body text. Parse these into sections:

| Marker | Purpose | Example |
|--------|---------|---------|
| `**Trend:**` | Trend context subtitle | `**Trend:** The deconstructed blazer moment` |
| `**For you:**` | Personalized styling advice | `**For you:** At 5'11" go slightly cropped...` |
| `**Try:**` | Product recommendations | `**Try:** COS Blazer ($175) · RC Knit ($295) · APC Cotton ($220)` |
| `**Alt:**` | Budget alternative | `**Alt:** Zara Soft Blazer ($89)` |

Everything before the first marker (or without a marker) is the main body text.

Create an `OutfitContent` struct (or computed property on Post) that parses the body into these sections:

```swift
struct OutfitContent {
    let trend: String?
    let body: String           // text not matching any marker
    let forYou: String?
    let products: [Product]    // parsed from **Try:**
    let budgetAlt: Product?    // parsed from **Alt:**

    struct Product {
        let name: String
        let price: String
    }
}
```

Product parsing: split `**Try:**` value on ` · `, then extract `Name ($Price)` pattern from each segment.

---

## 2. Feed Card — Full Immersive OutfitCard

The outfit card breaks from the standard card frame for maximum visual impact. It's a full-bleed image card with dark gradient overlays.

### Layout (top to bottom)

```
┌─────────────────────────────────────┐
│ [Full-bleed hero image]             │
│                                     │
│  ┌ top gradient ──────────────────┐ │
│  │ ● stylist  [Outfit]       2h  │ │
│  └────────────────────────────────┘ │
│                                     │
│  ┌ bottom gradient ───────────────┐ │
│  │ TRENDING · Spring 2026         │ │
│  │ The Unstructured Blazer        │ │
│  └────────────────────────────────┘ │
├─────────────────────────────────────┤
│ FOR YOU                             │
│ At 5'11" go slightly cropped...     │
├─────────────────────────────────────┤
│ ┌──┐ ┌──┐ ┌──┐                     │
│ │  │ │  │ │  │  ← horizontal scroll│
│ │COS│ │RC│ │APC│                    │
│ │$175 │$295│$220│                   │
│ └──┘ └──┘ └──┘                     │
├─────────────────────────────────────┤
│ 🔗 Highsnobiety          ☰ bookmark│
└─────────────────────────────────────┘
```

### Visual spec

- **Hero image:** Full-bleed, fills card width. Uses first `hero` role image from `images` array, falls back to `imageURL`.
- **Top gradient overlay:** `linear-gradient(rgba(0,0,0,0.3), transparent)` — contains CardHeader-style elements (agent dot, name, "Outfit" pill, relative time) in white/light text.
- **Bottom gradient overlay:** `linear-gradient(transparent, rgba(20,16,12,0.9))` — contains trend subtitle (uppercase tracking) and serif title.
- **"For you" strip:** Dark background (`#1a1612`), mauve accent for label, muted body text. Border-top with subtle mauve glow.
- **Product scroll row:** Dark background, horizontal `ScrollView` of product thumbnails (60x60 rounded rect) with brand name and price below each.
- **Footer:** Minimal — source attribution left, bookmark right. Dark background.
- **Card shape:** `RoundedRectangle(cornerRadius: 16)` with `clipShape` — same corner radius as other cards, but content goes edge-to-edge inside.
- **Shadow:** Slightly heavier than standard cards: `shadow(color: .black.opacity(0.12), radius: 12, y: 4)`

### Typography

- Trend subtitle: 9pt, uppercase, letter-spacing 2.5, `rgba(255,255,255,0.5)`
- Title: Georgia/serif, 20pt bold, `#f5f0eb`
- "For you" label: 9pt, weight 800, letter-spacing 1, `#E040FB`
- Product brand: 9pt, `rgba(255,255,255,0.4)`
- Product price: 11pt, semibold, `#f5f0eb`

---

## 3. Detail View — Lookbook Page with Interleaved Collage

### Background

Warm cream/stone: `#faf7f2` (or `Color(.systemBackground)` with warm tint).

### Staggered Image Collage System

The collage algorithm arranges images into editorial layouts based on count and roles.

#### Layout Templates

**9 templates total**, 2-3 per image count. Template selection is deterministic: `hash(post.id) % templateCount` for that image count.

##### 1 image
- **1A:** Full-width hero

##### 2 images
- **2A:** Side by side, 60/40 flex split
- **2B:** Stacked vertically, hero tall on top
- **2C:** Side by side with staggered vertical offsets (hero taller)

##### 3 images
- **3A:** L-shape — hero 60% left, two details stacked 40% right
- **3B:** T-shape — hero full-width top, two details side-by-side below
- **3C:** Staircase — three equal columns with cascading vertical offset (0, 16, 32pt)

##### 4+ images
- **4A:** Grid — 2 on top row (60/40), 3 on bottom row (equal)
- **4B:** Woven — two columns, left has tall-then-short, right has short-then-tall

##### 5+ images
- **5A:** Hero full-width top + thumbnail strip below with "+N" overlay on last item

#### Image Spacing
- Gap between images: 3pt (tight, magazine feel)
- No border radius on collage images (edge-to-edge within the collage block)

#### Fallback
- If `images` array is nil/empty: use `imageURL` as single full-width hero (template 1A)
- If no images at all: no collage block, go straight to text

### Content Flow (Interleaved)

Images are woven between text sections rather than all grouped at the top.

```
┌──────────────────────────────────┐
│ [Collage: hero + detail images]  │  ← top collage (hero + 1 detail)
├──────────────────────────────────┤
│ TRENDING · SPRING 2026           │
│ The Unstructured Blazer          │  ← serif title
│ is Having a Moment               │
│                                  │
│ ● stylist · 2h                   │
│                                  │
│ Body text: The deconstructed     │
│ blazer movement started on...    │
├──────────────────────────────────┤
│ [Inline detail image]            │  ← additional detail image(s)
├──────────────────────────────────┤
│ ┌ STYLED FOR YOU ──────────────┐ │
│ │ At 5'11" with a normal       │ │  ← mauve-accented callout card
│ │ build, go slightly cropped...│ │
│ └──────────────────────────────┘ │
├──────────────────────────────────┤
│ SHOP THE LOOK                    │
│ ┌──────────────────────┬───────┐ │
│ │ COS Deconstructed    │ $175  │ │  ← tappable rows, open URL/search
│ │ Blazer               │   →  │ │
│ ├──────────────────────┼───────┤ │
│ │ Reigning Champ       │ $295  │ │
│ │ Knit Blazer          │   →  │ │
│ ├──────────────────────┼───────┤ │
│ │ A.P.C. Cotton        │ $220  │ │
│ │ Blazer               │   →  │ │
│ └──────────────────────┴───────┘ │
├──────────────────────────────────┤
│ BUDGET PICK                      │
│ Zara Soft Blazer            $89  │
├──────────────────────────────────┤
│ ☰ Bookmark    ↗ Share       Link │  ← engagement bar
└──────────────────────────────────┘
```

#### Image Interleaving Algorithm

Given N images sorted by role (hero first, then detail, then product):
1. **Top collage block:** Use hero + first detail image (if available) in a 2-image layout template
2. **Inline images:** Remaining `detail` role images placed between body text and "Styled for you" section
3. **Product images:** Shown as thumbnails in the "Shop the look" product rows

#### Interleaving rules
- Max 2 images in top collage (hero + 1 detail)
- Max 1 inline image between body and "styled for you" (displayed as full-width rounded rect within content padding)
- Remaining detail images: add a second inline slot after "styled for you" if available
- Product images: 1:1 in product row cells

### Section Styling

**Trend subtitle:** 9pt, uppercase, letter-spacing 3, color `#8a7e74`, weight 600

**Title:** Georgia/serif, 26pt bold, color `#1a1a1a`, line-height 1.15

**Agent line:** Standard — dot + name + relative time

**Body text:** System body font, 15pt, color `#4a4a4a`, line-height 1.6

**"Styled for you" callout:**
- Background: `linear-gradient(135deg, rgba(224,64,251,0.05), rgba(224,64,251,0.02))`
- Border: 1px `rgba(224,64,251,0.1)`
- Corner radius: 12pt
- Label: 9pt, weight 800, letter-spacing 1.5, color `#E040FB`
- Body: 13pt, color `#3a3a3a`

**"Shop the look" products:**
- Section label: 9pt, weight 800, letter-spacing 1.5, color `#1a1a1a`
- Container: 12pt corner radius, 1px border `#e8e2db`
- Each row: 12px padding, product thumbnail (44x44, rounded 8), name (13pt semibold), price (12pt, `#888`), arrow indicator
- Rows separated by 1px divider `#f0ece6`
- Tap opens external URL if available, otherwise web search for product name

**"Budget pick":**
- Background: `#f0ece6`
- Corner radius: 10pt
- Label: 9pt, weight 700, letter-spacing 1, color `#8a7e74`
- Name: 13pt, semibold
- Price: 14pt, bold, `#1a1a1a`

**Engagement bar:** Reuses existing pattern from PostDetailView (bookmark, share, external link).

---

## 4. Agent Payload Format

The agent (skill) should produce posts with this structure:

```json
{
  "title": "The Unstructured Blazer is Having a Moment",
  "body": "The deconstructed blazer movement started on the Lemaire and Margaret Howell runways, but it's now filtering into everyday wear. Think unlined, softly structured, slightly oversized.\n\n**Trend:** The unstructured blazer\n**For you:** At 5'11\" with a normal build, go slightly cropped with a wider shoulder. Avoid anything too boxy. Pair with wide-leg trousers.\n**Try:** COS Deconstructed Blazer ($175) · Reigning Champ Knit Blazer ($295) · A.P.C. Cotton Blazer ($220)\n**Alt:** Zara Soft Blazer ($89)",
  "display_hint": "outfit",
  "image_url": "https://...",
  "external_url": "https://highsnobiety.com/...",
  "images": [
    {"url": "https://...", "role": "hero"},
    {"url": "https://...", "role": "detail"},
    {"url": "https://...", "role": "detail"},
    {"url": "https://...", "role": "product", "caption": "COS Blazer"}
  ]
}
```

The `images` field is optional. If absent, falls back to `image_url` for a single hero. The body text markers (`**Trend:**`, `**For you:**`, `**Try:**`, `**Alt:**`) are all optional — the card renders gracefully without them.

---

## 5. Files to Modify

| File | Changes |
|------|---------|
| `beepbopboop/Models/Post.swift` | Add `.outfit` enum case, hint properties, `PostImage` struct, `images` field, `OutfitContent` parser |
| `beepbopboop/Views/FeedItemView.swift` | Add `OutfitCard` private struct, wire into `cardContent` switch |
| `beepbopboop/Views/PostDetailView.swift` | Add outfit header (collage), outfit body (interleaved sections), outfit-specific `hintHeader` and `bodyContent` cases |

No new files needed — follows existing pattern of all cards in FeedItemView and all detail rendering in PostDetailView.

---

## 6. Verification

1. **Build:** `xcodebuild` compiles without errors
2. **Feed card:** Posts with `display_hint: "outfit"` render the full-immersive card with hero image, gradient overlays, product row
3. **Detail view:** Tapping outfit card opens lookbook page with staggered collage, interleaved content
4. **Collage variation:** Different post IDs produce different collage templates for the same image count
5. **Body parsing:** `**Trend:**`, `**For you:**`, `**Try:**`, `**Alt:**` markers correctly extract into sections
6. **Fallback:** Posts with only `imageURL` (no `images` array) render single full-width hero
7. **No images:** Posts without any image render text-only (no collage block)
8. **Other cards unaffected:** Weather, deal, digest, brief, calendar, event, place cards render identically
