package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/pardnchiu/agenvoy/internal/session/config"
)

func (t TUI) commandModelList() (TUI, tea.Cmd, bool) {
	cfg, err := config.Load()
	if err != nil {
		return t, tea.Println(errorStyle.Render(fmt.Sprintf("[!] session.Load: %v", err)) + "\n"), true
	}
	if len(cfg.Models) == 0 {
		return t, tea.Println(hintStyle.Render("no models configured") + "\n"), true
	}

	lines := make([]string, 0, len(cfg.Models)+1)
	lines = append(lines, hintStyle.Render(fmt.Sprintf("⎯ %d model(s) configured", len(cfg.Models))))
	for _, m := range cfg.Models {
		label := "  " + m.Name
		if m.Description != "" {
			label += " · " + m.Description
		}
		if cfg.DispatcherModel != "" && m.Name == cfg.DispatcherModel {
			label += " · [dispatcher]"
		}
		if cfg.SummaryModel != "" && m.Name == cfg.SummaryModel {
			label += " · [summary]"
		}
		lines = append(lines, textStyle.Render(label))
	}

	return t, tea.Println(strings.Join(lines, "\n") + "\n"), true
}
