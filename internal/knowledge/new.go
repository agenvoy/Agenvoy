package knowledge

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
	if _, err := c.Exec(migrateSQL); err != nil {
		return fmt.Errorf("sql.DB Exec [migrate knowledge]: %w", err)
	}
	if err := rebuildIndex(c); err != nil {
		return err
	}

	conn = c
	return nil
}

func rebuildIndex(c *go_sqlkit_core.Connector) error {
	if _, err := c.Exec(`INSERT INTO knowledge_fts5(knowledge_fts5) VALUES('rebuild')`); err != nil {
		return fmt.Errorf("sql.DB Exec [rebuild knowledge_fts5]: %w", err)
	}
	return nil
}

func Close() {
	conn = nil
}
