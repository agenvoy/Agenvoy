package fileHistory

import (
	"context"
	"fmt"
	"strings"

	historyStore "github.com/pardnchiu/agenvoy/internal/runtime/history"
	toolTypes "github.com/pardnchiu/agenvoy/internal/tools/types"
)

func Restore(ctx context.Context, e *toolTypes.Executor, version int64, taskID0 string, paths []string) (string, error) {
	taskID := strings.TrimSpace(taskID0)
	if taskID == "current" {
		taskID = e.PendingTask
	}

	meta := historyStore.Meta{
		SessionID: e.SessionID,
		TaskID:    e.PendingTask,
		Tool:      "edit_file",
	}

	switch {
	case version > 0:
		out, err := historyStore.RestoreTo(ctx, version, meta)
		if err != nil {
			return "", fmt.Errorf("internal/runtime/history: RestoreTo [%d]: %w", version, err)
		}
		return out, nil
	case taskID == "":
		return "", fmt.Errorf("version or task_id is required")
	}

	filters := []historyStore.Filter{{TaskID: taskID}}
	if len(paths) > 0 {
		filters = filters[:0]
		for _, raw := range paths {
			path, err := absPath(e, raw)
			if err != nil {
				return "", err
			}
			filters = append(filters, historyStore.Filter{TaskID: taskID, Path: path})
		}
	}

	var report []string
	for _, filter := range filters {
		lines, err := historyStore.Undo(ctx, filter, meta)
		if err != nil {
			return "", fmt.Errorf("internal/runtime/history: Undo: %w", err)
		}
		if len(lines) == 0 {
			if filter.Path != "" {
				report = append(report, fmt.Sprintf("%s was not changed by task %s", filter.Path, taskID))
			}
			continue
		}
		report = append(report, lines...)
	}
	if len(report) == 0 {
		return fmt.Sprintf("task %s changed no files, so there is nothing to undo", taskID), nil
	}
	return strings.Join(report, "\n"), nil
}
