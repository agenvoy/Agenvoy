package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

type ModelScopeSelect struct {
	scope string
}

func (t TUI) commandModel(parts []string) (TUI, tea.Cmd, bool) {
	if len(parts) > 1 {
		switch parts[1] {
		case "add":
			return t.commandModelAdd()
		case "remove":
			return t.commandModelRemove()
		case "dispatch":
			return t.commandDispatcher()
		case "summary":
			return t.commandSummaryModel()
		}
	}

	t.popup = &Popup{
		kind:  popupSingleSelect,
		title: "Model",
		options: []string{
			"add        add model from provider",
			"remove     remove model from registry",
			"dispatch   set dispatcher model",
			"summary    set summary model",
		},
		values: []string{"add", "remove", "dispatch", "summary"},
		onConfirm: func(chosen string) any {
			return ModelScopeSelect{scope: chosen}
		},
	}
	return t, nil, true
}
