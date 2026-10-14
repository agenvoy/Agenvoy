package fileHistory

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/pardnchiu/agenvoy/configs"
	historystore "github.com/pardnchiu/agenvoy/internal/runtime/store"
	actionhistory "github.com/pardnchiu/agenvoy/internal/tools/history/action"
	tooltypes "github.com/pardnchiu/agenvoy/internal/tools/types"
	"github.com/pardnchiu/agenvoy/internal/utils"
)

func list(ctx context.Context, e *tooltypes.Executor, rawPath, taskID, from, to string, limit int) (string, error) {
	path, err := absPath(e, rawPath)
	if err != nil {
		return "", err
	}

	filter := historystore.Filter{Path: path, TaskID: strings.TrimSpace(taskID)}
	if filter.TaskID == "current" {
		filter.TaskID = e.PendingTask
	}
	if filter.From, err = parseTime(from); err != nil {
		return "", err
	}
	if filter.To, err = parseTime(to); err != nil {
		return "", err
	}

	filter.Limit = min(limit, maxHistoryRows)
	if filter.Limit <= 0 {
		filter.Limit = defaultRows
	}

	list, err := historystore.List(ctx, filter)
	if err != nil {
		return "", fmt.Errorf("internal/runtime/store: List: %w", err)
	}
	if len(list) == 0 {
		return "no recorded changes", nil
	}

	out := make([]map[string]any, 0, len(list))
	for _, row := range list {
		item := map[string]any{
			"version":    row.ID,
			"path":       filepath.Join(row.Dir, row.Name),
			"action":     row.Action,
			"size":       row.Size,
			"tool":       row.Tool,
			"session_id": row.SessionID,
			"task_id":    row.TaskID,
			"objective":  actionhistory.Objective(row.SessionID, row.TaskID),
			"changed_at": time.Unix(0, row.ChangedAt).Format(configs.TIME_LAYOUT),
		}
		if reason := row.RestoreBlock(); reason != "" {
			item["restorable"] = false
			item["reason"] = reason
		}
		out = append(out, item)
	}

	raw, err := utils.MarshalPlain(out)
	if err != nil {
		return "", fmt.Errorf("encoding/json: Marshal: %w", err)
	}
	return string(raw), nil
}
