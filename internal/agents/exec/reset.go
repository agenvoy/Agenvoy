package exec

import (
	"context"
	"fmt"

	agentSummary "github.com/pardnchiu/agenvoy/internal/agents/exec/summary"
	sessionManager "github.com/pardnchiu/agenvoy/internal/session"
	sessionHistory "github.com/pardnchiu/agenvoy/internal/session/history"
	sessionSummary "github.com/pardnchiu/agenvoy/internal/session/summary"
)

func ForceSummary(ctx context.Context, sessionID string) (int, error) {
	if sessionID == "" {
		return 0, fmt.Errorf("session id is required")
	}

	_, histories := sessionHistory.Get(sessionID)
	if len(histories) == 0 {
		return 0, nil
	}

	if err := agentSummary.Generate(ctx, sessionID, histories); err != nil {
		return 0, fmt.Errorf("summary refresh failed: %w", err)
	}
	return len(histories), nil
}

func ResetSessionAll(sessionID string) (int, error) {
	if sessionID == "" {
		return 0, fmt.Errorf("session id is required")
	}
	ClearSteer(sessionID)
	return sessionManager.ResetAll(sessionID)
}

func ResetSessionWithSummary(ctx context.Context, sessionID string) (int, error) {
	if sessionID == "" {
		return 0, fmt.Errorf("session id is required")
	}

	_, histories := sessionHistory.Get(sessionID)

	if len(histories) > 0 {
		if err := agentSummary.Generate(ctx, sessionID, histories); err != nil {
			return 0, fmt.Errorf("summary refresh failed; reset aborted to avoid context loss: %w", err)
		}
	}

	if raw, _ := sessionSummary.Get(sessionID); len(raw) == 0 {
		return 0, fmt.Errorf("this session has no summary to keep; reset aborted to avoid context loss — clear it with mode=all instead")
	}

	ClearSteer(sessionID)
	return sessionManager.Reset(sessionID)
}
