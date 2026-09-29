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
			return t.modelPopup(3)
		case "image":
			return t.modelPopup(4)
		case "stt":
			return t.modelPopup(5)
		case "tts":
			return t.modelPopup(6)
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
		tabs:  []string{"model", "reasoning", "dispatch", "summary", "image", "stt", "tts"},
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
			if name, ok := strings.CutPrefix(chosen, summaryPrefix); ok {
				return SummaryModelSelect{name: name}
			}
			if name, ok := strings.CutPrefix(chosen, imagePrefix); ok {
				return ImageModelSelect{name: name}
			}
			if name, ok := strings.CutPrefix(chosen, sttPrefix); ok {
				return AudioModelSelect{kind: "stt", name: name}
			}
			if name, ok := strings.CutPrefix(chosen, ttsPrefix); ok {
				return AudioModelSelect{kind: "tts", name: name}
			}
			return ModelScopeSelect{scope: chosen}
		},
	}
	popup.onTab = func(p *Popup) tea.Cmd {
		p.styledLines = nil
		p.onDelete, p.onTag, p.onMove = nil, nil, nil
		p.searchable, p.allOptions, p.allValues = false, nil, nil

		switch p.tabIdx {
		case 1:
			options, values := reasoningOptions(sid)
			p.subtitle = "reasoning level for this session  auto follows the kind of work the agent selector reports"
			p.options, p.values = options, values
			p.cursor = max(slices.Index(values, sessionReasoningPrefix+currentReasoning(sid)), 0)
			return nil
		case 2:
			options, values, cursor := dispatcherOptions()
			p.subtitle = "model that routes each request  Jev replaces it with the CLM classifier"
			if len(options) == 0 {
				p.styledLines = []string{hintStyle.Render("  no models configured")}
			}
			p.options, p.values, p.cursor = options, values, cursor
			return nil
		case 3:
			options, values, cursor := summaryOptions()
			p.subtitle = "model that writes summary memory"
			if len(options) == 0 {
				p.styledLines = []string{hintStyle.Render("  no models configured")}
			}
			p.options, p.values, p.cursor = options, values, cursor
			return nil
		case 4:
			p.subtitle = "provider that generates images"
			return routingTabLoad(p, "image")
		case 5:
			p.subtitle = "model that transcribes audio"
			return routingTabLoad(p, "stt")
		case 6:
			p.subtitle = "model that generates speech"
			return routingTabLoad(p, "tts")
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
		return nil
	}
	popup.tabIdx = tab
	cmd := popup.onTab(popup)
	t.popup = popup
	return t, cmd, true
}
