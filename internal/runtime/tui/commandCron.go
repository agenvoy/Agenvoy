package tui

import (
	"sort"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/pardnchiu/agenvoy/internal/agents"
	"github.com/pardnchiu/agenvoy/internal/runtime"
	"github.com/pardnchiu/agenvoy/internal/runtime/ipc"
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
	if t.connecting {
		return t, nil
	}
	if content == "" {
		return t, nil
	}
	if len(agents.Registry().Entries) == 0 {
		return t, notice(msgWarn("no model configured  /model global add") + "\n")
	}
	t = t.recordInputHistory(content)
	t.running = true
	t.runStartedAt = time.Now()
	t.currentModel = ""
	t.runTarget = ""

	go runExec(ipc.Frame{Type: ipc.FrameRun, SessionID: t.currentSessionID, Rayload: &ipc.Payload{Input: content, WorkDir: t.cwd}})

	return t, tea.Batch(
		tea.Println(messageBlock(content)),
		t.spinner.Tick,
	)
}
