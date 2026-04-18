# Food Skill + RestaurantCard Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build the `beepbopboop-food` Claude skill and iOS `RestaurantCard` view for restaurant discovery posts using Yelp/Google Places data.

**Architecture:** The skill generates posts with `display_hint: "restaurant"` and structured JSON in `external_url` (FoodData). The iOS app decodes this via `Post.foodData`, routes it in `FeedItemView`, and renders it with `RestaurantCard` — a warm off-white self-contained card following the OutfitCard pattern.

**Tech Stack:** SwiftUI (iOS), Go (backend), Markdown skill file, Yelp Fusion API, Google Places API

---

### Task 1: Backend — add `"restaurant"` display hint

**Files:**
- Modify: `backend/internal/handler/post.go:32-47` (ValidDisplayHints map)
- Modify: `backend/internal/handler/post.go:171-172` (structuredHint check)
- Modify: `backend/internal/handler/post.go:231-245` (validation switch)

**Step 1: Add to ValidDisplayHints map**

In `ValidDisplayHints`, add after `"standings": true,`:
```go
"restaurant": true,
```

**Step 2: Add to structuredHint check (line ~171)**

Change:
```go
structuredHint := req.DisplayHint == "weather" || req.DisplayHint == "scoreboard" ||
    req.DisplayHint == "matchup" || req.DisplayHint == "standings"
```
To:
```go
structuredHint := req.DisplayHint == "weather" || req.DisplayHint == "scoreboard" ||
    req.DisplayHint == "matchup" || req.DisplayHint == "standings" ||
    req.DisplayHint == "restaurant"
```

**Step 3: Add to validation switch (line ~232)**

In the `switch req.DisplayHint` block, after `case "standings":`, add:
```go
case "restaurant":
    validateFoodData(req.ExternalURL, &errs, &warns)
```

Also add to the `else if` condition that requires external_url:
```go
} else if req.DisplayHint == "weather" || req.DisplayHint == "scoreboard" || req.DisplayHint == "matchup" || req.DisplayHint == "standings" || req.DisplayHint == "restaurant" {
```

**Step 4: Add validateFoodData function**

After the `validateStandingsData` function, add:
```go
type foodDataValidation struct {
    Name        *string  `json:"name"`
    Rating      *float64 `json:"rating"`
    ReviewCount *int     `json:"reviewCount"`
    Cuisine     []string `json:"cuisine"`
    Latitude    *float64 `json:"latitude"`
    Longitude   *float64 `json:"longitude"`
}

func validateFoodData(raw string, errs *[]validationIssue, warns *[]validationIssue) {
    var fd foodDataValidation
    if err := json.Unmarshal([]byte(raw), &fd); err != nil {
        *errs = append(*errs, validationIssue{
            Field:   "external_url",
            Code:    "invalid_json",
            Message: "restaurant external_url must be valid JSON",
        })
        return
    }
    if fd.Name == nil || *fd.Name == "" {
        *errs = append(*errs, validationIssue{Field: "external_url.name", Code: "required", Message: "restaurant data missing name"})
    }
    if fd.Rating == nil {
        *warns = append(*warns, validationIssue{Field: "external_url.rating", Code: "missing", Message: "restaurant data missing rating"})
    }
    if fd.Latitude == nil || fd.Longitude == nil {
        *warns = append(*warns, validationIssue{Field: "external_url.latitude", Code: "missing", Message: "restaurant data missing coordinates"})
    }
}
```

**Step 5: Build backend to verify**
```bash
cd /path/to/beepbopboop/backend && go build ./...
```
Expected: no errors

**Step 6: Commit**
```bash
git add backend/internal/handler/post.go
git commit -m "feat: add restaurant display_hint to backend validation"
```

---

### Task 2: iOS — FoodData model

**Files:**
- Create: `beepbopboop/beepbopboop/Models/FoodData.swift`

**Step 1: Create FoodData.swift**

```swift
import Foundation

struct FoodData: Codable {
    let yelpId: String?
    let name: String
    let imageUrl: String?
    let rating: Double
    let reviewCount: Int
    let cuisine: [String]
    let priceRange: String?
    let address: String
    let neighbourhood: String?
    let distanceM: Double?
    let isOpenNow: Bool?
    let phone: String?
    let yelpUrl: String?
    let latitude: Double
    let longitude: Double
    let mustTry: [String]
    let pricePerHead: String?
    let newOpening: Bool

    enum CodingKeys: String, CodingKey {
        case yelpId, name, imageUrl, rating, reviewCount, cuisine
        case priceRange, address, neighbourhood, distanceM, isOpenNow
        case phone, yelpUrl, latitude, longitude, mustTry, pricePerHead, newOpening
    }
}
```

