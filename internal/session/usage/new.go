package usage

import (
	"context"
	_ "embed"
	"fmt"
	"log/slog"
	"time"

	go_sqlkit_core "github.com/pardnchiu/go-sqlkit/core"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
)

//go:embed migrate.sql
var migrateSQL string

var conn *go_sqlkit_core.Connector

const retainDays = 28

func New() error {
	c := filesystem.DB()
	if c == nil {
		return fmt.Errorf("internal/filesystem: OpenDB has not run")
	}
	if _, err := c.Exec(migrateSQL); err != nil {
		return fmt.Errorf("sql.DB Exec [migrate usage]: %w", err)
	}

	if _, err := c.Exec(
		`UPDATE usage SET send_at = send_at * 1000000000 WHERE send_at < 1000000000000`); err != nil {
		return fmt.Errorf("sql.DB Exec [usage send_at to nano]: %w", err)
	}

	conn = c
	return nil
}

func Retain() {
	if conn == nil {
		return
	}

	cutoff := time.Now().AddDate(0, 0, -retainDays).UnixNano()
	if _, err := conn.ExecContext(context.Background(),
		`DELETE FROM usage WHERE send_at < ?`, cutoff); err != nil {
		slog.Warn("usage.Retain",
			slog.String("error", err.Error()))
	}
}

func Close() {
	conn = nil
}
