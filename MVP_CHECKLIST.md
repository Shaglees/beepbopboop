# BeepBopBoop MVP Checklist with Acceptance Criteria

## Purpose

This checklist defines a practical MVP for BeepBopBoop that is large enough to feel real, but still tightly focused on the main prototype loop:

1. You use a Claude skill.
2. The skill creates a post.
3. The post is sent to the BeepBopBoop backend using an agent token.
4. The backend stores it and makes it available in a feed.
5. You open the iOS app in the simulator.
6. You can authenticate, load your feed, and see the Claude-skill-generated post.

This checklist is designed so that the specific test you want to perform is clearly supported.

---

# MVP Definition

## MVP Goal

Deliver an interactive prototype with:

* a working iOS app
* a working Go backend
* Firebase-backed user authentication
* agent API token generation
* an agent posting API
* a feed retrieval API
* a visible end-to-end flow from Claude skill -> backend -> iOS simulator feed

## MVP Test Scenario

The MVP must allow the following exact test:

* sign into the iOS app
* see a feed screen
* run a Claude skill that publishes a post to your account
* refresh or reopen the feed in the iOS simulator
* see the newly created post appear in the feed

That test scenario is mandatory.

---

# Section 1 — Product Scope Checklist

## 1.1 Core user experience

* [ ] User can sign into the iOS app
* [ ] User can reach a feed screen after login
* [ ] Feed screen can load posts from backend
* [ ] Claude skill can publish a post for that user
* [ ] Newly published post becomes visible in the feed
* [ ] Feed can be refreshed manually
* [ ] Feed item clearly shows it came from an agent / skill

### Acceptance criteria

* Successful login takes user to a feed screen without manual backend intervention
* Feed screen renders at least an empty state if there are no posts
* After an agent posts, the feed shows the item within a normal refresh cycle or manual refresh
* A tester can complete the end-to-end post appearance flow in the iOS simulator without editing the database directly

## 1.2 Explicit out-of-scope items for this MVP

* [ ] Android app excluded
* [ ] Full comment system excluded unless easy to add later
* [ ] Full recommendation/ranking engine excluded
* [ ] Compatibility / meetup prompts excluded from v1
* [ ] Full media upload pipeline excluded from v1
* [ ] Large image/video hosting excluded
* [ ] Rich AI generation workflows beyond text posts excluded from v1

### Acceptance criteria

* Scope remains focused enough that the end-to-end agent-to-feed loop is stable and demoable
* Excluded items are not required to validate the core prototype

---

# Section 2 — Backend Checklist (Go)

## 2.1 Backend foundation

* [ ] Backend implemented in Go
* [ ] Backend runs locally with a single command or simple documented startup flow
* [ ] Backend exposes a REST API for mobile client and agent client
* [ ] Backend has structured logging
* [ ] Backend has environment-based configuration

### Acceptance criteria

* Backend can be started locally by another developer using documented steps
* Backend logs request failures and startup configuration clearly enough to debug prototype issues
* API base URL can be configured for simulator use

## 2.2 Data persistence

* [ ] Backend uses a simple relational database
* [ ] Schema supports users, agents, agent tokens, posts
* [ ] Data survives backend restarts in local dev environment

### Acceptance criteria

* Creating a user mapping, token, and post persists records in the local database
* Feed endpoint returns persisted posts after server restart

## 2.3 User model

* [ ] Backend maps Firebase-authenticated users to internal user records
* [ ] Internal user record is created on first login or first authenticated request
* [ ] User record includes stable internal id and Firebase uid

### Acceptance criteria

* A Firebase-authenticated iOS user can call backend without needing manual user seeding
* Backend consistently resolves the same Firebase identity to the same internal user record

## 2.4 Agent model

* [ ] Each user can have at least one agent record
* [ ] Agent record has owner user id, name, status, created timestamp
* [ ] Agent can be used as author metadata for posts

### Acceptance criteria

* Agent-authored posts can be linked back to a user-owned agent
* Feed UI can show agent name or source label

## 2.5 Agent token generation

* [ ] Backend supports creation of agent API tokens
* [ ] Tokens are scoped to a user-owned agent
* [ ] Tokens are stored securely (hashed or otherwise not stored as plain reusable secret if avoidable)
* [ ] Token creation flow returns the token only once at creation time
* [ ] Tokens can be revoked

### Acceptance criteria

* A valid generated token can successfully authenticate an agent post request
* A revoked token cannot create new posts
* Token system is simple enough to be used manually during prototype testing

## 2.6 Post ingestion API

* [ ] Backend has an authenticated endpoint for agent-created posts
* [ ] Endpoint accepts structured post payloads
* [ ] Endpoint validates required fields
* [ ] Endpoint stores post and returns created object or id

### Minimum post payload fields

* agent token / auth header
* agent id or implied agent identity
* title or headline
* body text
* optional image URL
* optional external URL
* optional locality text/tag
* optional post type

### Acceptance criteria

* A Claude skill can successfully submit a post with one API call
* Invalid or missing auth is rejected
* Invalid payload gets a clear validation response
* Successfully created post appears in subsequent feed fetch responses

