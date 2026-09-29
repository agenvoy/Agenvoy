package usage

import (
	"context"
	"log/slog"
	"strings"
	"time"

	provider "github.com/pardnchiu/go-llm-router/core"

	"github.com/pardnchiu/agenvoy/internal/agents/claudeCode"
)

func Append(sessionID, providerName, model string, u provider.Usage, elapsed time.Duration, toolCalls []provider.ToolCall) {
	if sessionID == "" || conn == nil {
		return
	}

	input := u.Input
	if (providerName == "claude" || providerName == claudeCode.Provider) && input < u.CacheCreate {
		input += u.CacheCreate
	}

	if _, err := conn.ExecContext(context.Background(), `
	INSERT INTO usage (session_id, send_at, model, input, output, write, hit, elapsed_ms, tool_calls)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		sessionID, time.Now().UnixNano(), providerName+"@"+model,
		input, u.Output, u.CacheCreate, u.CacheRead, max(elapsed.Milliseconds(), 0),
		joinToolCallIDs(toolCalls)); err != nil {
		slog.Debug("usage.Append",
			slog.String("session", sessionID),
			slog.String("error", err.Error()))
	}
}

func joinToolCallIDs(list []provider.ToolCall) string {
	ids := make([]string, 0, len(list))
	for _, one := range list {
		if id := strings.TrimSpace(one.ID); id != "" {
			ids = append(ids, id)
		}
	}
	return strings.Join(ids, ",")
}
