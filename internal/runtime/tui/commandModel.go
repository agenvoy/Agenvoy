package tui

import (
	"errors"
	"slices"
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
			return t.modelPopup(2)
		case "reasoning":
			return t.modelPopup(1)
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

	return t.modelPopup(0)
}

func (t TUI) modelPopup(tab int) (TUI, tea.Cmd, bool) {
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
		tabs:  []string{"model", "reasoning", "dispatch", "config"},
		onConfirm: func(chosen string) any {
			if name, ok := strings.CutPrefix(chosen, sessionModelPrefix); ok {
				return SessionModelSelect{name: name}
			}
			if level, ok := strings.CutPrefix(chosen, sessionReasoningPrefix); ok {
				return SessionReasoningSelect{level: level}
			}
			if name, ok := strings.CutPrefix(chosen, dispatcherPrefix); ok {
				return DispatcherSelect{name: name}
			}
			return ModelScopeSelect{scope: chosen}
		},
	}
	popup.onTab = func(p *Popup) {
		p.styledLines = nil
		p.onDelete, p.onTag, p.onMove = nil, nil, nil

		switch p.tabIdx {
		case 1:
			options, values := reasoningOptions(sid)
			p.subtitle = "reasoning level for this session  auto follows the kind of work the agent selector reports"
			p.options, p.values = options, values
			p.cursor = max(slices.Index(values, sessionReasoningPrefix+currentReasoning(sid)), 0)
			return
		case 2:
			options, values, cursor := dispatcherOptions()
			p.subtitle = "model that routes each request  Jev replaces it with the CLM classifier"
			if len(options) == 0 {
				p.styledLines = []string{hintStyle.Render("  no models configured")}
			}
			p.options, p.values, p.cursor = options, values, cursor
			return
		case 3:
			actions := []string{"summary", "image", "stt", "tts"}
			p.subtitle = ""
			p.options = optionColumn(actions, []string{
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
	popup.tabIdx = tab
	popup.onTab(popup)
	t.popup = popup
	return t, nil, true
}
