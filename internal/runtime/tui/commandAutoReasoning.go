package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/pardnchiu/go-pkg/filesystem/keychain"

	"github.com/pardnchiu/agenvoy/internal/session/config"
)

const (
	typesafeConsole    = "https://console.typesafe.ai/keys"
	typesafeLabel      = "TypeSafe/Jev(beta)"
	typesafeDispatcher = "typesafe"
	fieldAutoReasoning = "auto_reasoning"
	fieldDispatcher    = "dispatcher_beta"
)

type AutoReasoningPick struct {
	on bool
}

type TypesafeKeySubmit struct {
	field string
	value string
}

func (t TUI) commandAutoReasoning() (TUI, tea.Cmd, bool) {
	cursor := 0
	if autoReasoningActive() {
		cursor = 1
	}
	t.popup = &Popup{
		kind:     popupSingleSelect,
		title:    "Auto reasoning",
		subtitle: "reasoning effort per request  " + typesafeLabel,
		options:  []string{"off", "on"},
		values:   []string{"off", "on"},
		cursor:   cursor,
		onConfirm: func(chosen string) any {
			return AutoReasoningPick{on: chosen == "on"}
		},
	}
	return t, nil, true
}

func (t TUI) runAutoReasoningPick(on bool) (TUI, tea.Cmd) {
	if on {
		return t.enableTypesafe(fieldAutoReasoning)
	}
	return t, saveTypesafe(fieldAutoReasoning, false)
}

func (t TUI) enableTypesafe(field string) (TUI, tea.Cmd) {
	if strings.TrimSpace(keychain.Get(config.TypesafeKey)) != "" {
		return t, saveTypesafe(field, true)
	}
	t.popup = &Popup{
		kind:     popupText,
		title:    config.TypesafeKey + " is required for " + typesafeLabel,
		subtitle: "Create one in the TypeSafe Console  " + typesafeConsole,
		input:    newPopupInput("", false),
		onConfirm: func(value string) any {
			return TypesafeKeySubmit{field: field, value: strings.TrimSpace(value)}
		},
	}
	return t, nil
}

func (t TUI) runTypesafeKeySubmit(field, value string) (TUI, tea.Cmd) {
	if value == "" {
		return t, tea.Println(msgError(config.TypesafeKey+" is required") + "\n")
	}
	if err := keychain.Set(config.TypesafeKey, value); err != nil {
		return t, tea.Println(msgError(fmt.Sprintf("keychain.Set: %v", err)) + "\n")
	}
	if err := config.SaveKey(config.TypesafeKey); err != nil {
		return t, tea.Println(msgError(fmt.Sprintf("config.SaveKey: %v", err)) + "\n")
	}
	return t, saveTypesafe(field, true)
}

func saveTypesafe(field string, on bool) tea.Cmd {
	cfg, err := config.Load()
	if err != nil {
		return tea.Println(msgError(fmt.Sprintf("config.Load: %v", err)) + "\n")
	}
	var text string
	switch field {
	case fieldAutoReasoning:
		cfg.AutoReasoning = on
		text = "auto reasoning off"
		if on {
			text = "auto reasoning on"
		}
	case fieldDispatcher:
		cfg.DispatcherBeta = on
		text = "dispatcher: " + cfg.DispatcherModel
		if on {
			text = "dispatcher: " + typesafeLabel
		}
	}
	if err := config.Save(cfg); err != nil {
		return tea.Println(msgError(fmt.Sprintf("config.Save: %v", err)) + "\n")
	}
	return tea.Println(msgLog(text) + "\n")
}
