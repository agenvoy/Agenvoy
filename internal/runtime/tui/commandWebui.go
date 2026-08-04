package tui

import (
	"fmt"
	"os"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
	"github.com/pardnchiu/agenvoy/internal/runtime/webui"
)

type WebuiAction struct {
	action string
}

type WebuiDone struct {
	action string
	err    error
}

func (t TUI) commandWebui(parts []string) (TUI, tea.Cmd, bool) {
	if len(parts) > 1 {
		switch parts[1] {
		case "enable", "disable":
			action := parts[1]
			return t, func() tea.Msg { return WebuiAction{action: action} }, true
		}
	}

	engine := webui.Engine()
	_, running := webui.Status(engine)

	options := []string{"enable", "disable"}
	cursor := 0
	if running {
		cursor = 1
	}
	t.popup = &Popup{
		kind:        popupSingleSelect,
		title:       "Open WebUI",
		styledLines: webuiStatus(engine),
		options:     options,
		values:      options,
		cursor:      cursor,
		onConfirm: func(chosen string) any {
			return WebuiAction{action: chosen}
		},
	}
	return t, nil, true
}

func webuiStatus(engine string) []string {
	if engine == "" {
		return []string{hintStyle.Render("  engine  ") + errorStyle.Render("○ podman / docker not installed")}
	}
	prefix := hintStyle.Render(fmt.Sprintf("  %s  ", engine))
	exists, running := webui.Status(engine)
	switch {
	case running:
		return []string{prefix + okayStyle.Render("● running ("+webui.URL()+")")}
	case exists:
		return []string{prefix + errorStyle.Render("○ stopped")}
	default:
		return []string{prefix + errorStyle.Render("○ not deployed")}
	}
}

func runWebuiExec(action string) tea.Cmd {
	script := fmt.Sprintf(`set -e
curl -fsSL %s | bash -s -- %s
`, webui.InstallURL, action)

	cmd := exec.Command("bash", "-c", script)
	cmd.Env = append(os.Environ(), "AGENVOY_PORT="+filesystem.Port)
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		if err != nil {
			return WebuiDone{action: action, err: fmt.Errorf("webui.sh %s: %w", action, err)}
		}
		return WebuiDone{action: action}
	})
}
