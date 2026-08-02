package compact

import (
	"fmt"
	"strings"

	"github.com/pardnchiu/agenvoy/internal/agents/exec/todo"
	agentTypes "github.com/pardnchiu/agenvoy/internal/agents/types"
	provider "github.com/pardnchiu/go-llm-router/core"
)

func RawToolFallback(session *agentTypes.AgentSession, taskHash string) bool {
	// * step1: check tool histories is exist or not
	if len(session.ToolHistories) == 0 {
		return false
	}

	// * step2: extract todo result and get clean tool histories
	histories := session.ToolHistories
	var planPair []provider.Message
	if taskHash != "" {
		planPair = todo.LastPair(histories)
		histories = todo.Strip(histories, false)
	}

	// * step3: turn histories to string
	raw := convertToString(histories)
	if raw == "" {
		return false
	}

	session.OldHistories = nil
	session.SummaryMessage = provider.Message{}
	head := []provider.Message{
		{
			Role:    "user",
			Content: "Earlier tool results (summarization failed — kept raw). Required to answer the original question. User has not been replied to yet:\n\n" + raw,
		},
	}
	head = append(head, planPair...)
	session.ToolHistories = head
	return true
}

func convertToString(histories []provider.Message) string {
	nameByID := make(map[string]string)
	for _, e := range histories {
		for _, tc := range e.ToolCalls {
			nameByID[tc.ID] = tc.Function.Name
		}
	}

	var sb strings.Builder
	for _, e := range histories {
		if e.Role != "tool" {
			continue
		}

		content, _ := e.Content.(string)
		if strings.TrimSpace(content) == "" {
			continue
		}

		name := nameByID[e.ToolCallID]
		if strings.TrimSpace(name) == "" {
			name = "tool"
		}
		fmt.Fprintf(&sb, "%s:\n%s\n\n", strings.TrimSpace(name), strings.TrimSpace(content))
	}
	return strings.TrimSpace(sb.String())
}
