package agentSummary

import (
	"context"
	"log/slog"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/pardnchiu/agenvoy/internal/agents"
	"github.com/pardnchiu/agenvoy/internal/agents/exec/cooldown"
	agentTypes "github.com/pardnchiu/agenvoy/internal/agents/types"
	"github.com/pardnchiu/agenvoy/internal/filesystem"
	usagelog "github.com/pardnchiu/agenvoy/internal/session/usage"
	provider "github.com/pardnchiu/go-llm-router/core"
)

func Send(ctx context.Context, agent agentTypes.Agent, sessionID string, usage *provider.Usage, messages []provider.Message, reasoning provider.Reasoning, threshold func(string) int) string {
	// * step1: prefer the summary model, then steer off a cooling one
	sender := agent
	if bot := agents.SummaryBot(); bot != nil {
		sender = bot
	}
	if picked := cooldown.Check(sender, agents.Registry()); picked != nil {
		sender = picked
	}

	// * step2: gate on the model that actually sends, fall back when it cannot hold the payload
	if threshold != nil && sender.Name() != agent.Name() && payloadRunes(messages) >= threshold(sender.Name()) {
		sender = agent
	}

	resp, err := send(ctx, sender, messages, reasoning)
	if err != nil && sender.Name() != agent.Name() {
		slog.Warn("agentSummary.Send: summary model",
			slog.String("session", sessionID),
			slog.String("model", sender.Name()),
			slog.String("error", err.Error()))
		sender = agent
		resp, err = send(ctx, sender, messages, reasoning)
	}
	if err != nil {
		slog.Warn("agentSummary.Send",
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

func send(ctx context.Context, agent agentTypes.Agent, messages []provider.Message, reasoning provider.Reasoning) (*provider.Output, error) {
	sendCtx, cancel := context.WithTimeout(ctx, time.Duration(filesystem.AgentSendTimeoutSec)*time.Second)
	defer cancel()

	resp, _, err := agent.Send(sendCtx, messages, nil, reasoning)
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
