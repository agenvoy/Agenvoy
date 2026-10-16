package exec

import (
	"context"

	agentsexecsummary "github.com/pardnchiu/agenvoy/internal/agents/exec/summary"
	sessionmanager "github.com/pardnchiu/agenvoy/internal/session"
	sessionhistory "github.com/pardnchiu/agenvoy/internal/session/history"
	sessionsummary "github.com/pardnchiu/agenvoy/internal/session/summary"
	utilsmsg "github.com/pardnchiu/agenvoy/internal/utils/msg"
)

func Reset(ctx context.Context, sessionID string, withSummary bool) (int, error) {
	if sessionID == "" {
		return 0, utilsmsg.New(utilsmsg.Invalid, "session id", "required")
	}

	if withSummary {
		_, histories := sessionhistory.Get(sessionID)

		if len(histories) > 0 {
			if err := agentsexecsummary.Generate(ctx, sessionID, histories); err != nil {
				return 0, utilsmsg.Trace("agentsexecsummary.Generate", "", err)
			}
		}

		if raw, _ := sessionsummary.Get(sessionID); len(raw) == 0 {
			return 0, utilsmsg.New(utilsmsg.NotFound, "session summary", "none to keep; reset aborted to avoid context loss, clear it with mode=all instead")
		}
	}
	ClearSteer(sessionID)
	return sessionmanager.Reset(sessionID, withSummary)
}

func Summary(ctx context.Context, sessionID string) (int, error) {
	if sessionID == "" {
		return 0, utilsmsg.New(utilsmsg.Invalid, "session id", "required")
	}

	_, histories := sessionhistory.Get(sessionID)
	if len(histories) == 0 {
		return 0, nil
	}

	if err := agentsexecsummary.Generate(ctx, sessionID, histories); err != nil {
		return 0, utilsmsg.Trace("agentsexecsummary.Generate", "", err)
	}
	return len(histories), nil
}
