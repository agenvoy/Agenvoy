package pubsub

import (
	"context"
	"sync"

	agentTypes "github.com/pardnchiu/agenvoy/internal/agents/types"
)

var (
	mu   sync.RWMutex
	subs = map[string][]*Subscriber{}

	forwardMu sync.RWMutex
	forward   func(string, agentTypes.Event)
)

func Sub(sessionID string, length int) *Subscriber {
	if length <= 0 {
		length = 32
	}

	sub := &Subscriber{
		channel:   make(chan agentTypes.Event, length),
		sessionID: sessionID,
	}

	mu.Lock()
	subs[sessionID] = append(subs[sessionID], sub)
	mu.Unlock()
	return sub
}

func SetForwarder(fn func(string, agentTypes.Event)) {
	forwardMu.Lock()
	forward = fn
	forwardMu.Unlock()
}

func Pub(sessionID string, event agentTypes.Event) {
	if sessionID == "" {
		return
	}

	mu.RLock()
	list := subs[sessionID]
	mu.RUnlock()

	for _, s := range list {
		s.send(event)
	}

	forwardMu.RLock()
	fn := forward
	forwardMu.RUnlock()
	if fn != nil {
		fn(sessionID, event)
	}
}

func Wrap(ctx context.Context, sessionID string, channel chan agentTypes.Event, length int) chan agentTypes.Event {
	if sessionID == "" {
		return channel
	}

	if length <= 0 {
		length = cap(channel)
		if length <= 0 {
			length = 32
		}
	}

	src := make(chan agentTypes.Event, length)
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
