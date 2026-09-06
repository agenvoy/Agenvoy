package note

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
	if err := renameLegacyTable(c); err != nil {
		return err
	}
	if _, err := c.Exec(migrateSQL); err != nil {
		return fmt.Errorf("sql.DB Exec [migrate note]: %w", err)
	}
	if err := rebuildIndex(c); err != nil {
		return err
	}

	conn = c
	return nil
}

func renameLegacyTable(c *go_sqlkit_core.Connector) error {
	var legacy, current int
	if err := c.Read.QueryRow(`
	SELECT
		COALESCE(SUM(name = 'knowledge'), 0),
		COALESCE(SUM(name = 'note'), 0)
	FROM sqlite_master WHERE type = 'table'`).Scan(&legacy, &current); err != nil {
		return fmt.Errorf("sql.DB QueryRow [sqlite_master]: %w", err)
	}
	if legacy == 0 || current > 0 {
		return nil
	}

	for _, stmt := range []string{
		`DROP TRIGGER IF EXISTS trigger_knowledge_after_insert`,
		`DROP TRIGGER IF EXISTS trigger_knowledge_after_delete`,
		`DROP TRIGGER IF EXISTS trigger_knowledge_after_update`,
		`DROP TABLE IF EXISTS knowledge_fts5`,
		`ALTER TABLE knowledge RENAME TO note`,
	} {
		if _, err := c.Exec(stmt); err != nil {
			return fmt.Errorf("sql.DB Exec [%s]: %w", stmt, err)
		}
	}
	return nil
}

func rebuildIndex(c *go_sqlkit_core.Connector) error {
	if _, err := c.Exec(`INSERT INTO note_fts5(note_fts5) VALUES('rebuild')`); err != nil {
		return fmt.Errorf("sql.DB Exec [rebuild note_fts5]: %w", err)
	}
	return nil
}

func Close() {
	conn = nil
}
