package fileHistory

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	historyStore "github.com/pardnchiu/agenvoy/internal/runtime/history"
	actionHistory "github.com/pardnchiu/agenvoy/internal/tools/history/action"
	toolRegister "github.com/pardnchiu/agenvoy/internal/tools/register"
	toolTypes "github.com/pardnchiu/agenvoy/internal/tools/types"
)

const (
	maxHistoryRows = 24
	defaultRows    = 3
)

func registListFileHistory() {
	toolRegister.Regist(toolRegister.Def{
		Name:        "list_file_history",
		AlwaysAllow: true,
		Concurrent:  true,
		SystemUse:   true,
		Description: `
Recorded changes to a file, newest first, each with the version it produced and what the user asked for at the time.
Use for 這個檔案動過幾次 / 什麼時候改的 / 上一版是什麼, with task_id for 那個動作動了哪些檔案, and always before restoring: 還原 means asking the user which of these versions they mean.
Timestamps and objectives only — the content itself is read_file_history.`,
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "The file to look back over (e.g. '/Users/me/notes.md', '~/notes.md', 'src/main.go'). Omit with task_id to see every file that task changed.",
				},
				"task_id": map[string]any{
					"type":        "string",
					"description": "Only what one task changed, from list_action_history. 'current' for the task running now.",
				},
				"from": map[string]any{
					"type":        "string",
					"description": "Local time to start from: '2026-08-13' (that day at 00:00), '2026-08-13 15:04', '2026-08-13 15:04:05'.",
				},
				"to": map[string]any{
					"type":        "string",
					"description": "Local time to stop at, same formats as from.",
				},
				"limit": map[string]any{
					"type":        "integer",
					"description": "Rows to return, newest first. Never above 24.",
					"default":     defaultRows,
				},
			},
		},
		Handler: func(ctx context.Context, e *toolTypes.Executor, args json.RawMessage) (string, error) {
			var params struct {
				Path   string `json:"path"`
				TaskID string `json:"task_id"`
				From   string `json:"from"`
				To     string `json:"to"`
				Limit  int    `json:"limit"`
			}
			if err := json.Unmarshal(args, &params); err != nil {
				return "", fmt.Errorf("encoding/json: Unmarshal: %w", err)
			}

			path, err := absPath(e, params.Path)
			if err != nil {
				return "", err
			}

			filter := historyStore.Filter{Path: path, TaskID: strings.TrimSpace(params.TaskID)}
			if filter.TaskID == "current" {
				filter.TaskID = e.PendingTask
			}
			if filter.From, err = parseTime(params.From); err != nil {
				return "", err
			}
			if filter.To, err = parseTime(params.To); err != nil {
				return "", err
			}

			filter.Limit = min(params.Limit, maxHistoryRows)
			if filter.Limit <= 0 {
				filter.Limit = defaultRows
			}

			list, err := historyStore.List(ctx, filter)
			if err != nil {
				return "", fmt.Errorf("internal/runtime/history: List: %w", err)
			}
			if len(list) == 0 {
				return "no recorded changes", nil
			}

			out := make([]map[string]any, 0, len(list))
			for _, row := range list {
				item := map[string]any{
					"version":    row.ID,
					"path":       filepath.Join(row.Dir, row.Name),
					"action":     row.Action,
					"size":       row.Size,
					"tool":       row.Tool,
					"session_id": row.SessionID,
					"task_id":    row.TaskID,
					"objective":  actionHistory.Objective(row.SessionID, row.TaskID),
					"changed_at": time.Unix(0, row.ChangedAt).Format(historyStore.TimeLayout),
				}
				if reason := row.RestoreBlock(); reason != "" {
					item["restorable"] = false
					item["reason"] = reason
				}
				out = append(out, item)
			}

			raw, err := json.Marshal(out)
			if err != nil {
				return "", fmt.Errorf("encoding/json: Marshal: %w", err)
			}
			return string(raw), nil
		},
	})
}
