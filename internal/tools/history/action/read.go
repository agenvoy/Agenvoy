package actionHistory

import (
	"fmt"
	"strings"

	toolTypes "github.com/pardnchiu/agenvoy/internal/tools/types"
)

func read(e *toolTypes.Executor, taskIDs []string, full bool) (string, error) {
	list, err := entries(e)
	if err != nil {
		return "", err
	}

	idEntry := make(map[string]entry, len(list))
	for _, item := range list {
		idEntry[item.taskID] = item
	}

	blocks := make([]string, 0, len(taskIDs))
	var missing []string
	for _, id := range taskIDs {
		item, ok := idEntry[id]
		if !ok {
			missing = append(missing, id)
			continue
		}
		r, err := load(item)
		if err != nil {
			return "", err
		}
		blocks = append(blocks, render(item, r, full))
	}

	if len(blocks) == 0 {
		return "", fmt.Errorf("no finished task with id %s in this session", strings.Join(missing, ", "))
	}
	out := strings.Join(blocks, "\n\n---\n\n")
	if len(missing) > 0 {
		out += fmt.Sprintf("\n\n---\n\nnot found in this session: %s", strings.Join(missing, ", "))
	}
	return out, nil
}

func render(item entry, r record, full bool) string {
	var b strings.Builder

	fmt.Fprintf(&b, "task %s — %s", item.taskID, item.at.Format(displayLayout))
	if r.Model != "" {
		fmt.Fprintf(&b, " · %s", r.Model)
	}
	if r.Reasoning != "" {
		fmt.Fprintf(&b, " · reasoning %s", r.Reasoning)
	}
	if r.Objective != "" {
		fmt.Fprintf(&b, "\n\nobjective:\n%s", r.Objective)
	}

	var withheld []string
	for _, t := range r.ToolResults {
		if !full && !isReference(t.Name) {
			withheld = append(withheld, t.Name)
			continue
		}
		if t.Args == "" {
			fmt.Fprintf(&b, "\n\ncalled %s:\n%s", t.Name, t.Result)
			continue
		}
		fmt.Fprintf(&b, "\n\ncalled %s(%s):\n%s", t.Name, t.Args, t.Result)
	}
	if len(withheld) > 0 {
		fmt.Fprintf(&b, "\n\nalso called, output withheld by scope=reference — scope=full to include it: %s",
			strings.Join(withheld, ", "))
	}

	if r.Reply != "" {
		fmt.Fprintf(&b, "\n\nreplied:\n%s", r.Reply)
	}
	return b.String()
}
