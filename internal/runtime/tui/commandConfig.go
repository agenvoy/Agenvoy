package tui

import (
	"slices"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
	"github.com/pardnchiu/agenvoy/internal/startup"
)

const (
	configStartup   = "startup"
	configReplyLang = "reply_lang"
	configOutputDir = "output_dir"
)

type ConfigSelect struct {
	key string
}

type StartupDone struct {
	action string
	detail string
	err    error
}

func (t TUI) commandConfig() (TUI, tea.Cmd, bool) {
	return t.openConfig(""), nil, true
}

func (t TUI) openConfig(focus string) TUI {
	startupValue := hintStyle.Render("false")
	if startup.State() {
		startupValue = okayStyle.Render("true")
	}
	names := []string{"Startup on login", "Reply language", "Output dir"}
	settings := []string{
		startupValue,
		filesystem.CanonicalReplyLang(filesystem.ConfigReplyLang),
		filesystem.OutputDir(),
	}
	values := []string{configStartup, configReplyLang, configOutputDir}
	options := optionColumn(names, settings)

	input := newPopupInput("", false)
	input.Placeholder = "Search settings..."
	input.SetPromptFunc(2, func(int) string {
		return hintStyle.Render("/ ")
	})

	t.popup = &Popup{
		kind:        popupSingleSelect,
		title:       "Config",
		options:     options,
		values:      values,
		allOptions:  options,
		allValues:   values,
		searchable:  true,
		input:       input,
		cursor:      max(slices.Index(values, focus), 0),
		enterAction: "change",
		onConfirm: func(chosen string) any {
			return ConfigSelect{key: chosen}
		},
	}
	return t
}

func (t TUI) runConfigSelect(key string) (TUI, tea.Cmd) {
	switch key {
	case configStartup:
		if startup.State() {
			return t, setStartup("disable")
		}
		return t, setStartup("enable")

	case configReplyLang:
		next, cmd, _ := t.commandReplyLanguage()
		return next, cmd

	case configOutputDir:
		next, cmd, _ := t.commandOutputDir()
		return next, cmd
	}
	return t, nil
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
