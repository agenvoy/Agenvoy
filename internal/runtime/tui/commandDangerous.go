package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

type DangerousSelect struct {
	action string
}

func (t TUI) commandDangerous(parts []string) (TUI, tea.Cmd, bool) {
	if len(parts) > 1 {
		switch parts[1] {
		case "remove-session":
			return t.commandRemoveSession()
		case "allow-skill":
			return t.commandAllowSkill(parts[1:])
		}
	}

	t.popup = &Popup{
		kind:  popupSingleSelect,
		title: "Dangerous",
		options: []string{
			"remove-session  delete current session",
			"allow-skill     always-allow skill (skip permission)",
		},
		values: []string{"remove-session", "allow-skill"},
		onConfirm: func(chosen string) any {
			return DangerousSelect{action: chosen}
		},
	}
	return t, nil, true
}
