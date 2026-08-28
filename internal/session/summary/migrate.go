package summary

import (
	"log/slog"
	"os"
	"strings"

	go_pkg_filesystem "github.com/pardnchiu/go-pkg/filesystem"
	go_pkg_filesystem_reader "github.com/pardnchiu/go-pkg/filesystem/reader"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
)

type legacyMeta struct {
	LastMessageTime string `json:"last_message_time"`
}

func MigrateCursor() {
	dirs, err := go_pkg_filesystem_reader.ListDirs(filesystem.SessionsDir)
	if err != nil {
		slog.Warn("summary migrate: ListDirs",
			slog.String("dir", filesystem.SessionsDir),
			slog.String("error", err.Error()))
		return
	}

	var migrated, broken int
	for _, one := range dirs {
		if strings.HasPrefix(one.Name, ".") {
			continue
		}
		switch migrateCursorSession(one.Name) {
		case 1:
			migrated++
		case -1:
			broken++
		}
	}

	if migrated+broken > 0 {
		slog.Info("⎯ summary cursor migrated",
			slog.Int("migrated", migrated),
			slog.Int("broken", broken))
	}
}

func migrateCursorSession(sessionID string) int {
	legacy := filesystem.LegacySummaryMetaPath(sessionID)
	if !go_pkg_filesystem_reader.Exists(legacy) {
		return 0
	}

	defer func() {
		if err := os.Remove(legacy); err != nil && !os.IsNotExist(err) {
			slog.Warn("summary migrate: os.Remove",
				slog.String("path", legacy),
				slog.String("error", err.Error()))
		}
	}()

	meta, err := go_pkg_filesystem.ReadJSON[legacyMeta](legacy)
	if err != nil {
		slog.Warn("summary migrate: ReadJSON",
			slog.String("path", legacy),
			slog.String("error", err.Error()))
		return -1
	}

	cursor := strings.TrimSpace(meta.LastMessageTime)
	if cursor == "" {
		return -1
	}
	if go_pkg_filesystem_reader.Exists(filesystem.SummaryCursorPath(sessionID)) {
		return 0
	}

	SaveCursor(sessionID, cursor)
	return 1
}
