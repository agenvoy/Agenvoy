package scheduler

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pardnchiu/agenvoy/internal/runtime"
	toolTypes "github.com/pardnchiu/agenvoy/internal/tools/types"
)

func listSchedule(e *toolTypes.Executor, target string) (string, error) {
	sid := ""
	if e != nil {
		sid = strings.TrimSpace(e.SessionID)
	}

	type result struct {
		Tasks []runtime.TaskEntry `json:"tasks,omitempty"`
		Crons []runtime.CronEntry `json:"crons,omitempty"`
	}
	var r result

	if target == "task" || target == "all" {
		tasks, err := runtime.LoadTasks()
		if err != nil {
			return "", fmt.Errorf("LoadTasks: %w", err)
		}
		for _, t := range tasks {
			if strings.TrimSpace(t.SessionID) == sid {
				r.Tasks = append(r.Tasks, t)
			}
		}
	}

	if target == "cron" || target == "all" {
		crons, err := runtime.LoadCrons()
		if err != nil {
			return "", fmt.Errorf("LoadCrons: %w", err)
		}
		for _, c := range crons {
			if strings.TrimSpace(c.SessionID) == sid {
				r.Crons = append(r.Crons, c)
			}
		}
	}

	if len(r.Tasks) == 0 && len(r.Crons) == 0 {
		return "{}", nil
	}

	raw, err := json.Marshal(r)
	if err != nil {
		return "", fmt.Errorf("json.Marshal: %w", err)
	}
	return string(raw), nil
}
