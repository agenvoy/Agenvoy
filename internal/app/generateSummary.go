package app

import (
	"context"
	"log/slog"
	"time"

	"github.com/pardnchiu/agenvoy/configs"
	agentsummary "github.com/pardnchiu/agenvoy/internal/agents/exec/summary"
	sessionhistory "github.com/pardnchiu/agenvoy/internal/session/history"
	sessionsummary "github.com/pardnchiu/agenvoy/internal/session/summary"
)

func GenerateSummary() {
	sessions := sessionsummary.Pending()
	if len(sessions) == 0 {
		return
	}

	for _, sid := range sessions {
		_, histories := sessionhistory.Get(sid)
		if len(histories) == 0 {
			continue
		}
		bgCtx, cancel := context.WithTimeout(context.Background(), time.Duration(configs.AGENT_SEND_TIMEOUT_SEC)*time.Second)
		err := agentsummary.Generate(bgCtx, sid, histories)
		cancel()
		if err != nil {
			slog.Warn("agentsummary.Generate",
				slog.String("session", sid),
				slog.String("error", err.Error()))
		}
	}
}
