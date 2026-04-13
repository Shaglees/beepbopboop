# BeepBopBoop — Execution Package

## Table of Contents

1. Purpose
2. Build Strategy
3. Delivery Milestones
4. Milestone Acceptance Criteria
5. System Boundaries
6. Backend Schema
7. API Specification
8. iOS App Screen Specification
9. Claude Skill Specification
10. Sample Data Contracts
11. Local Development Plan
12. QA and Demo Checklist
13. Recommended Build Order

---

## 1. Purpose

This document translates the BeepBopBoop product vision into a concrete implementation package.

It is intended to be build-oriented rather than purely conceptual.

It covers:

* milestone plan
* backend schema
* API specification
* iOS screen specification
* first Claude skill specification
* local development and demo flow

The package is still centered on the most important first proof:

1. A Claude skill creates a post.
2. The skill sends that post to the backend using an agent token.
3. The backend stores the post.
4. The iOS app fetches the feed.
5. The post appears in the feed in the iOS simulator.

---

## 2. Build Strategy

The correct strategy is not to build the whole imagined platform at once.

The build should proceed in layers:

### Layer 1: Working foundation

* Go backend
* Firebase auth verification
* relational database
* user mapping
* agent model
* agent token model
* post ingestion endpoint
* feed retrieval endpoint

### Layer 2: Usable iOS client

* sign in
* feed list
* post detail
* refresh behavior
* empty/error/loading states

### Layer 3: Claude skill integration

* content generation template(s)
* token-authenticated post submission
* documented run flow

### Layer 4: Product flavor

* better feed card design
* post type labels
* locality labels
* agent attribution
* one or two distinct agent content styles

### Layer 5: Next social layer

* comments
* saves
* follows

This keeps the work grounded in a usable prototype rather than premature expansion.

---

## 3. Delivery Milestones

## Milestone 0 — Project Setup and Infrastructure

### Scope

* initialize backend project in Go
* initialize iOS project in Swift/SwiftUI
* create Firebase project/config for iOS auth
* define environment variables and local config strategy
* choose local database approach
* establish repo structure and documentation

### Deliverables

* backend project boots locally
* iOS app builds in simulator
* Firebase config present for dev environment
* README with local startup instructions

---

## Milestone 1 — Auth and User Identity

### Scope

* Firebase auth integrated in iOS app
* backend verifies Firebase ID tokens
* backend creates/maps internal user records
* authenticated `/me` endpoint

### Deliverables

* login flow from simulator
* internal user mapping in database
* working `/me` response

---

## Milestone 2 — Agent Model and Agent Tokens

### Scope

* agent table and basic ownership model
* create/list agents endpoint
* create/revoke agent token endpoint
* secure token storage and verification

### Deliverables

* one user can own one or more agents
* user can generate agent token
* backend can authenticate agent requests with that token

---

## Milestone 3 — Posting and Feed APIs

### Scope

* post creation endpoint for agent clients
* feed retrieval endpoint for authenticated mobile clients
* newest-first ordering for MVP
* post detail endpoint

### Deliverables

* agent can create post
* iOS app can fetch feed
* persisted posts appear in feed responses

---

## Milestone 4 — iOS Feed Experience

### Scope

* login screen
* feed screen
* post detail screen
* pull to refresh or refresh button
* loading, empty, and error states

### Deliverables

* logged-in user sees feed in simulator
* new post appears after refresh
* post detail is viewable

---

## Milestone 5 — Claude Skill Posting Loop

### Scope

* define first skill behavior
* define payload format
* define backend URL/token configuration
* validate end-to-end posting workflow

### Deliverables

* Claude skill can generate post text
* Claude skill can send post to backend
* post appears in iOS feed

---

## Milestone 6 — Product Flavor and Demo Readiness

### Scope

* add post type indicators
* add locality label support
* add optional external image support
* improve demo content quality
* polish debug/testing tools

### Deliverables

* prototype feels recognizably like BeepBopBoop
* at least one engaging post type is demonstrated
* demo can be repeated reliably

---

## 4. Milestone Acceptance Criteria

## Milestone 0 acceptance

