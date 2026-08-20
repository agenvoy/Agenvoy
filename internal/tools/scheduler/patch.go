package scheduler

import (
	"fmt"
	"strings"
	"time"

	"github.com/pardnchiu/agenvoy/internal/runtime"
)

func patchSchedule(target, when, skill string) (string, error) {
	switch target {
	case "task":
		at, err := parseTime(when)
		if err != nil {
			return "", err
		}
		if !at.After(time.Now()) {
			return "", fmt.Errorf("already gone: %s", at.Local().Format("2006-01-02 15:04:05"))
		}
		patched, err := runtime.PatchTask(skill, at.UTC())
		if err != nil {
			return "", err
		}
		if patched == 0 {
			return fmt.Sprintf("no task found for skill %q", skill), nil
		}
		return fmt.Sprintf("patched %d task(s) for skill %q; new fire time: %s",
			patched, skill, at.Local().Format("2006-01-02 15:04:05")), nil

	case "cron":
		expression := strings.TrimSpace(when)
		if len(strings.Fields(expression)) != 5 {
			return "", fmt.Errorf("expression must be 5 fields '{min} {hour} {dom} {mon} {dow}' (got %q)", expression)
		}
		patched, err := runtime.PatchCron(skill, expression)
		if err != nil {
			return "", err
		}
		if patched == 0 {
			return fmt.Sprintf("no cron found for skill %q", skill), nil
		}
		return fmt.Sprintf("patched %d cron(s) for skill %q; new expression: %s",
			patched, skill, expression), nil
	}
	return "", fmt.Errorf("target must be 'task' or 'cron' (got %q)", target)
}
