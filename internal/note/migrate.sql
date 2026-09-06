CREATE TABLE IF NOT EXISTS note (
    name       TEXT PRIMARY KEY,
    content    TEXT    NOT NULL DEFAULT '',
    updated_at INTEGER NOT NULL DEFAULT 0
);

CREATE VIRTUAL TABLE IF NOT EXISTS note_fts5 USING fts5(
    name, content,
    content=note, content_rowid=rowid,
    tokenize='trigram'
);

CREATE TRIGGER IF NOT EXISTS trigger_note_after_insert AFTER INSERT ON note BEGIN
    INSERT INTO note_fts5(rowid, name, content)
    VALUES (new.rowid, new.name, new.content);
END;

CREATE TRIGGER IF NOT EXISTS trigger_note_after_delete AFTER DELETE ON note BEGIN
    INSERT INTO note_fts5(note_fts5, rowid, name, content)
    VALUES ('delete', old.rowid, old.name, old.content);
END;

CREATE TRIGGER IF NOT EXISTS trigger_note_after_update AFTER UPDATE ON note BEGIN
    INSERT INTO note_fts5(note_fts5, rowid, name, content)
    VALUES ('delete', old.rowid, old.name, old.content);
    INSERT INTO note_fts5(rowid, name, content)
    VALUES (new.rowid, new.name, new.content);
END;
