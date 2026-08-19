package actionHistory

import (
	"fmt"
	"strings"

	toolTypes "github.com/pardnchiu/agenvoy/internal/tools/types"
)

func read(e *toolTypes.Executor, taskID string) (string, error) {
	list, err := entries(e)
	if err != nil {
		return "", err
	}

	for _, item := range list {
		if item.taskID != taskID {
			continue
		}
		r, err := load(item.path)
		if err != nil {
			return "", err
		}
		return render(item, r), nil
	}
	return "", fmt.Errorf("no finished task with id %s in this session", taskID)
}

func render(item entry, r record) string {
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

	section(&b, "completed", r.Completed)
	section(&b, "next steps", r.NextSteps)

	for _, a := range r.Answer {
		fmt.Fprintf(&b, "\n\nasked: %s\nanswered: %s", a.Question, a.Answer)
	}

	for _, t := range r.Todos {
		fmt.Fprintf(&b, "\n\ntodo [%s] %s", t.Status, t.Content)
	}

	for _, a := range r.ToolAttempts {
		fmt.Fprintf(&b, "\n\nattempted %s %s", a.Name, a.Args)
	}
	for _, t := range r.ToolResults {
		fmt.Fprintf(&b, "\n\ncalled %s:\n%s", t.Name, t.Result)
	}

	if r.Reply != "" {
		fmt.Fprintf(&b, "\n\nreplied:\n%s", r.Reply)
	}
	return b.String()
}

func section(b *strings.Builder, title string, lines []string) {
	if len(lines) == 0 {
		return
	}
	fmt.Fprintf(b, "\n\n%s:", title)
	for _, line := range lines {
		fmt.Fprintf(b, "\n- %s", line)
	}
}
