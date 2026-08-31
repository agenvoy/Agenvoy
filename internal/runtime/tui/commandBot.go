package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	historyStore "github.com/pardnchiu/agenvoy/internal/runtime/history"
	"github.com/pardnchiu/agenvoy/internal/session"
	configBot "github.com/pardnchiu/agenvoy/internal/session/config/bot"
)

type BotNameSubmit struct {
	name string
}

type BotSelfIDSubmit struct {
	name   string
	selfID string
}

type BotPromptSubmit struct {
	name   string
	selfID string
	body   string
}

type BotCustomSubmit struct {
	name   string
	selfID string
}

type BotSaved struct {
	name string
	err  error
}

func (t TUI) commandBot(parts []string) (TUI, tea.Cmd, bool) {
	sid := strings.TrimSpace(t.currentSessionID)
	if sid == "" {
		return t, tea.Println(errorStyle.Render("[!] no current session") + "\n"), true
	}

	if len(parts) >= 3 {
		name := strings.TrimSpace(parts[1])
		body := strings.TrimSpace(strings.Join(parts[2:], " "))
		if cmd, ok := t.botCheckConflict(sid, name); !ok {
			return t, cmd, true
		}
		selfID, _, _ := configBot.GetPersona(sid)
		return t, t.botSaveCmd(sid, selfID, name, body), true
	}

	refreshBotName(sid)
	_, existingName, existingBody := configBot.GetPersona(sid)
	t.popup = &Popup{
		kind:  popupText,
		title: "Bot name",
		input: newPopupInput(existingName, false),
		onConfirm: func(value string) any {
			return BotNameSubmit{name: strings.TrimSpace(value)}
		},
	}
	t.botBodyDraft = existingBody
	return t, nil, true
}

func (t TUI) botCheckConflict(sid, name string) (tea.Cmd, bool) {
	if name == "" {
		return tea.Println(errorStyle.Render("[!] bot name required") + "\n"), false
	}
	if owner := session.GetSessionID(name); owner != "" && owner != sid {
		return tea.Println(errorStyle.Render(fmt.Sprintf("[!] bot name %q already used by session %s", name, owner)) + "\n"), false
	}
	return nil, true
}

func (t TUI) botCheckSelfID(sid, selfID string) (tea.Cmd, bool) {
	if err := historyStore.ValidSelfID(selfID); err != nil {
		return tea.Println(errorStyle.Render("[!] "+err.Error()) + "\n"), false
	}
	if owner := session.GetSessionIDBySelfID(selfID); owner != "" && owner != sid {
		return tea.Println(errorStyle.Render(fmt.Sprintf("[!] self id %q already used by session %s", selfID, owner)) + "\n"), false
	}
	return nil, true
}

func (t TUI) showBotSelfIDPopup(sid, name string) (TUI, tea.Cmd) {
	existing, _, _ := configBot.GetPersona(sid)
	t.popup = &Popup{
		kind:  popupText,
		title: fmt.Sprintf("Bot self id (%s)", name),
		input: newPopupInput(existing, false),
		onConfirm: func(value string) any {
			return BotSelfIDSubmit{name: name, selfID: strings.TrimSpace(value)}
		},
	}
	return t, nil
}

func (t TUI) showBotPromptPicker(name, selfID string) (TUI, tea.Cmd) {
	options, values := listPromptTemplates()
	if len(options) == 0 {
		return t.showBotCustomPopup(name, selfID)
	}

	displayOptions := append(options, "Custom")
	displayValues := append(values, "")

	t.popup = &Popup{
		kind:    popupSingleSelect,
		title:   fmt.Sprintf("Bot description (%s)", name),
		options: displayOptions,
		values:  displayValues,
		cursor:  0,
		onConfirm: func(chosen string) any {
			if chosen == "" {
				return BotCustomSubmit{name: name, selfID: selfID}
			}
			return BotPromptSubmit{name: name, selfID: selfID, body: readPromptTemplate(chosen)}
		},
	}
	return t, nil
}

func (t TUI) showBotCustomPopup(name, selfID string) (TUI, tea.Cmd) {
	t.popup = &Popup{
		kind:      popupText,
		title:     fmt.Sprintf("Bot description (%s)", name),
		multiline: true,
		input:     newPopupInput(t.botBodyDraft, true),
		onConfirm: func(value string) any {
			return BotPromptSubmit{name: name, selfID: selfID, body: value}
		},
	}
	t.botBodyDraft = ""
	return t, nil
}

func (t TUI) botSaveCmd(sid, selfID, name, body string) tea.Cmd {
	err := configBot.SavePersona(sid, selfID, name, body)
	return func() tea.Msg { return BotSaved{name: name, err: err} }
}
