package tui

import (
	"fmt"
	"log/slog"

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
		return t, nil
	}

	cfg.SummaryModel = name
	if err := config.Save(cfg); err != nil {
		return t, notice(msgError(fmt.Sprintf("session.Save: %v", err)) + "\n")
	}
	slog.Debug("summary model updated", slog.String("model", name))
	return t, nil
}
