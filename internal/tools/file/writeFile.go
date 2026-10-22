package file

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	go_pkg_filesystem "github.com/pardnchiu/go-pkg/filesystem"
	go_pkg_utils "github.com/pardnchiu/go-pkg/utils"

	"github.com/pardnchiu/agenvoy/configs"
	"github.com/pardnchiu/agenvoy/internal/filesystem"
	historystore "github.com/pardnchiu/agenvoy/internal/runtime/store"
	"github.com/pardnchiu/agenvoy/internal/tools/file/boundary"
	tooltypes "github.com/pardnchiu/agenvoy/internal/tools/types"
	utilsmsg "github.com/pardnchiu/agenvoy/internal/utils/msg"
)

func writeFileContent(ctx context.Context, e *tooltypes.Executor, path0, content0, tool string) (string, error) {
	baseDir := e.WorkDir
	if baseDir == "" {
		baseDir = filesystem.DownloadDir
	}

	path := strings.TrimSpace(path0)
	absPath, err := boundary.Resolve(e.SessionID, baseDir, path)
	if err != nil {
		return "", utilsmsg.Trace("boundary.Resolve", "", err)
	}
	if absPath == "" {
		return "", utilsmsg.New(utilsmsg.Invalid, "path", "required")
	}

	content := content0
	if content == "" {
		return "", utilsmsg.New(utilsmsg.Invalid, "content", "required")
	}

	info, err := os.Stat(absPath)
	isNew := os.IsNotExist(err)
	if err != nil && !isNew {
		return "", utilsmsg.Wrap(utilsmsg.Internal, "os.Stat", "", err)
	}
	if !isNew && info.Size() > configs.MAX_DOCUMENT_BYTES {
		return "", utilsmsg.New(utilsmsg.Limit, "file", "too large (%d bytes, max %d MiB)", info.Size(), configs.MAX_DOCUMENT_BYTES>>20)
	}
	if !isNew {
		if err := requireFresh(e, absPath, info.ModTime()); err != nil {
			return "", err
		}
	}

	change, err := historystore.Capture(absPath)
	if err != nil {
		slog.Debug("historystore.Capture",
			slog.String("path", absPath),
			slog.String("error", err.Error()))
	}

	if err := go_pkg_filesystem.WriteFile(absPath, content, 0644); err != nil {
		return "", utilsmsg.Wrap(utilsmsg.Internal, "go_pkg_filesystem.WriteFile", "", err)
	}

	var unrecorded string
	markWritten(e, absPath)
	e.RecordFile(absPath)

	if err := historystore.Record(ctx, change.WithCreated(content), historystore.Meta{SessionID: e.SessionID, TaskID: e.PendingTask, Tool: tool}); err != nil {
		slog.Debug("historystore.Record",
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
