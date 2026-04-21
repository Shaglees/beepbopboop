# Graph Report - .  (2026-04-13)

## Corpus Check
- 48 files · ~33,311 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 271 nodes · 268 edges · 40 communities detected
- Extraction: 100% EXTRACTED · 0% INFERRED · 0% AMBIGUOUS
- Token cost: 0 input · 0 output

## Community Hubs (Navigation)
- [[_COMMUNITY_Community 0|Community 0]]
- [[_COMMUNITY_Community 1|Community 1]]
- [[_COMMUNITY_Community 2|Community 2]]
- [[_COMMUNITY_Community 3|Community 3]]
- [[_COMMUNITY_Community 4|Community 4]]
- [[_COMMUNITY_Community 5|Community 5]]
- [[_COMMUNITY_Community 6|Community 6]]
- [[_COMMUNITY_Community 7|Community 7]]
- [[_COMMUNITY_Community 8|Community 8]]
- [[_COMMUNITY_Community 9|Community 9]]
- [[_COMMUNITY_Community 10|Community 10]]
- [[_COMMUNITY_Community 11|Community 11]]
- [[_COMMUNITY_Community 12|Community 12]]
- [[_COMMUNITY_Community 13|Community 13]]
- [[_COMMUNITY_Community 14|Community 14]]
- [[_COMMUNITY_Community 15|Community 15]]
- [[_COMMUNITY_Community 16|Community 16]]
- [[_COMMUNITY_Community 17|Community 17]]
- [[_COMMUNITY_Community 18|Community 18]]
- [[_COMMUNITY_Community 19|Community 19]]
- [[_COMMUNITY_Community 20|Community 20]]
- [[_COMMUNITY_Community 21|Community 21]]
- [[_COMMUNITY_Community 22|Community 22]]
- [[_COMMUNITY_Community 23|Community 23]]
- [[_COMMUNITY_Community 24|Community 24]]
- [[_COMMUNITY_Community 25|Community 25]]
- [[_COMMUNITY_Community 26|Community 26]]
- [[_COMMUNITY_Community 27|Community 27]]
- [[_COMMUNITY_Community 28|Community 28]]
- [[_COMMUNITY_Community 29|Community 29]]
- [[_COMMUNITY_Community 30|Community 30]]
- [[_COMMUNITY_Community 31|Community 31]]
- [[_COMMUNITY_Community 32|Community 32]]
- [[_COMMUNITY_Community 33|Community 33]]
- [[_COMMUNITY_Community 34|Community 34]]
- [[_COMMUNITY_Community 35|Community 35]]
- [[_COMMUNITY_Community 36|Community 36]]
- [[_COMMUNITY_Community 37|Community 37]]
- [[_COMMUNITY_Community 38|Community 38]]
- [[_COMMUNITY_Community 39|Community 39]]

## God Nodes (most connected - your core abstractions)
1. `CodingKeys` - 20 edges
2. `SettingsViewModel` - 11 edges
3. `PostRepo` - 7 edges
4. `CodingKeys` - 7 edges
5. `scanPost()` - 6 edges
6. `beepbopboopUITests` - 6 edges
7. `FeedListViewModel` - 6 edges
8. `PostTypeValue` - 6 edges
9. `LinkableText` - 6 edges
10. `APIError` - 6 edges

## Surprising Connections (you probably didn't know these)
- `SettingsViewModel` --inherits--> `ObservableObject`  [EXTRACTED]
  beepbopboop/beepbopboop/Views/SettingsView.swift →   _Bridges community 1 → community 7_
- `SettingsView` --inherits--> `View`  [EXTRACTED]
  beepbopboop/beepbopboop/Views/SettingsView.swift →   _Bridges community 7 → community 2_

## Communities

### Community 0 - "Community 0"
Cohesion: 0.08
Nodes (25): CodingKey, CodingKeys, agentID, agentName, body, createdAt, externalURL, id (+17 more)

### Community 1 - "Community 1"
Cohesion: 0.12
Nodes (6): AuthService, Equatable, APIService.APIError, FeedListViewModel, FeedViewModel, ObservableObject

### Community 2 - "Community 2"
Cohesion: 0.12
Nodes (6): FeedItemView, FeedListView, FeedView, LoginView, PostDetailView, View

### Community 3 - "Community 3"
Cohesion: 0.15
Nodes (11): APIError, httpError, invalidResponse, invalidURL, locationRequired, APIService, FeedType, community (+3 more)

### Community 4 - "Community 4"
Cohesion: 0.27
Nodes (7): formatCursor(), nullFloat64(), nullString(), parseCursorString(), scanPost(), CreatePostParams, PostRepo

### Community 5 - "Community 5"
Cohesion: 0.17
Nodes (11): Codable, Identifiable, FeedResponse, Post, PostTypeValue, article, discovery, event (+3 more)

### Community 6 - "Community 6"
Cohesion: 0.17
Nodes (0): 

### Community 7 - "Community 7"
Cohesion: 0.18
Nodes (4): MKLocalSearchCompleterDelegate, NSObject, SettingsView, SettingsViewModel

