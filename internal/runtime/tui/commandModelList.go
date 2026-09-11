package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/pardnchiu/agenvoy/internal/session/config"
	configBot "github.com/pardnchiu/agenvoy/internal/session/config/bot"
)

const sessionModelPrefix = "model:"

type SessionModelSelect struct {
	name string
}

func registeredModelOptions(sid string) (options, values []string, cursor int) {
	cfg, err := config.Load()
	if err != nil || cfg == nil || len(cfg.Models) == 0 {
		return nil, nil, 0
	}

	current := ""
	if sid != "" {
		current, _ = configBot.GetModel(sid)
	}

	options = make([]string, 0, len(cfg.Models)+1)
	values = make([]string, 0, len(cfg.Models)+1)

	auto := configBot.DefaultModel
	if current == configBot.DefaultModel {
		auto += "  " + systemStyle.Render("[current]")
	}
	options = append(options, auto)
	values = append(values, sessionModelPrefix+configBot.DefaultModel)

	for _, m := range cfg.Models {
		label := m.Name
		if m.Name == current {
			label += "  " + systemStyle.Render("[current]")
			cursor = len(options)
		}
		if cfg.DispatcherModel != "" && m.Name == cfg.DispatcherModel {
			label += "  " + okayStyle.Render("[dispatcher]")
		}
		if cfg.SummaryModel != "" && m.Name == cfg.SummaryModel {
			label += "  " + okayStyle.Render("[summary]")
		}
		options = append(options, label)
		values = append(values, sessionModelPrefix+m.Name)
	}
	return options, values, cursor
}

func (t TUI) runSessionModelSelect(name string) (TUI, tea.Cmd) {
	sid := strings.TrimSpace(t.currentSessionID)
	if sid == "" {
		return t, tea.Println(msgLog("no active session") + "\n")
	}
	configBot.SetModel(sid, name, "")
	return t, tea.Println(msgLog(fmt.Sprintf("model: %s", name)) + "\n")
}
