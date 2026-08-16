package variant

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	go_pkg_filesystem "github.com/pardnchiu/go-pkg/filesystem"

	historyStore "github.com/pardnchiu/agenvoy/internal/runtime/history"
	toolTypes "github.com/pardnchiu/agenvoy/internal/tools/types"
)

func Register() {
	registWriteTool()
	registPatchTool()
	registTestTool()
	registRemoveTool()
	registWriteSkill()
	registPatchSkill()
	registRemoveSkill()
}

func capture(path string) historyStore.Change {
	change, err := historyStore.Capture(path)
	if err != nil {
		slog.Warn("historyStore.Capture",
			slog.String("path", path),
			slog.String("error", err.Error()))
	}
	return change
}

func record(ctx context.Context, e *toolTypes.Executor, change historyStore.Change, created, tool string) {
	if err := historyStore.Record(ctx, change.WithCreated(created), metaOf(e, tool)); err != nil {
		slog.Warn("historyStore.Record",
			slog.String("tool", tool),
			slog.String("error", err.Error()))
	}
}

func recordRemoval(ctx context.Context, e *toolTypes.Executor, path, trashPath, tool string) {
	if err := historyStore.RecordDelete(ctx, path, trashPath, metaOf(e, tool)); err != nil {
		slog.Warn("historyStore.RecordDelete",
			slog.String("path", path),
			slog.String("error", err.Error()))
	}
}

func metaOf(e *toolTypes.Executor, tool string) historyStore.Meta {
	if e == nil {
		return historyStore.Meta{Tool: tool}
	}
	return historyStore.Meta{SessionID: e.SessionID, TaskID: e.PendingTask, Tool: tool}
}

func patch(path, old, new string, replaceAll bool) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("os.Stat [%s]: %w", path, err)
	}
	if info.Size() > 1<<20 {
		return fmt.Errorf("file too large (%d bytes, max 1 MB)", info.Size())
	}

	content, err := go_pkg_filesystem.ReadText(path)
	if err != nil {
		return fmt.Errorf("github.com/pardnchiu/go-pkg/filesystem: ReadText [%s]: %w", path, err)
	}

	if !strings.Contains(content, old) {
		return fmt.Errorf("%s is not found in %s", old, path)
	}

	search := old
	if new == "" && !strings.HasSuffix(old, "\n") && strings.Contains(content, old+"\n") {
		search = old + "\n"
	}
	var updated string
	if replaceAll {
		updated = strings.ReplaceAll(content, search, new)
	} else {
		updated = strings.Replace(content, search, new, 1)
	}

	if err := go_pkg_filesystem.WriteFile(path, updated, 0644); err != nil {
		return fmt.Errorf("github.com/pardnchiu/go-pkg/filesystem: WriteFile [%s]: %w", path, err)
	}
	return nil
}
