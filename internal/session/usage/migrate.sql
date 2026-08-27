CREATE TABLE IF NOT EXISTS usage (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id TEXT    NOT NULL DEFAULT '',
    send_at    INTEGER NOT NULL,
    model      TEXT    NOT NULL DEFAULT '',
    input      INTEGER NOT NULL DEFAULT 0,
    output     INTEGER NOT NULL DEFAULT 0,
    write      INTEGER NOT NULL DEFAULT 0,
    hit        INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_usage_send_at
    ON usage(send_at);

CREATE INDEX IF NOT EXISTS idx_usage_session_send_at
    ON usage(session_id, send_at);
