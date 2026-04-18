# Feed Tuning Sliders — Design

**Issue**: #33  
**Date**: 2026-04-17

## Problem

`PUT /user/weights` accepts `labelWeights`, `typeWeights`, `freshnessBias`, `geoBias` — API is complete. iOS has zero UI for it. Engagement-derived weights can take days to converge on a preference the user could express in 5 seconds.

## Solution

Two sliders in Settings ("Tune your feed") that read/write `GET/PUT /user/weights` directly.

---

## Backend

### Auth mismatch

`GET/PUT /user/weights` are in the agent-auth route group. iOS uses Firebase auth. Need Firebase-auth variants.

### Changes

1. **`WeightsHandler`**: Add `userRepo` dependency. Add `GetWeightsFirebase` and `UpdateWeightsFirebase` methods using `middleware.FirebaseUIDFromContext` → `userRepo.FindOrCreateByFirebaseUID(uid)`.

2. **`main.go`**: Register in Firebase-auth group:
   ```
   r.Get("/user/weights", weightsH.GetWeightsFirebase)
   r.Put("/user/weights", weightsH.UpdateWeightsFirebase)
   ```

3. **PUT body** (flat):
   ```json
   { "label_weights": {}, "type_weights": {}, "freshness_bias": 0.65, "geo_bias": 0.8 }
   ```
   Handler re-encodes to `json.RawMessage` and passes to `weightsRepo.Upsert`.

4. **GET response**: Returns existing weights or `nil` (defaults applied client-side).

---

## iOS

### Model

```swift
struct FeedWeights: Codable {
    var labelWeights: [String: Double]?
    var typeWeights: [String: Double]?
    var freshnessBias: Double
    var geoBias: Double
    
    static let defaults = FeedWeights(freshnessBias: 0.8, geoBias: 0.5)
    
    enum CodingKeys: String, CodingKey {
        case labelWeights = "label_weights"
        case typeWeights = "type_weights"
        case freshnessBias = "freshness_bias"
        case geoBias = "geo_bias"
    }
}
```

### APIService

Two new methods following the existing URLSession + Bearer auth pattern:

- `getWeights() async throws -> FeedWeights?` — GET /user/weights, returns nil if 200 with null weights
- `updateWeights(_ weights: FeedWeights) async throws` — PUT /user/weights

### SettingsViewModel

New published properties:
- `@Published var geoBias: Double = 0.5`
- `@Published var freshnessBias: Double = 0.8`
- `@Published var feedUpdated = false` — transient flash

`loadSettings()` concurrently loads both user settings and weights. On weights load, populate slider values; fall back to defaults if nil.

Auto-save: on slider release (`onEditingChanged(false)`), cancel any pending save task and schedule a new one with 500ms delay. On success, flash `feedUpdated` for 2 seconds.

### SettingsView

New section below Radius:

```
Section("Tune your feed") {
    📍 More local  ——●——  🌍 More global
    Slider(value: $viewModel.geoBias)
    
    ⚡ Live & timely  ——●——  📚 Evergreen
    Slider(value: $viewModel.freshnessBias)
    
    [Reset to defaults]   Feed updated ✓ (transient)
}
```

- Sliders 0–1, emoji labels on each end in HStack
- `onEditingChanged(false)` triggers debounced auto-save (no separate Save button for this section)
- Inline "Feed updated ✓" fades in/out on success
- Reset button sends `FeedWeights.defaults` immediately

---

## Out of Scope

- Per-label interest toggles (issue's optional "Customise further" section) — deferring to keep it clean
- Keychain storage for agent tokens — not needed since we're using Firebase auth
