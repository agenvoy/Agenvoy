package actionHistory

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	toolRegister "github.com/pardnchiu/agenvoy/internal/tools/register"
	toolTypes "github.com/pardnchiu/agenvoy/internal/tools/types"
)

func registReadActionHistory() {
	toolRegister.Regist(toolRegister.Def{
		Name:        "read_action_history",
		AlwaysAllow: true,
		Concurrent:  true,
		Description: `
One task's whole run: every tool call with what it returned, what the user answered, and the reply it ended on.
Use for 那次到底做了什麼 / 上次是怎麼解決的 / 那個工具回了什麼 / 當初為什麼那樣做.
Ids come from list_action_history; file contents are in read_file_history.`,
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"task_id": map[string]any{
					"type":        "string",
					"description": "Task id, from a list_action_history row.",
				},
			},
			"required": []string{
				"task_id",
			},
		},
		Handler: func(ctx context.Context, e *toolTypes.Executor, args json.RawMessage) (string, error) {
			var params struct {
				TaskID string `json:"task_id"`
			}
			if err := json.Unmarshal(args, &params); err != nil {
				return "", fmt.Errorf("encoding/json: Unmarshal: %w", err)
			}
			params.TaskID = strings.TrimSpace(params.TaskID)
			if params.TaskID == "" {
				return "", fmt.Errorf("task_id is required")
			}

			list, err := entries(e)
			if err != nil {
				return "", err
			}

			for _, item := range list {
				if item.taskID != params.TaskID {
					continue
				}
				r, err := load(item.path)
				if err != nil {
					return "", err
				}
				return render(item, r), nil
			}
			return "", fmt.Errorf("no finished task with id %s in this session", params.TaskID)
		},
	})
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
