package subagent

import (
	"context"
	"strings"

	agenttypes "github.com/pardnchiu/agenvoy/internal/agents/types"
	"github.com/pardnchiu/agenvoy/internal/runtime/pubsub"
	provider "github.com/pardnchiu/go-llm-router/core"
)

func waitResume(ctx context.Context, sessionID string, parent chan<- agenttypes.Event, displayName string) (string, provider.Usage) {
	var usage provider.Usage
	sub := pubsub.Sub(sessionID, 128)
	defer sub.Close()

	var sb strings.Builder
	for {
		select {
		case <-ctx.Done():
			return "", usage

		case ev, ok := <-sub.Events():
			if !ok {
				return "", usage
			}
			sendEvent(parent, displayName, ev)

			switch ev.Type {
			case agenttypes.EventText:
				if ev.Text == "" {
					continue
				}
				if sb.Len() > 0 {
					sb.WriteByte('\n')
				}
				sb.WriteString(ev.Text)

			case agenttypes.EventDone:
				if ev.Usage != nil {
					usage.Input += ev.Usage.Input
					usage.Output += ev.Usage.Output
					usage.CacheCreate += ev.Usage.CacheCreate
					usage.CacheRead += ev.Usage.CacheRead
				}
				return strings.TrimSpace(sb.String()), usage

			case agenttypes.EventCanceled, agenttypes.EventError:
				return strings.TrimSpace(sb.String()), usage
			}
		}
	}
}
