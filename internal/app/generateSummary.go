package app

import (
	"context"
	"log/slog"
	"time"

	agentSummary "github.com/pardnchiu/agenvoy/internal/agents/exec/summary"
	"github.com/pardnchiu/agenvoy/internal/filesystem"
	sessionHistory "github.com/pardnchiu/agenvoy/internal/session/history"
	sessionSummary "github.com/pardnchiu/agenvoy/internal/session/summary"
)

func GenerateSummary() {
	sessions := sessionSummary.Pending()
	if len(sessions) == 0 {
		return
	}

	for _, sid := range sessions {
		_, histories := sessionHistory.Get(sid)
		if len(histories) == 0 {
			continue
		}
		bgCtx, cancel := context.WithTimeout(context.Background(), time.Duration(filesystem.AgentSendTimeoutSec)*time.Second)
		err := agentSummary.Generate(bgCtx, sid, histories)
		cancel()
		if err != nil {
			slog.Warn("agentSummary.Generate",
				slog.String("session", sid),
				slog.String("error", err.Error()))
		}
	}
}
