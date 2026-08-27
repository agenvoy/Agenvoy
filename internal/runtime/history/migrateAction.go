package historyStore

import (
	"context"
	"encoding/json"
	"log/slog"
	"path/filepath"
	"strings"
	"time"

	go_pkg_filesystem "github.com/pardnchiu/go-pkg/filesystem"
	go_pkg_filesystem_reader "github.com/pardnchiu/go-pkg/filesystem/reader"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
)

const (
	importedSuffix = ".imported"
	nameLayout     = "2006-01-02-15-04"
)

func MigrateAction() {
	if conn == nil {
		return
	}

	dirs, err := go_pkg_filesystem_reader.ListDirs(filesystem.SessionsDir)
	if err != nil {
		slog.Warn("action migrate: ListDirs",
			slog.String("dir", filesystem.SessionsDir),
			slog.String("error", err.Error()))
		return
	}

	var imported, broken int
	for _, one := range dirs {
		if strings.HasPrefix(one.Name, ".") {
			continue
		}
		i, b := migrateActionSession(one.Name)
		imported += i
		broken += b
	}

	if imported+broken > 0 {
		slog.Info("⎯ task history migrated into sqlite",
			slog.Int("imported", imported),
			slog.Int("broken", broken))
	}
}

func migrateActionSession(sessionID string) (int, int) {
	dir := filesystem.TaskHistoryDir(sessionID)
	if !go_pkg_filesystem_reader.IsDir(dir) {
		return 0, 0
	}

	staged := dir + importedSuffix
	if err := go_pkg_filesystem.Move(dir, staged); err != nil {
		return 0, 0
	}

	files, err := go_pkg_filesystem_reader.ListFiles(staged)
	if err != nil {
		slog.Warn("action migrate: ListFiles",
			slog.String("dir", staged),
			slog.String("error", err.Error()))
		if err := go_pkg_filesystem.Move(staged, dir); err != nil {
			slog.Warn("action migrate: restore",
				slog.String("dir", staged),
				slog.String("error", err.Error()))
		}
		return 0, 0
	}

	ctx := context.Background()
	loc := time.Now().Location()
	var imported, broken int
	for _, one := range files {
		name, ok := strings.CutSuffix(one.Name, ".json")
		if !ok || len(name) <= len(nameLayout)+1 {
			broken++
			continue
		}
		endAt, err := time.ParseInLocation(nameLayout, name[:len(nameLayout)], loc)
		if err != nil {
			broken++
			continue
		}

		raw, err := go_pkg_filesystem.ReadText(filepath.Join(staged, one.Name))
		if err != nil {
			broken++
			continue
		}

		var row ActionRecord
		if err := json.Unmarshal([]byte(raw), &row); err != nil {
			broken++
			continue
		}
		row.TaskHash = name[len(nameLayout)+1:]
		row.EndAt = endAt

		if err := WriteAction(ctx, sessionID, row); err != nil {
			slog.Warn("action migrate: WriteAction",
				slog.String("session", sessionID),
				slog.String("task", row.TaskHash),
				slog.String("error", err.Error()))
			broken++
			continue
		}
		imported++
	}
	return imported, broken
}
