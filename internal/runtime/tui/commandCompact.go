package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/pardnchiu/agenvoy/internal/agents/exec/compact"
	"github.com/pardnchiu/agenvoy/internal/utils"
)

const (
	compactPrefix = "compact:"
	resetPrefix   = "reset:"
)

type CompactConfirm struct {
	id  string
	yes bool
}

type CompactDone struct {
	id      string
	removed int
	err     error
}

func (t TUI) commandCompactReset(tab int) (TUI, tea.Cmd, bool) {
	sid := strings.TrimSpace(t.currentSessionID)
	if sid == "" {
		return t, notice(msgLog("no active session") + "\n"), true
	}

	label := utils.ShortenSessionID(sid)
	popup := &Popup{
		kind:  popupSingleSelect,
		title: "/compact",
		tabs:  []string{"compact", "reset"},
		onConfirm: func(chosen string) any {
			if mode, ok := strings.CutPrefix(chosen, resetPrefix); ok {
				return ResetSessionConfirm1{id: sid, mode: mode}
			}
			return CompactConfirm{id: sid, yes: chosen == compactPrefix+"yes"}
		},
	}
	popup.onTab = func(p *Popup) tea.Cmd {
		p.cursor = 0
		if p.tabIdx == 1 {
			p.subtitle = fmt.Sprintf("reset history for %s  summary: regenerate then keep  all: also wipe the summary", label)
			p.options = []string{"No", "Yes  summary first, keep it", "Yes  reset all (summary too)"}
			p.values = []string{resetPrefix + "no", resetPrefix + "summary", resetPrefix + "all"}
			return nil
		}
		p.subtitle = fmt.Sprintf("compact history for %s  redundant and meaningless exchanges are removed by LLM analysis", label)
		p.options = []string{"No", "Yes"}
		p.values = []string{compactPrefix + "no", compactPrefix + "yes"}
		return nil
	}
	popup.tabIdx = tab
	popup.onTab(popup)
	t.popup = popup
	return t, nil, true
}

func (t TUI) runCompact(sid string) (TUI, tea.Cmd) {
	t.running = true
	t.runStartedAt = time.Now()
	t.runTarget = utils.ShortenSessionID(sid)
	t.activity = "compacting history..."

	return t, tea.Batch(
		notice(msgLog(fmt.Sprintf("compacting history for %s...", utils.ShortenSessionID(sid)))+"\n"),
		t.spinner.Tick,
		func() tea.Msg {
			ctx := context.Background()
			removed, err := compact.SessionHistory(ctx, sid)
			return CompactDone{id: sid, removed: removed, err: err}
		},
	)
}

func (t TUI) finishCompact(msg CompactDone) (TUI, tea.Cmd) {
	t.running = false
	t.activity = ""
	t.runTarget = ""

	if msg.err != nil {
		return t, notice(msgError(fmt.Sprintf("compact failed: %v", msg.err)) + "\n")
	}

	t.tokens = 0
	t.lastIn = 0
	t.lastContext = 0
	t.lastOut = 0
	t.lastCacheRead = 0
	t.lastCacheCreate = 0

	hint := fmt.Sprintf("compact: %s (nothing to remove)", utils.ShortenSessionID(msg.id))
	if msg.removed > 0 {
		hint = fmt.Sprintf("compact: %s (%d messages removed)", utils.ShortenSessionID(msg.id), msg.removed)
	}

	seq := []tea.Cmd{
		tea.ClearScreen,
		tea.Println(headerBlock(t.daemonStatus, t.httpStatus, t.discordStatus, t.telegramStatus, t.lineStatus, t.currentSessionID)),
	}
	tail := loadSessionTail(msg.id, t.width, false)
	if len(tail) == 0 {
		seq = append(seq, notice(msgLog("no history yet")+"\n"))
	} else {
		seq = append(seq, tail...)
	}
	seq = append(seq, notice(msgLog(hint)+"\n"))
	return t, tea.Sequence(seq...)
}
