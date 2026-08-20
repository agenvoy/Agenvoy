package actionHistory

import (
	"encoding/json"
	"fmt"
	"slices"

	toolTypes "github.com/pardnchiu/agenvoy/internal/tools/types"
)

func list(e *toolTypes.Executor, limit int) (string, error) {
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
