package exec

import (
	"context"
	"fmt"

	agentsummary "github.com/pardnchiu/agenvoy/internal/agents/exec/summary"
	sessionmanager "github.com/pardnchiu/agenvoy/internal/session"
	sessionhistory "github.com/pardnchiu/agenvoy/internal/session/history"
	sessionsummary "github.com/pardnchiu/agenvoy/internal/session/summary"
)

func ForceSummary(ctx context.Context, sessionID string) (int, error) {
	if sessionID == "" {
		return 0, fmt.Errorf("session id is required")
	}

	_, histories := sessionhistory.Get(sessionID)
	if len(histories) == 0 {
		return 0, nil
	}

	if err := agentsummary.Generate(ctx, sessionID, histories); err != nil {
		return 0, fmt.Errorf("summary refresh failed: %w", err)
	}
	return len(histories), nil
}

func ResetSessionAll(sessionID string) (int, error) {
	if sessionID == "" {
		return 0, fmt.Errorf("session id is required")
	}
	ClearSteer(sessionID)
	return sessionmanager.ResetAll(sessionID)
}

func ResetSessionWithSummary(ctx context.Context, sessionID string) (int, error) {
	if sessionID == "" {
		return 0, fmt.Errorf("session id is required")
	}

	_, histories := sessionhistory.Get(sessionID)

	if len(histories) > 0 {
		if err := agentsummary.Generate(ctx, sessionID, histories); err != nil {
			return 0, fmt.Errorf("summary refresh failed; reset aborted to avoid context loss: %w", err)
		}
	}

	if raw, _ := sessionsummary.Get(sessionID); len(raw) == 0 {
		return 0, fmt.Errorf("this session has no summary to keep; reset aborted to avoid context loss — clear it with mode=all instead")
	}

	ClearSteer(sessionID)
	return sessionmanager.Reset(sessionID)
}