**Step 2: Commit**
```bash
git add beepbopboop/beepbopboop/Models/FoodData.swift
git commit -m "feat: add FoodData model for restaurant posts"
```

---

### Task 3: iOS — Update Post.swift

**Files:**
- Modify: `beepbopboop/beepbopboop/Models/Post.swift`

**Step 1: Add `.restaurant` to DisplayHintValue enum (line ~181)**

Add after `case scoreboard, matchup, standings`:
```swift
case restaurant
```

**Step 2: Add `"restaurant"` case to displayHintValue switch (line ~186)**

After `case "standings": return .standings`:
```swift
case "restaurant": return .restaurant
```

**Step 3: Add color/icon/label entries**

In `hintColor` switch, after `.standings: return .secondary`:
```swift
case .restaurant: return Color(red: 0.937, green: 0.267, blue: 0.267) // coral #EF4444
```

In `hintIcon` switch, after `.standings: return "list.number"`:
```swift
case .restaurant: return "fork.knife"
```

In `hintLabel` switch, after `.standings: return "Scores"`:
```swift
case .restaurant: return "Restaurant"
```

**Step 4: Add foodData computed property (after standingsData)**

```swift
/// Parsed restaurant data from externalURL (for restaurant display_hint posts).
var foodData: FoodData? {
    guard displayHintValue == .restaurant,
          let json = externalURL,
          let data = json.data(using: .utf8) else { return nil }
    return try? JSONDecoder().decode(FoodData.self, from: data)
}
```

**Step 5: Build app to verify no Swift errors**

**Step 6: Commit**
```bash
git add beepbopboop/beepbopboop/Models/Post.swift
git commit -m "feat: add restaurant display hint to Post model"
```

---

### Task 4: iOS — RestaurantCard view

**Files:**
- Create: `beepbopboop/beepbopboop/Views/FoodCards.swift`

**Step 1: Create FoodCards.swift**

Full implementation:

