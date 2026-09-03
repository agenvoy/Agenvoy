package actionHistory

import (
	"fmt"
	"slices"
	"strings"

	"github.com/pardnchiu/agenvoy/internal/utils"

	toolTypes "github.com/pardnchiu/agenvoy/internal/tools/types"
)

func toolList(e *toolTypes.Executor, limit int) (string, error) {
	if limit <= 0 {
		limit = defaultToolsLimit
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
		r, err := load(item)
		if err != nil {
			continue
		}
		for _, t := range r.ToolResults {
			if !isReference(t.Name) {
				continue
			}
			row := map[string]any{
				"at":      item.at.Format(displayLayout),
				"name":    t.Name,
				"task_id": item.taskID,
			}
			if t.Args != "" {
				row["args"] = truncateArgs(t.Args)
			}
			out = append(out, row)
		}
	}
	if len(out) == 0 {
		return fmt.Sprintf("no reusable tool calls in the last %d runs", len(list)), nil
	}

	raw, err := utils.MarshalPlain(out)
	if err != nil {
		return "", fmt.Errorf("encoding/json: Marshal: %w", err)
	}
	return string(raw), nil
}

func toolContent(e *toolTypes.Executor, taskIDs []string, name, args string) (string, error) {
	if len(taskIDs) == 0 {
		return "", fmt.Errorf("task_id is required when mode=tool")
	}
	if name == "" {
		return "", fmt.Errorf("name is required when mode=tool")
	}

	list, err := entries(e)
	if err != nil {
		return "", err
	}

	id := taskIDs[0]
	var item entry
	found := false
	for _, one := range list {
		if one.taskID == id {
			item, found = one, true
			break
		}
	}
	if !found {
		return "", fmt.Errorf("no finished task with id %s in this session", id)
	}

	r, err := load(item)
	if err != nil {
		return "", err
	}

	named := make([]result, 0, 4)
	names := make([]string, 0, len(r.ToolResults))
	for _, t := range r.ToolResults {
		names = append(names, t.Name)
		if t.Name == name {
			named = append(named, t)
		}
	}
	if len(named) == 0 {
		return "", fmt.Errorf("task %s never called %s; it called: %s", id, name, strings.Join(slices.Compact(names), ", "))
	}

	if args = strings.TrimSpace(args); args != "" {
		for _, t := range named {
			if t.Args == args {
				return t.Result, nil
			}
		}
		return "", fmt.Errorf("task %s called %s %d times but none with those arguments; the ones it used: %s",
			id, name, len(named), strings.Join(argsOf(named), " | "))
	}
	if len(named) > 1 {
		return "", fmt.Errorf("task %s called %s %d times; pass args to pick one: %s",
			id, name, len(named), strings.Join(argsOf(named), " | "))
	}
	return named[0].Result, nil
}

func argsOf(list []result) []string {
	out := make([]string, 0, len(list))
	for _, t := range list {
		out = append(out, truncateArgs(t.Args))
	}
	return out
}
