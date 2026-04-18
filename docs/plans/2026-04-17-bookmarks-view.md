# Bookmarks View Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a Saved Posts tab to the iOS app, backed by a new `GET /posts/saved` backend endpoint, and wire the bookmark button to fire save/unsave events to the backend.

**Architecture:** The backend gets a new `ListSaved` repo method and `GetSaved` handler on the Firebase-auth route group. The iOS `FeedType` enum gains a `.saved` case so the existing `FeedListViewModel` and `FeedListView` can be reused unchanged. The `CardFooter` bookmark button is updated to fire events via `APIService` in addition to toggling `@AppStorage`.

**Tech Stack:** Go (chi, database/sql, PostgreSQL), Swift/SwiftUI, Firebase Auth

---

## Task 1: Backend — `ListSaved` repo method

**Files:**
- Modify: `backend/internal/repository/post_repo.go`

**Context:**
- The cursor format is `"<RFC3339>|<seq>"` — but for saved posts the ordering column is `saved_at` (MAX of save-event `created_at`), not `p.created_at`.
- The cursor for saved posts should use `saved_at|seq` so pagination is stable.
- `postColumns` selects `p.seq` as the last column; `scanPost` returns it as the second value.
- `unsave` events invalidate a `save`: a post is "saved" iff the most-recent event for that (post, user) pair is `save`, i.e. there is no later `unsave`.

**Step 1: Add `ListSaved` to `post_repo.go`**

Append this method after `ListPersonal` (around line 261):

```go
// ListSaved returns posts saved by userID with cursor-based pagination.
// Ordering is by saved_at (most-recently saved first).
// The cursor format reuses "RFC3339|seq" where the timestamp is saved_at.
func (r *PostRepo) ListSaved(userID, cursor string, limit int) ([]model.Post, *string, error) {
	args := []any{userID}
	cursorClause := ""
	argIdx := 2

	if cursor != "" {
		t, seq, err := parseCursorString(cursor)
		if err != nil {
			return nil, nil, err
		}
		cursorClause = fmt.Sprintf(
			" AND (MAX(pe.created_at) < $%d OR (MAX(pe.created_at) = $%d AND p.seq < $%d))",
			argIdx, argIdx+1, argIdx+2,
		)
		args = append(args, t, t, seq)
		argIdx += 3
	}
	args = append(args, limit)

	query := fmt.Sprintf(`
		SELECT `+postColumns+`, MAX(pe.created_at) AS saved_at
		FROM posts p
		JOIN agents a ON a.id = p.agent_id
		JOIN post_events pe ON pe.post_id = p.id
			AND pe.user_id = $1
			AND pe.event_type = 'save'
		LEFT JOIN post_events unsave ON unsave.post_id = p.id
			AND unsave.user_id = $1
			AND unsave.event_type = 'unsave'
			AND unsave.created_at > pe.created_at
		WHERE unsave.post_id IS NULL
		GROUP BY p.id, a.id%s
		ORDER BY saved_at DESC, p.seq DESC
		LIMIT $%d`, cursorClause, argIdx)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, nil, fmt.Errorf("query saved feed: %w", err)
	}
	defer rows.Close()

	posts := make([]model.Post, 0)
	var lastSavedAt time.Time
	var lastSeq int64
	for rows.Next() {
		p, seq, err := scanPost(rows)
		if err != nil {
			return nil, nil, fmt.Errorf("scan post: %w", err)
		}
		// consume the extra saved_at column
		var savedAt time.Time
		// scanPost already consumed all postColumns; saved_at is next
		// We need to scan it separately — restructure to use a custom scan
		_ = savedAt
		posts = append(posts, p)
		lastSeq = seq
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("iterate posts: %w", err)
	}

	var nextCursor *string
	if len(posts) >= limit {
		c := formatCursor(lastSavedAt, lastSeq)
		nextCursor = &c
	}
	return posts, nextCursor, nil
}
```

**Wait — the query selects `postColumns + saved_at`. `scanPost` only scans `postColumns`. We need to also scan `saved_at` per row.** The cleanest approach is a local loop that calls `scanPost` then reads one more column:

