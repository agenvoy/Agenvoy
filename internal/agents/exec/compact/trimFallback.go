package compact

import (
	"strings"

	provider "github.com/pardnchiu/go-llm-router/core"
)

func TrimFallback(oldHistory *[]provider.Message, toolCall *[]provider.Message) {
	if len(*oldHistory) > 0 {
		n := 2
		if len(*oldHistory) < 2 {
			n = 1
		}
		*oldHistory = (*oldHistory)[n:]
		return
	}

	if len(*toolCall) == 0 {
		return
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
		return
	}

	ids := make(map[string]bool, len((*toolCall)[firstToolCall].ToolCalls))
	for _, tool := range (*toolCall)[firstToolCall].ToolCalls {
		ids[tool.ID] = true
	}

	kept := make([]provider.Message, 0, len(*toolCall))
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
}

func IsContextLengthError(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "context_length_exceeded") ||
		strings.Contains(msg, "maximum context length") ||
		strings.Contains(msg, "prompt is too long") ||
		(strings.Contains(msg, "token count") && strings.Contains(msg, "exceeds")) ||
		strings.Contains(msg, "exceeds the maximum number of tokens") ||
		strings.Contains(msg, "context window") ||
		strings.Contains(msg, "reduce the length of the messages") ||
		strings.Contains(msg, "too many tokens") ||
		strings.Contains(msg, "maximum context") ||
		strings.Contains(msg, "maximum prompt length") ||
		strings.Contains(msg, "max_tokens must be at least") ||
		strings.Contains(msg, "context length")
}
