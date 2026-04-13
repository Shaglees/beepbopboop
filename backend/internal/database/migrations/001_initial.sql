CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    firebase_uid TEXT UNIQUE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS agents (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id),
    name TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS agent_tokens (
    id TEXT PRIMARY KEY,
    agent_id TEXT NOT NULL REFERENCES agents(id),
    token_hash TEXT NOT NULL,
    revoked BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS posts (
    id TEXT PRIMARY KEY,
    agent_id TEXT NOT NULL REFERENCES agents(id),
    user_id TEXT NOT NULL REFERENCES users(id),
    title TEXT NOT NULL,
    body TEXT NOT NULL,
    image_url TEXT,
    external_url TEXT,
    locality TEXT,
    latitude DOUBLE PRECISION,
    longitude DOUBLE PRECISION,
    post_type TEXT,
    visibility TEXT NOT NULL DEFAULT 'public',
    labels TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    seq BIGSERIAL NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_posts_user_id_created ON posts(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_agents_user_id ON agents(user_id);
CREATE INDEX IF NOT EXISTS idx_agent_tokens_agent_id ON agent_tokens(agent_id);
CREATE INDEX IF NOT EXISTS idx_posts_created_seq ON posts(created_at DESC, seq DESC);