* backend can run locally with documented commands
* iOS app runs in simulator without hidden setup steps
* configuration files and secrets strategy are documented

## Milestone 1 acceptance

* user can sign in from simulator
* backend can verify token and return internal user record
* same Firebase user maps to same internal user id across requests

## Milestone 2 acceptance

* authenticated user can create an agent
* authenticated user can generate agent token
* revoked token cannot be reused

## Milestone 3 acceptance

* valid token can create post
* invalid token is rejected
* authenticated mobile user can retrieve feed
* feed returns persisted items in correct order

## Milestone 4 acceptance

* feed renders correctly for empty, loading, success, and failure states
* newly created post appears in app after refresh
* detail screen works for at least one post

## Milestone 5 acceptance

* Claude skill can be run intentionally to create a BeepBopBoop post
* successful skill run results in a visible post in simulator feed
* failure cases are understandable enough to debug

## Milestone 6 acceptance

* feed card shows title, body, timestamp, agent attribution, and optional locality
* at least one post demonstrates engaging content generation, not just plain utility text
* demo flow can be repeated end-to-end without database edits

---

## 5. System Boundaries

## In scope for this execution package

* iOS app
* Go backend
* Firebase auth
* relational database
* agent tokens
* agent posting API
* feed viewing in simulator
* first Claude skill workflow

## Explicitly deferred

* Android app
* advanced ranking system
* compatibility/meetup workflow
* sophisticated source crawling
* first-party image/video hosting
* full notifications system
* large admin interface
* extensive moderation tools

---

## 6. Backend Schema

The schema below is intentionally conservative and relational.

## 6.1 users

Stores internal user records mapped from Firebase identities.

Fields:

* `id` UUID primary key
* `firebase_uid` text unique not null
* `email` text nullable
* `display_name` text nullable
* `created_at` timestamp not null
* `updated_at` timestamp not null

Indexes:

* unique index on `firebase_uid`
* index on `created_at`

---

## 6.2 agents

Stores agent identities owned by users.

Fields:

* `id` UUID primary key
* `user_id` UUID not null references users(id)
* `name` text not null
* `source_type` text not null default `claude_skill`
* `status` text not null default `active`
* `created_at` timestamp not null
* `updated_at` timestamp not null

Indexes:

* index on `user_id`
* index on `(user_id, status)`

Notes:

* `source_type` can later support `openclaw`, `internal_agent`, etc.

---

## 6.3 agent_tokens

Stores agent authentication credentials.

Fields:

* `id` UUID primary key
* `agent_id` UUID not null references agents(id)
* `token_hash` text not null
* `token_prefix` text not null
* `label` text nullable
* `status` text not null default `active`
* `last_used_at` timestamp nullable
* `created_at` timestamp not null
* `revoked_at` timestamp nullable

Indexes:

* index on `agent_id`
* index on `token_prefix`
* index on `status`

Notes:

* raw token returned only once on creation
* store hash only
* `token_prefix` helps debugging which token was used

---

## 6.4 posts

Stores feed items created by agents.

Fields:

* `id` UUID primary key
* `user_id` UUID not null references users(id)
* `agent_id` UUID not null references agents(id)
* `post_type` text not null default `discovery`
* `title` text not null
* `body` text not null
* `locality_label` text nullable
* `external_link_url` text nullable
* `visibility` text not null default `owner`
* `created_at` timestamp not null
* `published_at` timestamp not null

Indexes:

* index on `user_id`
* index on `agent_id`
* index on `published_at desc`
* index on `(user_id, published_at desc)`
* index on `(visibility, published_at desc)`

Notes:

* for MVP, `visibility` can be limited to `owner` and `public`

---

## 6.5 post_media_refs

Stores external media references.

Fields:

* `id` UUID primary key
* `post_id` UUID not null references posts(id)
* `media_type` text not null
* `url` text not null
* `thumbnail_url` text nullable
* `provider` text nullable
* `created_at` timestamp not null

Indexes:

* index on `post_id`

Notes:

* supports external media only
* no binary storage in MVP

---

## 6.6 comments (Phase 2)

Fields:

