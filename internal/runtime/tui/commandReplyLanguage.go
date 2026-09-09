package tui

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
	"github.com/pardnchiu/agenvoy/internal/runtime/daemon"
)

const replyLanguageTimeout = 10 * time.Second

type ReplyLanguageSelect struct {
	code string
}

type replyLanguageState struct {
	ReplyLang string                       `json:"reply_lang"`
	Languages []filesystem.ReplyLangOption `json:"languages"`
}

func (t TUI) commandReplyLanguage() (TUI, tea.Cmd, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), replyLanguageTimeout)
	defer cancel()

	state, err := daemon.Get[replyLanguageState](ctx, "/v1/config/system", nil)
	if err != nil {
		return t, tea.Println(errorStyle.Render(fmt.Sprintf("[!] reply-language: %v", err)) + "\n"), true
	}
	if len(state.Languages) == 0 {
		return t, tea.Println(hintStyle.Render("no languages available") + "\n"), true
	}

	width := 0
	for _, one := range state.Languages {
		width = max(width, len(one.Code)+2)
	}

	options := make([]string, 0, len(state.Languages))
	values := make([]string, 0, len(state.Languages))
	cursor := 0
	for i, one := range state.Languages {
		label := fmt.Sprintf("%-*s %s", width, "["+one.Code+"]", one.Label)
		if one.Code == state.ReplyLang {
			label += "  " + systemStyle.Render("[current]")
			cursor = i
		}
		options = append(options, label)
		values = append(values, one.Code)
	}

	t.popup = &Popup{
		kind:    popupSingleSelect,
		title:   "Select reply language",
		options: options,
		values:  values,
		cursor:  cursor,
		onConfirm: func(chosen string) any {
			return ReplyLanguageSelect{code: chosen}
		},
	}
	return t, nil, true
}

func (t TUI) runReplyLanguageSelect(code string) (TUI, tea.Cmd) {
	ctx, cancel := context.WithTimeout(context.Background(), replyLanguageTimeout)
	defer cancel()

	state, err := daemon.Post[replyLanguageState](ctx, "/v1/config/system", map[string]any{"reply_lang": code})
	if err != nil {
		return t, tea.Println(errorStyle.Render(fmt.Sprintf("[!] reply-language: %v", err)) + "\n")
	}
	return t, tea.Println(hintStyle.Render("⎯ reply language: "+state.ReplyLang) + "\n")
}
