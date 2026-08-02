package todo

import (
	"slices"
	"strings"

	provider "github.com/pardnchiu/go-llm-router/core"
)

func Strip(messages []provider.Message, keepLast bool) []provider.Message {
	keepID := ""
	if keepLast {
		for i := len(messages) - 1; i >= 0 && keepID == ""; i-- {
			for _, tc := range messages[i].ToolCalls {
				if tc.Function.Name == "write_todo" {
					keepID = tc.ID
					break
				}
			}
		}
	}

	todoIDs := make(map[string]bool)
	for _, msg := range messages {
		for _, tc := range msg.ToolCalls {
			if tc.Function.Name == "write_todo" && tc.ID != keepID {
				todoIDs[tc.ID] = true
			}
		}
	}
	if len(todoIDs) == 0 {
		return messages
	}

	kept := make([]provider.Message, 0, len(messages))
	for _, msg := range messages {
		if msg.ToolCallID != "" && todoIDs[msg.ToolCallID] {
			continue
		}
		if len(msg.ToolCalls) > 0 {
			filtered := make([]provider.ToolCall, 0, len(msg.ToolCalls))
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

func LastPair(messages []provider.Message) []provider.Message {
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
			call.ToolCalls = []provider.ToolCall{}
			for _, tc := range msg.ToolCalls {
				if tc.ID == id {
					call.ToolCalls = append(call.ToolCalls, tc)
				}
			}
			return []provider.Message{call, m}
		}
		return nil
	}
	return nil
}