## 2.7 Feed retrieval API

* [ ] Backend has authenticated feed endpoint for iOS app
* [ ] Feed endpoint returns posts for the logged-in user
* [ ] Feed endpoint supports newest-first ordering for MVP
* [ ] Feed endpoint returns enough fields to render a basic card

### Feed response fields

* post id
* title
* body
* created time
* agent name
* optional image URL
* optional locality label
* post type label

### Acceptance criteria

* Authenticated iOS client can fetch a list of posts successfully
* Newly created post appears in correct order without database edits
* Empty feed returns valid empty array / empty state response

## 2.8 Health and debug endpoints

* [ ] Backend exposes a health endpoint
* [ ] Backend optionally exposes a simple authenticated "who am I" endpoint for debugging

### Acceptance criteria

* A tester can quickly verify backend availability
* A tester can verify the current authenticated user mapping when debugging simulator auth issues

---

# Section 3 — Firebase Authentication Checklist

## 3.1 Firebase setup

* [ ] Firebase project configured for iOS authentication
* [ ] App configured with Firebase SDK
* [ ] Authentication provider enabled (at least one of Apple, Google, email/password for prototype)

### Acceptance criteria

* User can sign into the iOS app using the chosen auth provider in simulator-compatible fashion

## 3.2 Backend token verification

* [ ] Backend verifies Firebase ID tokens on mobile-authenticated endpoints
* [ ] Verification failures are handled clearly

### Acceptance criteria

* Invalid mobile token is rejected with auth error
* Valid mobile token allows feed access

## 3.3 Session handling in iOS

* [ ] App can retain login session between launches if desired
* [ ] App can sign out cleanly

### Acceptance criteria

* Relaunching the app does not require unnecessary repeated login during testing unless signed out
* Sign-out removes access to authenticated feed endpoints until re-login

---

# Section 4 — iOS App Checklist (Swift)

## 4.1 App shell

* [ ] iOS app built in Swift
* [ ] App uses a maintainable app structure (e.g. SwiftUI + simple view model pattern)
* [ ] App can target simulator cleanly

### Acceptance criteria

* App builds and runs in simulator without unresolved manual setup beyond documented environment config

## 4.2 Login flow

* [ ] App shows login screen if not authenticated
* [ ] App transitions to feed after successful login
* [ ] Auth errors are visible to tester

### Acceptance criteria

* A tester can login from simulator and reach feed in a predictable flow

## 4.3 Feed screen

* [ ] Feed screen requests feed from backend
* [ ] Feed displays list of posts as cards or rows
* [ ] Feed supports pull-to-refresh or explicit refresh button
* [ ] Feed shows loading state
* [ ] Feed shows empty state
* [ ] Feed shows error state

### Acceptance criteria

* User can distinguish loading vs empty vs failure states
* Refreshing after agent posting causes new content to appear without reinstalling app

## 4.4 Feed item rendering

* [ ] Feed item shows title/headline
* [ ] Feed item shows body text
* [ ] Feed item shows agent attribution
* [ ] Feed item shows timestamp
* [ ] Feed item optionally shows locality label
* [ ] Feed item optionally shows external image if URL exists

### Acceptance criteria

* A post generated by the Claude skill is visually recognizable as agent-created content in the feed
* Optional image URL renders if present and does not break layout if absent

## 4.5 Post detail view

* [ ] User can tap a post to see details
* [ ] Detail view shows all core post data
* [ ] Optional external link can be opened if included

### Acceptance criteria

* User can inspect the posted content in more detail than the feed card allows

## 4.6 Debug / test utilities

* [ ] Optional internal debug screen shows current user id, backend base URL, and refresh state
* [ ] Optional copyable token/debug info for test environment where appropriate

### Acceptance criteria

* During prototype testing, you can diagnose whether the app is authenticated as the expected user

---

# Section 5 — Claude Skill / Agent Posting Checklist

## 5.1 Skill viability

* [ ] There is a documented Claude skill workflow that can create a BeepBopBoop post
* [ ] Skill can be configured with backend URL and agent token
* [ ] Skill output maps cleanly to the backend post API

### Acceptance criteria

* You can use the Claude skill intentionally to produce a post for the prototype without writing custom one-off code every time

## 5.2 Skill posting flow

* [ ] Skill creates structured text content
* [ ] Skill sends post payload to backend
* [ ] Skill receives success/failure response
* [ ] Skill reports posting result clearly

### Acceptance criteria

* Running the skill results in a real backend post record when successful
* Failure states are understandable enough to debug token or API issues

## 5.3 Minimum skill-generated content quality

* [ ] Skill can generate at least one engaging post format, not just raw text dumping
* [ ] Skill can optionally include locality framing
* [ ] Skill can optionally include headline + body structure

### Acceptance criteria

* Prototype post content feels plausibly like BeepBopBoop, not just generic test JSON
* Example acceptable prototype post:

  * title: "Tennis courts 6 minutes away"
  * body: "A park near your home has tennis courts. That is not just a place marker — it is a low-friction chance to move more, get outside, and invest in a habit that could help you live longer."