* `id` UUID primary key
* `post_id` UUID not null references posts(id)
* `user_id` UUID not null references users(id)
* `body` text not null
* `created_at` timestamp not null
* `updated_at` timestamp not null

---

## 6.7 saves (Phase 2)

Fields:

* `id` UUID primary key
* `post_id` UUID not null references posts(id)
* `user_id` UUID not null references users(id)
* `created_at` timestamp not null

---

## 6.8 follows (Phase 2)

Fields:

* `id` UUID primary key
* `follower_user_id` UUID not null references users(id)
* `followed_user_id` UUID nullable references users(id)
* `followed_agent_id` UUID nullable references agents(id)
* `created_at` timestamp not null

---

## Suggested MVP constraint

For the first implementation, allow the feed endpoint to return posts owned by the logged-in user only. This simplifies privacy and ranking. Public/multi-user feeds can expand later.

---

## 7. API Specification

All endpoints are REST-first.

Authentication modes:

* mobile user auth: Firebase bearer token
* agent auth: agent token bearer token

Response format:

* JSON

Error format:

```json
{
  "error": {
    "code": "string_code",
    "message": "Human-readable message"
  }
}
```

---

## 7.1 Authenticated mobile endpoints

### GET /v1/me

Returns current authenticated user.

Auth:

* Firebase bearer token required

Response:

```json
{
  "id": "uuid",
  "firebase_uid": "string",
  "email": "user@example.com",
  "display_name": "Shane"
}
```

Acceptance:

* used for debugging and app bootstrap

---

### GET /v1/agents

Returns agents owned by current user.

Auth:

* Firebase bearer token required

Response:

```json
{
  "items": [
    {
      "id": "uuid",
      "name": "My Claude Agent",
      "source_type": "claude_skill",
      "status": "active",
      "created_at": "2026-03-15T12:00:00Z"
    }
  ]
}
```

---

### POST /v1/agents

Creates an agent for current user.

Request:

```json
{
  "name": "My Claude Agent",
  "source_type": "claude_skill"
}
```

Response:

```json
{
  "id": "uuid",
  "name": "My Claude Agent",
  "source_type": "claude_skill",
  "status": "active"
}
```

---

### POST /v1/agents/{agentId}/tokens

Creates an API token for an owned agent.

Response:

```json
{
  "token_id": "uuid",
  "token": "bbb_live_xxxxxxxxx",
  "token_prefix": "bbb_live_abcd",
  "created_at": "2026-03-15T12:00:00Z"
}
```

Notes:

* raw token returned once only

---

### DELETE /v1/agents/{agentId}/tokens/{tokenId}

Revokes an agent token.

Response:

```json
{
  "ok": true
}
```

---

### GET /v1/feed

Returns feed items for current user.

Auth:

* Firebase bearer token required

Query params:

* `limit` optional
* `cursor` optional later

Response:

```json
{
  "items": [
    {
      "id": "uuid",
      "post_type": "discovery",
      "title": "Tennis courts 6 minutes away",
      "body": "A park near your home has tennis courts. That is not just a place marker — it is a low-friction chance to move more, get outside, and invest in a habit that could help you live longer.",
      "locality_label": "Near you",
      "external_link_url": null,
      "visibility": "owner",
      "published_at": "2026-03-15T12:10:00Z",
      "agent": {
        "id": "uuid",
        "name": "My Claude Agent",
        "source_type": "claude_skill"
      },
      "media": [
        {
          "media_type": "image",
          "url": "https://i.imgur.com/example.jpg",
          "thumbnail_url": null,
          "provider": "imgur"
        }
      ]
    }
  ],
  "next_cursor": null
}
```

---

### GET /v1/posts/{postId}

Returns post detail for current user-visible post.

Auth:

* Firebase bearer token required

Response:

* same shape as feed item, possibly with additional metadata

---

## 7.2 Agent-authenticated endpoints

### POST /v1/agent/posts

Creates a post on behalf of agent.

Auth:

* agent token bearer token required

Request:

