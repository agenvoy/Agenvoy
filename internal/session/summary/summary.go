package summary

import (
	"encoding/json"
	"log/slog"
	"os"
	"strings"

	go_pkg_filesystem "github.com/pardnchiu/go-pkg/filesystem"
	go_pkg_filesystem_reader "github.com/pardnchiu/go-pkg/filesystem/reader"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
)

func GetCursor(sessionID string) string {
	path := filesystem.SummaryCursorPath(sessionID)
	if !go_pkg_filesystem_reader.Exists(path) {
		return ""
	}

	text, err := go_pkg_filesystem.ReadText(path)
	if err != nil {
		slog.Warn("github.com/pardnchiu/go-pkg/filesystem ReadText",
			slog.String("path", path),
			slog.String("error", err.Error()))
		return ""
	}
	return strings.TrimSpace(text)
}

func SaveCursor(sessionID, cursor string) {
	if err := go_pkg_filesystem.WriteText(filesystem.SummaryCursorPath(sessionID), cursor); err != nil {
		slog.Warn("github.com/pardnchiu/go-pkg/filesystem WriteText",
			slog.String("path", filesystem.SummaryCursorPath(sessionID)),
			slog.String("error", err.Error()))
	}
}

func Get(sessionID string) ([]byte, map[string]any) {
	text, err := go_pkg_filesystem.ReadText(filesystem.SummaryPath(sessionID))
	if err != nil {
		return nil, nil
	}
	raw := []byte(text)

	var dic map[string]any
	if err := json.Unmarshal(raw, &dic); err != nil {
		slog.Warn("json Unmarshal",
			slog.String("path", filesystem.SummaryPath(sessionID)),
			slog.String("error", err.Error()))
		dic = map[string]any{}
		Save(sessionID, dic)
		return []byte("{}"), dic
	}
	return raw, dic
}

func Save(sessionID string, data any) {
	if err := go_pkg_filesystem.WriteJSON(filesystem.SummaryPath(sessionID), data, false); err != nil {
		slog.Warn("github.com/pardnchiu/go-pkg/filesystem WriteJSON",
			slog.String("path", filesystem.SummaryPath(sessionID)),
			slog.String("error", err.Error()))
	}
}

func Ensure(sessionID string) ([]byte, map[string]any) {
	raw, summary := Get(sessionID)
	if raw != nil {
		return raw, summary
	}

	empty := map[string]any{}
	Save(sessionID, empty)
	raw, summary = Get(sessionID)
	if raw != nil {
		return raw, summary
	}

	return []byte("{}"), empty
}

func Pending() []string {
	dirs, err := go_pkg_filesystem_reader.ListDirs(filesystem.SessionsDir)
	if err != nil {
		return nil
	}

	var list []string
	for _, dir := range dirs {
		sid := dir.Name
		if strings.HasPrefix(sid, "temp-") {
			continue
		}
		historyPath := filesystem.HistoryPath(sid)
		hInfo, err := os.Stat(historyPath)
		if err != nil {
			continue
		}

		summaryPath := filesystem.SummaryPath(sid)
		info, err := os.Stat(summaryPath)
		if err != nil || hInfo.ModTime().After(info.ModTime()) {
			list = append(list, sid)
		}
	}
	return list
}
