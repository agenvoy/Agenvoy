package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	configBot "github.com/pardnchiu/agenvoy/internal/session/config/bot"
	"github.com/pardnchiu/go-llm-router/core"
)

var reasoningLevels = []string{"none", "low", "medium", "high", "xhigh"}

func filteredReasoningLevels(model string) []string {
	providerName, modelName, _ := strings.Cut(model, "@")
	min := provider.MinReasoningLevel(providerName, modelName)
	for i, lvl := range reasoningLevels {
		if lvl == min {
			return reasoningLevels[i:]
		}
	}
	return reasoningLevels
}

func (t TUI) cycleReasoning(forward bool) (TUI, tea.Cmd) {
	sid := t.currentSessionID
	if sid == "" {
		return t, nil
	}

	model, current := configBot.GetModel(sid)
	providerName, modelName, _ := strings.Cut(model, "@")
	if !provider.SupportsReasoningSwitch(providerName, modelName) {
		return t, tea.Println(hintStyle.Render(fmt.Sprintf("⎯ %s does not support reasoning switching", model)) + "\n")
	}

	levels := filteredReasoningLevels(model)
	idx := 0
	for i, lvl := range levels {
		if lvl == current {
			idx = i
			break
		}
	}
	n := len(levels)
	if forward {
		idx = (idx + 1) % n
	} else {
		idx = (idx - 1 + n) % n
	}
	level := levels[idx]
	configBot.SetModel(sid, "", level)
	return t, nil
}
