package tui

import (
	"strings"

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

	actions := []string{"add", "remove", "dispatch", "summary", "image", "stt", "tts"}

	options, values, cursor := registeredModelOptions(t.currentSessionID)
	var styledLines []string
	if len(options) == 0 {
		styledLines = []string{hintStyle.Render("  no models configured")}
	} else {
		options = append(options, "")
		values = append(values, "")
	}
	options = append(options, optionColumn(actions, []string{
		"add model from provider",
		"remove model from registry",
		"set dispatcher model",
		"set summary model",
		"set image generator",
		"set speech-to-text model",
		"set text-to-speech model",
	})...)
	values = append(values, actions...)

	t.popup = &Popup{
		kind:        popupSingleSelect,
		title:       "Model",
		styledLines: styledLines,
		options:     options,
		values:      values,
		cursor:      cursor,
		maxVisible:  len(options),
		onConfirm: func(chosen string) any {
			if name, ok := strings.CutPrefix(chosen, sessionModelPrefix); ok {
				return SessionModelSelect{name: name}
			}
			return ModelScopeSelect{scope: chosen}
		},
	}
	return t, nil, true
}
