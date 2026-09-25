package tui

import (
	"errors"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	configBot "github.com/pardnchiu/agenvoy/internal/session/config/bot"
)

type ModelScopeSelect struct {
	scope string
}

func (t TUI) commandModel(parts []string) (TUI, tea.Cmd, bool) {
	if len(parts) > 1 {
		switch parts[1] {
		case "add":
			return t.commandModelAdd()
		case "dispatch":
			return t.commandDispatcher()
		case "reasoning":
			return t.commandAutoReasoning()
		case "summary":
			return t.commandSummaryModel()
		case "image":
			return t.commandImageModel()
		case "stt":
			return t.commandSTTModel()
		case "tts":
			return t.commandTTSModel()
		}
	}

	onDelete := func(chosen string) any {
		name, ok := strings.CutPrefix(chosen, sessionModelPrefix)
		if !ok || name == configBot.DefaultModel {
			return nil
		}
		return ModelRemovePick{name: name}
	}
	onTag := func(chosen string) any {
		name, ok := strings.CutPrefix(chosen, sessionModelPrefix)
		if !ok || name == configBot.DefaultModel {
			return nil
		}
		return ModelTagPick{name: name}
	}
	onMove := func(chosen, neighbor string) error {
		name, ok := strings.CutPrefix(chosen, sessionModelPrefix)
		other, otherOK := strings.CutPrefix(neighbor, sessionModelPrefix)
		if !ok || !otherOK || name == configBot.DefaultModel || other == configBot.DefaultModel {
			return errors.New("fallback order only moves between models")
		}
		return swapModelPriority(name, other)
	}

	sid := t.currentSessionID
	popup := &Popup{
		kind:  popupSingleSelect,
		title: "/model",
		tabs:  []string{"model", "config"},
		onConfirm: func(chosen string) any {
			if name, ok := strings.CutPrefix(chosen, sessionModelPrefix); ok {
				return SessionModelSelect{name: name}
			}
			return ModelScopeSelect{scope: chosen}
		},
	}
	popup.onTab = func(p *Popup) {
		if p.tabIdx == 1 {
			actions := []string{"dispatch", "reasoning", "summary", "image", "stt", "tts"}
			p.subtitle, p.styledLines = "", nil
			p.onDelete, p.onTag, p.onMove = nil, nil, nil
			p.options = optionColumn(actions, []string{
				"smart routing",
				"auto reasoning by using TypeSafe/Jev(beta)",
				"summary memory",
				"image generation",
				"audio analysis",
				"speech generation",
			})
			p.values = actions
			p.cursor = 0
			return
		}

		options, values, cursor := registeredModelOptions(sid)
		p.styledLines = nil
		if len(options) == 0 {
			p.styledLines = []string{hintStyle.Render("  no models configured")}
		} else {
			options = append(options, "")
			values = append(values, "")
		}
		p.subtitle = "fallback order applies to auto only  when a provider is unavailable, the models below are tried in order"
		p.onDelete, p.onTag, p.onMove = onDelete, onTag, onMove
		p.options = append(options, optionColumn([]string{"add"}, []string{"add model from provider"})...)
		p.values = append(values, "add")
		p.cursor = cursor
	}
	popup.onTab(popup)
	t.popup = popup
	return t, nil, true
}
