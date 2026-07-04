package exec

import (
	"strings"

	agentTypes "github.com/pardnchiu/agenvoy/internal/agents/types"
)

func assembleMessages(systemPart []agentTypes.Message, oldHistory []agentTypes.Message, summaryMessage agentTypes.Message, userInput agentTypes.Message, toolCall []agentTypes.Message, taskHash string) []agentTypes.Message {
	result := make([]agentTypes.Message, 0, len(systemPart)+len(oldHistory)+2+len(toolCall))
	result = append(result, systemPart...)
	for _, msg := range oldHistory {
		if content, ok := msg.Content.(string); ok && (strings.Contains(content, poisonRefusal) || strings.Contains(content, guardrailSentinel)) {
			continue
		}
		result = append(result, msg)
	}
	if summaryMessage.Role != "" {
		result = append(result, summaryMessage)
	}
	result = append(result, userInput)
	result = append(result, toolCall...)

	if taskHash != "" {
		result = stripWriteTodo(result, true)
	}
	return result
}

func stripWriteTodo(messages []agentTypes.Message, keepLast bool) []agentTypes.Message {
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

	kept := make([]agentTypes.Message, 0, len(messages))
	for _, msg := range messages {
		if msg.ToolCallID != "" && todoIDs[msg.ToolCallID] {
			continue
		}
		if len(msg.ToolCalls) > 0 {
			filtered := make([]agentTypes.ToolCall, 0, len(msg.ToolCalls))
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

func lastWriteTodoPair(messages []agentTypes.Message) []agentTypes.Message {
	for i := len(messages) - 1; i >= 0; i-- {
		id := ""
		for _, tc := range messages[i].ToolCalls {
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
			call := messages[i]
			call.Content = nil
			call.ToolCalls = []agentTypes.ToolCall{}
			for _, tc := range messages[i].ToolCalls {
				if tc.ID == id {
					call.ToolCalls = append(call.ToolCalls, tc)
				}
			}
			return []agentTypes.Message{call, m}
		}
		return nil
	}
	return nil
}

func containsWriteTodo(messages []agentTypes.Message) bool {
	for _, msg := range messages {
		for _, tc := range msg.ToolCalls {
			if tc.Function.Name == "write_todo" {
				return true
			}
		}
	}
	return false
}

func trimOnContextExceeded(oldHistory *[]agentTypes.Message, toolCall *[]agentTypes.Message) bool {
	if len(*oldHistory) > 0 {
		n := 2
		if len(*oldHistory) < 2 {
			n = 1
		}
		*oldHistory = (*oldHistory)[n:]
		return false
	}

	if len(*toolCall) == 0 {
		return false
	}

	firstToolCall := -1
	for i, message := range *toolCall {
		if message.Role == "assistant" && len(message.ToolCalls) > 0 {
			firstToolCall = i
			break
		}
	}

	if firstToolCall == -1 {
		*toolCall = (*toolCall)[1:]
		return false
	}

	ids := make(map[string]bool, len((*toolCall)[firstToolCall].ToolCalls))
	for _, tool := range (*toolCall)[firstToolCall].ToolCalls {
		ids[tool.ID] = true
	}

	kept := make([]agentTypes.Message, 0, len(*toolCall))
	for i, m := range *toolCall {
		if i == firstToolCall {
			continue
		}
		if m.ToolCallID != "" && ids[m.ToolCallID] {
			continue
		}
		kept = append(kept, m)
	}
	*toolCall = kept
	return true
}

func isContextLengthError(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "context_length_exceeded") ||
		strings.Contains(msg, "maximum context length") ||
		strings.Contains(msg, "prompt is too long") ||
		(strings.Contains(msg, "token count") && strings.Contains(msg, "exceeds")) ||
		strings.Contains(msg, "exceeds the maximum number of tokens")
}