```swift
import SwiftUI

// MARK: - RestaurantCard

struct RestaurantCard: View {
    let post: Post
    let food: FoodData
    @State private var activeReaction: String?

    private let warmBg = Color(red: 0.980, green: 0.980, blue: 0.969)   // #FAFAF7
    private let coral  = Color(red: 0.937, green: 0.267, blue: 0.267)   // #EF4444
    private let sage   = Color(red: 0.518, green: 0.800, blue: 0.086)   // #84CC16

    init?(post: Post) {
        guard let fd = post.foodData else { return nil }
        self.post = post
        self.food = fd
        self._activeReaction = State(initialValue: post.myReaction)
    }

    var body: some View {
        VStack(spacing: 0) {
            heroSection
            infoSection
            RestaurantFooter(post: post, coral: coral, activeReaction: $activeReaction)
        }
        .background(warmBg)
    }

    // MARK: Hero

    private var heroSection: some View {
        ZStack(alignment: .top) {
            heroImage
                .frame(height: 180)
                .clipped()

            // Header overlay
            HStack(spacing: 6) {
                Circle()
                    .fill(coral)
                    .frame(width: 8, height: 8)
                Text(post.agentName)
                    .font(.subheadline.weight(.medium))
                    .foregroundColor(.white)
                Text("Restaurant")
                    .font(.caption2.weight(.semibold))
                    .foregroundColor(.white)
                    .padding(.horizontal, 7)
                    .padding(.vertical, 3)
                    .background(.white.opacity(0.2))
                    .cornerRadius(4)
                Spacer()
                Text(post.relativeTime)
                    .font(.subheadline)
                    .foregroundColor(.white.opacity(0.7))
            }
            .padding(.horizontal, 16)
            .padding(.top, 14)
            .padding(.bottom, 32)
            .background(
                LinearGradient(
                    colors: [.black.opacity(0.35), .clear],
                    startPoint: .top,
                    endPoint: .bottom
                )
            )

            // NEW banner (diagonal, top-left)
            if food.newOpening {
                newBanner
            }

            // Open/Closed pill (top-right)
            if let isOpen = food.isOpenNow {
                openPill(isOpen: isOpen)
                    .padding(.top, 14)
                    .padding(.trailing, 16)
                    .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topTrailing)
            }
        }
        .frame(height: 180)
    }

    @ViewBuilder
    private var heroImage: some View {
        if let urlStr = food.imageUrl, let url = URL(string: urlStr) {
            AsyncImage(url: url) { phase in
                switch phase {
                case .success(let image):
                    image.resizable().aspectRatio(contentMode: .fill)
                case .failure:
                    placeholderHero
                default:
                    placeholderHero.overlay(ProgressView())
                }
            }
        } else {
            placeholderHero
        }
    }

    private var placeholderHero: some View {
        Rectangle()
            .fill(
                LinearGradient(
                    colors: [coral.opacity(0.3), coral.opacity(0.15)],
                    startPoint: .topLeading,
                    endPoint: .bottomTrailing
                )
            )
            .overlay(
                Image(systemName: "fork.knife")
                    .font(.system(size: 40))
                    .foregroundColor(coral.opacity(0.5))
            )
    }

    private var newBanner: some View {
        Text("NEW")
            .font(.system(size: 9, weight: .black))
            .tracking(1.5)
            .foregroundColor(.white)
            .padding(.horizontal, 20)
            .padding(.vertical, 4)
            .background(Color(red: 0.133, green: 0.773, blue: 0.369)) // #22C35E green
            .rotationEffect(.degrees(-45))
            .offset(x: -22, y: 18)
            .clipped()
    }

    private func openPill(isOpen: Bool) -> some View {
        HStack(spacing: 4) {
            Circle()
                .fill(isOpen ? Color(red: 0.133, green: 0.773, blue: 0.369) : .red)
                .frame(width: 6, height: 6)
            Text(isOpen ? "Open Now" : "Closed")
                .font(.caption2.weight(.semibold))
                .foregroundColor(.white)
        }
        .padding(.horizontal, 8)
        .padding(.vertical, 4)
        .background(.black.opacity(0.5))
        .clipShape(Capsule())
    }

    // MARK: Info

    private var infoSection: some View {
        VStack(alignment: .leading, spacing: 8) {
            // Name
            Text(food.name)
                .font(.system(size: 18, weight: .semibold))
                .foregroundColor(Color(red: 0.1, green: 0.1, blue: 0.1))
                .lineLimit(1)

            // Cuisine chips
            if !food.cuisine.isEmpty {
                cuisineChips
            }

            // Star rating row
            ratingRow

            // Distance + price
            distancePriceRow

            // Must Try strip
            if !food.mustTry.isEmpty {
                mustTryStrip
            }

            // Price per head
            if let pricePerHead = food.pricePerHead {
                Text("~\(pricePerHead)/person")
                    .font(.caption.weight(.medium))
                    .foregroundColor(sage)
            }
        }
        .padding(.horizontal, 16)
        .padding(.top, 12)
        .padding(.bottom, 10)
    }

    private var cuisineChips: some View {
        ScrollView(.horizontal, showsIndicators: false) {
            HStack(spacing: 6) {
                ForEach(food.cuisine, id: \.self) { tag in
                    Text(tag)
                        .font(.caption2.weight(.medium))
                        .foregroundColor(coral)
                        .padding(.horizontal, 8)
                        .padding(.vertical, 3)
                        .background(coral.opacity(0.1))
                        .clipShape(Capsule())
                }
            }
        }
    }

    private var ratingRow: some View {
        HStack(spacing: 6) {
            // Star display
            HStack(spacing: 2) {
                ForEach(0..<5) { i in
                    let filled = Double(i) < food.rating
                    let halfFilled = !filled && Double(i) < food.rating + 0.5
                    Image(systemName: filled ? "star.fill" : (halfFilled ? "star.leadinghalf.filled" : "star"))
                        .font(.system(size: 11))
                        .foregroundColor(coral)
                }
            }
            Text(String(format: "%.1f", food.rating))
                .font(.caption.weight(.semibold))
                .foregroundColor(Color(red: 0.2, green: 0.2, blue: 0.2))
            Text("(\(food.reviewCount.formatted()) reviews)")
                .font(.caption)
                .foregroundColor(.secondary)
        }
    }

    private var distancePriceRow: some View {
        HStack(spacing: 4) {
            Image(systemName: "location.fill")
                .font(.caption2)
                .foregroundColor(.secondary)
            if let distM = food.distanceM {
                Text(formatDistance(distM))
                    .font(.caption)
                    .foregroundColor(.secondary)
            }
            if let price = food.priceRange {
                Text("·")
                    .font(.caption)
                    .foregroundColor(.secondary)
                Text(price)
                    .font(.caption.weight(.semibold))
                    .foregroundColor(Color(red: 0.2, green: 0.2, blue: 0.2))
            }
            if let neighbourhood = food.neighbourhood, !neighbourhood.isEmpty {
                Text("·")
                    .font(.caption)
                    .foregroundColor(.secondary)
                Text(neighbourhood)
                    .font(.caption)
                    .foregroundColor(.secondary)
                    .lineLimit(1)
            }
        }
    }

    private var mustTryStrip: some View {
        HStack(spacing: 4) {
            Text("Try:")
                .font(.caption.weight(.semibold))
                .foregroundColor(.secondary)
            Text(food.mustTry.joined(separator: " · "))
                .font(.caption)
                .italic()
                .foregroundColor(.secondary)
                .lineLimit(1)
        }
    }

    private func formatDistance(_ metres: Double) -> String {
        if metres < 1000 {
            return "\(Int(metres))m away"
        } else {
            let km = metres / 1000
            return String(format: "%.1fkm away", km)
        }
    }
}

// MARK: - Restaurant Footer

private struct RestaurantFooter: View {
    let post: Post
    let coral: Color
    @Binding var activeReaction: String?
    @AppStorage var isBookmarked: Bool
    @EnvironmentObject private var apiService: APIService

    init(post: Post, coral: Color, activeReaction: Binding<String?>) {
        self.post = post
        self.coral = coral
        self._activeReaction = activeReaction
        self._isBookmarked = AppStorage(wrappedValue: false, "bookmark_\(post.id)")
    }

    var body: some View {
        HStack(spacing: 8) {
            // Yelp link
            if let yelpUrlStr = post.foodData?.yelpUrl, let yelpUrl = URL(string: yelpUrlStr) {
                Link(destination: yelpUrl) {
                    HStack(spacing: 4) {
                        Image(systemName: "arrow.up.right.square")
                            .font(.caption2)
                        Text("Open in Yelp")
                            .font(.caption2.weight(.medium))
                    }
                    .foregroundColor(coral)
                }
            }

            Spacer()

            ReactionPicker(
                activeReaction: $activeReaction,
                postID: post.id,
                style: .feedCompact
            )

            Button {
                UIImpactFeedbackGenerator(style: .light).impactOccurred()
                isBookmarked.toggle()
            } label: {
                Image(systemName: isBookmarked ? "bookmark.fill" : "bookmark")
                    .font(.caption)
                    .foregroundColor(isBookmarked ? coral : .secondary)
                    .contentTransition(.symbolEffect(.replace))
            }
            .buttonStyle(.plain)
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 10)
        .background(Color(red: 0.980, green: 0.980, blue: 0.969)) // warmBg
    }
}
```

