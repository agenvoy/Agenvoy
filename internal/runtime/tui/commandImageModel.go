package tui

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/pardnchiu/agenvoy/internal/session/config"
	imageTool "github.com/pardnchiu/agenvoy/internal/tools/external/image"
)

type ImageModelSelect struct {
	name string
}

type ImageModelLoaded struct {
	current   string
	available []string
	err       error
	back      *Popup
}

func (t TUI) commandImageModel() (TUI, tea.Cmd, bool) {
	back := t.popupOrigin
	load := func() tea.Msg {
		imageTool.Prune(context.Background())

		cfg, err := config.Load()
		if err != nil {
			return ImageModelLoaded{err: err}
		}
		return ImageModelLoaded{current: cfg.ImageGenerator, available: imageTool.Available(context.Background()), back: back}
	}
	return t, tea.Sequence(tea.Println(msgLog("loading models...")+"\n"), load), true
}

func (t TUI) openImageModelPopup(msg ImageModelLoaded) (TUI, tea.Cmd) {
	if msg.err != nil {
		return t, tea.Println(msgError(fmt.Sprintf("session.Load: %v", msg.err)) + "\n")
	}

	available := msg.available
	if len(available) == 0 {
		return t, tea.Println(msgLog("no image-capable provider has credentials  add one with /model add") + "\n")
	}

	options := make([]string, 0, len(available)+2)
	values := make([]string, 0, len(available)+2)
	cursor := 0

	for i, name := range available {
		label := name
		if msg.current == name {
			label += "  " + systemStyle.Render("[current]")
			cursor = i
		}
		options = append(options, label)
		values = append(values, name)
	}

	disable := hintStyle.Render("disable")
	if msg.current == "" || msg.current == "off" {
		disable += "  " + systemStyle.Render("[current]")
		cursor = len(options) + 1
	}
	options = append(options, "", disable)
	values = append(values, "", "")

	t.popup = &Popup{
		kind:    popupSingleSelect,
		title:   "Select image generator",
		options: options,
		values:  values,
		cursor:  cursor,
		back:    msg.back,
		onConfirm: func(chosen string) any {
			return ImageModelSelect{name: chosen}
		},
	}
	return t, nil
}

func (t TUI) runImageModelSelect(name string) (TUI, tea.Cmd) {
	cfg, err := config.Load()
	if err != nil {
		return t, tea.Println(msgError(fmt.Sprintf("session.Load: %v", err)) + "\n")
	}
	if name == "off" {
		name = ""
	}

	if cfg.ImageGenerator == name {
		return t, nil
	}

	cfg.ImageGenerator = name
	if err := config.Save(cfg); err != nil {
		return t, tea.Println(msgError(fmt.Sprintf("session.Save: %v", err)) + "\n")
	}
	return t, nil
}
