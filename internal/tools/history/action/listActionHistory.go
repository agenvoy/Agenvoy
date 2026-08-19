package actionHistory

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"

	toolRegister "github.com/pardnchiu/agenvoy/internal/tools/register"
	toolTypes "github.com/pardnchiu/agenvoy/internal/tools/types"
)

const defaultListLimit = 10

func registListActionHistory() {
	toolRegister.Regist(toolRegister.Def{
		Name:        "list_action_history",
		AlwaysAllow: true,
		Concurrent:  true,
		SystemUse:   true,
		Description: `
What each finished task actually ran: what it was asked for, which tools it called, and its task_id.
Use for 最近做了什麼 / 上一個動作怎麼處理的 / 之前處理過這個嗎 — the run itself, not the conversation.
One task in full goes to read_action_history; what a file's contents became goes to list_file_history.`,
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"limit": map[string]any{
					"type":        "integer",
					"description": "Tasks to return, newest first.",
					"default":     defaultListLimit,
				},
			},
		},
		Handler: func(ctx context.Context, e *toolTypes.Executor, args json.RawMessage) (string, error) {
			var params struct {
				Limit int `json:"limit"`
			}
			if err := json.Unmarshal(args, &params); err != nil {
				return "", fmt.Errorf("encoding/json: Unmarshal: %w", err)
			}
			if params.Limit <= 0 {
				params.Limit = defaultListLimit
			}

			list, err := entries(e)
			if err != nil {
				return "", err
			}
			if len(list) == 0 {
				return "no finished tasks yet", nil
			}
			if len(list) > params.Limit {
				list = list[:params.Limit]
			}

			out := make([]map[string]any, 0, len(list))
			for _, item := range list {
				row := map[string]any{
					"task_id": item.taskID,
					"at":      item.at.Format(displayLayout),
				}

				if r, err := load(item.path); err == nil {
					row["objective"] = r.Objective
					row["did"] = toolsUsed(r)
					row["model"] = r.Model
					if n := len(r.Answer); n > 0 {
						row["answered_questions"] = n
					}
					if n := len(r.Todos); n > 0 {
						row["todos"] = n
					}
				} else {
					row["unreadable"] = err.Error()
				}
				out = append(out, row)
			}

			raw, err := json.Marshal(out)
			if err != nil {
				return "", fmt.Errorf("encoding/json: Marshal: %w", err)
			}
			return string(raw), nil
		},
	})
}

func toolsUsed(r record) []string {
	var used []string
	for _, t := range r.ToolResults {
		if !slices.Contains(used, t.Name) {
			used = append(used, t.Name)
		}
	}
	for _, a := range r.ToolAttempts {
		if !slices.Contains(used, a.Name) {
			used = append(used, a.Name)
		}
	}
	return used
}
