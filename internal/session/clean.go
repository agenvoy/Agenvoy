package session

import (
	"context"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
	historyStore "github.com/pardnchiu/agenvoy/internal/runtime/history"
)

func Clean() {
	entries, err := os.ReadDir(filesystem.SessionsDir)
	if err != nil {
		slog.Warn("os ReadDir",
			slog.String("dir", filesystem.SessionsDir),
			slog.String("error", err.Error()))
		return
	}
	now := time.Now()
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		sessionDir := filesystem.SessionDir(entry.Name())

		if strings.HasPrefix(entry.Name(), "temp-") {
			if now.Sub(latestModTime(sessionDir)) > 30*time.Minute {
				if !clearSessionRows(entry.Name()) {
					continue
				}
				if err := os.RemoveAll(sessionDir); err != nil {
					slog.Debug("os RemoveAll",
						slog.String("dir", entry.Name()),
						slog.String("error", err.Error()))
				}
			}
			continue
		}

	}
}

func latestModTime(dir string) time.Time {
	var latest time.Time
	_ = filepath.WalkDir(dir, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			slog.Debug("DirEntry Info",
				slog.String("entry", d.Name()),
				slog.String("error", err.Error()))
			return nil
		}
		if t := info.ModTime(); t.After(latest) {
			latest = t
		}
		return nil
	})
	return latest
}

func clearSessionRows(sessionID string) bool {
	ctx := context.Background()
	if err := historyStore.Clear(sessionID); err != nil {
		slog.Warn("historyStore.Clear",
			slog.String("session", sessionID),
			slog.String("error", err.Error()))
		return false
	}
	if err := historyStore.DeleteState(ctx, sessionID); err != nil {
		slog.Warn("historyStore.DeleteState",
			slog.String("session", sessionID),
			slog.String("error", err.Error()))
		return false
	}
	if err := historyStore.DeleteSession(ctx, sessionID); err != nil {
		slog.Warn("historyStore.DeleteSession",
			slog.String("session", sessionID),
			slog.String("error", err.Error()))
		return false
	}
	return true
}
