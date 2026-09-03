package actionHistory

import (
	"fmt"

	"github.com/pardnchiu/agenvoy/internal/utils"

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

		if r, err := load(item); err == nil {
			row["objective"] = r.Objective
		} else {
			row["unreadable"] = err.Error()
		}
		out = append(out, row)
	}

	raw, err := utils.MarshalPlain(out)
	if err != nil {
		return "", fmt.Errorf("encoding/json: Marshal: %w", err)
	}
	return string(raw), nil
}

func truncateArgs(text string) string {
	runes := []rune(text)
	if len(runes) <= listArgsRunes {
		return text
	}
	return string(runes[:listArgsRunes]) +
		fmt.Sprintf("... truncated; %d bytes total, mode=read for the whole run", len(text))
}
