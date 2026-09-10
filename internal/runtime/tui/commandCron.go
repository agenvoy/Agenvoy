package tui

import (
	"sort"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/pardnchiu/agenvoy/internal/agents"
	"github.com/pardnchiu/agenvoy/internal/runtime"
)

func listCronEntries() []runtime.CronEntry {
	crons, err := runtime.LoadCrons()
	if err != nil {
		return nil
	}
	sort.Slice(crons, func(i, j int) bool {
		if crons[i].Skill != crons[j].Skill {
			return crons[i].Skill < crons[j].Skill
		}
		return crons[i].Expression < crons[j].Expression
	})
	return crons
}

func (t TUI) dispatchAgent(content string) (TUI, tea.Cmd) {
	if content == "" {
		return t, nil
	}
	if len(agents.Registry().Entries) == 0 {
		return t, tea.Println(msgWarn("no model configured  /model global add") + "\n")
	}
	t = t.recordInputHistory(content)
	t.running = true
	t.runStartedAt = time.Now()
	t.runTarget = targetSession(content, t.currentSessionID)

	go runExec(t.ctx, content, false, t.cwd, t.currentSessionID, "", "")

	return t, tea.Batch(
		tea.Println(messageBlock(content)),
		t.spinner.Tick,
	)
}
