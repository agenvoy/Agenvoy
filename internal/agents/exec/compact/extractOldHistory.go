package compact

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/pardnchiu/agenvoy/configs"
	agentSummary "github.com/pardnchiu/agenvoy/internal/agents/exec/summary"
	agentTypes "github.com/pardnchiu/agenvoy/internal/agents/types"
	provider "github.com/pardnchiu/go-llm-router/core"
)

const (
	minOldHistoryRunes = 32_000
)

func ExtractOldHistories(ctx context.Context, agent agentTypes.Agent, session *agentTypes.AgentSession, usage *provider.Usage, events chan<- agentTypes.Event) bool {
	// * step1: check user input is exist or not
	userInput := extractUserInput(session.UserInput)
	if userInput == "" {
		return false
	}

	// * step2: check old history is exist or not
	if len(session.OldHistories) == 0 {
		return false
	}

	// * step3: exract old history content
	var sb strings.Builder
	for _, msg := range session.OldHistories {
		content, _ := msg.Content.(string)
		if content == "" {
			continue
		}

		switch msg.Role {
		case "user":
			fmt.Fprintf(&sb, "[user] %s\n\n", content)
		case "assistant":
			fmt.Fprintf(&sb, "[assistant] %s\n\n", content)
		}
	}

	// * step4: check length is over threshold or not
	raw := sb.String()
	if utf8.RuneCountInString(raw) < minOldHistoryRunes {
		return false
	}

	// * step5: sent compact notify
	events <- agentTypes.Event{Type: agentTypes.EventCompact, Text: "history"}

	// * step6: prepare conpact system prompt
	prompt := strings.NewReplacer(
		"{{.UserQuestion}}", userInput,
	).Replace(strings.TrimSpace(configs.OldHistoryExtractPrompt))

	messages := []provider.Message{
		{Role: "system", Content: prompt},
		{Role: "user", Content: raw},
	}

	// * step7: start to compact old history
	result := agentSummary.Send(ctx, agent, session.ID, usage, messages, provider.ReasoningNone, CheckThreshold)
	if result == "" {
		return false
	}

	session.OldHistories = []provider.Message{
		{
			Role:    "user",
			Content: "Prior conversation summary (background only — not complete data for this question. Call tools for up-to-date information. The user has not been replied to yet):\n\n" + result,
		},
	}
	return true
}

func extractUserInput(input provider.Message) string {
	switch val := input.Content.(type) {
	case string:
		return val
	case []provider.ContentPart:
		for _, part := range val {
			if part.Type == "text" {
				return part.Text
			}
		}
	}
	return ""
}
