package usage

import (
	"context"
	_ "embed"
	"log/slog"
	"strings"
	"time"

	go_sqlkit_core "github.com/pardnchiu/go-sqlkit/core"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
	utilsmsg "github.com/pardnchiu/agenvoy/internal/utils/msg"
)

//go:embed migrate.sql
var migrateSQL string

var conn *go_sqlkit_core.Connector

const retainDays = 28

func New() error {
	c := filesystem.DB()
	if c == nil {
		return utilsmsg.New(utilsmsg.Internal, "filesystem.DB", "OpenDB has not run")
	}
	if _, err := c.Exec(migrateSQL); err != nil {
		return utilsmsg.Wrap(utilsmsg.Internal, "sql.DB Exec", "[migrate usage]", err)
	}

	if _, err := c.Exec(
		`ALTER TABLE usage ADD COLUMN elapsed_ms INTEGER NOT NULL DEFAULT 0`); err != nil &&
		!strings.Contains(err.Error(), "duplicate column name") {
		return utilsmsg.Wrap(utilsmsg.Internal, "sql.DB Exec", "[usage add elapsed_ms]", err)
	}

	if _, err := c.Exec(
		`ALTER TABLE usage ADD COLUMN tool_calls TEXT NOT NULL DEFAULT ''`); err != nil &&
		!strings.Contains(err.Error(), "duplicate column name") {
		return utilsmsg.Wrap(utilsmsg.Internal, "sql.DB Exec", "[usage add tool_calls]", err)
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
