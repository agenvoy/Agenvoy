package handler

import (
	agenttypes "github.com/pardnchiu/agenvoy/internal/agents/types"
)

func drainEvents(events <-chan agenttypes.Event) {
	go func() {
		for range events {
		}
	}()
}
