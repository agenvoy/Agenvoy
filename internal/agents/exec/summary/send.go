package agentSummary

import (
	"context"
	"log/slog"
	"strings"
	"time"
	"unicode/utf8"

	go_llm_router "github.com/pardnchiu/go-llm-router/core"

	"github.com/pardnchiu/agenvoy/configs"
	"github.com/pardnchiu/agenvoy/internal/agents"
	"github.com/pardnchiu/agenvoy/internal/agents/exec/fast"
	"github.com/pardnchiu/agenvoy/internal/agents/exec/retryHandler"
	agenttypes "github.com/pardnchiu/agenvoy/internal/agents/types"
	usagelog "github.com/pardnchiu/agenvoy/internal/session/usage"
)

func Send(ctx context.Context, agent agenttypes.Agent, sessionID string, usage *go_llm_router.Usage, messages []go_llm_router.Message, reasoning go_llm_router.Reasoning, threshold func(string) int) string {
	// * step1: prefer the summary model, then steer off a cooling one
	sender := agent
	if bot := agents.SummaryBot(); bot != nil {
		sender = bot
	}
	if picked := retryHandler.Check(sender, agents.ModelSet()); picked != nil {
		sender = picked
	}

	// * step2: gate on the model that actually sends, fall back when it cannot hold the payload
	if threshold != nil && sender.Name() != agent.Name() && payloadRunes(messages) >= threshold(sender.Name()) {
		sender = agent
	}

	resp, sendElapsed, err := send(ctx, sender, messages, reasoning)
	if err != nil && sender.Name() != agent.Name() {
		slog.Debug("agentSummary.Send: summary model",
			slog.String("session", sessionID),
			slog.String("model", sender.Name()),
			slog.String("error", err.Error()))
		sender = agent
		resp, sendElapsed, err = send(ctx, sender, messages, reasoning)
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
	usagelog.Append(sessionID, prov, model, resp.Usage, sendElapsed, nil)

	result, ok := resp.Choices[0].Message.Content.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(result)
}

func send(ctx context.Context, agent agenttypes.Agent, messages []go_llm_router.Message, reasoning go_llm_router.Reasoning) (*go_llm_router.Output, time.Duration, error) {
	sendCtx, cancel := context.WithTimeout(ctx, time.Duration(configs.AGENT_SEND_TIMEOUT_SEC)*time.Second)
	defer cancel()

	sendStart := time.Now()
	resp, _, err := agent.Send(sendCtx, messages, nil, reasoning, fast.Mode(ctx))
	return resp, time.Since(sendStart), err
}

func payloadRunes(messages []go_llm_router.Message) int {
	total := 0
	for _, msg := range messages {
		if content, ok := msg.Content.(string); ok {
			total += utf8.RuneCountInString(content)
		}
	}
	return total
}
