package tool

import (
	"context"
	"log/slog"
	"strings"
	"time"

	agenttypes "github.com/pardnchiu/agenvoy/internal/agents/types"
	historystore "github.com/pardnchiu/agenvoy/internal/runtime/store"
	sessionhistory "github.com/pardnchiu/agenvoy/internal/session/history"
	sessionlog "github.com/pardnchiu/agenvoy/internal/session/log"
)

const outboundModel = "forwarded"

func resolveChatbotSession(prefix, field, id string) string {
	ctx := context.Background()
	if field == "chat_id" {
		return matchPrefix(historystore.FindSessionByChat(ctx, id), prefix)
	}
	return matchPrefix(historystore.FindSessionByChannel(ctx, id), prefix)
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

	if err := sessionhistory.Append(sessionID, []sessionhistory.Record{{
		Role:    "assistant",
		Content: message,
		SendAt:  time.Now().UnixNano(),
	}}); err != nil {
		slog.Debug("sessionhistory.Append (push)",
			slog.String("session", sessionID),
			slog.String("error", err.Error()))
	}

	sessionlog.Record(sessionID, agenttypes.Event{Type: agenttypes.EventText, Text: message})
	sessionlog.Record(sessionID, agenttypes.Event{Type: agenttypes.EventTextDone})
	sessionlog.Record(sessionID, agenttypes.Event{Type: agenttypes.EventDone, Model: outboundModel})
}