```go
func (r *PostRepo) ListSaved(userID, cursor string, limit int) ([]model.Post, *string, error) {
	args := []any{userID}
	cursorClause := ""
	argIdx := 2

	if cursor != "" {
		t, seq, err := parseCursorString(cursor)
		if err != nil {
			return nil, nil, err
		}
		cursorClause = fmt.Sprintf(
			" HAVING (MAX(pe.created_at) < $%d OR (MAX(pe.created_at) = $%d AND p.seq < $%d))",
			argIdx, argIdx+1, argIdx+2,
		)
		args = append(args, t, t, seq)
		argIdx += 3
	}
	args = append(args, limit)

	query := fmt.Sprintf(`
		SELECT `+postColumns+`, MAX(pe.created_at) AS saved_at
		FROM posts p
		JOIN agents a ON a.id = p.agent_id
		JOIN post_events pe ON pe.post_id = p.id
			AND pe.user_id = $1
			AND pe.event_type = 'save'
		LEFT JOIN post_events unsave ON unsave.post_id = p.id
			AND unsave.user_id = $1
			AND unsave.event_type = 'unsave'
			AND unsave.created_at > pe.created_at
		WHERE unsave.post_id IS NULL
		GROUP BY p.id, a.id%s
		ORDER BY saved_at DESC, p.seq DESC
		LIMIT $%d`, cursorClause, argIdx)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, nil, fmt.Errorf("query saved feed: %w", err)
	}
	defer rows.Close()

	type savedRow struct {
		post    model.Post
		seq     int64
		savedAt time.Time
	}
	var scanned []savedRow
	for rows.Next() {
		var p model.Post
		var imageURL, externalURL, locality, postType, labelsJSON, imagesJSON sql.NullString
		var latitude, longitude sql.NullFloat64
		var scheduledAt sql.NullTime
		var seq int64
		var savedAt time.Time
		err := rows.Scan(
			&p.ID, &p.AgentID, &p.AgentName, &p.UserID,
			&p.Title, &p.Body,
			&imageURL, &externalURL, &locality, &latitude, &longitude,
			&postType, &p.Visibility, &p.DisplayHint, &labelsJSON, &imagesJSON,
			&p.Status, &scheduledAt, &p.CreatedAt, &seq,
			&savedAt,
		)
		if err != nil {
			return nil, nil, fmt.Errorf("scan saved post: %w", err)
		}
		p.ImageURL = imageURL.String
		p.ExternalURL = externalURL.String
		p.Locality = locality.String
		if latitude.Valid {
			p.Latitude = &latitude.Float64
		}
		if longitude.Valid {
			p.Longitude = &longitude.Float64
		}
		p.PostType = postType.String
		if labelsJSON.Valid {
			json.Unmarshal([]byte(labelsJSON.String), &p.Labels)
		}
		if imagesJSON.Valid {
			p.Images = json.RawMessage(imagesJSON.String)
		}
		if scheduledAt.Valid {
			p.ScheduledAt = &scheduledAt.Time
		}
		scanned = append(scanned, savedRow{post: p, seq: seq, savedAt: savedAt})
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("iterate saved posts: %w", err)
	}

	posts := make([]model.Post, len(scanned))
	for i, r := range scanned {
		posts[i] = r.post
	}

	var nextCursor *string
	if len(scanned) >= limit {
		last := scanned[len(scanned)-1]
		c := formatCursor(last.savedAt, last.seq)
		nextCursor = &c
	}
	return posts, nextCursor, nil
}
```

**Step 2: Build the backend to check compilation**

```bash
cd backend && go build ./...
```

Expected: no errors.

**Step 3: Commit**

```bash
git add backend/internal/repository/post_repo.go
git commit -m "feat: add ListSaved repo method for saved posts feed"
```

---

## Task 2: Backend — `GetSaved` handler and route

**Files:**
- Modify: `backend/internal/handler/multi_feed.go`
- Modify: `backend/cmd/server/main.go`

**Step 1: Add `GetSaved` to `MultiFeedHandler` in `multi_feed.go`**

Append after `GetForYou`:

```go
func (h *MultiFeedHandler) GetSaved(w http.ResponseWriter, r *http.Request) {
	uid := middleware.FirebaseUIDFromContext(r.Context())

	user, err := h.userRepo.FindOrCreateByFirebaseUID(uid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to resolve user"})
		return
	}

	cursor, limit, err := parsePagination(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_cursor"})
		return
	}

	posts, nextCursor, err := h.postRepo.ListSaved(user.ID, cursor, limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load saved feed"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(model.FeedResponse{Posts: posts, NextCursor: nextCursor})
}
```

**Step 2: Register the route in `main.go`**

In the Firebase-authenticated route group (around line 102), add after `r.Get("/feeds/foryou", ...)`:

```go
r.Get("/posts/saved", multiFeedH.GetSaved)
```

**Step 3: Build the backend**

```bash
cd backend && go build ./...
```

Expected: no errors.

**Step 4: Commit**

```bash
git add backend/internal/handler/multi_feed.go backend/cmd/server/main.go
git commit -m "feat: add GET /posts/saved endpoint for saved posts feed"
```

---

## Task 3: iOS — Add `trackEvent` to `APIService`

The bookmark button needs to fire `save`/`unsave` events. `POST /posts/{postID}/events` already exists and accepts `{"event_type": "save"}`.

**Files:**
- Modify: `beepbopboop/beepbopboop/Services/APIService.swift`

**Step 1: Add `trackEvent` method**

