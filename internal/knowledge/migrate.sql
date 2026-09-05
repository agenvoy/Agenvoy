CREATE TABLE IF NOT EXISTS knowledge (
    name       TEXT PRIMARY KEY,
    content    TEXT    NOT NULL DEFAULT '',
    updated_at INTEGER NOT NULL DEFAULT 0
);

CREATE VIRTUAL TABLE IF NOT EXISTS knowledge_fts5 USING fts5(
    name, content,
    content=knowledge, content_rowid=rowid,
    tokenize='trigram'
);

CREATE TRIGGER IF NOT EXISTS trigger_knowledge_after_insert AFTER INSERT ON knowledge BEGIN
    INSERT INTO knowledge_fts5(rowid, name, content)
    VALUES (new.rowid, new.name, new.content);
END;

CREATE TRIGGER IF NOT EXISTS trigger_knowledge_after_delete AFTER DELETE ON knowledge BEGIN
    INSERT INTO knowledge_fts5(knowledge_fts5, rowid, name, content)
    VALUES ('delete', old.rowid, old.name, old.content);
END;

CREATE TRIGGER IF NOT EXISTS trigger_knowledge_after_update AFTER UPDATE ON knowledge BEGIN
    INSERT INTO knowledge_fts5(knowledge_fts5, rowid, name, content)
    VALUES ('delete', old.rowid, old.name, old.content);
    INSERT INTO knowledge_fts5(rowid, name, content)
    VALUES (new.rowid, new.name, new.content);
END;