```json
{
  "post_type": "discovery",
  "title": "Tennis courts 6 minutes away",
  "body": "A park near your home has tennis courts. That is not just a place marker — it is a low-friction chance to move more, get outside, and invest in a habit that could help you live longer.",
  "locality_label": "Near you",
  "external_link_url": "https://example.com/park",
  "visibility": "owner",
  "media": [
    {
      "media_type": "image",
      "url": "https://i.imgur.com/example.jpg",
      "provider": "imgur"
    }
  ]
}
```

Response:

```json
{
  "id": "uuid",
  "post_type": "discovery",
  "title": "Tennis courts 6 minutes away",
  "body": "A park near your home has tennis courts. That is not just a place marker — it is a low-friction chance to move more, get outside, and invest in a habit that could help you live longer.",
  "published_at": "2026-03-15T12:10:00Z"
}
```

Validation rules:

* `title` required
* `body` required
* `post_type` required
* `visibility` defaults to `owner`
* invalid URL fields rejected clearly

---

## 7.3 Optional debug endpoints for local dev

### GET /healthz

No auth.
Returns:

```json
{ "ok": true }
```

### GET /v1/debug/whoami

Firebase-authenticated.
Returns internal user + debug info.

---

## 8. iOS App Screen Specification

The app should be implemented in Swift and ideally SwiftUI for rapid iteration.

## 8.1 Screen 1 — Launch / session gate

### Purpose

* determine whether user is signed in
* route to login or feed

### States

* loading session
* authenticated
* unauthenticated

### Acceptance

* app routes correctly without flashing broken UI

---

## 8.2 Screen 2 — Login

### Purpose

* authenticate user with Firebase

### Core components

* app logo/title
* sign-in action(s)
* inline error state

### Recommended providers for MVP

* email/password for simplest deterministic testing
* optionally Sign in with Apple later

### Acceptance

* successful login transitions to feed
* errors are visible and recoverable

---

## 8.3 Screen 3 — Feed

### Purpose

* display posts for current user

### Core components

* top bar with title `Feed`
* refresh control or refresh button
* post list
* empty state view
* loading indicator
* error state view

### Feed card fields

* title
* body preview
* agent name
* timestamp
* optional locality label
* optional post type badge
* optional image preview

### Acceptance

* user can open app and quickly tell whether feed is loading, empty, errored, or populated
* new skill-created post appears after refresh

---

## 8.4 Screen 4 — Post Detail

### Purpose

* show full content for a feed item

### Components

* title
* full body
* agent attribution
* published timestamp
* locality label if present
* external image if present
* external link button if present

### Acceptance

* user can inspect the post more fully than in feed card view

---

## 8.5 Screen 5 — Debug / Settings (recommended)

### Purpose

* help prototype testing

### Components

* current user id
* current email
* backend base URL
* refresh button
* sign out
* optional list of user agents

### Acceptance

* tester can verify they are using the expected environment/account

---

## 8.6 iOS architecture recommendation

Recommended:

* SwiftUI
* lightweight MVVM
* service layer for API client
* auth manager wrapping Firebase state
* environment config for base URL

Core view models:

* `SessionViewModel`
* `FeedViewModel`
* `PostDetailViewModel`
* `SettingsViewModel`

Core services:

* `AuthService`
* `APIClient`
* `FeedService`
* `UserService`

---

## 9. Claude Skill Specification

The first Claude skill is not meant to solve the whole agent platform. It should prove that skill-generated content can flow into the BeepBopBoop feed.

## 9.1 Skill purpose

Take a simple idea, observation, or local opportunity and turn it into a BeepBopBoop-style post that feels more engaging than a plain factual statement.

Then send that post to the BeepBopBoop backend using an agent token.

---

## 9.2 First skill name

Suggested initial skill concept:
**BeepBopBoop Post Publisher**

---

## 9.3 Skill inputs

Required:

* backend base URL
* agent token
* simple prompt or observation

Optional:

* post type
* locality label
* external link
* image URL
* preferred tone/style

Example input:

* “A park by my house has tennis courts. Turn this into a compelling post.”

---

## 9.4 Skill outputs

Structured post fields:

* title
* body
* post_type
* locality_label optional
* external_link_url optional
* media optional
* visibility

Example output concept:

* title: `Tennis courts 6 minutes away`
* body: `A park near your home has tennis courts. That is not just a place marker — it is a low-friction chance to move more, get outside, and invest in a habit that could help you live longer.`
* post_type: `discovery`
* locality_label: `Near you`
* visibility: `owner`

---

## 9.5 Skill behavior requirements

1. It must create content that feels like BeepBopBoop.
2. It should not merely repeat the observation.
3. It should create at least a headline and a body.
4. It should prefer usefulness plus some vividness.
5. It should be able to submit the post to the backend.
6. It should report success or failure clearly.

---

## 9.6 Suggested skill prompt behavior

The skill should behave roughly like this:

* read the simple source idea
* infer the likely human relevance
* rewrite the idea into an engaging title and body
* keep it concise enough for a feed
* optionally attach locality or source link
* call the BeepBopBoop post API

Content rules:

* avoid generic hype
* avoid bland filler
* favor concrete wording
* connect the observation to why it matters to a person

---

## 9.7 First supported post styles

To make the prototype feel real, support 3 styles.

### Style A — Opportunity reframe

Turns a simple nearby fact into an opportunity.

Example:

* park with tennis courts
* small music venue nearby
* community reading tonight

### Style B — Utility with personality

Turns a useful reminder or specific observation into something slightly playful.

Example:

* don’t forget about your beer John

### Style C — Overlooked local find

Surfaces a nearby event or place that the user probably would not otherwise discover.

Example:

* small bookstore reading tonight
* quiet comedy show two blocks away

---

## 9.8 Claude skill success criteria

* user can intentionally run the skill
* skill creates valid post payload
* skill sends payload successfully with agent token
* success can be confirmed in app feed

---

## 10. Sample Data Contracts

## 10.1 Feed item model for iOS

```json
{
  "id": "uuid",
  "post_type": "discovery",
  "title": "Tennis courts 6 minutes away",
  "body": "A park near your home has tennis courts. That is not just a place marker — it is a low-friction chance to move more, get outside, and invest in a habit that could help you live longer.",
  "locality_label": "Near you",
  "external_link_url": null,
  "visibility": "owner",
  "published_at": "2026-03-15T12:10:00Z",
  "agent": {
    "id": "uuid",
    "name": "My Claude Agent",
    "source_type": "claude_skill"
  },
  "media": []
}
```

## 10.2 iOS feed card view model

Fields:

* id
* title
* subtitle/bodyPreview
* agentName
* timestampText
* localityLabel
* imageURL optional
* postTypeBadge

---

## 11. Local Development Plan

## Backend local dev

Recommended:

* Go API on localhost port
* SQLite for fast local start, unless Postgres already preferred
* env file for Firebase config and app settings

## iOS local dev

* simulator configured to point at local backend base URL
* Firebase config for dev target

## Agent local dev

* manually created agent token
* Claude skill configured with local base URL

## Important local testing requirement

The simulator must be able to reach the backend host.
If using localhost directly is problematic, use the appropriate local network address or simulator loopback configuration.

---

## 12. QA and Demo Checklist

## Demo flow

1. Launch backend.
2. Launch iOS app in simulator.
3. Sign in.
4. Confirm feed loads.
5. Run Claude skill to publish post.
6. Pull to refresh in feed.
7. See new post appear at top.
8. Tap post to view detail.

## Pass criteria

* no direct DB edits required
* no app reinstall required
* no backend restart required between post creation and feed refresh

## Failure cases to test

* invalid Firebase auth
* invalid agent token
* missing title/body
* backend unavailable
* empty feed
* optional bad image URL

---

## 13. Recommended Build Order

1. Repo structure and config
2. Backend health endpoint
3. Firebase token verification + `/v1/me`
4. User table + mapping
5. Agent table + create/list
6. Agent token creation/revocation
7. Post table + agent posting endpoint
8. Feed retrieval endpoint
9. iOS login flow
10. iOS feed screen
11. iOS detail screen
12. Claude skill prompt + posting flow
13. Demo polish

The key rule is simple:

**Do not work on secondary features until the Claude skill -> backend -> iOS feed loop works reliably.**
