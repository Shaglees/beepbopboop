# Wave 4: New Features — Implementation Plan (Overview)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship community local news, fashion try-on with user photos, and interest-driven calendar events as feed content.

**Architecture:** Three independent sub-systems sharing the existing feed, hint catalog, and publish pipeline. Local news gets a new skill + adaptive card. Fashion try-on extends the existing skill + card. Calendar events use a server worker with Go templates and existing card types.

**Tech Stack:** Go (chi router, database/sql, pq), SwiftUI, PostgreSQL, TMDB API, OpenAI image-2 API

**Spec:** `docs/superpowers/specs/2026-04-25-wave4-new-features-design.md`

**Implementation order:** A → B → C (each sub-system is independent)

---

## Plan Documents

| Part | File | Sub-system | Tasks |
|------|------|-----------|-------|
| A | `2026-04-25-wave4-part-a-local-news.md` | Community Local News (#189, #190) | Tasks 1–6 |
| B | `2026-04-25-wave4-part-b-fashion-tryon.md` | Fashion Try-On (#191) | Tasks 7–11 |
| C | `2026-04-25-wave4-part-c-interest-calendar.md` | Interest Calendar (#156) | Tasks 12–17 |

---

## File Structure

### Sub-system A: Local News

#### Backend — New Files
| File | Responsibility |
|------|---------------|
| `backend/internal/model/news_source.go` | `NewsSource` struct |
| `backend/internal/repository/news_source_repo.go` | CRUD + geo query for `news_sources` table |
| `backend/internal/repository/news_source_repo_test.go` | Repo tests |
| `backend/internal/handler/news_source.go` | REST endpoints for news source registry |
| `backend/internal/handler/news_source_test.go` | Handler tests |

#### Backend — Modified Files
| File | Changes |
|------|---------|
| `backend/internal/database/database.go:~397` | Add `news_sources` CREATE TABLE before `return db, nil` |
| `backend/internal/handler/hints.go:~546` | Add `local_news` hint entry |
| `backend/cmd/server/main.go:65-86` | Add `newsSourceRepo` |
| `backend/cmd/server/main.go:110-167` | Add `newsSourceH` handler |
| `backend/cmd/server/main.go:230-254` | Add agent-auth routes for news sources |

#### iOS — New Files
| File | Responsibility |
|------|---------------|
| `beepbopboop/beepbopboop/Views/LocalNewsCard.swift` | Adaptive card for article/video/hybrid |

#### iOS — Modified Files
| File | Changes |
|------|---------|
| `beepbopboop/beepbopboop/Models/Post.swift` | Add `.localNews` to `DisplayHint` enum |
| `beepbopboop/beepbopboop/Views/FeedItemView.swift:21-158` | Add `case .localNews:` routing |

#### Skills — New Files
| File | Responsibility |
|------|---------------|
| `.claude/skills/beepbopboop-local-news/SKILL.md` | Main skill file |
| `.claude/skills/beepbopboop-local-news/MODE_FETCH.md` | Fetch + compose mode |
| `.claude/skills/beepbopboop-local-news/MODE_DISCOVER.md` | Source discovery mode |
| `.claude/skills/beepbopboop-local-news/MODE_VIDEO.md` | Video news mode |

---

### Sub-system B: Fashion Try-On

#### Backend — New Files
| File | Responsibility |
|------|---------------|
| `backend/internal/repository/user_photo_repo.go` | Save/get/delete headshot+bodyshot |
| `backend/internal/repository/user_photo_repo_test.go` | Repo tests |
| `backend/internal/handler/photo.go` | Upload/download/delete endpoints |
| `backend/internal/handler/photo_test.go` | Handler tests |

#### Backend — Modified Files
| File | Changes |
|------|---------|
| `backend/internal/database/database.go:~397` | ALTER TABLE users ADD COLUMN headshot/bodyshot columns |
| `backend/cmd/server/main.go:65-86` | Add `photoRepo` |
| `backend/cmd/server/main.go:110-167` | Add `photoH` handler |
| `backend/cmd/server/main.go:182-228` | Add Firebase routes for photo CRUD |
| `backend/cmd/server/main.go:230-254` | Add agent-auth read-only photo routes |

#### iOS — Modified Files
| File | Changes |
|------|---------|
| `beepbopboop/beepbopboop/Services/APIService.swift:~757` | Add photo upload/download/delete methods |
| `beepbopboop/beepbopboop/Views/ProfileView.swift` | Add "My Photos" section |
| `beepbopboop/beepbopboop/Views/FeedItemView.swift:~1197` | OutfitCard try-on overlay |

#### Skills — New/Modified Files
| File | Responsibility |
|------|---------------|
| `.claude/skills/beepbopboop-fashion/MODE_TRYON.md` | New try-on mode |
| `.claude/skills/beepbopboop-fashion/SKILL.md` | Add try-on routing entry |

---

### Sub-system C: Interest Calendar

#### Backend — New Files
| File | Responsibility |
|------|---------------|
| `backend/internal/model/calendar_event.go` | `CalendarEvent`, `CalendarPostLog` structs |
| `backend/internal/repository/calendar_event_repo.go` | Upsert, query, dedup log |
| `backend/internal/repository/calendar_event_repo_test.go` | Repo tests |
| `backend/internal/entertainment/worker.go` | TMDB ingest worker |
| `backend/internal/entertainment/worker_test.go` | Worker tests |
| `backend/internal/calendar/materialize.go` | Materialization worker |
| `backend/internal/calendar/materialize_test.go` | Worker tests |
| `backend/internal/calendar/templates.go` | Go templates for post composition |

#### Backend — Modified Files
| File | Changes |
|------|---------|
| `backend/internal/database/database.go:~397` | CREATE TABLE interest_calendar_events + calendar_post_log |
| `backend/internal/sports/worker.go` | Extend cycle() to upsert upcoming games to calendar events |
| `backend/cmd/server/main.go:65-86` | Add `calendarEventRepo` |
| `backend/cmd/server/main.go:256-281` | Start entertainment + materialize workers |

#### No iOS Changes
Calendar posts use existing `matchup` and `event` display hints — no new card views needed.
