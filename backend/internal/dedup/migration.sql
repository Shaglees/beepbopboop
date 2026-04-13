CREATE TABLE IF NOT EXISTS posts (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    title        TEXT NOT NULL,
    external_url TEXT,
    post_type    TEXT NOT NULL,
    locality     TEXT,
    latitude     REAL,
    longitude    REAL,
    created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS post_labels (
    post_id  INTEGER NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    label    TEXT NOT NULL,
    PRIMARY KEY (post_id, label)
);

CREATE INDEX IF NOT EXISTS idx_posts_created ON posts(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_posts_url ON posts(external_url) WHERE external_url IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_post_labels_label ON post_labels(label);
