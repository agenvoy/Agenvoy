package historyStore

import (
	_ "embed"
	"fmt"

	go_sqlkit_core "github.com/pardnchiu/go-sqlkit/core"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
)

//go:embed migrate.sql
var migrateSQL string

var conn *go_sqlkit_core.Connector

func New() error {
	c := filesystem.DB()
	if c == nil {
		return fmt.Errorf("internal/filesystem: OpenDB has not run")
	}
	if err := renameSessionMeta(c); err != nil {
		return err
	}
	if err := addSessionSelfID(c); err != nil {
		return err
	}
	if _, err := c.Exec(migrateSQL); err != nil {
		return fmt.Errorf("sql.DB Exec [migrate]: %w", err)
	}
	if err := syncColumns(c); err != nil {
		return err
	}

	conn = c
	return nil
}

func syncColumns(c *go_sqlkit_core.Connector) error {
	rows, err := c.Query(`PRAGMA table_info(messages)`)
	if err != nil {
		return fmt.Errorf("sql.DB Query [PRAGMA table_info]: %w", err)
	}
	defer rows.Close()

	existing := make(map[string]bool)
	for rows.Next() {
		var (
			cid, notNull, pk int
			name, dataType   string
			defaultValue     any
		)
		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &pk); err != nil {
			return fmt.Errorf("sql.Rows Scan [PRAGMA table_info]: %w", err)
		}
		existing[name] = true
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("sql.Rows Err [PRAGMA table_info]: %w", err)
	}

	for _, name := range []string{"sender"} {
		if existing[name] {
			continue
		}
		if _, err := c.Exec(fmt.Sprintf(`ALTER TABLE messages ADD COLUMN %s TEXT NOT NULL DEFAULT ''`, name)); err != nil {
			return fmt.Errorf("sql.DB Exec [ALTER TABLE messages ADD COLUMN %s]: %w", name, err)
		}
	}

	for _, name := range []string{"channel_id"} {
		if !existing[name] {
			continue
		}
		if _, err := c.Exec(fmt.Sprintf(`ALTER TABLE messages DROP COLUMN %s`, name)); err != nil {
			return fmt.Errorf("sql.DB Exec [ALTER TABLE messages DROP COLUMN %s]: %w", name, err)
		}
	}

	if err := backfillSessionDefaults(c); err != nil {
		return err
	}
	return dropActionColumns(c)
}

func backfillSessionDefaults(c *go_sqlkit_core.Connector) error {
	if _, err := c.Exec(`UPDATE session SET model = ? WHERE model = ''`, DefaultModel); err != nil {
		return fmt.Errorf("sql.DB Exec [UPDATE session model]: %w", err)
	}
	if _, err := c.Exec(`UPDATE session SET reasoning = ? WHERE reasoning = ''`, DefaultReasoning); err != nil {
		return fmt.Errorf("sql.DB Exec [UPDATE session reasoning]: %w", err)
	}
	if _, err := c.Exec(`UPDATE session SET self_id = LOWER(self_id) WHERE self_id <> LOWER(self_id)`); err != nil {
		return fmt.Errorf("sql.DB Exec [UPDATE session self_id]: %w", err)
	}
	return nil
}

func addSessionSelfID(c *go_sqlkit_core.Connector) error {
	rows, err := c.Query(`PRAGMA table_info(session)`)
	if err != nil {
		return fmt.Errorf("sql.DB Query [PRAGMA table_info session]: %w", err)
	}
	defer rows.Close()

	var columns, found int
	for rows.Next() {
		var (
			cid, notNull, pk int
			name, dataType   string
			defaultValue     any
		)
		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &pk); err != nil {
			return fmt.Errorf("sql.Rows Scan [PRAGMA table_info session]: %w", err)
		}
		columns++
		if name == "self_id" {
			found++
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("sql.Rows Err [PRAGMA table_info session]: %w", err)
	}
	if columns == 0 || found > 0 {
		return nil
	}

	if _, err := c.Exec(`ALTER TABLE session ADD COLUMN self_id TEXT NOT NULL DEFAULT ''`); err != nil {
		return fmt.Errorf("sql.DB Exec [ALTER TABLE session ADD COLUMN self_id]: %w", err)
	}
	return nil
}

func renameSessionMeta(c *go_sqlkit_core.Connector) error {
	var legacy, current int
	if err := c.Read.QueryRow(`
	SELECT
		COALESCE(SUM(name = 'session_meta'), 0),
		COALESCE(SUM(name = 'message_meta'), 0)
	FROM sqlite_master WHERE type = 'table'`).Scan(&legacy, &current); err != nil {
		return fmt.Errorf("sql.DB QueryRow [sqlite_master]: %w", err)
	}
	if legacy == 0 || current > 0 {
		return nil
	}

	if _, err := c.Exec(`ALTER TABLE session_meta RENAME TO message_meta`); err != nil {
		return fmt.Errorf("sql.DB Exec [ALTER TABLE session_meta RENAME]: %w", err)
	}
	return nil
}

func dropActionColumns(c *go_sqlkit_core.Connector) error {
	rows, err := c.Query(`PRAGMA table_info(action_history)`)
	if err != nil {
		return fmt.Errorf("sql.DB Query [PRAGMA table_info action_history]: %w", err)
	}
	defer rows.Close()

	existing := make(map[string]bool)
	for rows.Next() {
		var (
			cid, notNull, pk int
			name, dataType   string
			defaultValue     any
		)
		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &pk); err != nil {
			return fmt.Errorf("sql.Rows Scan [PRAGMA table_info action_history]: %w", err)
		}
		existing[name] = true
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("sql.Rows Err [PRAGMA table_info action_history]: %w", err)
	}

	for _, name := range []string{"completed", "next_steps", "answer", "tool_attempts"} {
		if !existing[name] {
			continue
		}
		if _, err := c.Exec(fmt.Sprintf(`ALTER TABLE action_history DROP COLUMN %s`, name)); err != nil {
			return fmt.Errorf("sql.DB Exec [ALTER TABLE action_history DROP COLUMN %s]: %w", name, err)
		}
	}

	if _, err := c.Exec(
		`UPDATE action_history SET end_at = end_at * 1000000000 WHERE end_at < 1000000000000`); err != nil {
		return fmt.Errorf("sql.DB Exec [action_history end_at to nano]: %w", err)
	}
	return nil
}

func Close() {
	conn = nil
}

func IsReady() bool {
	return conn != nil
}

func IsExist(sessionID string) bool {
	if conn == nil {
		return false
	}

	var exists bool
	conn.Read.QueryRow(`
	SELECT EXISTS(SELECT 1 FROM messages WHERE session_id = ?)
	`, sessionID).Scan(&exists)
	return exists
}

func SetStartAt(sessionID string, timestamp int64) error {
	if conn == nil {
		return nil
	}

	_, err := conn.Exec(`
	INSERT INTO message_meta (session_id, start_at)
	VALUES (?, ?)
	ON CONFLICT(session_id)
	DO UPDATE SET start_at = excluded.start_at
	`, sessionID, timestamp)
	return err
}

func GetStartAt(sessionID string) int64 {
	if conn == nil {
		return 0
	}

	var ts int64
	conn.Read.QueryRow(`
	SELECT start_at
	FROM message_meta
	WHERE session_id = ?
	`, sessionID).Scan(&ts)
	return ts
}
