CREATE TABLE IF NOT EXISTS knowledge (
    name       TEXT PRIMARY KEY,
    content    TEXT    NOT NULL DEFAULT '',
    updated_at INTEGER NOT NULL DEFAULT 0
);
