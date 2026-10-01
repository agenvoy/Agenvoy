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
	if err := addSessionRole(c); err != nil {
		return err
	}
	if _, err := c.Exec(migrateSQL); err != nil {
		return fmt.Errorf("sql.DB Exec [migrate]: %w", err)
	}

	conn = c
	return nil
}

func addSessionRole(c *go_sqlkit_core.Connector) error {
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
		if name == "role" {
			found++
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("sql.Rows Err [PRAGMA table_info session]: %w", err)
	}
	if columns == 0 || found > 0 {
		return nil
	}

	if _, err := c.Exec(`ALTER TABLE session ADD COLUMN role TEXT NOT NULL DEFAULT ''`); err != nil {
		return fmt.Errorf("sql.DB Exec [ALTER TABLE session ADD COLUMN role]: %w", err)
	}
	if _, err := c.Exec(`UPDATE session SET role = rule WHERE role = '' AND rule <> ''`); err != nil {
		return fmt.Errorf("sql.DB Exec [UPDATE session SET role]: %w", err)
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
