package handler

import (
	agentTypes "github.com/pardnchiu/agenvoy/internal/agents/types"
)

func drainEvents(events <-chan agentTypes.Event) {
	go func() {
		for range events {
		}
	}()
}
