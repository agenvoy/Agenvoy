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

var modelTagDetails = map[string]string{
	"S":    "strongest  code and work that asks for depth or precision",
	"A":    "default for most work  one step below the flagship  e.g. sonnet, terra, pro",
	"B":    "mainstream mid tier  e.g. haiku, luna, flash",
	"C":    "fast and cheap  calls tools reliably as instructed",
	"pass": "never picked by auto routing or subagents  last in fallback  or set for a session",
}

func (t TUI) openModelTagPicker(name string) (TUI, tea.Cmd) {
	current := ""
	if cfg, err := config.Load(); err == nil {
		current = cfg.ModelTag[name]
	}

	details := make([]string, len(config.ModelTags))
	cursor := len(config.ModelTags) + 1
	for i, tag := range config.ModelTags {
		details[i] = modelTagDetails[tag]
		if tag == current {
			details[i] += "  " + systemStyle.Render("[current]")
			cursor = i
		}
	}
	options := optionColumn(config.ModelTags, details)
	values := append([]string{}, config.ModelTags...)

	none := hintStyle.Render("none") + "  " + hintStyle.Render("follow the built-in naming rules")
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
