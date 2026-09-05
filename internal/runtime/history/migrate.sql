CREATE TABLE IF NOT EXISTS messages (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id   TEXT    NOT NULL,
    send_at      INTEGER NOT NULL DEFAULT 0,
    role         TEXT    NOT NULL,
    content      TEXT    NOT NULL,
    sender       TEXT    NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_messages_session_send_at
    ON messages(session_id, send_at);

CREATE TABLE IF NOT EXISTS message_meta (
    session_id TEXT PRIMARY KEY,
    start_at   INTEGER NOT NULL DEFAULT 0
);

CREATE VIRTUAL TABLE IF NOT EXISTS messages_fts5 USING fts5(
    role, content,
    content=messages, content_rowid=id
);

CREATE TRIGGER IF NOT EXISTS trigger_messages_after_insert AFTER INSERT ON messages BEGIN
    INSERT INTO messages_fts5(rowid, role, content)
    VALUES (new.id, new.role, new.content);
END;

CREATE TRIGGER IF NOT EXISTS trigger_messages_after_delete AFTER DELETE ON messages BEGIN
    INSERT INTO messages_fts5(messages_fts5, rowid, role, content)
    VALUES ('delete', old.id, old.role, old.content);
END;
CREATE TABLE IF NOT EXISTS file_history (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    dir        TEXT    NOT NULL,
    name       TEXT    NOT NULL,
    action     TEXT    NOT NULL,
    content    BLOB,
    hash       TEXT    NOT NULL DEFAULT '',
    size       INTEGER NOT NULL DEFAULT 0,
    truncated  INTEGER NOT NULL DEFAULT 0,
    trash_path TEXT,
    session_id TEXT    NOT NULL DEFAULT '',
    task_id    TEXT    NOT NULL DEFAULT '',
    tool       TEXT    NOT NULL DEFAULT '',
    changed_at INTEGER NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_fh_unique ON file_history(dir, name, changed_at, task_id, session_id);

CREATE INDEX IF NOT EXISTS idx_fh_path    ON file_history(dir, name, changed_at DESC);
CREATE INDEX IF NOT EXISTS idx_fh_dir     ON file_history(dir, changed_at DESC);
CREATE INDEX IF NOT EXISTS idx_fh_time    ON file_history(changed_at DESC);
CREATE INDEX IF NOT EXISTS idx_fh_session ON file_history(session_id, changed_at DESC);

CREATE TABLE IF NOT EXISTS action_history (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id    TEXT    NOT NULL,
    task_hash     TEXT    NOT NULL,
    end_at        INTEGER NOT NULL,
    model         TEXT    NOT NULL DEFAULT '',
    reasoning     TEXT    NOT NULL DEFAULT '',
    objective     TEXT    NOT NULL DEFAULT '',
    tool_results  TEXT    NOT NULL DEFAULT '',
    reply         TEXT    NOT NULL DEFAULT ''
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_ah_unique
    ON action_history(session_id, task_hash);

CREATE INDEX IF NOT EXISTS idx_ah_session_end
    ON action_history(session_id, end_at DESC);

CREATE INDEX IF NOT EXISTS idx_ah_end
    ON action_history(end_at);

CREATE TABLE IF NOT EXISTS session (
    session_id TEXT PRIMARY KEY,
    self_id    TEXT NOT NULL DEFAULT '',
    name       TEXT NOT NULL DEFAULT '',
    model      TEXT NOT NULL DEFAULT 'auto',
    reasoning  TEXT NOT NULL DEFAULT 'medium',
    rule       TEXT NOT NULL DEFAULT '',
    chat_id    TEXT NOT NULL DEFAULT '',
    guild_id   TEXT NOT NULL DEFAULT '',
    channel_id TEXT NOT NULL DEFAULT '',
    user_id    TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_session_chat    ON session(chat_id)    WHERE chat_id    <> '';
CREATE INDEX IF NOT EXISTS idx_session_channel ON session(channel_id) WHERE channel_id <> '';
CREATE INDEX IF NOT EXISTS idx_session_name    ON session(name)       WHERE name       <> '';

CREATE UNIQUE INDEX IF NOT EXISTS idx_session_self_id
    ON session(self_id) WHERE self_id <> '';

DROP TABLE IF EXISTS state;
