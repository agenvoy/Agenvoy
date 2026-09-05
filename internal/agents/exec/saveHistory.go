package exec

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/pardnchiu/agenvoy/configs"
	agentTypes "github.com/pardnchiu/agenvoy/internal/agents/types"
	"github.com/pardnchiu/agenvoy/internal/runtime/torii"
	sessionHistory "github.com/pardnchiu/agenvoy/internal/session/history"
	provider "github.com/pardnchiu/go-llm-router/core"
)

func saveNewHistory(ctx context.Context, choice provider.OutputChoices, session *agentTypes.AgentSession) error {
	session.Histories = append(session.Histories, choice.Message)

	if session.Stateless {
		return nil
	}

	base := min(max(session.BaseLen, 0), len(session.Histories))
	now := time.Now().UnixNano()
	delta := make([]sessionHistory.Record, 0, len(session.Histories)-base)
	for _, message := range session.Histories[base:] {
		if message.Role == "system" ||
			message.Role == "tool" ||
			(message.Role == "assistant" && len(message.ToolCalls) > 0) {
			continue
		}
		if content, ok := message.Content.(string); ok && (strings.Contains(content, configs.PoisonRefusal) || strings.Contains(content, configs.GuardrailSentinel)) {
			continue
		}

		record := sessionHistory.Record{Role: message.Role, Content: message.Content, SendAt: now}
		if content, ok := message.Content.(string); ok {
			record.Content = sessionHistory.StripPrefix(content)
		}
		if message.Role == "user" {
			record.Sender = session.Sender
			if session.UserSendAt > 0 {
				record.SendAt = session.UserSendAt
			}
		}
		delta = append(delta, record)
	}

	if err := sessionHistory.Append(session.ID, delta); err != nil {
		return fmt.Errorf("sessionHistory.Append: %w", err)
	}

	writeSessionHistEntry(ctx, session.ID, choice.Message)
	return nil
}

func SaveUserInputHistory(ctx context.Context, sessionID, userText string) {
	if sessionID == "" || strings.TrimSpace(userText) == "" {
		return
	}
	writeSessionHistEntry(ctx, sessionID, provider.Message{
		Role:    "user",
		Content: userText,
	})
}

func writeSessionHistEntry(ctx context.Context, sessionID string, msg provider.Message) {
	if !torii.HasEmbedder() {
		return
	}
	if content, ok := msg.Content.(string); ok {
		msg.Content = sessionHistory.StripPrefix(content)
	}
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return
	}
	key := fmt.Sprintf("%s:%d", sessionID, time.Now().UnixNano())
	db := torii.DB(torii.DBSessionHist)
	value := string(msgBytes)

	if setErr := db.SetVector(ctx, key, value, nil); setErr != nil {
		slog.Warn("store.DB.Set",
			slog.String("session", sessionID),
			slog.String("error", setErr.Error()))
	}
}
