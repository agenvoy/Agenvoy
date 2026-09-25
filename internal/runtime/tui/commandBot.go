package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	historyStore "github.com/pardnchiu/agenvoy/internal/runtime/store"
	"github.com/pardnchiu/agenvoy/internal/session"
	configBot "github.com/pardnchiu/agenvoy/internal/session/config/bot"
)

type BotFieldPick struct {
	field string
}

type BotFieldSubmit struct {
	field string
	value string
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

func (t TUI) openBotField(sid, field string) (TUI, tea.Cmd) {
	selfID, name, body := configBot.GetPersona(sid)
	if field == "role" {
		t.botBodyDraft = body
		return t.showBotPromptPicker(name, selfID)
	}

	existing, subtitle := name, "session display name"
	if field == "id" {
		existing, subtitle = selfID, "self id  A-Z a-z 0-9 _ - only  used to call this session by name"
	}
	t.popup = &Popup{
		kind:     popupText,
		title:    "/session " + field,
		subtitle: subtitle,
		input:    newPopupInput(existing, false),
		onConfirm: func(value string) any {
			return BotFieldSubmit{field: field, value: strings.TrimSpace(value)}
		},
	}
	return t, nil
}

func (t TUI) runBotFieldSubmit(msg BotFieldSubmit) (TUI, tea.Cmd) {
	sid := strings.TrimSpace(t.currentSessionID)
	if sid == "" {
		return t, tea.Println(msgError("no current session") + "\n")
	}
	selfID, name, body := configBot.GetPersona(sid)
	switch msg.field {
	case "name":
		if cmd, ok := t.botCheckConflict(sid, msg.value); !ok {
			return t, cmd
		}
		name = msg.value
	case "id":
		if cmd, ok := t.botCheckSelfID(sid, msg.value); !ok {
			return t, cmd
		}
		selfID = msg.value
	}
	return t, t.botSaveCmd(sid, selfID, name, body)
}

func (t TUI) botCheckConflict(sid, name string) (tea.Cmd, bool) {
	if name == "" {
		return tea.Println(msgError("bot name required") + "\n"), false
	}
	return nil, true
}

func (t TUI) botCheckSelfID(sid, selfID string) (tea.Cmd, bool) {
	if err := historyStore.ValidSelfID(selfID); err != nil {
		return tea.Println(msgError(""+err.Error()) + "\n"), false
	}
	if owner := session.GetSessionIDBySelfID(selfID); owner != "" && owner != sid {
		return tea.Println(msgError(fmt.Sprintf("self id %q already used by session %s", selfID, owner)) + "\n"), false
	}
	return nil, true
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
		title:   "/session role",
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
		title:     "/session role",
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
