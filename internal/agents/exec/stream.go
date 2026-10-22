package exec

import (
	"context"

	agenttypes "github.com/pardnchiu/agenvoy/internal/agents/types"
	"github.com/pardnchiu/agenvoy/internal/runtime/pubsub"
)

func Stream(ctx context.Context, sessionID string, size int, run func(events chan<- agenttypes.Event) error) (<-chan agenttypes.Event, func() error) {
	events := make(chan agenttypes.Event, size)
	wrapped := pubsub.Wrap(ctx, sessionID, events, size)
	errCh := make(chan error, 1)
	go func() {
		defer close(wrapped)
		errCh <- run(wrapped)
	}()
	return events, func() error {
		return <-errCh
	}
}