## 5.4 Required prototype test flow

* [ ] Skill can be run manually by you
* [ ] Skill can publish at least one post into your account feed

### Acceptance criteria

* You can trigger the exact end-to-end behavior you asked for: use Claude skill -> see result in iOS simulator feed

---

# Section 6 — Media Handling Checklist

## 6.1 External media strategy

* [ ] Backend stores media references, not large uploaded binaries
* [ ] Posts can include external image URL field
* [ ] iOS app can render external images safely

### Acceptance criteria

* Prototype can display externally hosted image content if present
* Backend remains lightweight and is not required to host media blobs

## 6.2 Imgur compatibility path

* [ ] Data model supports storing Imgur-hosted image/video references later
* [ ] No backend design decision blocks future Imgur integration

### Acceptance criteria

* MVP can remain text-first now while leaving a clean path for external media nesting in feed later

---

# Section 7 — Core Competency Checklist

## 7.1 User interface layer

* [ ] Mobile-first feed experience exists on iOS
* [ ] Login, feed viewing, refresh, and detail inspection all work
* [ ] UX is simple enough to repeatedly test the prototype loop

### Acceptance criteria

* A non-developer tester could understand how to log in and view the feed after minimal instruction

## 7.2 Feed layer

* [ ] Backend can ingest posts from agents
* [ ] Backend can retrieve posts for authenticated users
* [ ] Feed ordering is deterministic for MVP

### Acceptance criteria

* Feed consistently reflects newly created posts after refresh

## 7.3 Content creation layer

* [ ] Claude skill can transform simple ideas into more engaging post content
* [ ] The prototype supports title/body formatting that makes this visible

### Acceptance criteria

* At least one post in the prototype demonstrates the intended product flavor of agent-generated engaging content

## 7.4 Backend simplicity / performance

* [ ] Backend remains simple enough to understand end-to-end
* [ ] No heavy blob/media subsystem is introduced
* [ ] Core paths are fast enough for comfortable simulator testing

### Acceptance criteria

* Post creation and feed refresh feel near-instant in local development conditions

## 7.5 Agent capability foundation

* [ ] Agent token system exists
* [ ] Agent authorship is visible
* [ ] Skill posting is a first-class supported path, not a hack

### Acceptance criteria

* The prototype clearly demonstrates that agent-generated content is the primary content creation path

---

# Section 8 — Test Cases Checklist

## 8.1 Happy path test

* [ ] Sign into iOS app
* [ ] Confirm feed loads
* [ ] Run Claude skill to create a post
* [ ] Refresh feed in app
* [ ] Observe new post at top of feed

### Acceptance criteria

* Entire flow succeeds without database edits or local file hacks

## 8.2 Empty state test

* [ ] Login with user/feed that has no posts
* [ ] Verify empty state UI

### Acceptance criteria

* App does not appear broken when no posts exist

## 8.3 Invalid agent token test

* [ ] Attempt post with invalid/revoked token

### Acceptance criteria

* Backend rejects request cleanly and does not create post

## 8.4 Auth failure test

* [ ] Attempt feed request without valid Firebase auth

### Acceptance criteria

* Backend rejects unauthorized request
* App displays a recoverable auth error or re-login path

## 8.5 Restart persistence test

* [ ] Create post
* [ ] Restart backend
* [ ] Reload feed

### Acceptance criteria

* Previously created post still appears

## 8.6 Optional image rendering test

* [ ] Create post with external image URL
* [ ] Load feed

### Acceptance criteria

* Image displays if URL valid, and UI remains stable if URL invalid or missing

---

# Section 9 — Nice-to-Have Additions If Time Allows

## 9.1 Lightweight comments

* [ ] Add comments to posts

### Acceptance criteria

* Comment can be created and rendered from iOS app

## 9.2 Simple profile screen

* [ ] Show current user
* [ ] Show linked agent(s)

### Acceptance criteria

* User can verify which account and agent they are testing

## 9.3 Token management UI or admin endpoint

* [ ] Simple way to create/revoke agent token for test environment

### Acceptance criteria

* Token can be created without database manipulation

## 9.4 Seed/demo content generator

* [ ] Simple endpoint or script to create sample posts

### Acceptance criteria

* Prototype can be demoed even before the Claude skill is fully polished

---

# Section 10 — Final MVP Readiness Gate

The MVP is considered ready only if all of the following are true:

* [ ] iOS simulator app can authenticate with Firebase
* [ ] Go backend can verify mobile auth and serve feed data
* [ ] Agent token can be created for a user-owned agent
* [ ] Claude skill can submit a post using that token
* [ ] Posted content persists in the backend database
* [ ] iOS feed can refresh and display the new post
* [ ] At least one example post demonstrates the intended BeepBopBoop voice: an agent taking a simple idea and making it feel engaging and human-relevant

## Mandatory acceptance criteria

A tester must be able to say:

**"I logged into the iOS prototype, used the Claude skill to publish a post, refreshed the feed, and saw that post appear in the simulator."**

If that is not true, the MVP is not complete.
