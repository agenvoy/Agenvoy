package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

type MemoryScopeSelect struct {
	scope string
}

func (t TUI) commandMemory(parts []string) (TUI, tea.Cmd, bool) {
	if len(parts) > 1 {
		switch parts[1] {
		case "compact":
			return t.commandCompact()
		case "reset":
			return t.commandReset()
		case "summary":
			return t.commandSummary()
		}
	}

	t.popup = &Popup{
		kind:  popupSingleSelect,
		title: "Memory",
		options: []string{
			"compact  remove redundant / meaningless exchanges from history via LLM analysis · confirm required",
			"reset    reset / refresh current session · double-confirm · summary regen first then drop history + task history + action.log",
			"summary  force / regenerate summary now · no confirm · runs the hourly cron pass on demand",
		},
		values: []string{"compact", "reset", "summary"},
		onConfirm: func(chosen string) any {
			return MemoryScopeSelect{scope: chosen}
		},
	}
	return t, nil, true
}
