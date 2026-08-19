package scheduler

import (
	"context"
	"fmt"

	"github.com/pardnchiu/agenvoy/internal/filesystem/skill"
	"github.com/pardnchiu/agenvoy/internal/runtime"
	historyStore "github.com/pardnchiu/agenvoy/internal/runtime/history"
	toolTypes "github.com/pardnchiu/agenvoy/internal/tools/types"
)

func removeSchedule(ctx context.Context, e *toolTypes.Executor, target, skillName string) (string, error) {
	var removed int
	var err error

	switch target {
	case "task":
		removed, err = runtime.RemoveTask(skillName)
	case "cron":
		removed, err = runtime.RemoveCron(skillName)
	default:
		return "", fmt.Errorf("target must be 'task' or 'cron' (got %q)", target)
	}
	if err != nil {
		return "", err
	}
	if removed == 0 {
		return fmt.Sprintf("no %s found for skill %q", target, skillName), nil
	}
	if err := skill.TrashSchedule(ctx, skillName, historyStore.Meta{SessionID: e.SessionID, TaskID: e.PendingTask}); err != nil {
		return "", fmt.Errorf("TrashScheduleSkill: %w", err)
	}
	return fmt.Sprintf("removed %d %s(s) for skill %q and moved skill to .Trash", removed, target, skillName), nil
}
