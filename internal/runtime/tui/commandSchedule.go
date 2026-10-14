package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	go_pkg_filesystem_reader "github.com/pardnchiu/go-pkg/filesystem/reader"

	"github.com/pardnchiu/agenvoy/configs"
	"github.com/pardnchiu/agenvoy/internal/filesystem"
	"github.com/pardnchiu/agenvoy/internal/filesystem/skill"
)

func (t TUI) commandSchedule(parts []string) (TUI, tea.Cmd, bool) {
	name := strings.TrimPrefix(parts[0], "/sched-")
	if name == "" {
		return t, notice(msgError("scheduler skill name required") + "\n"), true
	}
	if !go_pkg_filesystem_reader.Exists(filesystem.ScheduleSkillPath(name)) {
		return t, notice(msgError(fmt.Sprintf("scheduler skill %q not found", name)) + "\n"), true
	}
	body, err := skill.GetSchedule(name)
	if err != nil {
		return t, notice(msgError(fmt.Sprintf("read scheduler skill: %v", err)) + "\n"), true
	}
	body = strings.TrimSpace(body)
	if body == "" {
		return t, notice(msgError(fmt.Sprintf("scheduler skill %q is empty", name)) + "\n"), true
	}

	extra := strings.TrimSpace(strings.Join(parts[1:], " "))
	prompt := strings.NewReplacer(
		"{{.Name}}", name,
		"{{.Body}}", body,
	).Replace(strings.TrimSpace(configs.InjectScheduleRun))
	if extra != "" {
		prompt = prompt + "\n\n---\nAdditional instructions: " + extra
	}
	next, cmd := t.dispatchAgent(prompt)
	return next, cmd, true
}
