package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/pardnchiu/agenvoy/internal/session/config"
)

type ModelTagPick struct {
	name string
}

type ModelTagSubmit struct {
	name string
	tag  string
}

func (t TUI) openModelTagPicker(name string) (TUI, tea.Cmd) {
	current := ""
	if cfg, err := config.Load(); err == nil {
		current = cfg.ModelTag[name]
	}

	details := make([]string, len(config.ModelTags))
	cursor := len(config.ModelTags) + 1
	for i, tag := range config.ModelTags {
		details[i] = config.ModelTagDetails[tag]
		if tag == current {
			details[i] += "  " + systemStyle.Render("[current]")
			cursor = i
		}
	}
	options := optionColumn(config.ModelTags, details)
	values := append([]string{}, config.ModelTags...)

	none := hintStyle.Render("none") + "  " + hintStyle.Render(config.ModelTagNoneDetail)
	if current == "" {
		none += "  " + systemStyle.Render("[current]")
	}
	options = append(options, "", none)
	values = append(values, "", "")

	t.popup = &Popup{
		kind:    popupSingleSelect,
		title:   "Tier  " + name,
		options: options,
		values:  values,
		cursor:  cursor,
		onConfirm: func(chosen string) any {
			return ModelTagSubmit{name: name, tag: chosen}
		},
	}
	return t, nil
}
