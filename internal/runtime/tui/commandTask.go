package tui

import (
	"sort"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/pardnchiu/agenvoy/internal/runtime"
)

func (t TUI) commandTask() (TUI, tea.Cmd, bool) {
	return t.commandTaskRemove()
}

func listTaskEntries() []runtime.TaskEntry {
	tasks, err := runtime.LoadTasks()
	if err != nil {
		return nil
	}
	sort.Slice(tasks, func(i, j int) bool {
		if !tasks[i].At.Equal(tasks[j].At) {
			return tasks[i].At.Before(tasks[j].At)
		}
		return tasks[i].Skill < tasks[j].Skill
	})
	return tasks
}

func taskOptions(tasks []runtime.TaskEntry) (labels []string) {
	labels = make([]string, len(tasks))
	for i, t := range tasks {
		labels[i] = t.At.Local().Format("2006-01-02 15:04") + "  " + t.Skill
	}
	return labels
}
