package tui

import (
	"context"
	"fmt"
	"slices"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/pardnchiu/go-pkg/filesystem/keychain"

	"github.com/pardnchiu/agenvoy/internal/session/config"
	imageTool "github.com/pardnchiu/agenvoy/internal/tools/external/image"
)

type KeySelect struct {
	key string
}

type KeySubmit struct {
	key   string
	value string
}

type KeyDeletePick struct {
	key string
}

type KeyDeleteConfirm struct {
	key string
	yes bool
}

func (t TUI) commandKey(parts []string) (TUI, tea.Cmd, bool) {
	cfg, err := config.Load()
	if err != nil {
		return t, tea.Println(msgError(fmt.Sprintf("session.Load: %v", err)) + "\n"), true
	}
	if len(cfg.Keys) == 0 {
		return t, tea.Println(msgLog("no keys recorded  run /model add or store_secret first") + "\n"), true
	}

	if len(parts) > 1 {
		target := strings.TrimSpace(parts[1])
		if !slices.Contains(cfg.Keys, target) {
			return t, tea.Println(msgError(fmt.Sprintf("key not recorded: %q", target)) + "\n"), true
		}
		next, cmd := t.openKeyValuePrompt(target)
		return next, cmd, true
	}

	t.popup = &Popup{
		kind:        popupSingleSelect,
		title:       "Key  update keychain value",
		options:     cfg.Keys,
		values:      cfg.Keys,
		enterAction: "edit",
		onConfirm: func(chosen string) any {
			return KeySelect{key: chosen}
		},
		onDelete: func(chosen string) any {
			return KeyDeletePick{key: chosen}
		},
	}
	return t, nil, true
}

func (t TUI) openKeyDeleteConfirm(key string) (TUI, tea.Cmd) {
	t.popup = &Popup{
		kind:     popupSingleSelect,
		title:    "Delete " + key + " ?",
		subtitle: "removed from the OS keychain and from the recorded key list",
		options:  []string{"No", "Yes"},
		values:   []string{"no", "yes"},
		onConfirm: func(chosen string) any {
			return KeyDeleteConfirm{key: key, yes: chosen == "yes"}
		},
		onCancel: func() any {
			return KeyDeleteConfirm{key: key}
		},
	}
	return t, nil
}

func (t TUI) runKeyDelete(key string) (TUI, tea.Cmd) {
	if err := keychain.Delete(key); err != nil {
		return t, tea.Println(msgError(fmt.Sprintf("keychain.Delete %s: %v", key, err)) + "\n")
	}
	if err := config.DeleteKey(key); err != nil {
		return t, tea.Println(msgError(fmt.Sprintf("config.DeleteKey %s: %v", key, err)) + "\n")
	}
	imageTool.Prune(context.Background())

	next, cmd, _ := t.commandKey(nil)
	return next, tea.Sequence(tea.Println(msgLog("key deleted: "+key)+"\n"), cmd)
}

func (t TUI) openKeyValuePrompt(key string) (TUI, tea.Cmd) {
	t.popup = &Popup{
		kind:     popupText,
		title:    fmt.Sprintf("Key  %s", key),
		input:    newPopupInput("", false),
		subtitle: "Enter new value  Enter to submit  Esc to cancel",
		onConfirm: func(value string) any {
			return KeySubmit{key: key, value: strings.TrimSpace(value)}
		},
	}
	return t, nil
}
