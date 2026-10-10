package tui

import (
	"fmt"
	"slices"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/pardnchiu/agenvoy/internal/agents"
	agentTypes "github.com/pardnchiu/agenvoy/internal/agents/types"
	"github.com/pardnchiu/agenvoy/internal/session/config"
	configBot "github.com/pardnchiu/agenvoy/internal/session/config/bot"

	"github.com/pardnchiu/go-llm-router/core/claudeCode"
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

	skipClaudeCode := !agentTypes.ClaudeCodeEnabled()
	for _, m := range cfg.Models {
		if skipClaudeCode && claudeCode.Is(m.Name) {
			continue
		}
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
		switch tag := cfg.ModelTag[m.Name]; tag {
		case "":
		case config.ModelTagPass:
			label += "  " + warnStyle.Render(tag)
		default:
			label += "  " + warnStyle.Render(tag+"-tier")
		}
		options = append(options, label)
		values = append(values, sessionModelPrefix+m.Name)
	}
	return options, values, cursor
}

func (t TUI) runSessionModelSelect(name string) (TUI, tea.Cmd) {
	sid := strings.TrimSpace(t.currentSessionID)
	if sid == "" {
		return t, notice(msgLog("no active session") + "\n")
	}
	configBot.SetModel(sid, name, "")
	return t, nil
}

type ModelOrderSave struct {
	names []string
	next  any
}

func saveModelOrder(names []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config.Load: %w", err)
	}
	var slots []int
	for i, m := range cfg.Models {
		if slices.Contains(names, m.Name) {
			slots = append(slots, i)
		}
	}
	if len(slots) != len(names) {
		return fmt.Errorf("model list changed, reorder again")
	}
	for k, i := range slots {
		cfg.Models[i] = config.ModelEntry{Name: names[k]}
	}
	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("config.Save: %w", err)
	}
	agents.Reload()
	return nil
}

func (t TUI) runModelOrderSave(msg ModelOrderSave) (tea.Model, tea.Cmd) {
	if err := saveModelOrder(msg.names); err != nil {
		return t, notice(msgError(fmt.Sprintf("fallback order: %v", err)) + "\n")
	}
	if msg.next == nil {
		return t, nil
	}
	return t.update(msg.next)
}
