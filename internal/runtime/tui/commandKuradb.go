package tui

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/pardnchiu/agenvoy/internal/runtime/kuradb"
)

type KuradbAction struct {
	action string
}

type KuradbDone struct {
	action string
	err    error
}

func (t TUI) commandKuradb(parts []string) (TUI, tea.Cmd, bool) {
	if len(parts) > 1 {
		switch parts[1] {
		case "setup", "update", "reconnect":
			action := parts[1]
			return t, func() tea.Msg { return KuradbAction{action: action} }, true
		}
	}

	options := kuradbActions()
	t.popup = &Popup{
		kind:        popupSingleSelect,
		title:       "KuraDB",
		styledLines: kuradbStatus(),
		options:     options,
		values:      options,
		onConfirm: func(chosen string) any {
			return KuradbAction{action: chosen}
		},
	}
	return t, nil, true
}

func kuradbActions() []string {
	if !kuradb.Installed() {
		return []string{"setup"}
	}
	name, registered := kuradb.Registered()
	if !registered {
		return []string{"setup", "update"}
	}
	if info, ok := mcpServerStatus(name); !ok || !info.Connected {
		return []string{"reconnect", "update"}
	}
	return []string{"update"}
}

func kuradbStatus() []string {
	prefix := hintStyle.Render("  kura  ")

	name, registered := kuradb.Registered()
	if !registered {
		if kuradb.Installed() {
			return []string{prefix + errorStyle.Render("○ installed · not registered in mcp.json")}
		}
		return []string{prefix + errorStyle.Render("○ not installed")}
	}

	info, ok := mcpServerStatus(name)
	switch {
	case !ok:
		return []string{prefix + errorStyle.Render("○ registered · no client")}
	case !info.Connected:
		lines := []string{prefix + errorStyle.Render("○ "+name+" · disconnected")}
		if info.Error != "" {
			lines = append(lines, hintStyle.Render("         ")+errorStyle.Render(info.Error))
		}
		return lines
	case info.Error != "":
		return []string{prefix + warnStyle.Render("● "+name+" · tools unavailable")}
	default:
		return []string{prefix + okayStyle.Render("● "+name+" · connected")}
	}
}

func connectError() error {
	name, registered := kuradb.Registered()
	if !registered {
		return fmt.Errorf("not registered in mcp.json")
	}
	info, ok := mcpServerStatus(name)
	switch {
	case !ok || !info.Connected:
		if ok && info.Error != "" {
			return fmt.Errorf("%s did not connect: %s", name, info.Error)
		}
		return fmt.Errorf("%s did not connect", name)
	default:
		return nil
	}
}

func reconnectUntilUp(sessionID string) error {
	var err error
	for attempt := range 2 {
		if attempt > 0 {
			time.Sleep(time.Second)
		}
		if err = kuradb.Reconnect(context.Background(), sessionID); err != nil {
			continue
		}
		if err = connectError(); err == nil {
			return nil
		}
	}
	return err
}

func runKuradbSetup(sessionID string) tea.Cmd {
	if kuradb.Installed() {
		return func() tea.Msg {
			if err := kuradb.Register(context.Background(), sessionID); err != nil {
				return KuradbDone{action: "setup", err: err}
			}
			return KuradbDone{action: "setup", err: connectError()}
		}
	}
	return runKuradbScript("setup", sessionID, false)
}

func runKuradbUpdate(sessionID string) tea.Cmd {
	return runKuradbScript("update", sessionID, true)
}

func runKuradbReconnect(sessionID string) tea.Cmd {
	return func() tea.Msg {
		return KuradbDone{action: "reconnect", err: reconnectUntilUp(sessionID)}
	}
}

func runKuradbScript(action, sessionID string, force bool) tea.Cmd {
	script := fmt.Sprintf("set -e\ncurl -fsSL %s | bash", kuradb.InstallURL)
	if force {
		script += " -s -- --force"
	}

	cmd := exec.Command("bash", "-c", script+"\n")
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		if err != nil {
			return KuradbDone{action: action, err: fmt.Errorf("kuradb.sh: %w", err)}
		}
		if err := kuradb.Register(context.Background(), sessionID); err != nil {
			return KuradbDone{action: action, err: err}
		}
		return KuradbDone{action: action, err: reconnectUntilUp(sessionID)}
	})
}
