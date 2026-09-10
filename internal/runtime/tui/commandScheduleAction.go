package tui

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/pardnchiu/agenvoy/internal/filesystem/skill"
	"github.com/pardnchiu/agenvoy/internal/runtime"
	historyStore "github.com/pardnchiu/agenvoy/internal/runtime/store"
)

type ScheduleTestPick struct {
	skill string
}

type ScheduleRemovePick struct {
	kind  string
	skill string
}

type ScheduleRemoveConfirm struct {
	kind  string
	skill string
	yes   bool
}

func (t TUI) reopenSchedule() (TUI, tea.Cmd) {
	next, cmd, _ := t.commandScheduleMenu(nil)
	return next, cmd
}

func (t TUI) runScheduleTest(skillName string) (TUI, tea.Cmd) {
	next, cmd, _ := t.commandSchedule([]string{"/sched-" + skillName})
	return next, cmd
}

func (t TUI) openScheduleRemoveConfirm(kind, skillName string) (TUI, tea.Cmd) {
	t.popup = &Popup{
		kind:     popupSingleSelect,
		title:    fmt.Sprintf("Delete %s %q ?", kind, skillName),
		subtitle: "the schedule entry is dropped and its skill directory moves to Trash",
		options:  []string{"No", "Yes"},
		values:   []string{"no", "yes"},
		cursor:   0,
		onConfirm: func(chosen string) any {
			return ScheduleRemoveConfirm{kind: kind, skill: skillName, yes: chosen == "yes"}
		},
		onCancel: func() any {
			return ScheduleRemoveConfirm{kind: kind, skill: skillName}
		},
	}
	return t, nil
}

func (t TUI) runScheduleRemove(kind, skillName string) (TUI, tea.Cmd) {
	var (
		removed int
		err     error
	)
	if kind == "cron" {
		removed, err = runtime.RemoveCron(skillName)
	} else {
		removed, err = runtime.RemoveTask(skillName)
	}
	if err != nil {
		return t, tea.Println(msgError(fmt.Sprintf("%s remove: %v", kind, err)) + "\n")
	}
	if removed == 0 {
		return t, tea.Println(msgLog(fmt.Sprintf("no %s found for %s", kind, skillName)) + "\n")
	}
	if err := skill.TrashSchedule(context.Background(), skillName, historyStore.Meta{SessionID: t.currentSessionID}); err != nil {
		return t, tea.Println(msgError(fmt.Sprintf("TrashSchedule: %v", err)) + "\n")
	}

	next, cmd := t.reopenSchedule()
	return next, tea.Sequence(
		tea.Println(msgLog(fmt.Sprintf("removed %s: %s  skill trashed", kind, skillName))+"\n"),
		cmd,
	)
}

func scheduleLabels(fields, names []string) (labels []string) {
	width := 0
	for _, one := range fields {
		width = max(width, len(one)+3)
	}
	labels = make([]string, len(fields))
	for i, one := range fields {
		labels[i] = padToWidth("["+one+"]", width) + names[i]
	}
	return labels
}
