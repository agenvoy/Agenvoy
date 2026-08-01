package compact

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/pardnchiu/agenvoy/configs"
	"github.com/pardnchiu/agenvoy/internal/agents"
	agentTypes "github.com/pardnchiu/agenvoy/internal/agents/types"
	"github.com/pardnchiu/agenvoy/internal/filesystem"
	usagelog "github.com/pardnchiu/agenvoy/internal/session/usage"
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
	result := sendCompact(ctx, agent, session.ID, usage, messages)
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

func sendCompact(ctx context.Context, agent agentTypes.Agent, sessionID string, usage *provider.Usage, messages []provider.Message) string {
	sender := agent
	usedSummary := false
	if summaryBot := agents.SummaryBot(); summaryBot != nil && payloadRunes(messages) < CheckThreshold(summaryBot.Name()) {
		sender = summaryBot
		usedSummary = true
	}

	resp, err := send(ctx, sender, messages)
	if err != nil && usedSummary {
		slog.Warn("sendCompact: summary model send",
			slog.String("session", sessionID),
			slog.String("model", sender.Name()),
			slog.String("error", err.Error()))
		sender = agent
		resp, err = send(ctx, sender, messages)
	}
	if err != nil {
		slog.Warn("sendCompact: send",
			slog.String("session", sessionID),
			slog.String("model", sender.Name()),
			slog.String("error", err.Error()))
		return ""
	}
	if len(resp.Choices) == 0 {
		return ""
	}

	if usage != nil {
		usage.Input += resp.Usage.Input
		usage.Output += resp.Usage.Output
		usage.CacheCreate += resp.Usage.CacheCreate
		usage.CacheRead += resp.Usage.CacheRead
	}

	prov, model, _ := strings.Cut(sender.Name(), "@")
	usagelog.Append(sessionID, prov, model, resp.Usage)

	result, ok := resp.Choices[0].Message.Content.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(result)
}

func send(ctx context.Context, agent agentTypes.Agent, messages []provider.Message) (*provider.Output, error) {
	compactCtx, cancel := context.WithTimeout(ctx, time.Duration(filesystem.AgentSendTimeoutSec)*time.Second)
	defer cancel()

	resp, _, err := agent.Send(compactCtx, messages, nil, provider.ReasoningNone)
	return resp, err
}

func payloadRunes(messages []provider.Message) int {
	total := 0
	for _, msg := range messages {
		if content, ok := msg.Content.(string); ok {
			total += utf8.RuneCountInString(content)
		}
	}
	return total
}
