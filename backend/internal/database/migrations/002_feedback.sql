-- user_feedback stores raw responses to feedback posts
CREATE TABLE IF NOT EXISTS user_feedback (
    id          BIGSERIAL PRIMARY KEY,
    post_id     TEXT NOT NULL REFERENCES posts(id),
    user_id     TEXT NOT NULL REFERENCES users(id),
    response    JSONB NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_user_feedback_post_id   ON user_feedback(post_id);
CREATE INDEX IF NOT EXISTS idx_user_feedback_user_id   ON user_feedback(user_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_user_feedback_post_user ON user_feedback(post_id, user_id);

-- preference_context: agent-writable distilled preference summary injected into prompts
ALTER TABLE user_settings ADD COLUMN IF NOT EXISTS preference_context JSONB;
