package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/pardnchiu/agenvoy/internal/session/config"
)

type ImageModelSelect struct {
	name string
}

func (t TUI) runImageModelSelect(name string) (TUI, tea.Cmd) {
	cfg, err := config.Load()
	if err != nil {
		return t, notice(msgError(fmt.Sprintf("session.Load: %v", err)) + "\n")
	}
	if name == "off" {
		name = ""
	}

	if cfg.ImageGenerator == name {
		return t, nil
	}

	cfg.ImageGenerator = name
	if err := config.Save(cfg); err != nil {
		return t, notice(msgError(fmt.Sprintf("session.Save: %v", err)) + "\n")
	}
	return t, nil
}
