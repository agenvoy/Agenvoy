package actionHistory

import (
	"encoding/json"
	"fmt"

	toolTypes "github.com/pardnchiu/agenvoy/internal/tools/types"
)

type call struct {
	Name string `json:"name"`
	Args string `json:"args,omitempty"`
}

func list(e *toolTypes.Executor, limit int, full bool) (string, error) {
	if limit <= 0 {
		limit = defaultListLimit
	}

	list, err := entries(e)
	if err != nil {
		return "", err
	}
	if len(list) == 0 {
		return "no finished tasks yet", nil
	}
	if len(list) > limit {
		list = list[:limit]
	}

	out := make([]map[string]any, 0, len(list))
	for _, item := range list {
		row := map[string]any{
			"task_id": item.taskID,
			"at":      item.at.Format(displayLayout),
		}

		if r, err := load(item); err == nil {
			row["objective"] = r.Objective
			if r.Model != "" {
				row["model"] = r.Model
			}
			if n := len(r.Todos); n > 0 {
				row["todos"] = n
			}
			if calls := toolCalls(r, full); len(calls) > 0 {
				row["calls"] = calls
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
}

func toolCalls(r record, full bool) []call {
	calls := make([]call, 0, len(r.ToolResults))
	for _, t := range r.ToolResults {
		one := call{Name: t.Name}
		if full || isReference(t.Name) {
			one.Args = truncateArgs(t.Args)
		}
		calls = append(calls, one)
	}
	return calls
}

func truncateArgs(text string) string {
	runes := []rune(text)
	if len(runes) <= listArgsRunes {
		return text
	}
	return string(runes[:listArgsRunes]) +
		fmt.Sprintf("… truncated; %d bytes total, mode=read for the whole run", len(text))
}
