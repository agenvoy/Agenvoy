package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type SessionSelect struct {
	id string
}

type SessionNew struct{}

func (t TUI) handleCommand(cmd string) (TUI, tea.Cmd, bool) {
	parts := strings.Fields(cmd)
	if strings.HasPrefix(parts[0], "/sched-") {
		return t.commandSchedule(parts)
	}
	switch parts[0] {
	case "/exit", "/quit":
		return t, tea.Sequence(
			tea.Println(msgLog("bye.")+"\n"),
			tea.Quit,
		), true

	case "/clear":
		t.tokens = 0
		t.lastIn = 0
		t.lastOut = 0
		t.lastCacheRead = 0
		t.lastCacheCreate = 0
		return t, tea.Sequence(
			tea.ClearScreen,
			tea.Println(headerBlock(t.daemonStatus, t.httpStatus, t.discordStatus, t.telegramStatus)),
		), true

	case "/switch":
		return t.commandSwitch(parts)

	case "/new":
		return t.commandNew(parts)

	case "/remove-session":
		return t.commandRemoveSession()

	case "/allow-skill":
		return t.commandAllowSkill(parts)

	case "/compact":
		return t.commandCompact()

	case "/reset":
		return t.commandReset()

	case "/bot":
		return t.commandBot(parts)

	case "/rule":
		return t.commandNote("rule")

	case "/note":
		return t.commandNote("note")

	case "/model":
		return t.commandModel(parts)

	case "/mcp":
		return t.commandMcp(parts)

	case "/channel":
		return t.commandChannel(parts)

	case "/startup":
		return t.commandStartup(parts)

	case "/schedule":
		return t.commandScheduleMenu(parts)

	case "/update":
		return t.commandUpdate()

	case "/resume":
		return t.commandHistory()

	case "/log":
		return t.commandLog()

	case "/usage":
		return t.commandUsage(parts)

	case "/key":
		return t.commandKey(parts)

	case "/reply-language":
		return t.commandReplyLanguage()

	case "/pending":
		return t.commandPending()

	}
	return t, nil, false
}

func (t TUI) commandHistory() (TUI, tea.Cmd, bool) {
	sid := strings.TrimSpace(t.currentSessionID)
	if sid == "" {
		return t, tea.Println(msgLog("no active session") + "\n"), true
	}
	seq := []tea.Cmd{
		tea.ClearScreen,
		tea.Println(headerBlock(t.daemonStatus, t.httpStatus, t.discordStatus, t.telegramStatus)),
	}
	tail := loadSessionTail(sid, t.width, true)
	if len(tail) == 0 {
		seq = append(seq, tea.Println(msgLog("no history yet")+"\n"))
	} else {
		seq = append(seq, tail...)
	}
	return t, tea.Sequence(seq...), true
}
