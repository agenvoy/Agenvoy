package fileHistory

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	historyStore "github.com/pardnchiu/agenvoy/internal/runtime/history"
	toolRegister "github.com/pardnchiu/agenvoy/internal/tools/register"
	toolTypes "github.com/pardnchiu/agenvoy/internal/tools/types"
)

func registRestoreFile() {
	toolRegister.Regist(toolRegister.Def{
		Name: "restore_file",
		Description: `
Puts files back to a recorded version: one version by id, or every file a task touched.
Use for 還原 / 復原 / 撤銷剛剛的修改 / 改回上一版 — never guess which version they meant: file_history, then ask_user with one option per version carrying its date, its objective and its task id, then restore what they chose.
Versions come from file_history; task ids from that or chat_history.`,
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"version": map[string]any{
					"type":        "integer",
					"description": "The version to go back to, from a file_history row. Enough on its own — it names both the file and the state.",
				},
				"task_id": map[string]any{
					"type":        "string",
					"description": "Undo a whole task instead, when no version is given: every file it touched, back to before it ran. 'current' for the task running now.",
				},
				"paths": map[string]any{
					"type":        "array",
					"description": "Narrow a task_id undo to these files (e.g. '/abs/path/foo.go', '~/notes.md'). Omit to undo every file that task touched.",
					"items": map[string]any{
						"type": "string",
					},
				},
			},
		},
		Handler: func(ctx context.Context, e *toolTypes.Executor, args json.RawMessage) (string, error) {
			var params struct {
				Version int64    `json:"version"`
				TaskID  string   `json:"task_id"`
				Paths   []string `json:"paths"`
			}
			if err := json.Unmarshal(args, &params); err != nil {
				return "", fmt.Errorf("encoding/json: Unmarshal: %w", err)
			}

			taskID := strings.TrimSpace(params.TaskID)
			if taskID == "current" {
				taskID = e.PendingTask
			}

			meta := historyStore.Meta{
				SessionID: e.SessionID,
				TaskID:    e.PendingTask,
				Tool:      "restore_file",
			}

			switch {
			case params.Version > 0:
				out, err := historyStore.RestoreTo(ctx, params.Version, meta)
				if err != nil {
					return "", fmt.Errorf("internal/runtime/history: RestoreTo [%d]: %w", params.Version, err)
				}
				return out, nil
			case taskID == "":
				return "", fmt.Errorf("version or task_id is required")
			}

			filters := []historyStore.Filter{{TaskID: taskID}}
			if len(params.Paths) > 0 {
				filters = filters[:0]
				for _, raw := range params.Paths {
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
		},
	})
}
