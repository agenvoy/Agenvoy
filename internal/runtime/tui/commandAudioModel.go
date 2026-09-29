package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/pardnchiu/agenvoy/internal/session/config"
)

type AudioModelSelect struct {
	kind string
	name string
}

func (t TUI) runAudioModelSelect(kind, name string) (TUI, tea.Cmd) {
	cfg, err := config.Load()
	if err != nil {
		return t, notice(msgError(fmt.Sprintf("session.Load: %v", err)) + "\n")
	}
	if name == "off" {
		name = ""
	}

	current := &cfg.STTModel
	if kind == "tts" {
		current = &cfg.TTSModel
	}
	if *current == name {
		return t, nil
	}

	*current = name
	if err := config.Save(cfg); err != nil {
		return t, notice(msgError(fmt.Sprintf("session.Save: %v", err)) + "\n")
	}
	return t, nil
}