### Community 8 - "Community 8"
Cohesion: 0.18
Nodes (3): beepbopboopUITests, beepbopboopUITestsLaunchTests, XCTestCase

### Community 9 - "Community 9"
Cohesion: 0.29
Nodes (2): HealthHandler, MeHandler

### Community 10 - "Community 10"
Cohesion: 0.29
Nodes (6): Agent, AgentToken, FeedResponse, Post, User, UserSettings

### Community 11 - "Community 11"
Cohesion: 0.29
Nodes (2): LinkableText, UIViewRepresentable

### Community 12 - "Community 12"
Cohesion: 0.33
Nodes (1): MultiFeedHandler

### Community 13 - "Community 13"
Cohesion: 0.33
Nodes (2): SettingsHandler, updateSettingsRequest

### Community 14 - "Community 14"
Cohesion: 0.47
Nodes (2): writeJSON(), AgentHandler

### Community 15 - "Community 15"
Cohesion: 0.4
Nodes (3): AgentAuth(), writeJSON(), contextKey

### Community 16 - "Community 16"
Cohesion: 0.4
Nodes (1): AgentRepo

### Community 17 - "Community 17"
Cohesion: 0.4
Nodes (2): createPostRequest, PostHandler

### Community 18 - "Community 18"
Cohesion: 0.5
Nodes (1): UserSettingsRepo

### Community 19 - "Community 19"
Cohesion: 0.4
Nodes (0): 

### Community 20 - "Community 20"
Cohesion: 0.5
Nodes (2): UserRepo, generateID()

### Community 21 - "Community 21"
Cohesion: 0.5
Nodes (3): Config, envOr(), Load()

### Community 22 - "Community 22"
Cohesion: 0.5
Nodes (1): FeedHandler

### Community 23 - "Community 23"
Cohesion: 0.83
Nodes (3): setupAgentTest(), TestAgentHandler_CreateAgent(), TestAgentHandler_CreateToken()

### Community 24 - "Community 24"
Cohesion: 0.5
Nodes (0): 

### Community 25 - "Community 25"
Cohesion: 0.5
Nodes (0): 

### Community 26 - "Community 26"
Cohesion: 0.5
Nodes (0): 

### Community 27 - "Community 27"
Cohesion: 0.5
Nodes (0): 

### Community 28 - "Community 28"
Cohesion: 0.83
Nodes (3): BoundingBox(), HaversineKm(), toRad()

### Community 29 - "Community 29"
Cohesion: 0.5
Nodes (0): 

### Community 30 - "Community 30"
Cohesion: 0.67
Nodes (0): 

### Community 31 - "Community 31"
Cohesion: 0.67
Nodes (0): 

### Community 32 - "Community 32"
Cohesion: 0.67
Nodes (0): 

### Community 33 - "Community 33"
Cohesion: 0.67
Nodes (1): beepbopboopTests

### Community 34 - "Community 34"
Cohesion: 0.67
Nodes (2): App, beepbopboopApp

### Community 35 - "Community 35"
Cohesion: 1.0
Nodes (0): 

### Community 36 - "Community 36"
Cohesion: 1.0
Nodes (0): 

### Community 37 - "Community 37"
Cohesion: 1.0
Nodes (0): 

### Community 38 - "Community 38"
Cohesion: 1.0
Nodes (0): 

### Community 39 - "Community 39"
Cohesion: 1.0
Nodes (0): 

## Knowledge Gaps
- **42 isolated node(s):** `createPostRequest`, `updateSettingsRequest`, `contextKey`, `CreatePostParams`, `User` (+37 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **Thin community `Community 35`** (2 nodes): `main.go`, `main()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 36`** (2 nodes): `pagination.go`, `parsePagination()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 37`** (2 nodes): `database.go`, `Open()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 38`** (2 nodes): `database_test.go`, `TestOpenAndMigrate()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 39`** (2 nodes): `user_repo_test.go`, `TestUserRepo_FindOrCreateByFirebaseUID()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `SettingsViewModel` connect `Community 7` to `Community 1`?**
  _High betweenness centrality (0.018) - this node is a cross-community bridge._
- **Why does `CodingKeys` connect `Community 0` to `Community 5`?**
  _High betweenness centrality (0.015) - this node is a cross-community bridge._
- **Why does `SettingsView` connect `Community 7` to `Community 2`?**
  _High betweenness centrality (0.013) - this node is a cross-community bridge._
- **What connects `createPostRequest`, `updateSettingsRequest`, `contextKey` to the rest of the system?**
  _42 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `Community 0` be split into smaller, more focused modules?**
  _Cohesion score 0.08 - nodes in this community are weakly interconnected._
- **Should `Community 1` be split into smaller, more focused modules?**
  _Cohesion score 0.12 - nodes in this community are weakly interconnected._
- **Should `Community 2` be split into smaller, more focused modules?**
  _Cohesion score 0.12 - nodes in this community are weakly interconnected._