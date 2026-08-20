package fileHistory

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	toolRegister "github.com/pardnchiu/agenvoy/internal/tools/register"
	toolTypes "github.com/pardnchiu/agenvoy/internal/tools/types"
)

const (
	maxHistoryRows = 24
	defaultRows    = 3
)

func registFileHistory() {
	toolRegister.Regist(toolRegister.Def{
		Name:        "file_history",
		AlwaysAllow: true,
		Concurrent:  true,
		SystemUse:   true,
		Description: `
Recorded versions of files changed through the tools: when each changed, what the task was after, and what the content was.
Use for 這個檔案動過幾次 / 什麼時候改的 / 上一版是什麼 / 改了什麼 / 跟之前差在哪 / 誤刪的內容是什麼.
把檔案改回去 → edit_file(mode=restore); 那個動作跑了什麼 → chat_history.`,
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"mode": map[string]any{
					"type":        "string",
					"enum":        []string{"list", "read"},
					"description": "list: every recorded version with its time and objective, newest first. read: the newest version of each file, diffed against what is on disk now. Omitted: paths selects read, otherwise list.",
					"default":     "list",
				},
				"path": map[string]any{
					"type":        "string",
					"description": "mode=list: the file to look back over (e.g. '/Users/me/notes.md', '~/notes.md', 'src/main.go'). Omit with task_id to see every file that task changed.",
				},
				"paths": map[string]any{
					"type":        "array",
					"description": "mode=read: files to compare. Batch every path into one call.",
					"items": map[string]any{
						"type": "string",
					},
				},
				"task_id": map[string]any{
					"type":        "string",
					"description": "mode=list: only what one task changed, from chat_history. 'current' for the task running now.",
				},
				"from": map[string]any{
					"type":        "string",
					"description": "mode=list: local time to start from: '2026-08-13' (that day at 00:00), '2026-08-13 15:04', '2026-08-13 15:04:05'.",
				},
				"to": map[string]any{
					"type":        "string",
					"description": "mode=list: local time to stop at, same formats as from.",
				},
				"limit": map[string]any{
					"type":        "integer",
					"description": "mode=list: rows to return, newest first. Never above 24.",
					"default":     defaultRows,
				},
			},
		},
		Handler: func(ctx context.Context, e *toolTypes.Executor, args json.RawMessage) (string, error) {
			var params struct {
				Mode   string   `json:"mode"`
				Path   string   `json:"path"`
				Paths  []string `json:"paths"`
				TaskID string   `json:"task_id"`
				From   string   `json:"from"`
				To     string   `json:"to"`
				Limit  int      `json:"limit"`
			}
			if len(args) > 0 {
				if err := json.Unmarshal(args, &params); err != nil {
					return "", fmt.Errorf("encoding/json: Unmarshal: %w", err)
				}
			}

			params.Mode = strings.TrimSpace(params.Mode)
			if params.Mode == "" {
				params.Mode = "list"
				if len(params.Paths) > 0 {
					params.Mode = "read"
				}
			}

			switch params.Mode {
			case "list":
				path := strings.TrimSpace(params.Path)
				if path == "" && len(params.Paths) == 1 {
					path = params.Paths[0]
				}
				if path == "" && len(params.Paths) > 1 {
					return "", fmt.Errorf("mode=list takes one path; pass them as paths with mode=read, or call once per file")
				}
				return list(ctx, e, path, params.TaskID, params.From, params.To, params.Limit)
			case "read":
				paths := params.Paths
				if len(paths) == 0 && strings.TrimSpace(params.Path) != "" {
					paths = []string{params.Path}
				}
				if len(paths) == 0 {
					return "", fmt.Errorf("paths is required when mode=read")
				}
				return read(ctx, e, paths)
			}
			return "", fmt.Errorf("unknown mode %q; available: list, read", params.Mode)
		},
	})
}
