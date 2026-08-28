package configStatus

import (
	"context"
	"log/slog"

	historyStore "github.com/pardnchiu/agenvoy/internal/runtime/history"
)

func Get(sessionID string) Status {
	if sessionID == "" {
		return Status{}
	}

	row, ok, err := historyStore.ReadState(context.Background(), sessionID)
	if err != nil {
		slog.Debug("historyStore.ReadState",
			slog.String("session", sessionID),
			slog.String("error", err.Error()))
	}
	if !ok {
		return Status{State: StatusIdle}
	}
	return FromRow(row)
}

func FromRow(row historyStore.StateRow) Status {
	status := Status{State: row.State, Count: row.InAction}
	if status.Count < 0 {
		status.Count = 0
	}
	if status.State == "" {
		status.State = StatusIdle
		if status.Count > 0 {
			status.State = StatusOnline
		}
	}
	return status
}

func Reset() {
	if err := historyStore.ResetState(context.Background(), StatusIdle); err != nil {
		slog.Warn("historyStore.ResetState",
			slog.String("error", err.Error()))
	}
}
