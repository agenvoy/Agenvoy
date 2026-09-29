package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

const noticeKeepLines = 32

type noticeMsg struct {
	text string
}

func notice(text string) tea.Cmd {
	text = strings.Trim(text, "\n")
	return func() tea.Msg {
		return noticeMsg{text: text}
	}
}

func appendNotice(current, text string) string {
	if text == "" {
		return current
	}
	if current != "" {
		text = current + "\n" + text
	}
	return lastLines(text, noticeKeepLines)
}

func msgError(text string) string {
	return errorStyle.Render("[!] " + text)
}

func msgWarn(text string) string {
	return warnStyle.Render("[~] " + text)
}

func msgLog(text string) string {
	return hintStyle.Render("[*] " + text)
}

func (t TUI) scrollNotice() TUI {
	top := max(strings.Count(t.notice, "\n")+1-noticeMaxLines, 0)
	switch {
	case t.noticeOffset >= top:
		t.noticeOffset = 0
	default:
		t.noticeOffset = min(t.noticeOffset+noticeMaxLines, top)
	}
	return t
}