Add after `removeReaction` (before the `APIError` enum, around line 163):

```swift
// MARK: - Events

@MainActor
func trackEvent(postID: String, eventType: String) async {
    guard let token = try? authService.getToken() else { return }
    guard let url = URL(string: "\(baseURL)/posts/\(postID)/events") else { return }
    var request = URLRequest(url: url)
    request.httpMethod = "POST"
    request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
    request.setValue("application/json", forHTTPHeaderField: "Content-Type")
    request.httpBody = try? JSONEncoder().encode(["event_type": eventType])
    _ = try? await URLSession.shared.data(for: request)
}
```

**Note on `authService.getToken()`:** Check how `getToken()` is called elsewhere in `APIService.swift` — it is called as `authService.getToken()` (non-throwing). If it doesn't throw, remove the `try?`:

```swift
let token = authService.getToken()
```

**Step 2: Add `.saved` FeedType case**

In `APIService.swift`, extend `FeedType`:

```swift
enum FeedType {
    case forYou, community, personal, saved

    var path: String {
        switch self {
        case .forYou: return "/feeds/foryou"
        case .community: return "/feeds/community"
        case .personal: return "/feeds/personal"
        case .saved: return "/posts/saved"
        }
    }
}
```

**Step 3: Build iOS app to check compilation**

```bash
cd beepbopboop && xcodebuild -scheme beepbopboop -destination 'platform=iOS Simulator,name=iPhone 16' build 2>&1 | tail -20
```

Expected: `BUILD SUCCEEDED`

**Step 4: Commit**

```bash
git add beepbopboop/beepbopboop/Services/APIService.swift
git commit -m "feat: add trackEvent API method and .saved FeedType"
```

---

## Task 4: iOS — Wire bookmark button to fire backend events

**Files:**
- Modify: `beepbopboop/beepbopboop/Views/FeedItemView.swift`

**Context:**
- `CardFooter` (line 99) uses `@AppStorage` for `isBookmarked` and `@EnvironmentObject private var apiService: APIService`.
- The button at line 128 toggles `isBookmarked` — we need to also call `apiService.trackEvent`.

**Step 1: Update the bookmark button in `CardFooter`**

Replace the Button action (lines 128–137):

```swift
Button {
    UIImpactFeedbackGenerator(style: .light).impactOccurred()
    isBookmarked.toggle()
    let eventType = isBookmarked ? "save" : "unsave"
    Task { await apiService.trackEvent(postID: post.id, eventType: eventType) }
} label: {
    Image(systemName: isBookmarked ? "bookmark.fill" : "bookmark")
        .font(.caption)
        .foregroundColor(isBookmarked ? post.hintColor : .secondary)
        .contentTransition(.symbolEffect(.replace))
}
.buttonStyle(.plain)
```

**Step 2: Update the `OutfitBookmarkButton` similarly**

Find `OutfitBookmarkButton` (around line 928). It also has a Button that toggles a bookmarked state. Apply the same pattern — add `Task { await apiService.trackEvent(postID: post.id, eventType: isBookmarked ? "save" : "unsave") }` after the toggle.

**Step 3: Build**

```bash
cd beepbopboop && xcodebuild -scheme beepbopboop -destination 'platform=iOS Simulator,name=iPhone 16' build 2>&1 | tail -20
```

Expected: `BUILD SUCCEEDED`

**Step 4: Commit**

```bash
git add beepbopboop/beepbopboop/Views/FeedItemView.swift
git commit -m "feat: wire bookmark button to fire save/unsave events to backend"
```

---

## Task 5: iOS — Add Saved FeedListViewModel and Saved tab

**Files:**
- Modify: `beepbopboop/beepbopboop/Views/FeedView.swift`
- Modify: `beepbopboop/beepbopboop/ViewModels/FeedListViewModel.swift`

### 5a: Update `FeedListViewModel` empty message for `.saved`

In `FeedListViewModel.swift`, update `emptyMessage`:

```swift
var emptyMessage: String {
    switch feedType {
    case .personal: return "Your agent hasn't posted anything yet."
    case .community: return "No posts from your community yet."
    case .forYou: return "Nothing here yet. Check back soon!"
    case .saved: return "Nothing saved yet — tap the bookmark icon on any post."
    }
}
```

### 5b: Add Saved tab to `FeedView`

**Step 1: Add `savedVM` StateObject**

In `FeedView`, add after `personalVM`:

```swift
@StateObject private var savedVM: FeedListViewModel
```

In `init`, add:

```swift
_savedVM = StateObject(wrappedValue: FeedListViewModel(feedType: .saved, apiService: apiService))
```

**Step 2: Add the Saved tab page**

In the `TabView`, after the Personal tab (tag: 2):

