package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

type ScheduleSelect struct {
	kind string
}

func (t TUI) commandScheduleMenu(parts []string) (TUI, tea.Cmd, bool) {
	if len(parts) > 1 {
		switch parts[1] {
		case "cron":
			return t.commandCron()
		case "task":
			return t.commandTask()
		}
	}

	values := []string{"cron", "task"}
	t.popup = &Popup{
		kind:    popupSingleSelect,
		title:   "Schedule",
		options: optionColumn(values, []string{"recurring", "one-shot"}),
		values:  values,
		onConfirm: func(chosen string) any {
			return ScheduleSelect{kind: chosen}
		},
	}
	return t, nil, true
}
