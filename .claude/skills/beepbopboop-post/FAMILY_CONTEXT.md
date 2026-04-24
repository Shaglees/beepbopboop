# Family context rules

**Parse once after Step 0 loads config.** Only applies when `BEEPBOPBOOP_FAMILY` is set.

## Parse the family string

Format: semicolon-separated `role:name:age_or_na:interests` per member.
- Roles: `partner`, `child`, `pet`
- Age: number for children, `na` for partner/pet
- Interests: comma-separated

## Derive flags

- `has_children` — at least one member with role `child`
- `has_young_children` — at least one child with age ≤ 6
- `has_school_age_children` — at least one child with age 7–17
- `has_partner` — at least one member with role `partner`
- `has_pets` — at least one member with role `pet`
- `children_interests` — combined interests from all children
- `partner_interests` — interests from partner

## How family flags modify existing modes

- **Weather (W2):** when `has_children`, include kid-friendly activities (playgrounds, family venues). When `has_pets`, include dog-friendly venues/walks. When `has_partner`, frame ~20% as date-night options.
- **Local (`BASE_LOCAL.md` Step 2):** when idea is "activities"/"things to do" and `has_children`, include playgrounds and kid-friendly venues.
- **Batch (`MODE_BATCH.md` BT3 Phase 2):** when `has_children`, add 1–2 family-relevant posts (kid-friendly events, activities matching `children_interests`). When `has_partner`, occasionally include a date-spot suggestion.
- **Post body texture:** naturally mention family where relevant — "bring the kids — playground next to the patio", or children's names sparingly: "Max would love this — dinosaur exhibit until April". Never forced, never the primary angle.

## Key rule

**Family context is never the primary driver of a post.** It adds texture to already-relevant content. An AI news article never mentions family. A coffee shop post might mention "kid-friendly" if it has a play area, but the coffee is still the lead.
