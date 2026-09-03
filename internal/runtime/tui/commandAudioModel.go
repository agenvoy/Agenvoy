package tui

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/pardnchiu/agenvoy/internal/session/config"
	audioTool "github.com/pardnchiu/agenvoy/internal/tools/external/audio"
)

type AudioModelSelect struct {
	kind string
	name string
}

func (t TUI) commandSTTModel() (TUI, tea.Cmd, bool) {
	return t.commandAudioModel("stt")
}

func (t TUI) commandTTSModel() (TUI, tea.Cmd, bool) {
	return t.commandAudioModel("tts")
}

func (t TUI) commandAudioModel(kind string) (TUI, tea.Cmd, bool) {
	cfg, err := config.Load()
	if err != nil {
		return t, tea.Println(errorStyle.Render(fmt.Sprintf("[!] session.Load: %v", err)) + "\n"), true
	}

	label, current := "speech-to-text", cfg.STTModel
	available := audioTool.STTOptions(context.Background())
	if kind == "tts" {
		label, current = "text-to-speech", cfg.TTSModel
		available = audioTool.TTSOptions(context.Background())
	}
	if len(available) == 0 {
		return t, tea.Println(hintStyle.Render(fmt.Sprintf("no %s model available · add openai or gemini with /model add", label)) + "\n"), true
	}

	options := make([]string, 0, len(available)+1)
	values := make([]string, 0, len(available)+1)
	cursor := 0

	options = append(options, hintStyle.Render("off"))
	values = append(values, "")
	if current == "" || current == "off" {
		options[0] += "  " + systemStyle.Render("[current]")
	}

	for i, name := range available {
		option := name
		if current == name {
			option += "  " + systemStyle.Render("[current]")
			cursor = i + 1
		}
		options = append(options, option)
		values = append(values, name)
	}

	t.popup = &Popup{
		kind:    popupSingleSelect,
		title:   "Select " + label + " model",
		options: options,
		values:  values,
		cursor:  cursor,
		onConfirm: func(chosen string) any {
			return AudioModelSelect{kind: kind, name: chosen}
		},
	}
	return t, nil, true
}

func (t TUI) runAudioModelSelect(kind, name string) (TUI, tea.Cmd) {
	cfg, err := config.Load()
	if err != nil {
		return t, tea.Println(errorStyle.Render(fmt.Sprintf("[!] session.Load: %v", err)) + "\n")
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
		return t, tea.Println(errorStyle.Render(fmt.Sprintf("[!] session.Save: %v", err)) + "\n")
	}
	return t, nil
}
