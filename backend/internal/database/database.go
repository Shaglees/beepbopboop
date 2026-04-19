package database

import (
	"database/sql"
	_ "embed"
	"fmt"

	_ "github.com/lib/pq"
)

//go:embed migrations/001_initial.sql
var migrationSQL string

func Open(url string) (*sql.DB, error) {
	db, err := sql.Open("postgres", url)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	if _, err := db.Exec(migrationSQL); err != nil {
		db.Close()
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	// Add new columns to existing databases (ignore "already exists" errors).
	db.Exec("ALTER TABLE posts ADD COLUMN IF NOT EXISTS visibility TEXT NOT NULL DEFAULT 'public'")
	db.Exec("ALTER TABLE posts ADD COLUMN IF NOT EXISTS labels TEXT")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_posts_visibility ON posts(visibility, created_at DESC)")

	// User settings table for location preferences
	db.Exec(`CREATE TABLE IF NOT EXISTS user_settings (
		user_id TEXT PRIMARY KEY REFERENCES users(id),
		location_name TEXT,
		latitude DOUBLE PRECISION,
		longitude DOUBLE PRECISION,
		radius_km DOUBLE PRECISION NOT NULL DEFAULT 25.0,
		updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`)

	db.Exec("ALTER TABLE user_settings ADD COLUMN IF NOT EXISTS followed_teams JSONB")

	// Geo index for community feed queries
	db.Exec("CREATE INDEX IF NOT EXISTS idx_posts_geo ON posts(visibility, latitude, longitude, created_at DESC)")

	// Engagement events table
	db.Exec(`CREATE TABLE IF NOT EXISTS post_events (
		id         BIGSERIAL PRIMARY KEY,
		post_id    TEXT NOT NULL REFERENCES posts(id),
		user_id    TEXT NOT NULL REFERENCES users(id),
		event_type TEXT NOT NULL,
		dwell_ms   INTEGER,
		created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`)
	db.Exec("CREATE INDEX IF NOT EXISTS idx_post_events_post ON post_events(post_id, event_type)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_post_events_user ON post_events(user_id, created_at DESC)")

	// Display hint for post rendering
	db.Exec("ALTER TABLE posts ADD COLUMN IF NOT EXISTS display_hint TEXT NOT NULL DEFAULT 'card'")

	// Images JSONB for multi-image posts (outfit collages, product thumbnails)
	db.Exec("ALTER TABLE posts ADD COLUMN IF NOT EXISTS images JSONB")

	// Custom display templates per user
	db.Exec(`CREATE TABLE IF NOT EXISTS display_templates (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL REFERENCES users(id),
		hint_name TEXT NOT NULL,
		definition JSONB NOT NULL,
		created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(user_id, hint_name)
	)`)

	// User preference weights (pushed by agent, applied in ForYou feed)
	db.Exec(`CREATE TABLE IF NOT EXISTS user_weights (
		user_id    TEXT NOT NULL REFERENCES users(id) PRIMARY KEY,
		weights    JSONB NOT NULL,
		updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`)

	// System user + worker agents for server-generated posts.
	db.Exec("INSERT INTO users (id, firebase_uid) VALUES ('system', 'system') ON CONFLICT DO NOTHING")
	db.Exec("INSERT INTO agents (id, user_id, name, status) VALUES ('weather-bot', 'system', 'Weather', 'active') ON CONFLICT DO NOTHING")
	db.Exec("INSERT INTO agents (id, user_id, name, status) VALUES ('sports-bot', 'system', 'Sports', 'active') ON CONFLICT DO NOTHING")

	// Post scheduling: status tracks published vs scheduled, scheduled_at holds publish time
	db.Exec("ALTER TABLE posts ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'published'")
	db.Exec("ALTER TABLE posts ADD COLUMN IF NOT EXISTS scheduled_at TIMESTAMPTZ")
	db.Exec("ALTER TABLE posts ADD COLUMN IF NOT EXISTS source_published_at TIMESTAMPTZ")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_posts_scheduled ON posts(status, scheduled_at) WHERE status = 'scheduled'")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_posts_source_published_at ON posts(source_published_at DESC)")

	// Post reactions (explicit user feedback for agent content tuning)
	db.Exec(`CREATE TABLE IF NOT EXISTS post_reactions (
		post_id    TEXT NOT NULL REFERENCES posts(id),
		user_id    TEXT NOT NULL REFERENCES users(id),
		reaction   TEXT NOT NULL,
		created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (post_id, user_id)
	)`)
	db.Exec("CREATE INDEX IF NOT EXISTS idx_post_reactions_user ON post_reactions(user_id, updated_at DESC)")

	// Denormalized engagement counts for feed ranking (avoids JOIN/subquery at query time)
	db.Exec("ALTER TABLE posts ADD COLUMN IF NOT EXISTS view_count INT NOT NULL DEFAULT 0")
	db.Exec("ALTER TABLE posts ADD COLUMN IF NOT EXISTS save_count INT NOT NULL DEFAULT 0")
	db.Exec("ALTER TABLE posts ADD COLUMN IF NOT EXISTS reaction_count INT NOT NULL DEFAULT 0")

	// Push notification tokens (APNs device tokens from iOS)
	db.Exec(`CREATE TABLE IF NOT EXISTS push_tokens (
		user_id    TEXT NOT NULL REFERENCES users(id),
		token      TEXT NOT NULL,
		platform   TEXT NOT NULL DEFAULT 'apns',
		updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (user_id, token)
	)`)

	// Notification preferences in user_settings
	db.Exec("ALTER TABLE user_settings ADD COLUMN IF NOT EXISTS notifications_enabled BOOLEAN NOT NULL DEFAULT TRUE")
	db.Exec("ALTER TABLE user_settings ADD COLUMN IF NOT EXISTS digest_hour INT NOT NULL DEFAULT 8")

	// user_feedback stores raw responses to feedback posts
	db.Exec(`CREATE TABLE IF NOT EXISTS user_feedback (
    id          BIGSERIAL PRIMARY KEY,
    post_id     TEXT NOT NULL REFERENCES posts(id),
    user_id     TEXT NOT NULL REFERENCES users(id),
    response    JSONB NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
)`)
	db.Exec("CREATE INDEX IF NOT EXISTS idx_user_feedback_post_id ON user_feedback(post_id)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_user_feedback_user_id ON user_feedback(user_id)")
	db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_user_feedback_post_user ON user_feedback(post_id, user_id)")
	// preference_context: agent-writable summary injected into agent prompts
	db.Exec("ALTER TABLE user_settings ADD COLUMN IF NOT EXISTS preference_context JSONB")

	// local_creators: cached creator profiles from agent research (research-once, serve-many)
	db.Exec(`CREATE TABLE IF NOT EXISTS local_creators (
		id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		name          TEXT NOT NULL,
		designation   TEXT NOT NULL,
		bio           TEXT,
		lat           DOUBLE PRECISION,
		lon           DOUBLE PRECISION,
		area_name     TEXT,
		links         JSONB,
		notable_works TEXT,
		tags          JSONB,
		source        TEXT NOT NULL,
		image_url     TEXT,
		discovered_at TIMESTAMPTZ NOT NULL DEFAULT now(),
		verified_at   TIMESTAMPTZ,
		UNIQUE (name, lat, lon)
	)`)
	db.Exec("CREATE INDEX IF NOT EXISTS idx_local_creators_geo ON local_creators (lat, lon)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_local_creators_designation ON local_creators (designation)")

	// Calendar integration opt-in flag per user
	db.Exec("ALTER TABLE user_settings ADD COLUMN IF NOT EXISTS calendar_enabled BOOLEAN NOT NULL DEFAULT FALSE")

	// Calendar events synced from device (EventKit/iOS)
	db.Exec(`CREATE TABLE IF NOT EXISTS calendar_events (
		id          TEXT NOT NULL,
		user_id     TEXT NOT NULL REFERENCES users(id),
		title       TEXT NOT NULL,
		start_time  TIMESTAMPTZ NOT NULL,
		end_time    TIMESTAMPTZ,
		location    TEXT,
		notes       TEXT,
		synced_at   TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (user_id, id)
	)`)
	db.Exec("CREATE INDEX IF NOT EXISTS idx_calendar_events_user_start ON calendar_events(user_id, start_time)")

	// Anticipatory worker agent
	db.Exec("INSERT INTO agents (id, user_id, name, status) VALUES ('calendar-bot', 'system', 'Anticipatory', 'active') ON CONFLICT DO NOTHING")

	// Agent following: social graph for agent discovery and follower feed.
	db.Exec(`CREATE TABLE IF NOT EXISTS agent_follows (
		user_id     TEXT        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		agent_id    TEXT        NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
		followed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
		PRIMARY KEY (user_id, agent_id)
	)`)
	db.Exec("CREATE INDEX IF NOT EXISTS agent_follows_agent_id ON agent_follows (agent_id)")
	db.Exec("CREATE INDEX IF NOT EXISTS agent_follows_user_id  ON agent_follows (user_id)")

	// Denormalized follower count + profile fields on agents table.
	db.Exec("ALTER TABLE agents ADD COLUMN IF NOT EXISTS follower_count INTEGER NOT NULL DEFAULT 0")
	db.Exec("ALTER TABLE agents ADD COLUMN IF NOT EXISTS description TEXT")
	db.Exec("ALTER TABLE agents ADD COLUMN IF NOT EXISTS avatar_url TEXT")

	return db, nil
}
