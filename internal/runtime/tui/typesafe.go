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
	fieldDispatcher    = "dispatcher_beta"
)

type TypesafeKeySubmit struct {
	field string
	value string
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
		return t, notice(msgError(config.TypesafeKey+" is required") + "\n")
	}
	if err := keychain.Set(config.TypesafeKey, value); err != nil {
		return t, notice(msgError(fmt.Sprintf("keychain.Set: %v", err)) + "\n")
	}
	if err := config.SaveKey(config.TypesafeKey); err != nil {
		return t, notice(msgError(fmt.Sprintf("config.SaveKey: %v", err)) + "\n")
	}
	return t, saveTypesafe(field, true)
}

func saveTypesafe(field string, on bool) tea.Cmd {
	cfg, err := config.Load()
	if err != nil {
		return notice(msgError(fmt.Sprintf("config.Load: %v", err)) + "\n")
	}
	var text string
	switch field {
	case fieldDispatcher:
		cfg.DispatcherBeta = on
		text = "dispatcher: " + cfg.DispatcherModel
		if on {
			text = "dispatcher: " + typesafeLabel
		}
	}
	if err := config.Save(cfg); err != nil {
		return notice(msgError(fmt.Sprintf("config.Save: %v", err)) + "\n")
	}
	return notice(msgLog(text) + "\n")
}
