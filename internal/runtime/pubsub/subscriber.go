package pubsub

import (
	"log/slog"
	"sync"
	"sync/atomic"

	agentTypes "github.com/pardnchiu/agenvoy/internal/agents/types"
)

type Subscriber struct {
	channel   chan agentTypes.Event
	sessionID string
	closed    bool
	dropped   atomic.Int64
	mu        sync.Mutex
}

func (s *Subscriber) Events() <-chan agentTypes.Event {
	return s.channel
}

func (s *Subscriber) TakeDropped() int64 {
	return s.dropped.Swap(0)
}

func (s *Subscriber) Close() {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}

	s.closed = true
	close(s.channel)
	s.mu.Unlock()

	s.unsub()
}

func (s *Subscriber) send(event agentTypes.Event) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return
	}

	select {
	case s.channel <- event:
	default:
		s.countDrop(event)
	}
}

func (s *Subscriber) countDrop(event agentTypes.Event) {
	n := s.dropped.Add(1)
	slog.Debug("pubsub subscriber overflow, event dropped",
		slog.String("session", s.sessionID),
		slog.String("event", event.Type.String()),
		slog.Int("buffer", cap(s.channel)),
		slog.Int64("dropped_total", n))
}

func (s *Subscriber) unsub() {
	mu.Lock()
	defer mu.Unlock()

	list := subs[s.sessionID]
	for i, sub := range list {
		if sub == s {
			subs[s.sessionID] = append(list[:i], list[i+1:]...)
			if len(subs[s.sessionID]) == 0 {
				delete(subs, s.sessionID)
			}
			return
		}
	}
}
