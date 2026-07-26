package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	configBot "github.com/pardnchiu/agenvoy/internal/session/config/bot"
	provider "github.com/pardnchiu/go-llm-router/core"
)

var reasoningLevels = func() []string {
	out := make([]string, 0, int(provider.ReasoningMax)+1)
	for r := provider.ReasoningNone; r <= provider.ReasoningMax; r++ {
		out = append(out, r.String())
	}
	return out
}()

func (t TUI) cycleReasoning(forward bool) (TUI, tea.Cmd) {
	sid := t.currentSessionID
	if sid == "" {
		return t, nil
	}

	_, current := configBot.GetModel(sid)

	idx := 0
	for i, lvl := range reasoningLevels {
		if lvl == current {
			idx = i
			break
		}
	}
	n := len(reasoningLevels)
	if forward {
		idx = (idx + 1) % n
	} else {
		idx = (idx - 1 + n) % n
	}
	level := reasoningLevels[idx]
	configBot.SetModel(sid, "", level)
	return t, nil
}
