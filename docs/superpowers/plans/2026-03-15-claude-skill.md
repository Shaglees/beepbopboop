# BeepBopBoop Claude Skill Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a Claude Code skill that generates engaging BeepBopBoop posts and publishes them to the backend via the agent token API.

**Architecture:** A Claude Code skill (markdown file with YAML frontmatter) that uses Bash tool calls to POST to the backend API. The skill prompts Claude to generate engaging, locality-aware content from a simple idea, then submits it as a structured post.

**Tech Stack:** Claude Code skills (markdown + YAML), curl for API calls, jq for response parsing

**Depends on:** Plan 1 (Go Backend) — needs running backend with agent token

**Ref docs:** `MVP_CHECKLIST.md` Section 5 | `PRD.md` Sections 8, 9

---

## File Structure

```
.claude/
└── skills/
    └── beepbopboop-post/
        └── SKILL.md                     # The Claude Code skill definition
```

---

## Chunk 1: Skill Implementation

### Task 1: Create the skill directory

**Files:**
- Create: `.claude/skills/beepbopboop-post/SKILL.md`

- [ ] **Step 1: Create skill file**

Create `.claude/skills/beepbopboop-post/SKILL.md`:

````markdown
---
name: beepbopboop-post
description: Generate and publish an engaging BeepBopBoop post from a simple idea
argument-hint: <idea> [locality] [post_type]
disable-model-invocation: true
allowed-tools: Bash(curl *), Bash(jq *)
---

# BeepBopBoop Post Skill

You are a BeepBopBoop agent. Your job is to take a simple idea and transform it into engaging, personalized, human-relevant content.

## Important

You are NOT a generic content writer. You are a discovery agent. Your posts should:

- Turn mundane observations into compelling discoveries
- Make the reader feel like they're learning something about their own life
- Be specific and grounded, not generic or fluffy
- Feel like a smart friend pointing something out, not a marketing bot
- Be concise — a headline that hooks, and a body that delivers

## Configuration

Before using this skill, set these environment variables:

```bash
export BEEPBOPBOOP_API_URL="http://localhost:8080"
export BEEPBOPBOOP_AGENT_TOKEN="bbp_your_token_here"
```

## Steps

### Step 1: Generate the post content

Based on the idea provided: `$0`

Generate:
- **title**: A compelling, specific headline (max 80 chars). Not clickbait — genuinely interesting.
- **body**: 2-4 sentences that expand on the title. Make it personal, actionable, or thought-provoking.

Locality context (if provided as second argument): `$1`
Post type (if provided as third argument): `$2`

### Step 2: Publish to the backend

Use the Bash tool to POST the generated content:

```bash
curl -s -X POST "${BEEPBOPBOOP_API_URL}/posts" \
  -H "Authorization: Bearer ${BEEPBOPBOOP_AGENT_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "<GENERATED_TITLE>",
    "body": "<GENERATED_BODY>",
    "image_url": "",
    "external_url": "",
    "locality": "<LOCALITY_OR_EMPTY>",
    "post_type": "<POST_TYPE_OR_DISCOVERY>"
  }' | jq .
```

### Step 3: Report the result

If the response contains an `id` field, the post was created successfully. Show:
- The post title
- The post ID
- Confirmation it's now in the user's feed

If the response contains an `error` field, show the error and suggest fixes:
- If 401: "Token may be invalid or revoked. Check BEEPBOPBOOP_AGENT_TOKEN."
- If connection refused: "Backend may not be running. Start it with: cd backend && go run ./cmd/server"

## Example

Given the idea "park near my house has tennis courts":

**title**: "Tennis courts 6 minutes away"
**body**: "A park near your home has tennis courts. That is not just a place marker — it is a low-friction chance to move more, get outside, and invest in a habit that could help you live longer."
**locality**: (from second argument or empty)
**post_type**: "discovery"
````

- [ ] **Step 2: Commit**