```swift
SavedFeedView(viewModel: savedVM, isHeaderVisible: $isHeaderVisible)
    .tag(3)
    .task { if savedVM.posts.isEmpty && !savedVM.isLoading { await savedVM.refresh() } }
```

**Step 3: Add "Saved" tab button to the tab bar**

In `tabBar`, after `tabButton("Personal", tag: 2)`:

```swift
tabButton("Saved", tag: 3, systemImage: "bookmark")
```

Update `tabButton` to optionally show a system image:

```swift
private func tabButton(_ title: String, tag: Int, systemImage: String? = nil) -> some View {
    Button {
        withAnimation(.bouncy) {
            selectedTab = tag
        }
    } label: {
        HStack(spacing: 4) {
            if let systemImage {
                Image(systemName: selectedTab == tag ? systemImage + ".fill" : systemImage)
                    .font(.subheadline)
            }
            Text(title)
                .font(.subheadline.weight(selectedTab == tag ? .semibold : .regular))
        }
        .foregroundStyle(selectedTab == tag ? .primary : .secondary)
        .padding(.horizontal, 16)
        .padding(.vertical, 8)
        .glassEffect(
            selectedTab == tag ? .regular.tint(.accentColor).interactive() : .regular,
            in: .capsule
        )
    }
    .buttonStyle(.plain)
}
```

**Step 4: Add `SavedFeedView`**

Create a minimal wrapper that reuses `FeedListView` but with a no-op `onSettingsTapped` (the Saved tab doesn't need settings access):

Add in `FeedView.swift` (or as a new file — inline in FeedView.swift is fine since it's small):

```swift
private struct SavedFeedView: View {
    @ObservedObject var viewModel: FeedListViewModel
    @Binding var isHeaderVisible: Bool

    var body: some View {
        FeedListView(
            viewModel: viewModel,
            isHeaderVisible: $isHeaderVisible,
            onSettingsTapped: {}
        )
    }
}
```

**Step 5: Build**

```bash
cd beepbopboop && xcodebuild -scheme beepbopboop -destination 'platform=iOS Simulator,name=iPhone 16' build 2>&1 | tail -20
```

Expected: `BUILD SUCCEEDED`

**Step 6: Commit**

```bash
git add beepbopboop/beepbopboop/Views/FeedView.swift beepbopboop/beepbopboop/ViewModels/FeedListViewModel.swift
git commit -m "feat: add Saved tab to feed view"
```

---

## Task 6: Final build verification + PR

**Step 1: Full backend build**

```bash
cd backend && go build ./... && go vet ./...
```

Expected: no output (clean).

**Step 2: Full iOS build**

```bash
cd beepbopboop && xcodebuild -scheme beepbopboop -destination 'platform=iOS Simulator,name=iPhone 16' build 2>&1 | grep -E 'error:|BUILD'
```

Expected: `BUILD SUCCEEDED` with no `error:` lines.

**Step 3: Comment on issue**

```bash
gh issue comment 31 --body "Implementation complete on branch \`claude/beautiful-cray-f9fc5b\`:

- **Backend:** Added \`GET /posts/saved\` (Firebase-auth, cursor-paginated) — returns posts the user has saved where no later \`unsave\` event exists.
- **iOS:** Bookmark button now fires \`save\`/\`unsave\` events to the backend (alongside existing local \`@AppStorage\` state).
- **iOS:** Added a 4th **Saved** tab reusing \`FeedListViewModel\` + \`FeedListView\` — pull-to-refresh, pagination, empty state all included.
- Both backend and iOS build clean."
```

**Step 4: Create PR**

```bash
gh pr create \
  --title "feat: bookmarks view — saved posts tab and backend endpoint" \
  --body "$(cat <<'EOF'
## Summary

- Adds `GET /posts/saved` backend endpoint (Firebase-auth, cursor-paginated via `post_events`)
- Wires iOS bookmark button to fire `save`/`unsave` events to the backend
- Adds a **Saved** tab (4th tab) reusing the existing feed infrastructure

Closes #31

## Details

**Backend (`backend/`)**
- `PostRepo.ListSaved` — queries `post_events` for saves with no subsequent unsave, ordered by `saved_at DESC`
- `MultiFeedHandler.GetSaved` — handler following the same pattern as `GetPersonal`
- Route: `GET /posts/saved` added to Firebase-auth group

**iOS (`beepbopboop/`)**
- `APIService.trackEvent` — fires `POST /posts/{id}/events` (fire-and-forget)
- `FeedType.saved` case mapping to `/posts/saved`
- `CardFooter` bookmark button fires `save`/`unsave` event on toggle
- `SavedFeedView` + `savedVM` in `FeedView` — 4th tab with bookmark icon

## Notes

- Saved state on cards remains local-only (`@AppStorage`) for now — cross-device sync deferred
- Empty state: "Nothing saved yet — tap the bookmark icon on any post"
EOF
)"
```
