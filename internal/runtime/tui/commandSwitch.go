package tui

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	go_pkg_filesystem_reader "github.com/pardnchiu/go-pkg/filesystem/reader"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
	sessionManager "github.com/pardnchiu/agenvoy/internal/session"
	configBot "github.com/pardnchiu/agenvoy/internal/session/config/bot"
	"github.com/pardnchiu/agenvoy/internal/utils"
)

type Session struct {
	id   string
	name string
}

func (t TUI) commandSwitch(parts []string) (TUI, tea.Cmd, bool) {
	if len(parts) >= 2 {
		name := strings.Join(parts[1:], " ")
		id := sessionManager.GetSessionID(name)
		if id == "" {
			return t, tea.Println(errorStyle.Render(fmt.Sprintf("[!] session %q not found", name)) + "\n"), true
		}
		next, cmd := t.runCommandSwitch(id)
		return next, cmd, true
	}

	popup := popupSwitch(t.currentSessionID)
	if popup == nil {
		return t, tea.Println(hintStyle.Render("no sessions available") + "\n"), true
	}
	popup.onConfirm = func(chosen string) any {
		if chosen == "" {
			return SessionNew{}
		}
		return SessionSelect{id: chosen}
	}
	t.popup = popup
	return t, nil, true
}

func (t TUI) runCommandSwitch(id string) (TUI, tea.Cmd) {
	if id == t.currentSessionID {
		return t, tea.Println(hintStyle.Render(fmt.Sprintf("⎯ already on: %s", utils.ShortenSessionID(id))) + "\n")
	}
	previous := t.currentSessionID
	t.currentSessionID = id
	t.currentSessionName, _ = configBot.Get(id)
	t.inputHistory = loadInputHistory(id)
	t.inputHistoryIdx = -1
	t = t.restartTailer()

	t.tokens = 0
	t.lastIn = 0
	t.lastOut = 0
	t.lastCacheRead = 0
	t.lastCacheCreate = 0
	t.currentModel = ""
	t.activity = ""

	switchLines := []string{hintStyle.Render(fmt.Sprintf("⎯ switched to: %s", utils.ShortenSessionID(id)))}
	if previous != "" && previous != id {
		switchLines = append(switchLines, hintStyle.Render(fmt.Sprintf("  previous: %s", utils.ShortenSessionID(previous))))
	}
	switchBlock := tea.Println(strings.Join(switchLines, "\n") + "\n")

	return t, tea.Sequence(
		tea.ClearScreen,
		tea.Println(headerBlock(t.daemonStatus, t.httpStatus, t.discordStatus, t.telegramStatus, t.lineStatus)),
		switchBlock,
	)
}

func listSessions() []Session {
	dirs, err := go_pkg_filesystem_reader.ListDirs(filesystem.SessionsDir)
	if err != nil {
		return nil
	}

	results := make([]Session, 0, len(dirs))
	for _, dir := range dirs {
		sid := dir.Name
		if strings.HasPrefix(sid, "temp-") || strings.HasPrefix(sid, ".") {
			continue
		}
		refreshBotName(sid)
		name, _ := configBot.Get(sid)
		results = append(results, Session{
			id:   sid,
			name: name,
		})
	}
	sort.Slice(results, func(i, j int) bool {
		if a, b := sessionRank(results[i].id), sessionRank(results[j].id); a != b {
			return a < b
		}
		return results[i].id < results[j].id
	})
	return results
}

func sessionRank(sessionID string) int {
	switch sessionPrefix(sessionID) {
	case "cli-":
		return 0
	case "tg-":
		return 1
	case "dc-":
		return 2
	case "chat-":
		return 4
	case "temp-":
		return 5
	default:
		return 3
	}
}

func sessionPrefix(sessionID string) string {
	head, _, ok := strings.Cut(sessionID, "-")
	if !ok {
		return sessionID
	}
	return head + "-"
}

func sessionTabs(sessions []Session) []string {
	seen := map[string]bool{}
	prefixes := make([]string, 0, len(sessions))
	for _, e := range sessions {
		prefix := sessionPrefix(e.id)
		if seen[prefix] {
			continue
		}
		seen[prefix] = true
		prefixes = append(prefixes, prefix)
	}
	sort.Slice(prefixes, func(i, j int) bool {
		if a, b := sessionRank(prefixes[i]), sessionRank(prefixes[j]); a != b {
			return a < b
		}
		return prefixes[i] < prefixes[j]
	})
	if len(prefixes) < 2 {
		return nil
	}
	return append([]string{"all"}, prefixes...)
}

func popupSwitch(sid string) *Popup {
	sessions := listSessions()

	sort.SliceStable(sessions, func(i, j int) bool {
		if sessions[i].id == sid && sessions[j].id != sid {
			return true
		}
		if sessions[j].id == sid && sessions[i].id != sid {
			return false
		}
		return false
	})

	popup := &Popup{
		kind:       popupSingleSelect,
		title:      "Switch session",
		maxVisible: cmdSelectorMaxVisible,
		tabs:       sessionTabs(sessions),
	}
	popup.onTab = func(p *Popup) {
		fillSwitchOptions(p, sessions, sid)
	}
	popup.onTab(popup)
	return popup
}

func fillSwitchOptions(p *Popup, sessions []Session, sid string) {
	tab := ""
	if p.tabIdx > 0 && p.tabIdx < len(p.tabs) {
		tab = p.tabs[p.tabIdx]
	}

	list := make([]Session, 0, len(sessions))
	for _, e := range sessions {
		if tab == "" || strings.HasPrefix(e.id, tab) {
			list = append(list, e)
		}
	}

	shorts := make([]string, len(list))
	sidMax := 0
	for i, e := range list {
		shorts[i] = utils.ShortenSessionID(e.id)
		if n := len(shorts[i]); n > sidMax {
			sidMax = n
		}
	}

	names := make([]string, 0, len(list)+1)
	sids := make([]string, 0, len(list)+1)
	cursor := 0
	for i, e := range list {
		padded := shorts[i]
		if len(padded) < sidMax {
			padded += strings.Repeat(" ", sidMax-len(padded))
		}
		label := padded
		if e.name != "" && e.name != e.id {
			label += "  (" + e.name + ")"
		}
		if e.id == sid {
			label += " " + systemStyle.Render("[current]")
			cursor = i
		}
		names = append(names, label)
		sids = append(sids, e.id)
	}

	names = append(names, "(new session)")
	sids = append(sids, "")

	p.options = names
	p.values = sids
	p.cursor = cursor
}
