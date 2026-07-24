package handler

import (
	agentTypes "github.com/pardnchiu/agenvoy/internal/agents/types"
)

func drainEvents(events <-chan agentTypes.Event) {
	go func() {
		for ev := range events {
			if ev.Type == agentTypes.EventToolConfirm && ev.ReplyCh != nil {
				ev.ReplyCh <- true
			}
		}
	}()
}
