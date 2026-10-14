package pubsub

import (
	"context"
	"sync"

	agenttypes "github.com/pardnchiu/agenvoy/internal/agents/types"
	sessiontui "github.com/pardnchiu/agenvoy/internal/session/tui"
)

var (
	mu   sync.RWMutex
	subs = map[string][]*Subscriber{}
)

func Sub(sessionID string, length int) *Subscriber {
	if length <= 0 {
		length = 32
	}

	sub := &Subscriber{
		channel:   make(chan agenttypes.Event, length),
		sessionID: sessionID,
	}

	mu.Lock()
	subs[sessionID] = append(subs[sessionID], sub)
	mu.Unlock()
	return sub
}

func Pub(sessionID string, event agenttypes.Event) {
	if sessionID == "" {
		return
	}
	if event.WindowHash == "" {
		event.WindowHash = sessiontui.Get()
	}

	mu.RLock()
	list := subs[sessionID]
	mu.RUnlock()

	for _, s := range list {
		s.send(event)
	}
}

func Wrap(ctx context.Context, sessionID string, channel chan agenttypes.Event, length int) chan agenttypes.Event {
	if sessionID == "" {
		return channel
	}

	if length <= 0 {
		length = cap(channel)
		if length <= 0 {
			length = 32
		}
	}

	src := make(chan agenttypes.Event, length)
	go func() {
		defer close(channel)
		for event := range src {
			Pub(sessionID, event)

			select {
			case channel <- event:
			case <-ctx.Done():
				for next := range src {
					Pub(sessionID, next)
				}
				return
			}
		}
	}()
	return src
}
