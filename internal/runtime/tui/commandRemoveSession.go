package tui

import (
	"fmt"
	"os"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/pardnchiu/agenvoy/internal/agents/exec"
	"github.com/pardnchiu/agenvoy/internal/filesystem"
	historyStore "github.com/pardnchiu/agenvoy/internal/runtime/history"
	"github.com/pardnchiu/agenvoy/internal/runtime/torii"
	"github.com/pardnchiu/agenvoy/internal/session"
	configBot "github.com/pardnchiu/agenvoy/internal/session/config/bot"
	sessionHistory "github.com/pardnchiu/agenvoy/internal/session/history"
	"github.com/pardnchiu/agenvoy/internal/utils"
)

type RemoveSessionPick struct{ chosen string }

type RemoveSessionConfirm struct {
	ids []string
	yes bool
}

func (t TUI) commandRemoveSession() (TUI, tea.Cmd, bool) {
	sessions := listSessions()
	if len(sessions) == 0 {
		return t, tea.Println(hintStyle.Render("no sessions") + "\n"), true
	}

	currentSID := strings.TrimSpace(t.currentSessionID)

	sort.SliceStable(sessions, func(i, j int) bool {
		if sessions[i].id == currentSID {
			return true
		}
		if sessions[j].id == currentSID {
			return false
		}
		return false
	})

	popup := &Popup{
		kind:       popupMultiSelect,
		title:      "Select sessions to remove",
		maxVisible: cmdSelectorMaxVisible,
		tabs:       sessionTabs(sessions),
		multi:      make(map[int]bool),
	}

	picked := map[string]bool{}
	harvest := func(p *Popup) {
		for i, on := range p.multi {
			if i >= len(p.values) {
				continue
			}
			if on {
				picked[p.values[i]] = true
				continue
			}
			delete(picked, p.values[i])
		}
	}

	popup.onTab = func(p *Popup) {
		harvest(p)
		fillRemoveOptions(p, sessions, currentSID)
		p.multi = make(map[int]bool, len(p.values))
		for i, id := range p.values {
			if picked[id] {
				p.multi[i] = true
			}
		}
	}
	popup.onTab(popup)

	popup.onConfirm = func(string) any {
		harvest(popup)
		ids := make([]string, 0, len(picked))
		for _, s := range sessions {
			if picked[s.id] {
				ids = append(ids, s.id)
			}
		}
		return RemoveSessionPick{chosen: strings.Join(ids, "\x1F")}
	}

	t.popup = popup
	return t, nil, true
}

func fillRemoveOptions(p *Popup, sessions []Session, currentSID string) {
	tab := ""
	if p.tabIdx > 0 && p.tabIdx < len(p.tabs) {
		tab = p.tabs[p.tabIdx]
	}

	options := make([]string, 0, len(sessions))
	values := make([]string, 0, len(sessions))
	for _, s := range sessions {
		if tab != "" && !strings.HasPrefix(s.id, tab) {
			continue
		}
		short := utils.ShortenSessionID(s.id)
		label := short
		if s.name != "" && s.name != s.id {
			label = fmt.Sprintf("%s (%s)", s.name, short)
		}
		if s.id == currentSID {
			label += " · (current)"
		}
		options = append(options, label)
		values = append(values, s.id)
	}

	p.options = options
	p.values = values
	p.cursor = 0
}

func (t TUI) runRemoveSessionPick(chosen string) (TUI, tea.Cmd) {
	if chosen == "" {
		return t, tea.Println(hintStyle.Render("⎯ no sessions selected") + "\n")
	}

	var ids []string
	for _, entry := range strings.Split(chosen, "\x1F") {
		id := strings.TrimSpace(strings.SplitN(entry, "\x00", 2)[0])
		if id != "" {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return t, tea.Println(hintStyle.Render("⎯ no sessions selected") + "\n")
	}

	labels := make([]string, len(ids))
	for i, id := range ids {
		labels[i] = utils.ShortenSessionID(id)
	}

	t.popup = &Popup{
		kind:     popupSingleSelect,
		title:    fmt.Sprintf("Remove %d session(s)? This cannot be undone.", len(ids)),
		subtitle: strings.Join(labels, ", "),
		options:  []string{"No", "Yes  delete them"},
		values:   []string{"no", "yes"},
		cursor:   0,
		onConfirm: func(chosen string) any {
			return RemoveSessionConfirm{ids: ids, yes: chosen == "yes"}
		},
	}
	return t, nil
}

func (t TUI) runRemoveSessionConfirm(msg RemoveSessionConfirm) (TUI, tea.Cmd) {
	if !msg.yes {
		return t, tea.Println(hintStyle.Render("⎯ cancelled") + "\n")
	}

	removedCurrent := false
	var removed []string
	for _, sid := range msg.ids {
		deleteSessionHistKeys(sid)
		historyStore.Clear(sid)
		sessionHistory.ClearMutex(sid)
		exec.ClearSteer(sid)
		if err := os.RemoveAll(filesystem.SessionDir(sid)); err != nil {
			continue
		}
		removed = append(removed, utils.ShortenSessionID(sid))
		if sid == t.currentSessionID {
			removedCurrent = true
		}
	}

	if len(removed) == 0 {
		return t, tea.Println(errorStyle.Render("[!] failed to remove sessions") + "\n")
	}

	if removedCurrent {
		fallback := pickAlternateSession(msg.ids...)
		if fallback == "" {
			created, err := session.New("cli-")
			if err != nil {
				return t, tea.Println(errorStyle.Render(fmt.Sprintf("[!] create fallback session: %v", err)) + "\n")
			}
			fallback = created
		}
		t.currentSessionID = fallback
		t.currentSessionName, _ = configBot.Get(fallback)
		t = t.restartTailer()
		t.tokens = 0
		t.lastIn = 0
		t.lastOut = 0
		t.lastCacheRead = 0
		t.lastCacheCreate = 0
		t.currentModel = ""
		t.activity = ""
	}

	return t, tea.Sequence(
		tea.ClearScreen,
		tea.Println(headerBlock(t.daemonStatus, t.httpStatus, t.discordStatus, t.telegramStatus, t.lineStatus)),
		tea.Println(hintStyle.Render(fmt.Sprintf("⎯ removed: %s", strings.Join(removed, ", ")))+"\n"),
	)
}

func pickAlternateSession(exclude ...string) string {
	excluded := make(map[string]bool, len(exclude))
	for _, id := range exclude {
		excluded[id] = true
	}
	for _, s := range listSessions() {
		if excluded[s.id] {
			continue
		}
		return s.id
	}
	return ""
}

func deleteSessionHistKeys(sid string) int {
	db := torii.DB(torii.DBSessionHist)
	keys := db.Keys(sid + ":*")
	if len(keys) == 0 {
		return 0
	}
	return db.Del(keys...)
}
