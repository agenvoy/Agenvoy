package tui

import (
	"sort"

	"github.com/pardnchiu/agenvoy/internal/runtime"
)

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
