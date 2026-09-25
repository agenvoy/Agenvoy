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
	loading   *Popup
}

func (t TUI) commandImageModel() (TUI, tea.Cmd, bool) {
	back := t.popupOrigin
	loading := loadingPopup(back)
	t.popup = loading
	load := func() tea.Msg {
		imageTool.Prune(context.Background())

		cfg, err := config.Load()
		if err != nil {
			return ImageModelLoaded{err: err, loading: loading}
		}
		return ImageModelLoaded{current: cfg.ImageGenerator, available: imageTool.Available(context.Background()), back: back, loading: loading}
	}
	return t, load, true
}

func (t TUI) openImageModelPopup(msg ImageModelLoaded) (TUI, tea.Cmd) {
	if t.popup != msg.loading {
		return t, nil
	}
	if msg.err != nil {
		t.popup = nil
		return t, tea.Println(msgError(fmt.Sprintf("session.Load: %v", msg.err)) + "\n")
	}

	available := msg.available
	if len(available) == 0 {
		t.popup = nil
		return t, tea.Println(msgLog("no image-capable provider has credentials  add one with /model add") + "\n")
	}

	t.popup = providerPopup("/model image", available, msg.current, msg.back, func(chosen string) any {
		return ImageModelSelect{name: chosen}
	})
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
