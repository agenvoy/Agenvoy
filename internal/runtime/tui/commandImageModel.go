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

func (t TUI) commandImageModel() (TUI, tea.Cmd, bool) {
	imageTool.Prune(context.Background())

	cfg, err := config.Load()
	if err != nil {
		return t, tea.Println(errorStyle.Render(fmt.Sprintf("[!] session.Load: %v", err)) + "\n"), true
	}

	available := imageTool.Available(context.Background())
	if len(available) == 0 {
		return t, tea.Println(hintStyle.Render("no image-capable provider has credentials · add one with /model add") + "\n"), true
	}

	options := make([]string, 0, len(available)+1)
	values := make([]string, 0, len(available)+1)
	cursor := 0

	options = append(options, hintStyle.Render("off"))
	values = append(values, "")
	if cfg.ImageGenerator == "" || cfg.ImageGenerator == "off" {
		options[0] += "  " + systemStyle.Render("[current]")
	}

	for i, name := range available {
		label := name
		if cfg.ImageGenerator == name {
			label += "  " + systemStyle.Render("[current]")
			cursor = i + 1
		}
		options = append(options, label)
		values = append(values, name)
	}

	t.popup = &Popup{
		kind:    popupSingleSelect,
		title:   "Select image generator",
		options: options,
		values:  values,
		cursor:  cursor,
		onConfirm: func(chosen string) any {
			return ImageModelSelect{name: chosen}
		},
	}
	return t, nil, true
}

func (t TUI) runImageModelSelect(name string) (TUI, tea.Cmd) {
	cfg, err := config.Load()
	if err != nil {
		return t, tea.Println(errorStyle.Render(fmt.Sprintf("[!] session.Load: %v", err)) + "\n")
	}
	if name == "off" {
		name = ""
	}

	if cfg.ImageGenerator == name {
		return t, nil
	}

	cfg.ImageGenerator = name
	if err := config.Save(cfg); err != nil {
		return t, tea.Println(errorStyle.Render(fmt.Sprintf("[!] session.Save: %v", err)) + "\n")
	}
	return t, nil
}
