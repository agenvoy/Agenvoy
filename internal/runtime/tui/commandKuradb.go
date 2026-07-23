package tui

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
	"github.com/pardnchiu/agenvoy/internal/runtime/kuradb"
	"github.com/pardnchiu/agenvoy/internal/session/config"
)

type KuradbAction struct {
	action string
}

type KuradbKeySubmit struct {
	token string
}

type KuradbDone struct {
	action string
	err    error
}

func (t TUI) commandKuradb(parts []string) (TUI, tea.Cmd, bool) {
	if len(parts) > 1 {
		switch parts[1] {
		case "enable", "disable", "update", "start", "stop":
			action := parts[1]
			return t, func() tea.Msg { return KuradbAction{action: action} }, true
		}
	}

	installed := kuradb.IsInstalled()

	var options []string
	if !installed {
		options = []string{"enable"}
	} else {
		options = []string{"update", "disable"}
		if kuradb.IsRunning() {
			options = append(options, "stop")
		} else {
			options = append(options, "start")
		}
	}
	t.popup = &Popup{
		kind:        popupSingleSelect,
		title:       "KuraDB",
		styledLines: kuradbStatus(installed),
		options:     options,
		values:      options,
		onConfirm: func(chosen string) any {
			return KuradbAction{action: chosen}
		},
	}
	return t, nil, true
}

func kuradbStatus(installed bool) []string {
	if !installed {
		return nil
	}
	version := "unknown"
	if v, err := kuradb.Version(); err == nil && v != "" {
		version = v
	}
	prefix := hintStyle.Render(fmt.Sprintf("  kura %s  ", version))
	if kuradb.IsRunning() {
		status := "● running"
		if endpoint, err := filesystem.GetKuradbEndpoint(); err == nil && endpoint != "" {
			status += " (" + endpoint + ")"
		}
		return []string{prefix + okayStyle.Render(status)}
	}
	return []string{prefix + errorStyle.Render("○ stopped")}
}

func (t TUI) openKuradbKeyPrompt() (TUI, tea.Cmd) {
	t.popup = &Popup{
		kind:     popupText,
		title:    "KuraDB · OPENAI_API_KEY",
		input:    newPopupInput("", false),
		subtitle: "required for embedding (text-embedding-3-small) · Enter to submit · Esc to cancel",
		onConfirm: func(value string) any {
			return KuradbKeySubmit{token: strings.TrimSpace(value)}
		},
	}
	return t, nil
}

func runKuradbEnableExec() tea.Cmd {
	script := fmt.Sprintf(`set -e
curl -fsSL %s | bash
kura add agenvoy 2>/dev/null || true
`, kuradb.InstallURL)

	cmd := exec.Command("bash", "-c", script)
	cmd.Env = os.Environ()
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		if err != nil {
			return KuradbDone{action: "enable", err: fmt.Errorf("install script: %w", err)}
		}
		if !kuradb.IsInstalled() {
			return KuradbDone{action: "enable", err: fmt.Errorf("kura binary not at %s after install", kuradb.BinaryPath)}
		}
		cfg, err := config.Load()
		if err != nil {
			return KuradbDone{action: "enable", err: fmt.Errorf("session.Load: %w", err)}
		}
		cfg.KuradbEnabled = true
		if err := config.Save(cfg); err != nil {
			return KuradbDone{action: "enable", err: fmt.Errorf("session.Save: %w", err)}
		}
		return KuradbDone{action: "enable"}
	})
}

func runKuradbUpdateExec() tea.Cmd {
	if cfg, err := config.Load(); err == nil && cfg != nil {
		cfg.KuradbEnabled = false
		if err := config.Save(cfg); err != nil {
			return func() tea.Msg {
				return KuradbDone{action: "update", err: fmt.Errorf("session.Save(stop): %w", err)}
			}
		}
	}

	script := fmt.Sprintf(`set -e
curl -fsSL %s | bash
kura add agenvoy 2>/dev/null || true
`, kuradb.InstallURL)

	cmd := exec.Command("bash", "-c", script)
	cmd.Env = os.Environ()
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		if err != nil {
			if kuradb.IsInstalled() {
				if cfg, lerr := config.Load(); lerr == nil && cfg != nil {
					cfg.KuradbEnabled = true
					_ = config.Save(cfg)
				}
			}
			return KuradbDone{action: "update", err: fmt.Errorf("install script: %w", err)}
		}
		if !kuradb.IsInstalled() {
			return KuradbDone{action: "update", err: fmt.Errorf("kura binary not at %s after install", kuradb.BinaryPath)}
		}
		cfg, err := config.Load()
		if err != nil {
			return KuradbDone{action: "update", err: fmt.Errorf("session.Load: %w", err)}
		}
		cfg.KuradbEnabled = true
		if err := config.Save(cfg); err != nil {
			return KuradbDone{action: "update", err: fmt.Errorf("session.Save: %w", err)}
		}
		return KuradbDone{action: "update"}
	})
}

func startKuradb() tea.Cmd {
	cmd := exec.Command(kuradb.BinaryPath)
	cmd.Env = os.Environ()
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		if err != nil {
			return KuradbDone{action: "start", err: fmt.Errorf("kura: %w", err)}
		}
		return KuradbDone{action: "start"}
	})
}

func stopKuradb() tea.Cmd {
	cmd := exec.Command(kuradb.BinaryPath, "stop")
	cmd.Env = os.Environ()
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		if err != nil {
			return KuradbDone{action: "stop", err: fmt.Errorf("kura stop: %w", err)}
		}
		return KuradbDone{action: "stop"}
	})
}

func runKuradbDisableExec() tea.Cmd {
	cmd := exec.Command("sudo", "rm", "-f", kuradb.BinaryPath)
	cmd.Env = os.Environ()
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		if err != nil {
			return KuradbDone{action: "disable", err: fmt.Errorf("rm %s: %w", kuradb.BinaryPath, err)}
		}
		cfg, err := config.Load()
		if err != nil {
			return KuradbDone{action: "disable", err: fmt.Errorf("session.Load: %w", err)}
		}
		cfg.KuradbEnabled = false
		if err := config.Save(cfg); err != nil {
			return KuradbDone{action: "disable", err: fmt.Errorf("session.Save: %w", err)}
		}
		return KuradbDone{action: "disable"}
	})
}
