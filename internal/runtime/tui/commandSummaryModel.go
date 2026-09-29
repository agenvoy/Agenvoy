package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/pardnchiu/agenvoy/internal/session/config"
)

type SummaryModelSelect struct {
	name string
}

func (t TUI) runSummaryModelSelect(name string) (TUI, tea.Cmd) {
	cfg, err := config.Load()
	if err != nil {
		return t, notice(msgError(fmt.Sprintf("session.Load: %v", err)) + "\n")
	}
	if cfg.SummaryModel == name {
		if name == "" {
			return t, notice(msgLog("summary unchanged: auto") + "\n")
		}
		return t, notice(msgLog(fmt.Sprintf("summary unchanged: %s", name)) + "\n")
	}

	cfg.SummaryModel = name
	if err := config.Save(cfg); err != nil {
		return t, notice(msgError(fmt.Sprintf("session.Save: %v", err)) + "\n")
	}
	if name == "" {
		return t, notice(msgLog("summary: auto") + "\n")
	}
	return t, notice(msgLog(fmt.Sprintf("summary: %s", name)) + "\n")
}