```bash
git add .claude/skills/
git commit -m "feat(skill): add beepbopboop-post Claude Code skill"
```

---

### Task 2: Create skill README

**Files:**
- Create: `.claude/skills/beepbopboop-post/README.md`

- [ ] **Step 1: Create README**

Create `.claude/skills/beepbopboop-post/README.md`:

```markdown
# BeepBopBoop Post Skill

A Claude Code skill for generating and publishing posts to the BeepBopBoop backend.

## Setup

1. Start the backend:
   ```bash
   cd backend && go run ./cmd/server
   ```

2. Create a user and agent (dev mode):
   ```bash
   # Create user
   curl -s -H "Authorization: Bearer my-test-user" http://localhost:8080/me | jq .

   # Create agent
   AGENT_ID=$(curl -s -X POST \
     -H "Authorization: Bearer my-test-user" \
     -H "Content-Type: application/json" \
     -d '{"name": "My Discovery Agent"}' \
     http://localhost:8080/agents | jq -r .id)

   # Generate token
   TOKEN=$(curl -s -X POST \
     -H "Authorization: Bearer my-test-user" \
     http://localhost:8080/agents/$AGENT_ID/tokens | jq -r .token)

   echo "Agent Token: $TOKEN"
   ```

3. Set environment variables:
   ```bash
   export BEEPBOPBOOP_API_URL="http://localhost:8080"
   export BEEPBOPBOOP_AGENT_TOKEN="$TOKEN"
   ```

## Usage

```
/beepbopboop-post "park near my house has tennis courts"
/beepbopboop-post "there's a comedy show tonight at the local pub" "Phibsborough"
/beepbopboop-post "new coffee shop opened on the corner" "Dublin 7" "discovery"
```

## Verify

After posting, check the feed:

```bash
curl -s -H "Authorization: Bearer my-test-user" http://localhost:8080/feed | jq .
```
```

- [ ] **Step 2: Commit**

```bash
git add .claude/skills/beepbopboop-post/README.md
git commit -m "docs(skill): add README for beepbopboop-post skill"
```

---

### Task 3: End-to-end skill test

- [ ] **Step 1: Start backend**

```bash
cd backend && go run ./cmd/server &
```

- [ ] **Step 2: Set up agent and token**

```bash
# Create user
curl -s -H "Authorization: Bearer test-user-1" http://localhost:8080/me > /dev/null

# Create agent
AGENT_ID=$(curl -s -X POST \
  -H "Authorization: Bearer test-user-1" \
  -H "Content-Type: application/json" \
  -d '{"name": "Discovery Agent"}' \
  http://localhost:8080/agents | jq -r .id)

# Generate token
TOKEN=$(curl -s -X POST \
  -H "Authorization: Bearer test-user-1" \
  http://localhost:8080/agents/$AGENT_ID/tokens | jq -r .token)

export BEEPBOPBOOP_API_URL="http://localhost:8080"
export BEEPBOPBOOP_AGENT_TOKEN="$TOKEN"
```

- [ ] **Step 3: Run the skill**

```
/beepbopboop-post "park near my house has tennis courts"
```

Expected: Skill generates an engaging post title and body, POSTs it, and reports success with post ID.

- [ ] **Step 4: Verify post in feed**

```bash
curl -s -H "Authorization: Bearer test-user-1" http://localhost:8080/feed | jq .
```

Expected: Feed contains the new post with agent attribution.

- [ ] **Step 5: Stop backend**

```bash
kill %1
```

- [ ] **Step 6: Verify the MVP test scenario**

At this point, the full end-to-end loop should be testable:

1. Start backend
2. Open iOS app in simulator
3. Sign in
4. Run `/beepbopboop-post "your idea here"`
5. Pull-to-refresh in the iOS app
6. See the new post appear in the feed

This satisfies the mandatory MVP acceptance criteria from `MVP_CHECKLIST.md` Section 10.
