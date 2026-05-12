package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
)

func (t TUI) commandSchedule(parts []string) (TUI, tea.Cmd, bool) {
	name := strings.TrimPrefix(parts[0], "/sched-")
	if name == "" {
		return t, tea.Println(errorStyle.Render("[!] scheduler skill name required") + "\n"), true
	}
	if !filesystem.ScheduleSkillExists(name) {
		return t, tea.Println(errorStyle.Render(fmt.Sprintf("[!] scheduler skill %q not found", name)) + "\n"), true
	}
	body, err := filesystem.ScheduleSkillBody(name)
	if err != nil {
		return t, tea.Println(errorStyle.Render(fmt.Sprintf("[!] read scheduler skill: %v", err)) + "\n"), true
	}
	body = strings.TrimSpace(body)
	if body == "" {
		return t, tea.Println(errorStyle.Render(fmt.Sprintf("[!] scheduler skill %q is empty", name)) + "\n"), true
	}

	extra := strings.TrimSpace(strings.Join(parts[1:], " "))
	prompt := body
	if extra != "" {
		prompt = body + "\n\n---\n" + extra
	}
	next, cmd := t.dispatchAgent(prompt)
	return next, cmd, true
}
