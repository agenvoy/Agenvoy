package compact

import (
	"strings"

	"github.com/pardnchiu/agenvoy/configs"
	"github.com/pardnchiu/agenvoy/internal/agents/exec/todo"
	agentTypes "github.com/pardnchiu/agenvoy/internal/agents/types"
	provider "github.com/pardnchiu/go-llm-router/core"
)

func AssembleMessages(session *agentTypes.AgentSession, taskHash string) []provider.Message {
	result := make([]provider.Message, 0, len(session.SystemPrompts)+len(session.OldHistories)+2+len(session.ToolHistories))
	result = append(result, session.SystemPrompts...)
	for _, msg := range session.OldHistories {
		if content, ok := msg.Content.(string); ok && (strings.Contains(content, configs.PoisonRefusal) || strings.Contains(content, configs.GuardrailSentinel)) {
			continue
		}
		result = append(result, msg)
	}
	if session.SummaryMessage.Role != "" {
		result = append(result, session.SummaryMessage)
	}
	result = append(result, session.UserInput)
	result = append(result, session.ToolHistories...)

	if taskHash != "" {
		result = todo.Strip(result, true)
	}
	return result
}
