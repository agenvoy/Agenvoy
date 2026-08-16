package file

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"

	go_pkg_filesystem "github.com/pardnchiu/go-pkg/filesystem"
	go_pkg_utils "github.com/pardnchiu/go-pkg/utils"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
	historyStore "github.com/pardnchiu/agenvoy/internal/runtime/history"
	"github.com/pardnchiu/agenvoy/internal/tools/file/denied"
	toolRegister "github.com/pardnchiu/agenvoy/internal/tools/register"
	toolTypes "github.com/pardnchiu/agenvoy/internal/tools/types"
)

func registWriteFile() {
	toolRegister.Regist(toolRegister.Def{
		Name: "write_file",
		Description: `
Create a file that does not exist yet, or deliberately replace one wholesale — regenerating from scratch, or an explicit "overwrite this" request.
Any edit to a file that already exists goes to patch_file instead: re-sending a whole file to change part of it throws away text that was already correct, and leaves the same edit to be made again.
The result reports the file's real byte count, line count and first/last lines — check those to confirm the write landed, rather than writing again.`,
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "File to write (e.g. '/abs/path/foo.go', '~/notes.md'). Exports with no path given belong in ~/Downloads or ~/.config/agenvoy/download/.",
					"default":     "",
				},
				"content": map[string]any{
					"type":        "string",
					"description": "Content to write.",
				},
			},
			"required": []string{
				"content",
			},
		},
		Handler: func(ctx context.Context, e *toolTypes.Executor, args json.RawMessage) (string, error) {
			var params struct {
				Path    string `json:"path"`
				Content string `json:"content"`
			}
			if err := json.Unmarshal(args, &params); err != nil {
				return "", fmt.Errorf("json.Unmarshal: %w", err)
			}

			baseDir := e.WorkDir
			if baseDir == "" {
				baseDir = filesystem.DownloadDir
			}

			path := strings.TrimSpace(params.Path)
			absPath, err := go_pkg_filesystem.AbsPath(baseDir, path, go_pkg_filesystem.AbsPathOption{HomeOnly: true})
			if err != nil {
				return "", fmt.Errorf("github.com/pardnchiu/go-pkg/filesystem: AbsPath: %w", err)
			}
			if absPath == "" {
				return "", fmt.Errorf("path is required")
			}

			content := params.Content
			if content == "" {
				return "", fmt.Errorf("content is required")
			}

			if parent, ok := denied.Hit(e.SessionID, absPath); ok {
				return "", fmt.Errorf("permission denied: %s is under previously rejected %s; not retried", absPath, parent)
			}

			info, err := os.Stat(absPath)
			isNew := os.IsNotExist(err)
			if err != nil && !isNew {
				if denied.IsPermission(err) {
					denied.Register(e.SessionID, absPath)
					return "", fmt.Errorf("permission denied: %s (recorded; further writes under this path will be skipped)", absPath)
				}
				return "", fmt.Errorf("os.Stat: %w", err)
			}
			if !isNew && info.Size() > maxReadSize {
				return "", fmt.Errorf("file too large (%d bytes, max 1 MB)", info.Size())
			}

			change, err := historyStore.Capture(absPath)
			if err != nil {
				slog.Warn("historyStore.Capture",
					slog.String("path", absPath),
					slog.String("error", err.Error()))
			}

			if err := go_pkg_filesystem.WriteFile(absPath, content, 0644); err != nil {
				if denied.IsPermission(err) {
					denied.Register(e.SessionID, absPath)
					return "", fmt.Errorf("permission denied: %s (recorded; further writes under this path will be skipped)", absPath)
				}
				return "", fmt.Errorf("github.com/pardnchiu/go-pkg/filesystem: WriteFile: %w", err)
			}

			var unrecorded string
			if err := historyStore.Record(ctx, change.WithCreated(content), historyStore.Meta{SessionID: e.SessionID, TaskID: e.PendingTask, Tool: "write_file"}); err != nil {
				slog.Warn("historyStore.Record",
					slog.String("path", absPath),
					slog.String("error", err.Error()))
				unrecorded = fmt.Sprintf("\nthe previous version was not recorded (%v), so this edit cannot be undone", err)
			}

			return writeReceipt(isNew, absPath, content) + unrecorded, nil
		},
	})
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
