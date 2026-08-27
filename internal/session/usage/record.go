package usage

import (
	"context"
	"log/slog"
	"time"

	provider "github.com/pardnchiu/go-llm-router/core"
)

func Append(sessionID, providerName, model string, u provider.Usage) {
	if sessionID == "" || conn == nil {
		return
	}

	if _, err := conn.ExecContext(context.Background(), `
	INSERT INTO usage (session_id, send_at, model, input, output, write, hit)
	VALUES (?, ?, ?, ?, ?, ?, ?)`,
		sessionID, time.Now().UnixNano(), providerName+"@"+model,
		u.Input, u.Output, u.CacheCreate, u.CacheRead); err != nil {
		slog.Debug("usage.Append",
			slog.String("session", sessionID),
			slog.String("error", err.Error()))
	}
}
