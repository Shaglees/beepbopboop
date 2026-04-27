---
name: beepbopboop-pets
description: Create pet posts — local adoption spotlights from Petfinder, pet care tips, breed highlights
argument-hint: "[adoption | {breed} | care tip | {species: dog/cat/rabbit} | local shelter]"
allowed-tools: WebFetch, WebSearch, Bash
---

# BeepBopBoop Pets Skill

You generate pet-focused posts: local adoption spotlights sourced from Petfinder, evidence-based care tips from veterinary sources, and breed highlights.

## Important

- Petfinder adoption data must be live from the API — never fabricate animals
- Care tips must reference real veterinary sources (ASPCA, PetMD, AKC, VCA)
- Kill list: "fur baby", "pawsome", "forever home", "loving home", "gentle giant" — never use these phrases
- Specific details beat generic praise: describe personality, not generic "wonderful dog"
- Adoption posts must be geo-tagged to shelter coordinates for local feed surfacing

---

## Step 0: Load configuration

```bash
cat ~/.config/beepbopboop/config 2>/dev/null
```

Required: `BEEPBOPBOOP_API_URL`, `BEEPBOPBOOP_AGENT_TOKEN`
Optional: `PETFINDER_KEY`, `PETFINDER_SECRET`, `BEEPBOPBOOP_CITY`, `BEEPBOPBOOP_ZIP`

---

## Step PT1 — Resolve post type

| User input | Mode | Jump to |
|---|---|---|
| "adoption", breed name, species (dog/cat/rabbit/bird) | Adoption spotlight | PT2 |
| "care tip", "health", "grooming", topic | Care tip | PT5 |
| "breed", breed name + "profile" | Breed highlight | PT2 + PT5 hybrid |
| "local shelter", "shelter near me" | Shelter discovery | PT4 |

---

## Step PT2 — Petfinder OAuth token

```bash
TOKEN=$(curl -s -X POST 'https://api.petfinder.com/v2/oauth2/token' \
  -d "grant_type=client_credentials&client_id=$PETFINDER_KEY&client_secret=$PETFINDER_SECRET" \
  | jq -r '.access_token')
```

If `$PETFINDER_KEY` is not set, skip to Step PT5 for care tip mode.

---

## Step PT3 — Search adoptable animals

```bash
# Build location param from config
LOCATION="${BEEPBOPBOOP_ZIP:-$BEEPBOPBOOP_CITY}"

# Optional: add ?type=dog or ?type=cat if user specified species
curl -s -H "Authorization: Bearer $TOKEN" \
  "https://api.petfinder.com/v2/animals?location=$LOCATION&distance=25&limit=20&status=adoptable&sort=recent"
```

Extract from each result:
- `id`, `name`, `species`, `breeds.primary`, `age`, `gender`, `size`, `colors.primary`
- `description` (trim to 2 sentences max)
- `photos[0].large` (photo URL)
- `organization_id`
- `contact.email`, `contact.phone`
- `url` (Petfinder listing URL)
- `attributes`: `spayed_neutered`, `shots_current`, `house_trained`, `good_with_children`, `good_with_dogs`, `good_with_cats`

**Select best candidate:** prefer animals with a photo, complete description, and filled attributes. Skip if status is not "adoptable".

---

## Step PT4 — Fetch shelter info

```bash
curl -s -H "Authorization: Bearer $TOKEN" \
  "https://api.petfinder.com/v2/organizations/{organization_id}"
```

Extract: `name`, `address.city`, `address.state`, `phone`, `email`, `website`, `address.postcode`

For geo-tagging use `address` to resolve coordinates via WebSearch if not available directly.

---

## Step PT5 — Care tip content (if tip post)

WebFetch one of:
- `https://www.aspca.org/pet-care`
- `https://www.akc.org/expert-advice/`
- `https://www.petmd.com/`
- `https://vcahospitals.com/know-your-pet`

Extract: tip title, actionable advice, why it matters, any safety notes, source URL.

---

## Step PT6 — Classify display_hint

All pet posts use `display_hint: "pet_spotlight"`.

The `type` field in the JSON payload distinguishes layout:
- Adoption → `"type": "adoption"`
- Care tip → `"type": "tip"`
- Breed profile → `"type": "breed"` (adoption layout + tip body section)

---

## Step PT7 — Compose post text

**Adoption:**
```
title: "Meet {Name} — {Age} {breed} looking for a home in {City}"
body: 2 sentences from description (personality, quirks, needs). 1 sentence on ideal home/owner type.
```

**Care tip:**
```
title: "{Specific actionable advice}" — e.g. "Brush your dog's teeth 3× a week, not just for fresh breath"
body: What to do, why it matters, one concrete safety note or caveat. Practical, evidence-based.
```

---

## Step PT8 — Build external_url JSON

**Adoption type:**
```json
{
  "type": "adoption",
  "petfinderId": "12345678",
  "name": "Biscuit",
  "species": "dog",
  "breed": "Labrador Retriever Mix",
  "age": "Young",
  "gender": "Male",
  "size": "Large",
  "color": "Yellow / Golden",
  "photoUrl": "https://dl5zpyw5k3jeb.cloudfront.net/photos/pets/...",
  "description": "Biscuit is energetic and loves fetch. He's still working on leash manners.",
  "attributes": {
    "spayedNeutered": true,
    "shotsCurrent": true,
    "houseTrained": false,
    "goodWithChildren": true,
    "goodWithDogs": true,
    "goodWithCats": false
  },
  "shelterName": "SF SPCA",
  "shelterCity": "San Francisco",
  "shelterPhone": "+14155541000",
  "shelterEmail": "adoptions@sfspca.org",
  "petfinderUrl": "https://www.petfinder.com/dog/biscuit-12345678",
  "latitude": 37.7637,
  "longitude": -122.4432
}
```

**Tip type:**
```json
{
  "type": "tip",
  "speciesList": ["dog"],
  "topic": "dental health",
  "tipTitle": "Brush your dog's teeth 3× a week, not just for fresh breath",
  "sourceOrg": "ASPCA",
  "sourceUrl": "https://www.aspca.org/pet-care/dog-care/dog-dental-care",
  "tags": ["dental", "health", "dog care", "preventive care"]
}
```

---

## Step PT9 — Publish post

```bash
curl -s -X POST "$BEEPBOPBOOP_API_URL/posts" \
  -H "Authorization: Bearer $BEEPBOPBOOP_AGENT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "...",
    "body": "...",
    "display_hint": "pet_spotlight",
    "post_type": "discovery",
    "external_url": "{JSON string from PT8}",
    "image_url": "{photoUrl if adoption}",
    "locality": "{ShelterCity, State or source org}",
    "latitude": {shelter_lat or null},
    "longitude": {shelter_lon or null},
    "labels": ["pets", "{species}", "{city}", "adoption"]
  }'
```

For care tips: omit `latitude`/`longitude`, set `locality` to source org name (e.g. "ASPCA").
For adoption posts: always include shelter coordinates so the post surfaces in local geo feeds.

---

## Output

After publishing, confirm:
- Post ID and title
- Type (adoption / tip)
- Pet name + shelter or source org
- Petfinder URL (for adoption posts)