**Step 2: Build to verify no Swift compile errors**

**Step 3: Commit**
```bash
git add beepbopboop/beepbopboop/Views/FoodCards.swift
git commit -m "feat: add RestaurantCard view for food posts"
```

---

### Task 5: iOS — Wire up FeedItemView

**Files:**
- Modify: `beepbopboop/beepbopboop/Views/FeedItemView.swift`

**Step 1: Add `.restaurant` to special-style array (line 8)**

Change:
```swift
if [.outfit, .weather, .scoreboard, .matchup, .standings].contains(post.displayHintValue) {
```
To:
```swift
if [.outfit, .weather, .scoreboard, .matchup, .standings, .restaurant].contains(post.displayHintValue) {
```

**Step 2: Add `.restaurant` case in cardContent switch (after `.standings` case)**

```swift
case .restaurant:
    if let card = RestaurantCard(post: post) {
        card
    } else {
        StandardCard(post: post)
    }
```

**Step 3: Build to verify no Swift compile errors**

**Step 4: Commit**
```bash
git add beepbopboop/beepbopboop/Views/FeedItemView.swift
git commit -m "feat: wire RestaurantCard into FeedItemView"
```

---

### Task 6: Skill file

**Files:**
- Create: `.claude/skills/beepbopboop-food/SKILL.md`

**Step 1: Create the skill directory and SKILL.md with full content per issue #55 spec**

**Step 2: Commit**
```bash
git add .claude/skills/beepbopboop-food/SKILL.md
git commit -m "feat: add beepbopboop-food Claude skill"
```

---

### Task 7: Create PR

```bash
git push -u origin claude/stupefied-hertz-7e10e0
gh pr create --title "feat: food skill + RestaurantCard iOS view (#55)" \
  --body "..."
```

Update issue #55 with progress comment.
