package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/pardnchiu/agenvoy/internal/session/config"
	configBot "github.com/pardnchiu/agenvoy/internal/session/config/bot"
)

const dispatcherPrefix = "dispatch:"

type DispatcherSelect struct {
	name string
}

func dispatcherOptions() (options, values []string, cursor int) {
	cfg, err := config.Load()
	if err != nil || len(cfg.Models) == 0 {
		return nil, nil, 0
	}

	options = make([]string, len(cfg.Models))
	values = make([]string, len(cfg.Models))
	for i, m := range cfg.Models {
		label := m.Name
		if !cfg.DispatcherBeta && cfg.DispatcherModel != "" && m.Name == cfg.DispatcherModel {
			label += "  " + systemStyle.Render("[current]")
			cursor = i
		}
		options[i] = label
		values[i] = dispatcherPrefix + m.Name
	}

	jev := "Jev  " + hintStyle.Render("use CLM model "+config.TypesafeModel)
	if cfg.DispatcherBeta {
		jev += "  " + systemStyle.Render("[current]")
		cursor = len(options) + 1
	}
	options = append(options, "", jev)
	values = append(values, "", dispatcherPrefix+typesafeDispatcher)
	return options, values, cursor
}

func (t TUI) cycleDispatcher(forward bool) (TUI, tea.Cmd) {
	sid := t.currentSessionID
	if sid == "" {
		return t, nil
	}

	cfg, err := config.Load()
	if err != nil || len(cfg.Models) == 0 {
		return t, nil
	}

	candidates := make([]string, 0, len(cfg.Models)+1)
	candidates = append(candidates, configBot.DefaultModel)
	for _, m := range cfg.Models {
		candidates = append(candidates, m.Name)
	}

	currentModel, _ := configBot.GetModel(sid)
	current := 0
	for i, c := range candidates {
		if c == currentModel {
			current = i
			break
		}
	}

	n := len(candidates)
	var next int
	if forward {
		next = (current + 1) % n
	} else {
		next = (current - 1 + n) % n
	}

	picked := candidates[next]
	configBot.SetModel(sid, picked, "")
	return t, nil
}

func (t TUI) runDispatcherSelect(name string) (TUI, tea.Cmd) {
	cfg, err := config.Load()
	if err != nil {
		return t, tea.Println(msgError(fmt.Sprintf("session.Load: %v", err)) + "\n")
	}
	if name == typesafeDispatcher {
		return t.enableTypesafe(fieldDispatcher)
	}
	if cfg.DispatcherModel == name && !cfg.DispatcherBeta {
		return t, tea.Println(msgLog(fmt.Sprintf("dispatcher unchanged: %s", name)) + "\n")
	}

	cfg.DispatcherModel = name
	cfg.DispatcherBeta = false
	if err := config.Save(cfg); err != nil {
		return t, tea.Println(msgError(fmt.Sprintf("session.Save: %v", err)) + "\n")
	}
	return t, tea.Println(msgLog(fmt.Sprintf("dispatcher: %s", name)) + "\n")
}
