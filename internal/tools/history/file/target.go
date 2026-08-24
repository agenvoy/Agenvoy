package fileHistory

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
	historyStore "github.com/pardnchiu/agenvoy/internal/runtime/history"
	"github.com/pardnchiu/agenvoy/internal/tools/file/boundary"
	toolTypes "github.com/pardnchiu/agenvoy/internal/tools/types"
)

var timeLayouts = []string{
	time.RFC3339,
	historyStore.TimeLayout,
	"2006-01-02T15:04:05",
	"2006-01-02 15:04",
	"2006-01-02",
}

func absPath(e *toolTypes.Executor, path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", nil
	}

	baseDir := e.WorkDir
	if baseDir == "" {
		baseDir = filesystem.DownloadDir
	}

	abs, err := boundary.Resolve(e.SessionID, baseDir, path)
	if err != nil {
		return "", fmt.Errorf("boundary.Resolve: %w", err)
	}
	return filepath.Clean(abs), nil
}

func parseTime(value string) (int64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, nil
	}

	for _, layout := range timeLayouts {
		if t, err := time.ParseInLocation(layout, value, time.Local); err == nil {
			return t.UnixNano(), nil
		}
	}
	return 0, fmt.Errorf("cannot read %q as a time: use '2026-08-13', '2026-08-13 15:04', '2026-08-13 15:04:05'", value)
}
