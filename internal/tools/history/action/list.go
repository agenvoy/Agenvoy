package actionHistory

import (
	"encoding/json"
	"fmt"
	"slices"

	toolTypes "github.com/pardnchiu/agenvoy/internal/tools/types"
)

func list(e *toolTypes.Executor, limit, resultIndex int) (string, error) {
	if limit <= 0 {
		limit = defaultListLimit
	}
	if resultIndex <= 0 {
		resultIndex = 1
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
	for i, item := range list {
		row := map[string]any{
			"task_id": item.taskID,
			"at":      item.at.Format(displayLayout),
		}

		if r, err := load(item); err == nil {
			row["objective"] = r.Objective
			row["did"] = toolsUsed(r)
			row["model"] = r.Model
			if n := len(r.Todos); n > 0 {
				row["todos"] = n
			}
			if i == 0 && len(r.ToolResults) > 0 {
				row["tool_result_count"] = len(r.ToolResults)
				row["tool_result_index"] = resultIndex
				if one, ok := resultAt(r.ToolResults, resultIndex); ok {
					row["tool_result"] = one
				} else {
					row["tool_result"] = fmt.Sprintf("no result at index %d; this run made %d tool calls",
						resultIndex, len(r.ToolResults))
				}
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

func resultAt(list []result, index int) (result, bool) {
	at := len(list) - index
	if at < 0 || at >= len(list) {
		return result{}, false
	}

	one := list[at]
	runes := []rune(one.Result)
	if len(runes) > recentResultRunes {
		one.Result = string(runes[:recentResultRunes]) +
			fmt.Sprintf("\n… truncated here; %d bytes total, read the task for the rest", len(one.Result))
	}
	return one, true
}

func toolsUsed(r record) []string {
	var used []string
	for _, t := range r.ToolResults {
		if !slices.Contains(used, t.Name) {
			used = append(used, t.Name)
		}
	}
	return used
}
