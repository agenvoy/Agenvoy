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

func registRestoreFileHistory() {
	toolRegister.Regist(toolRegister.Def{
		Name: "restore_file_history",
		Description: `
Undoes one task: every file it touched goes back to how it stood before it ran.
Use for 還原 / 復原 / 撤銷剛剛的修改 / 改回上一版, once the user has agreed to what read_file_history showed.
Ids come from list_action_history or any list_file_history row.`,
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"task_id": map[string]any{
					"type":        "string",
					"description": "Task to undo, from list_action_history or a list_file_history row. 'current' for the task running now.",
				},
			},
			"required": []string{
				"task_id",
			},
		},
		Handler: func(ctx context.Context, e *toolTypes.Executor, args json.RawMessage) (string, error) {
			var params struct {
				TaskID string `json:"task_id"`
			}
			if err := json.Unmarshal(args, &params); err != nil {
				return "", fmt.Errorf("encoding/json: Unmarshal: %w", err)
			}

			taskID := strings.TrimSpace(params.TaskID)
			if taskID == "current" {
				taskID = e.PendingTask
			}
			if taskID == "" {
				return "", fmt.Errorf("task_id is required")
			}

			report, err := historyStore.Undo(ctx, historyStore.Filter{TaskID: taskID}, historyStore.Meta{
				SessionID: e.SessionID,
				TaskID:    e.PendingTask,
				Tool:      "restore_file_history",
			})
			if err != nil {
				return "", fmt.Errorf("internal/runtime/history: Undo: %w", err)
			}
			if len(report) == 0 {
				return fmt.Sprintf("task %s changed no files, so there is nothing to undo", taskID), nil
			}
			return strings.Join(report, "\n"), nil
		},
	})
}
