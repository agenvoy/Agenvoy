package compact

import (
	"strings"

	"github.com/pardnchiu/agenvoy/configs"
	agentTypes "github.com/pardnchiu/agenvoy/internal/agents/types"
	provider "github.com/pardnchiu/go-llm-router/core"
)

func AssembleMessages(session *agentTypes.AgentSession) []provider.Message {
	result := make([]provider.Message, 0, len(session.SystemPrompts)+len(session.OldHistories)+3+len(session.ToolHistories))
	result = append(result, session.SystemPrompts...)
	for _, msg := range session.OldHistories {
		if content, ok := msg.Content.(string); ok && strings.Contains(content, configs.BAN_TAG) {
			continue
		}
		result = append(result, msg)
	}
	if session.SummaryMessage.Role != "" {
		result = append(result, session.SummaryMessage)
	}
	if session.TurnContext.Role != "" {
		result = append(result, session.TurnContext)
	}
	result = append(result, session.UserInput)
	result = append(result, session.ToolHistories...)

	return result
}
