package tui

import (
	"context"
	"time"

	agentTypes "github.com/pardnchiu/agenvoy/internal/agents/types"
	"github.com/pardnchiu/agenvoy/internal/runtime/daemon"
)

const publishTimeout = 3 * time.Second

func publishEventToDaemon(ctx context.Context, sessionID string, ev agentTypes.Event) {
	if sessionID == "" {
		return
	}
	sendCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), publishTimeout)
	defer cancel()

	daemon.Publish(sendCtx, "/v1/session/"+sessionID+"/event", ev)
}

func wrapEventsPublish(ctx context.Context, sessionID string, dst chan agentTypes.Event) chan agentTypes.Event {
	if sessionID == "" {
		return dst
	}
	src := make(chan agentTypes.Event, cap(dst))
	go func() {
		defer close(dst)
		for ev := range src {
			publishEventToDaemon(ctx, sessionID, ev)
			dst <- ev
		}
	}()
	return src
}
