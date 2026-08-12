package handler

import (
	"context"
	"log/slog"

	"github.com/pardnchiu/agenvoy/internal/agents/exec/followup"
	agentTypes "github.com/pardnchiu/agenvoy/internal/agents/types"
	configBot "github.com/pardnchiu/agenvoy/internal/session/config/bot"
	sessionHistory "github.com/pardnchiu/agenvoy/internal/session/history"
)

func withFollowup(ctx context.Context, sessionID string, dst chan agentTypes.Event) chan agentTypes.Event {
	src := make(chan agentTypes.Event, cap(dst))

	go func() {
		defer close(dst)

		failed := false
		for event := range src {
			switch event.Type {
			case agentTypes.EventError, agentTypes.EventExecError, agentTypes.EventCanceled:
				failed = true

			case agentTypes.EventDone:
				if !failed && event.Source == "" {
					if suggest, ok := generateFollowup(ctx, sessionID); ok {
						dst <- suggest
					}
				}
			}
			dst <- event
		}
	}()

	return src
}

func generateFollowup(ctx context.Context, sessionID string) (agentTypes.Event, bool) {
	_, histories := sessionHistory.Get(sessionID)
	if len(histories) == 0 {
		return agentTypes.Event{}, false
	}

	needTitle := configBot.NeedTitle(sessionID)
	result := followup.Generate(ctx, sessionID, histories, needTitle)
	if result.Empty() {
		return agentTypes.Event{}, false
	}

	if result.Title != "" {
		if err := configBot.SetTitle(sessionID, result.Title); err != nil {
			slog.Warn("configBot.SetTitle",
				slog.String("session", sessionID),
				slog.String("error", err.Error()))
		}
	}

	return agentTypes.Event{
		Type:     agentTypes.EventSuggest,
		Text:     result.Title,
		Suggests: result.Suggests,
	}, true
}
