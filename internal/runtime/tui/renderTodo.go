package tui

import (
	"fmt"
	"strings"

	agenttypes "github.com/pardnchiu/agenvoy/internal/agents/types"
)

func renderTodoList(todos []agenttypes.TodoItem, spinner string) string {
	if len(todos) == 0 {
		return ""
	}

	var done int
	for _, td := range todos {
		if td.Status == agenttypes.TodoCompleted {
			done++
		}
	}

	mark := "⏺"
	if done < len(todos) && spinner != "" {
		mark = spinner
	}

	var sb strings.Builder
	sb.WriteString(systemStyle.Render(mark + " Plan"))
	sb.WriteString(hintStyle.Render(fmt.Sprintf(" (%d/%d)", done, len(todos))))
	for _, td := range todos {
		sb.WriteByte('\n')
		switch td.Status {
		case agenttypes.TodoCompleted:
			sb.WriteString(okayStyle.Render("  ✔ "))
			sb.WriteString(hintStyle.Render(td.Content))
		case agenttypes.TodoInProgress:
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
