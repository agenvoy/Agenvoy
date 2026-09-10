package tui

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/pardnchiu/agenvoy/internal/agents/exec"
	"github.com/pardnchiu/agenvoy/internal/filesystem"
	historyStore "github.com/pardnchiu/agenvoy/internal/runtime/store"
	"github.com/pardnchiu/agenvoy/internal/runtime/torii"
	"github.com/pardnchiu/agenvoy/internal/session"
	configBot "github.com/pardnchiu/agenvoy/internal/session/config/bot"
	sessionHistory "github.com/pardnchiu/agenvoy/internal/session/history"
	"github.com/pardnchiu/agenvoy/internal/utils"
)

type SessionDeletePick struct{ id string }

type RemoveSessionConfirm struct {
	ids []string
	yes bool
}

func (t TUI) openSessionDeleteConfirm(id string) (TUI, tea.Cmd) {
	label := utils.ShortenSessionID(id)
	if name, _ := configBot.Get(id); name != "" && name != id {
		label = name + " (" + label + ")"
	}

	t.popup = &Popup{
		kind:     popupSingleSelect,
		title:    "Remove " + label + " ?",
		subtitle: "history, task history and action.log are deleted with it",
		options:  []string{"No", "Yes"},
		values:   []string{"no", "yes"},
		onConfirm: func(chosen string) any {
			return RemoveSessionConfirm{ids: []string{id}, yes: chosen == "yes"}
		},
		onCancel: func() any {
			return RemoveSessionConfirm{ids: []string{id}}
		},
	}
	return t, nil
}

func (t TUI) runRemoveSessionConfirm(msg RemoveSessionConfirm) (TUI, tea.Cmd) {
	if !msg.yes {
		next, cmd, _ := t.commandSessions(nil)
		return next, cmd
	}

	removedCurrent := false
	var removed []string
	for _, sid := range msg.ids {
		deleteSessionHistKeys(sid)
		historyStore.Clear(sid)
		if err := historyStore.DeleteSession(context.Background(), sid); err != nil {
			slog.Warn("historyStore.DeleteSession",
				slog.String("session", sid),
				slog.String("error", err.Error()))
		}
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
		return t, tea.Println(msgError("failed to remove sessions") + "\n")
	}

	if removedCurrent {
		fallback := pickAlternateSession(msg.ids...)
		if fallback == "" {
			created, err := session.New("cli-")
			if err != nil {
				return t, tea.Println(msgError(fmt.Sprintf("create fallback session: %v", err)) + "\n")
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

	next, _, _ := t.commandSessions(nil)
	return next, tea.Sequence(
		tea.ClearScreen,
		tea.Println(headerBlock(t.daemonStatus, t.httpStatus, t.discordStatus, t.telegramStatus, t.lineStatus)),
		tea.Println(msgLog(fmt.Sprintf("removed: %s", strings.Join(removed, ", ")))+"\n"),
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
	keys := db.Keys(context.Background(), sid+":*")
	if len(keys) == 0 {
		return 0
	}
	return db.Del(context.Background(), keys...)
}
