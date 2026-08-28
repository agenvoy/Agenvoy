package tool

import (
	"context"
	"log/slog"
	"strings"
	"time"

	agentTypes "github.com/pardnchiu/agenvoy/internal/agents/types"
	historyStore "github.com/pardnchiu/agenvoy/internal/runtime/history"
	sessionHistory "github.com/pardnchiu/agenvoy/internal/session/history"
	sessionLog "github.com/pardnchiu/agenvoy/internal/session/log"
)

const outboundModel = "forwarded"

func resolveChatbotSession(prefix, field, id string) string {
	ctx := context.Background()
	if field == "chat_id" {
		return matchPrefix(historyStore.FindSessionByChat(ctx, id), prefix)
	}
	return matchPrefix(historyStore.FindSessionByChannel(ctx, id), prefix)
}

func matchPrefix(sessionID, prefix string) string {
	if !strings.HasPrefix(sessionID, prefix) {
		return ""
	}
	return sessionID
}

func recordOutbound(sessionID, message string) {
	if sessionID == "" || message == "" {
		return
	}

	if err := sessionHistory.Append(sessionID, []sessionHistory.Record{{
		Role:    "assistant",
		Content: message,
		SendAt:  time.Now().UnixNano(),
	}}); err != nil {
		slog.Debug("sessionHistory.Append (push)",
			slog.String("session", sessionID),
			slog.String("error", err.Error()))
	}

	sessionLog.Record(sessionID, agentTypes.Event{Type: agentTypes.EventText, Text: message})
	sessionLog.Record(sessionID, agentTypes.Event{Type: agentTypes.EventTextDone})
	sessionLog.Record(sessionID, agentTypes.Event{Type: agentTypes.EventDone, Model: outboundModel})
}
