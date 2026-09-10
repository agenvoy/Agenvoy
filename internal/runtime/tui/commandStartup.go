package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/pardnchiu/agenvoy/internal/runtime/startup"
)

type StartupAction struct {
	action string
}

type StartupDone struct {
	action string
	detail string
	err    error
}

func (t TUI) commandStartup(parts []string) (TUI, tea.Cmd, bool) {
	if len(parts) > 1 {
		switch parts[1] {
		case "enable", "disable":
			return t, setStartup(parts[1]), true
		}
	}

	if !startup.State() {
		return t, setStartup("enable"), true
	}

	t.popup = &Popup{
		kind:    popupSingleSelect,
		title:   "Startup",
		options: []string{"disable"},
		values:  []string{"disable"},
		onConfirm: func(chosen string) any {
			return StartupAction{action: chosen}
		},
	}
	return t, nil, true
}

func setStartup(action string) tea.Cmd {
	return func() tea.Msg {
		var (
			detail string
			err    error
		)
		if action == "enable" {
			detail, err = startup.Enable()
		} else {
			detail, err = startup.Disable()
		}
		return StartupDone{action: action, detail: detail, err: err}
	}
}
