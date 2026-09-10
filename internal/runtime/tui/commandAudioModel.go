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

type AudioModelLoaded struct {
	kind      string
	current   string
	available []string
	err       error
	back      *Popup
}

func (t TUI) commandAudioModel(kind string) (TUI, tea.Cmd, bool) {
	back := t.popupOrigin
	load := func() tea.Msg {
		cfg, err := config.Load()
		if err != nil {
			return AudioModelLoaded{kind: kind, err: err}
		}
		if kind == "tts" {
			return AudioModelLoaded{kind: kind, current: cfg.TTSModel, available: audioTool.TTSOptions(context.Background()), back: back}
		}
		return AudioModelLoaded{kind: kind, current: cfg.STTModel, available: audioTool.STTOptions(context.Background()), back: back}
	}
	return t, tea.Sequence(tea.Println(msgLog("loading models...")+"\n"), load), true
}

func (t TUI) openAudioModelPopup(msg AudioModelLoaded) (TUI, tea.Cmd) {
	if msg.err != nil {
		return t, tea.Println(msgError(fmt.Sprintf("session.Load: %v", msg.err)) + "\n")
	}

	kind, current, available := msg.kind, msg.current, msg.available
	label := "speech-to-text"
	if kind == "tts" {
		label = "text-to-speech"
	}
	if len(available) == 0 {
		return t, tea.Println(msgLog(fmt.Sprintf("no %s model available  add openai or gemini with /model add", label)) + "\n")
	}

	options := make([]string, 0, len(available)+2)
	values := make([]string, 0, len(available)+2)
	cursor := 0

	for i, name := range available {
		option := name
		if current == name {
			option += "  " + systemStyle.Render("[current]")
			cursor = i
		}
		options = append(options, option)
		values = append(values, name)
	}

	disable := hintStyle.Render("disable")
	if current == "" || current == "off" {
		disable += "  " + systemStyle.Render("[current]")
		cursor = len(options) + 1
	}
	options = append(options, "", disable)
	values = append(values, "", "")

	t.popup = &Popup{
		kind:    popupSingleSelect,
		title:   "Select " + label + " model",
		options: options,
		values:  values,
		cursor:  cursor,
		back:    msg.back,
		onConfirm: func(chosen string) any {
			return AudioModelSelect{kind: kind, name: chosen}
		},
	}
	return t, nil
}

func (t TUI) runAudioModelSelect(kind, name string) (TUI, tea.Cmd) {
	cfg, err := config.Load()
	if err != nil {
		return t, tea.Println(msgError(fmt.Sprintf("session.Load: %v", err)) + "\n")
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
		return t, tea.Println(msgError(fmt.Sprintf("session.Save: %v", err)) + "\n")
	}
	return t, nil
}
