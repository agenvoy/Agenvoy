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
	listArgsRunes    = 256
)

func registChatHistory() {
	toolRegister.Regist(toolRegister.Def{
		Name:        "chat_history",
		SystemUse:   true,
		AlwaysLoad:  false,
		AlwaysAllow: true,
		Concurrent:  true,
		Description: `This session's action log: each past run's objective, its tool calls with arguments and output, and the messages.
Use for 最近做了什麼 / 之前處理過這個嗎 / 那次到底做了什麼 / 那個工具回了什麼 / 上一個動作的 action log / 我們之前聊過什麼.
Before repeating a tool call, mode=list shows what past runs called with which arguments; mode=read returns their output — reuse instead of re-running.
Not the workspace — file listing never reaches them. Versions → file_history; failures → error_history.`,
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"mode": map[string]any{
					"type":        "string",
					"enum":        []string{"list", "read", "search"},
					"description": "list: recent runs, newest first — each row carries its task_id, objective and every tool it called with the arguments it was given, no tool output. read: the full runs behind one or more task_ids, tool output untruncated. search: past messages, needs keyword. Omitted: task_id selects read, keyword selects search, otherwise list.",
					"default":     "list",
				},
				"task_id": map[string]any{
					"type":        "array",
					"items":       map[string]any{"type": "string"},
					"description": "mode=read: task ids taken from list rows; pass every run you want in one call. Required.",
				},
				"scope": map[string]any{
					"type":        "string",
					"enum":        []string{"reference", "full"},
					"description": "mode=list/read: reference keeps what costs money or time to obtain again — fetch_page, search_web, http_request, subagents reports, transcribe_media, generate_image, download_file, and every mcp__/api_/script_/ext_ payload — and reduces the rest to bare tool names. full includes every tool's arguments and output; use it to reconstruct what a run actually did.",
					"default":     "reference",
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
				Mode      string          `json:"mode"`
				Scope     string          `json:"scope"`
				TaskID    json.RawMessage `json:"task_id"`
				Keyword   string          `json:"keyword"`
				Match     string          `json:"match"`
				TimeRange string          `json:"time_range"`
				Limit     int             `json:"limit"`
			}
			if len(args) > 0 {
				if err := json.Unmarshal(args, &params); err != nil {
					return "", fmt.Errorf("encoding/json: Unmarshal: %w", err)
				}
			}

			params.Mode = strings.TrimSpace(params.Mode)
			taskIDs := parseTaskIDs(params.TaskID)

			full := false
			switch strings.TrimSpace(params.Scope) {
			case "", "reference":
			case "full":
				full = true
			default:
				return "", fmt.Errorf("unknown scope %q; available: reference, full", params.Scope)
			}
			if params.Mode == "" {
				params.Mode = "list"
				switch {
				case len(taskIDs) > 0:
					params.Mode = "read"
				case strings.TrimSpace(params.Keyword) != "":
					params.Mode = "search"
				}
			}

			switch params.Mode {
			case "list":
				return list(e, params.Limit, full)
			case "read":
				if len(taskIDs) == 0 {
					return "", fmt.Errorf("task_id is required when mode=read")
				}
				return read(e, taskIDs, full)
			case "search":
				return searchMessages(ctx, e, params.Keyword, params.Match, params.TimeRange, params.Limit)
			}
			return "", fmt.Errorf("unknown mode %q; available: list, read, search", params.Mode)
		},
	})
}

func parseTaskIDs(raw json.RawMessage) []string {
	if len(raw) == 0 {
		return nil
	}

	var list []string
	if err := json.Unmarshal(raw, &list); err != nil {
		var one string
		if err := json.Unmarshal(raw, &one); err != nil {
			return nil
		}
		list = []string{one}
	}

	ids := make([]string, 0, len(list))
	seen := make(map[string]bool, len(list))
	for _, id := range list {
		id = strings.TrimSpace(id)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		ids = append(ids, id)
	}
	return ids
}
