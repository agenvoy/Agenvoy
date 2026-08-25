package file

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	go_pkg_filesystem "github.com/pardnchiu/go-pkg/filesystem"
	go_pkg_utils "github.com/pardnchiu/go-pkg/utils"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
	historyStore "github.com/pardnchiu/agenvoy/internal/runtime/history"
	"github.com/pardnchiu/agenvoy/internal/tools/file/boundary"
	toolTypes "github.com/pardnchiu/agenvoy/internal/tools/types"
)

func writeFileContent(ctx context.Context, e *toolTypes.Executor, path0, content0 string) (string, error) {
	baseDir := e.WorkDir
	if baseDir == "" {
		baseDir = filesystem.DownloadDir
	}

	path := strings.TrimSpace(path0)
	absPath, err := boundary.Resolve(e.SessionID, baseDir, path)
	if err != nil {
		return "", fmt.Errorf("boundary.Resolve: %w", err)
	}
	if absPath == "" {
		return "", fmt.Errorf("path is required")
	}

	content := content0
	if content == "" {
		return "", fmt.Errorf("content is required")
	}

	info, err := os.Stat(absPath)
	isNew := os.IsNotExist(err)
	if err != nil && !isNew {
		return "", fmt.Errorf("os.Stat: %w", err)
	}
	if !isNew && info.Size() > maxReadSize {
		return "", fmt.Errorf("file too large (%d bytes, max 1 MB)", info.Size())
	}

	change, err := historyStore.Capture(absPath)
	if err != nil {
		slog.Debug("historyStore.Capture",
			slog.String("path", absPath),
			slog.String("error", err.Error()))
	}

	if err := go_pkg_filesystem.WriteFile(absPath, content, 0644); err != nil {
		return "", fmt.Errorf("github.com/pardnchiu/go-pkg/filesystem: WriteFile: %w", err)
	}

	var unrecorded string
	e.RecordFile(absPath)

	if err := historyStore.Record(ctx, change.WithCreated(content), historyStore.Meta{SessionID: e.SessionID, TaskID: e.PendingTask, Tool: "edit_file"}); err != nil {
		slog.Debug("historyStore.Record",
			slog.String("path", absPath),
			slog.String("error", err.Error()))
		unrecorded = fmt.Sprintf("\nthe previous version was not recorded (%v), so this edit cannot be undone", err)
	}

	return writeReceipt(isNew, absPath, content) + unrecorded, nil
}
func writeReceipt(isNew bool, path, content string) string {
	verb := "updated"
	if isNew {
		verb = "created"
	}

	size := len(content)
	if info, err := os.Stat(path); err == nil {
		size = int(info.Size())
	}

	lines := strings.Split(strings.TrimRight(content, "\n"), "\n")
	return fmt.Sprintf(
		"successfully %s: %s (%d bytes, %d lines on disk)\nfirst line: %s\nlast line: %s\nThis is the file's real content. The elided write argument left in history is a context-saving marker, not what was written — do not rewrite the file to \"restore\" it.",
		verb, path, size, len(lines),
		excerptLine(lines[0]), excerptLine(lines[len(lines)-1]),
	)
}

func excerptLine(line string) string {
	line = strings.TrimSpace(line)
	if line == "" {
		return "(blank)"
	}
	return go_pkg_utils.TruncateString(line, 128)
}
