package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/pardnchiu/agenvoy/internal/agents/exec"
	"github.com/pardnchiu/agenvoy/internal/utils"
)

type ResetSessionConfirm1 struct {
	id   string
	mode string
}

type ResetSessionConfirm2 struct {
	id   string
	mode string
	yes  bool
}

func (t TUI) commandReset() (TUI, tea.Cmd, bool) {
	sid := strings.TrimSpace(t.currentSessionID)
	if sid == "" {
		return t, tea.Println(hintStyle.Render("no active session") + "\n"), true
	}

	label := utils.ShortenSessionID(sid)
	t.popup = &Popup{
		kind:     popupSingleSelect,
		title:    fmt.Sprintf("Reset history for %s ?", label),
		subtitle: "summary: regenerate then keep · all: also wipe the summary",
		options:  []string{"No", "Yes  summary first, keep it", "Yes  reset all (summary too)"},
		values:   []string{"no", "summary", "all"},
		cursor:   0,
		onConfirm: func(chosen string) any {
			return ResetSessionConfirm1{id: sid, mode: chosen}
		},
	}
	return t, nil, true
}

func (t TUI) openResetConfirm2(sid, mode string) (TUI, tea.Cmd) {
	title := "Are you sure? Raw history will be permanently dropped."
	subtitle := fmt.Sprintf("%s — summary refresh runs first; abort if refresh fails.", utils.ShortenSessionID(sid))
	if mode == "all" {
		title = "Are you sure? History AND summary will be permanently dropped."
		subtitle = fmt.Sprintf("%s — no refresh; this session's memory is fully wiped.", utils.ShortenSessionID(sid))
	}

	t.popup = &Popup{
		kind:     popupSingleSelect,
		title:    title,
		subtitle: subtitle,
		options:  []string{"No", "Yes  reset it"},
		values:   []string{"no", "yes"},
		cursor:   0,
		onConfirm: func(chosen string) any {
			return ResetSessionConfirm2{id: sid, mode: mode, yes: chosen == "yes"}
		},
	}
	return t, nil
}

type ResetSessionDone struct {
	id   string
	mode string
	keys int
	err  error
}

func (t TUI) runResetSession(sid, mode string) (TUI, tea.Cmd) {
	t.running = true
	t.runStartedAt = time.Now()
	t.runTarget = utils.ShortenSessionID(sid)

	label := utils.ShortenSessionID(sid)
	if mode == "all" {
		t.activity = "resetting (history + summary)…"
		return t, tea.Batch(
			tea.Println(hintStyle.Render(fmt.Sprintf("⎯ clearing history and summary for %s…", label))+"\n"),
			t.spinner.Tick,
			func() tea.Msg {
				keys, err := exec.ResetSessionAll(sid)
				return ResetSessionDone{id: sid, mode: mode, keys: keys, err: err}
			},
		)
	}

	t.activity = "resetting (summary refresh first)…"
	return t, tea.Batch(
		tea.Println(hintStyle.Render(fmt.Sprintf("⎯ refreshing summary for %s, then clearing history…", label))+"\n"),
		t.spinner.Tick,
		func() tea.Msg {
			ctx := context.Background()
			keys, err := exec.ResetSessionWithSummary(ctx, sid)
			return ResetSessionDone{id: sid, mode: mode, keys: keys, err: err}
		},
	)
}

func (t TUI) finishResetSession(msg ResetSessionDone) (TUI, tea.Cmd) {
	t.running = false
	t.activity = ""
	t.runTarget = ""

	if msg.err != nil {
		return t, tea.Println(errorStyle.Render(fmt.Sprintf("[!] reset failed: %v", msg.err)) + "\n")
	}

	t.tokens = 0
	t.lastIn = 0
	t.lastOut = 0
	t.lastCacheRead = 0

	summaryNote := "summary kept"
	if msg.mode == "all" {
		summaryNote = "summary cleared"
	}

	seq := []tea.Cmd{
		tea.ClearScreen,
		tea.Println(headerBlock(t.daemonStatus, t.httpStatus, t.discordStatus, t.telegramStatus, t.lineStatus)),
		tea.Println(hintStyle.Render(fmt.Sprintf("⎯ reset: %s (%s, %d torii keys purged)", utils.ShortenSessionID(msg.id), summaryNote, msg.keys)) + "\n"),
	}
	return t, tea.Sequence(seq...)
}
