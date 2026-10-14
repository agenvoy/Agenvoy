package todo

import (
	"slices"
	"strings"

	go_llm_router "github.com/pardnchiu/go-llm-router/core"
)

func Strip(messages []go_llm_router.Message) []go_llm_router.Message {
	todoIDs := make(map[string]bool)
	for _, msg := range messages {
		for _, tc := range msg.ToolCalls {
			if tc.Function.Name == "write_todo" {
				todoIDs[tc.ID] = true
			}
		}
	}
	if len(todoIDs) == 0 {
		return messages
	}

	kept := make([]go_llm_router.Message, 0, len(messages))
	for _, msg := range messages {
		if msg.ToolCallID != "" && todoIDs[msg.ToolCallID] {
			continue
		}
		if len(msg.ToolCalls) > 0 {
			filtered := make([]go_llm_router.ToolCall, 0, len(msg.ToolCalls))
			for _, tc := range msg.ToolCalls {
				if !todoIDs[tc.ID] {
					filtered = append(filtered, tc)
				}
			}
			if len(filtered) == 0 {
				if content, ok := msg.Content.(string); !ok || strings.TrimSpace(content) == "" {
					continue
				}
				msg.ToolCalls = nil
			} else {
				msg.ToolCalls = filtered
			}
		}
		kept = append(kept, msg)
	}
	return kept
}

func LastPair(messages []go_llm_router.Message) []go_llm_router.Message {
	for i, msg := range slices.Backward(messages) {
		id := ""
		for _, tc := range msg.ToolCalls {
			if tc.Function.Name == "write_todo" {
				id = tc.ID
				break
			}
		}
		if id == "" {
			continue
		}
		for _, m := range messages[i+1:] {
			if m.ToolCallID != id {
				continue
			}
			call := msg
			call.Content = nil
			call.ToolCalls = []go_llm_router.ToolCall{}
			for _, tc := range msg.ToolCalls {
				if tc.ID == id {
					call.ToolCalls = append(call.ToolCalls, tc)
				}
			}
			return []go_llm_router.Message{call, m}
		}
		return nil
	}
	return nil
}
