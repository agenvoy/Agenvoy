package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/pardnchiu/agenvoy/internal/session/config"
)

type SummaryModelSelect struct {
	name string
}

func (t TUI) commandSummaryModel() (TUI, tea.Cmd, bool) {
	cfg, err := config.Load()
	if err != nil {
		return t, tea.Println(msgError(fmt.Sprintf("session.Load: %v", err)) + "\n"), true
	}
	if len(cfg.Models) == 0 {
		return t, tea.Println(msgLog("no models configured  use /model") + "\n"), true
	}

	options := make([]string, 0, len(cfg.Models)+2)
	values := make([]string, 0, len(cfg.Models)+2)
	cursor := 0

	for i, m := range cfg.Models {
		label := m.Name
		if cfg.SummaryModel != "" && m.Name == cfg.SummaryModel {
			label += "  " + systemStyle.Render("[current]")
			cursor = i
		}
		options = append(options, label)
		values = append(values, m.Name)
	}

	auto := hintStyle.Render("auto")
	if cfg.SummaryModel == "" {
		auto += "  " + systemStyle.Render("[current]")
		cursor = len(options) + 1
	}
	options = append(options, "", auto)
	values = append(values, "", "")

	t.popup = &Popup{
		kind:    popupSingleSelect,
		title:   "Select summary model",
		options: options,
		values:  values,
		cursor:  cursor,
		onConfirm: func(chosen string) any {
			return SummaryModelSelect{name: chosen}
		},
	}
	return t, nil, true
}

func (t TUI) runSummaryModelSelect(name string) (TUI, tea.Cmd) {
	cfg, err := config.Load()
	if err != nil {
		return t, tea.Println(msgError(fmt.Sprintf("session.Load: %v", err)) + "\n")
	}
	if cfg.SummaryModel == name {
		if name == "" {
			return t, tea.Println(msgLog("summary unchanged: auto") + "\n")
		}
		return t, tea.Println(msgLog(fmt.Sprintf("summary unchanged: %s", name)) + "\n")
	}

	cfg.SummaryModel = name
	if err := config.Save(cfg); err != nil {
		return t, tea.Println(msgError(fmt.Sprintf("session.Save: %v", err)) + "\n")
	}
	if name == "" {
		return t, tea.Println(msgLog("summary: auto") + "\n")
	}
	return t, tea.Println(msgLog(fmt.Sprintf("summary: %s", name)) + "\n")
}
