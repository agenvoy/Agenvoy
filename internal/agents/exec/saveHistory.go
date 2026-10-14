package exec

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	go_llm_router "github.com/pardnchiu/go-llm-router/core"

	"github.com/pardnchiu/agenvoy/configs"
	agenttypes "github.com/pardnchiu/agenvoy/internal/agents/types"
	"github.com/pardnchiu/agenvoy/internal/runtime/torii"
	sessionhistory "github.com/pardnchiu/agenvoy/internal/session/history"
)

func saveNewHistory(ctx context.Context, choice go_llm_router.OutputChoices, session *agenttypes.Session) error {
	session.RawHistories = append(session.RawHistories, choice.Message)

	if session.Stateless {
		return nil
	}

	now := time.Now().UnixNano()
	delta := make([]sessionhistory.Record, 0, len(session.RawHistories))
	for _, message := range session.RawHistories {
		if message.Role == "system" ||
			message.Role == "tool" ||
			(message.Role == "assistant" && len(message.ToolCalls) > 0) {
			continue
		}
		if content, ok := message.Content.(string); ok && strings.Contains(content, configs.BAN_TAG) {
			continue
		}

		record := sessionhistory.Record{Role: message.Role, Content: message.Content, SendAt: now}
		if content, ok := message.Content.(string); ok {
			record.Content = configs.MESSAGE_PREFIX_REGEX.ReplaceAllString(content, "")
		}
		if message.Role == "user" {
			record.Sender = session.Sender
			if session.SendAt > 0 {
				record.SendAt = session.SendAt
			}
		}
		delta = append(delta, record)
	}

	if err := sessionhistory.Append(session.ID, delta); err != nil {
		return fmt.Errorf("sessionhistory.Append: %w", err)
	}

	writeSessionHistEntry(ctx, session.ID, choice.Message)
	return nil
}

func SaveUserInputHistory(ctx context.Context, sessionID, userText string) {
	if sessionID == "" || strings.TrimSpace(userText) == "" {
		return
	}
	writeSessionHistEntry(ctx, sessionID, go_llm_router.Message{
		Role:    "user",
		Content: userText,
	})
}

func writeSessionHistEntry(ctx context.Context, sessionID string, msg go_llm_router.Message) {
	if !torii.HasEmbedder() {
		return
	}
	if content, ok := msg.Content.(string); ok {
		msg.Content = configs.MESSAGE_PREFIX_REGEX.ReplaceAllString(content, "")
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
