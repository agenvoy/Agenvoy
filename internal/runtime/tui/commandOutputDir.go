package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
	"github.com/pardnchiu/agenvoy/internal/session/config"
)

type OutputDirSubmit struct {
	value string
}

func (t TUI) commandOutputDir() (TUI, tea.Cmd, bool) {
	t.popup = &Popup{
		kind:     popupText,
		title:    "Output dir",
		subtitle: "blank uses ~/Downloads, or ~/.config/agenvoy/download without it  now " + filesystem.OutputDir(),
		input:    newPopupInput(filesystem.ConfigOutputDir, false),
		onConfirm: func(value string) any {
			return OutputDirSubmit{value: strings.TrimSpace(value)}
		},
	}
	return t, nil, true
}

func (t TUI) runOutputDirSubmit(value string) (TUI, tea.Cmd) {
	resolved, err := filesystem.ResolveOutputDir(value)
	if err != nil {
		return t, tea.Println(msgError(fmt.Sprintf("output-dir: %v", err)) + "\n")
	}

	dic, err := config.Get()
	if err != nil {
		return t, tea.Println(msgError(fmt.Sprintf("output-dir: %v", err)) + "\n")
	}
	dic["output_dir"] = value
	if err := config.Write(dic); err != nil {
		return t, tea.Println(msgError(fmt.Sprintf("output-dir: %v", err)) + "\n")
	}
	filesystem.ConfigOutputDir = value

	return t, tea.Println(msgLog("output dir: "+resolved) + "\n")
}
