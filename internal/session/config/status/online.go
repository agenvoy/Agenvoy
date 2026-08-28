package configStatus

import (
	"context"
	"log/slog"

	historyStore "github.com/pardnchiu/agenvoy/internal/runtime/history"
)

const (
	StatusOnline = "online"
	StatusIdle   = "idle"
)

type Status struct {
	State string `json:"state"`
	Count int    `json:"count"`
}

func Online(sessionID string) {
	if err := historyStore.EnterState(context.Background(), sessionID, StatusOnline); err != nil {
		slog.Debug("historyStore.EnterState",
			slog.String("session", sessionID),
			slog.String("error", err.Error()))
	}
}

func Idle(sessionID string) {
	if err := historyStore.LeaveState(context.Background(), sessionID, StatusOnline, StatusIdle); err != nil {
		slog.Debug("historyStore.LeaveState",
			slog.String("session", sessionID),
			slog.String("error", err.Error()))
	}
}
