package tui

import (
	"fmt"
	"strings"

	agentTypes "github.com/pardnchiu/agenvoy/internal/agents/types"
)

func renderTodoList(todos []agentTypes.TodoItem) string {
	if len(todos) == 0 {
		return ""
	}

	var done int
	for _, td := range todos {
		if td.Status == agentTypes.TodoCompleted {
			done++
		}
	}

	var sb strings.Builder
	sb.WriteString(systemStyle.Render("⏺ Plan"))
	sb.WriteString(hintStyle.Render(fmt.Sprintf(" (%d/%d)", done, len(todos))))
	for _, td := range todos {
		sb.WriteByte('\n')
		switch td.Status {
		case agentTypes.TodoCompleted:
			sb.WriteString(okayStyle.Render("  ✔ "))
			sb.WriteString(hintStyle.Render(td.Content))
		case agentTypes.TodoInProgress:
			label := strings.TrimSpace(td.ActiveForm)
			if label == "" {
				label = td.Content
			}
			sb.WriteString(systemStyle.Render("  ▶ "))
			sb.WriteString(whiteStyle.Render(label))
		default:
			sb.WriteString(hintStyle.Render("  ○ " + td.Content))
		}
	}
	return sb.String()
}
