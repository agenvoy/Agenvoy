package actionHistory

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"

	toolRegister "github.com/pardnchiu/agenvoy/internal/tools/register"
	toolTypes "github.com/pardnchiu/agenvoy/internal/tools/types"
)

const (
	defaultListLimit = 10
)

func registChatHistory() {
	toolRegister.Regist(toolRegister.Def{
		Name:        "chat_history",
		AlwaysAllow: true,
		Concurrent:  true,
		SystemUse:   true,
		Description: `This session's own record — the action log: what each finished run was asked for, every tool it called with what came back, the reply it ended on, and the messages themselves.
Use for 最近做了什麼 / 之前處理過這個嗎 / 那次到底做了什麼 / 那個工具回了什麼 / 上一個動作的 action log / 我們之前聊過什麼.
The records live in the daemon's own state directory, so listing, globbing or opening files never reaches them. File versions → file_history; failures → error_history.`,
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"mode": map[string]any{
					"type":        "string",
					"enum":        []string{"list", "read", "search"},
					"description": "list: recent runs, one row each, with their task_id. read: one run in full, needs task_id. search: past messages, needs keyword. Omitted: task_id selects read, keyword selects search, otherwise list.",
					"default":     "list",
				},
				"task_id": map[string]any{
					"type":        "string",
					"description": "mode=read: task id taken from a list row; required.",
				},
				"keyword": map[string]any{
					"type":        "string",
					"description": "mode=search: the core noun to look for (e.g. 'redis TTL', 'bwrap sandbox decision').",
				},
				"match": map[string]any{
					"type":        "string",
					"enum":        []string{"semantic", "keyword"},
					"description": "mode=search: semantic matches meaning in recent messages; keyword matches text across the full history including archive.",
					"default":     "semantic",
				},
				"time_range": map[string]any{
					"type":        "string",
					"enum":        slices.Sorted(maps.Keys(historyTimeRanges)),
					"description": "mode=search: how far back to look.",
					"default":     defaultTimeRange,
				},
				"limit": map[string]any{
					"type":        "integer",
					"description": "Rows to return, newest first; hit cap per source when mode=search. Ignored when mode=read.",
					"default":     defaultListLimit,
				},
			},
		},
		Handler: func(ctx context.Context, e *toolTypes.Executor, args json.RawMessage) (string, error) {
			var params struct {
				Mode      string `json:"mode"`
				TaskID    string `json:"task_id"`
				Keyword   string `json:"keyword"`
				Match     string `json:"match"`
				TimeRange string `json:"time_range"`
				Limit     int    `json:"limit"`
			}
			if len(args) > 0 {
				if err := json.Unmarshal(args, &params); err != nil {
					return "", fmt.Errorf("encoding/json: Unmarshal: %w", err)
				}
			}

			params.Mode = strings.TrimSpace(params.Mode)
			params.TaskID = strings.TrimSpace(params.TaskID)
			if params.Mode == "" {
				params.Mode = "list"
				switch {
				case params.TaskID != "":
					params.Mode = "read"
				case strings.TrimSpace(params.Keyword) != "":
					params.Mode = "search"
				}
			}

			switch params.Mode {
			case "list":
				return list(e, params.Limit)
			case "read":
				if params.TaskID == "" {
					return "", fmt.Errorf("task_id is required when mode=read")
				}
				return read(e, params.TaskID)
			case "search":
				return searchMessages(ctx, e, params.Keyword, params.Match, params.TimeRange, params.Limit)
			}
			return "", fmt.Errorf("unknown mode %q; available: list, read, search", params.Mode)
		},
	})
}
