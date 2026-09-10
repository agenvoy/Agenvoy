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
		case "image":
			return t.commandImageModel()
		case "stt":
			return t.commandSTTModel()
		case "tts":
			return t.commandTTSModel()
		}
	}

	values := []string{"add", "remove", "dispatch", "summary", "image", "stt", "tts"}

	t.popup = &Popup{
		kind:        popupSingleSelect,
		title:       "Model",
		styledLines: registeredModelLines(),
		options: optionColumn(values, []string{
			"add model from provider",
			"remove model from registry",
			"set dispatcher model",
			"set summary model",
			"set image generator",
			"set speech-to-text model",
			"set text-to-speech model",
		}),
		values: values,
		onConfirm: func(chosen string) any {
			return ModelScopeSelect{scope: chosen}
		},
	}
	return t, nil, true
}
