package usage

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
	"time"

	go_pkg_filesystem "github.com/pardnchiu/go-pkg/filesystem"
	go_pkg_filesystem_reader "github.com/pardnchiu/go-pkg/filesystem/reader"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
)

const (
	importedSuffix  = ".imported"
	timestampLayout = "2006-01-02 15:04:05.000"
)

var linePattern = regexp.MustCompile(`^\[(\d{4}-\d{2}-\d{2} [^\]]+)\]\[([^\]]+)\]\s+in/\s*(\d+)\s+out/\s*(\d+)\s+write/\s*(\d+)\s+hit/\s*(\d+)`)

func Migrate() {
	if conn == nil {
		return
	}

	dirs, err := go_pkg_filesystem_reader.ListDirs(filesystem.SessionsDir)
	if err != nil {
		slog.Warn("usage migrate: ListDirs",
			slog.String("dir", filesystem.SessionsDir),
			slog.String("error", err.Error()))
		return
	}

	cutoff := time.Now().AddDate(0, 0, -retainDays).UnixNano()
	var imported, expired, broken int
	for _, one := range dirs {
		if strings.HasPrefix(one.Name, ".") {
			continue
		}
		i, e, b := migrateSession(one.Name, cutoff)
		imported += i
		expired += e
		broken += b
	}

	if imported+expired+broken > 0 {
		slog.Info("⎯ usage.log migrated into sqlite",
			slog.Int("imported", imported),
			slog.Int("expired", expired),
			slog.Int("broken", broken))
	}
}

func migrateSession(sessionID string, cutoff int64) (int, int, int) {
	path := filesystem.UsageLogPath(sessionID)
	staged := path + importedSuffix
	if err := go_pkg_filesystem.Move(path, staged); err != nil {
		return 0, 0, 0
	}

	imported, expired, broken, err := importFile(sessionID, staged, cutoff)
	if err == nil {
		return imported, expired, broken
	}

	slog.Warn("usage migrate",
		slog.String("session", sessionID),
		slog.String("error", err.Error()))
	if err := go_pkg_filesystem.Move(staged, path); err != nil {
		slog.Warn("usage migrate: restore",
			slog.String("file", staged),
			slog.String("error", err.Error()))
	}
	return 0, 0, 0
}

func importFile(sessionID, staged string, cutoff int64) (int, int, int, error) {
	text, err := go_pkg_filesystem.ReadText(staged)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("github.com/pardnchiu/go-pkg/filesystem ReadText [%s]: %w", staged, err)
	}

	ctx := context.Background()
	tx, err := conn.Write.BeginTx(ctx, nil)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("sql.DB BeginTx: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
	INSERT INTO usage (session_id, send_at, model, input, output, write, hit)
	VALUES (?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("sql.Tx PrepareContext [INSERT usage]: %w", err)
	}
	defer stmt.Close()

	loc := time.Now().Location()
	var imported, expired, broken int
	for line := range strings.SplitSeq(text, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}

		matches := linePattern.FindStringSubmatch(line)
		if len(matches) != 7 {
			broken++
			continue
		}
		at, parseErr := time.ParseInLocation(timestampLayout, matches[1], loc)
		if parseErr != nil {
			broken++
			continue
		}
		if at.UnixNano() < cutoff {
			expired++
			continue
		}

		values, ok := parseCounts(matches[3:7])
		if !ok {
			broken++
			continue
		}

		if _, err := stmt.ExecContext(ctx, sessionID, at.UnixNano(), matches[2],
			values[0], values[1], values[2], values[3]); err != nil {
			return 0, 0, 0, fmt.Errorf("sql.Stmt ExecContext: %w", err)
		}
		imported++
	}

	if err := tx.Commit(); err != nil {
		return 0, 0, 0, fmt.Errorf("sql.Tx Commit: %w", err)
	}
	return imported, expired, broken, nil
}

func parseCounts(fields []string) ([4]int64, bool) {
	var values [4]int64
	for i, one := range fields {
		n, err := strconv.ParseInt(one, 10, 64)
		if err != nil {
			return values, false
		}
		values[i] = n
	}
	return values, true
}
