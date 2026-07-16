package exec

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/pardnchiu/agenvoy/configs"
	"github.com/pardnchiu/agenvoy/internal/agents/provider"
	agentTypes "github.com/pardnchiu/agenvoy/internal/agents/types"
	"github.com/pardnchiu/agenvoy/internal/filesystem"
	usagelog "github.com/pardnchiu/agenvoy/internal/session/usage"
)

func compactThreshold(modelName string) int {
	if strings.Contains(modelName, "oss-gpt-120") {
		return int(128_000 * 0.8)
	}

	switch {
	case strings.Contains(modelName, "gemini"):
		return int(1_000_000 * 0.8)
	case strings.Contains(modelName, "gpt"):
		return int(400_000 * 0.8)
	case strings.Contains(modelName, "claude"):
		return int(200_000 * 0.8)
	case strings.Contains(modelName, "grok-4.5"):
		return int(500_000 * 0.8)
	case strings.Contains(modelName, "grok"):
		return int(256_000 * 0.8)
	case strings.Contains(modelName, "deepseek"):
		return int(128_000 * 0.8)
	default:
		return int(128_000 * 0.8)
	}
}

func compactExec(ctx context.Context, agent agentTypes.Agent, session *agentTypes.AgentSession, usage *provider.Usage, taskHash string) bool {
	if len(session.ToolHistories) == 0 {
		return false
	}

	groupStarts := groupStartIndices(session.ToolHistories)
	total := len(groupStarts)
	if total < 3 {
		return false
	}
	batchSize := min(max(int(math.Round(float64(total)*0.2)), 2), total-1)
	return compactRange(ctx, agent, session, usage, taskHash, groupStarts[batchSize])
}

func extractOldHistories(ctx context.Context, agent agentTypes.Agent, session *agentTypes.AgentSession, usage *provider.Usage, events chan<- agentTypes.Event) bool {
	if len(session.OldHistories) == 0 {
		return false
	}
	userQuestion := extractUserText(session.UserInput)
	if userQuestion == "" {
		return false
	}

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
	if utf8.RuneCountInString(sb.String()) < compactThreshold(agent.Name())/3 {
		return false
	}

	events <- agentTypes.Event{Type: agentTypes.EventCompact, Text: "history"}

	prompt := strings.NewReplacer(
		"{{.UserQuestion}}", userQuestion,
	).Replace(strings.TrimSpace(configs.OldHistoryExtractPrompt))

	messages := []provider.Message{
		{Role: "system", Content: prompt},
		{Role: "user", Content: sb.String()},
	}

	compactCtx, cancel := context.WithTimeout(ctx, time.Duration(filesystem.AgentSendTimeoutSec)*time.Second)
	defer cancel()

	resp, err := agent.Send(compactCtx, messages, nil)
	if err != nil {
		slog.Warn("extractOldHistories agent.Send",
			slog.String("session", session.ID),
			slog.String("error", err.Error()))
		return false
	}
	if len(resp.Choices) == 0 {
		return false
	}

	if usage != nil {
		usage.Input += resp.Usage.Input
		usage.Output += resp.Usage.Output
		usage.CacheCreate += resp.Usage.CacheCreate
		usage.CacheRead += resp.Usage.CacheRead
	}

	prov, model, _ := strings.Cut(agent.Name(), "@")
	usagelog.Append(session.ID, prov, model, resp.Usage)

	result, ok := resp.Choices[0].Message.Content.(string)
	if !ok || strings.TrimSpace(result) == "" {
		return false
	}

	session.OldHistories = []provider.Message{
		{Role: "user", Content: "以下是先前對話的整併摘要（僅為背景參考，不代表本次問題所需的完整資料——若需要最新資訊仍須查找或呼叫工具，尚未回覆使用者）：\n\n" + strings.TrimSpace(result)},
	}
	return true
}

func rawToolDumpFallback(session *agentTypes.AgentSession, taskHash string) bool {
	if len(session.ToolHistories) == 0 {
		return false
	}

	messages := session.ToolHistories
	var planPair []provider.Message
	if taskHash != "" {
		planPair = lastWriteTodoPair(messages)
		messages = stripWriteTodo(messages, false)
	}

	raw := rawToolDump(messages)
	if raw == "" {
		return false
	}

	session.OldHistories = nil
	session.SummaryMessage = provider.Message{}
	head := []provider.Message{
		{Role: "user", Content: "以下是先前工具查詢的原始結果（摘要失敗，未經萃取，回答原始問題所需的需求資料，尚未回覆使用者）：\n\n" + raw},
	}
	head = append(head, planPair...)
	session.ToolHistories = head
	return true
}

func rawToolDump(messages []provider.Message) string {
	nameByID := make(map[string]string)
	for _, msg := range messages {
		for _, tc := range msg.ToolCalls {
			nameByID[tc.ID] = tc.Function.Name
		}
	}

	var sb strings.Builder
	for _, msg := range messages {
		if msg.Role != "tool" {
			continue
		}
		content, _ := msg.Content.(string)
		if strings.TrimSpace(content) == "" {
			continue
		}
		name := nameByID[msg.ToolCallID]
		if name == "" {
			name = "tool"
		}
		fmt.Fprintf(&sb, "%s:\n%s\n\n", name, content)
	}
	return strings.TrimSpace(sb.String())
}

func compactRange(ctx context.Context, agent agentTypes.Agent, session *agentTypes.AgentSession, usage *provider.Usage, taskHash string, boundaryIdx int) bool {
	userQuestion := extractUserText(session.UserInput)
	if userQuestion == "" {
		return false
	}

	tail := session.ToolHistories[boundaryIdx:]

	prefix := session.ToolHistories[:boundaryIdx]
	var planPair []provider.Message
	if taskHash != "" {
		if !containsWriteTodo(tail) {
			planPair = lastWriteTodoPair(prefix)
		}
		prefix = stripWriteTodo(prefix, false)
	}

	var sb strings.Builder
	for _, msg := range prefix {
		switch {
		case msg.Role == "assistant" && len(msg.ToolCalls) > 0:
			for _, tc := range msg.ToolCalls {
				fmt.Fprintf(&sb, "[call] %s(%s)\n", tc.Function.Name, tc.Function.Arguments)
			}
		case msg.Role == "tool":
			content, _ := msg.Content.(string)
			fmt.Fprintf(&sb, "[result] %s\n\n", content)
		case msg.Role == "assistant":
			content, _ := msg.Content.(string)
			if content != "" {
				fmt.Fprintf(&sb, "[assistant] %s\n\n", content)
			}
		case msg.Role == "user":
			content, _ := msg.Content.(string)
			if content != "" {
				fmt.Fprintf(&sb, "[context] %s\n\n", content)
			}
		}
	}
	if sb.Len() == 0 {
		return false
	}

	prompt := strings.NewReplacer(
		"{{.UserQuestion}}", userQuestion,
	).Replace(strings.TrimSpace(configs.CompactExecPrompt))

	messages := []provider.Message{
		{Role: "system", Content: prompt},
		{Role: "user", Content: sb.String()},
	}

	compactCtx, cancel := context.WithTimeout(ctx, time.Duration(filesystem.AgentSendTimeoutSec)*time.Second)
	defer cancel()

	resp, err := agent.Send(compactCtx, messages, nil)
	if err != nil {
		slog.Warn("compactRange agent.Send",
			slog.String("session", session.ID),
			slog.String("error", err.Error()))
		return false
	}
	if len(resp.Choices) == 0 {
		return false
	}

	if usage != nil {
		usage.Input += resp.Usage.Input
		usage.Output += resp.Usage.Output
		usage.CacheCreate += resp.Usage.CacheCreate
		usage.CacheRead += resp.Usage.CacheRead
	}

	prov, model, _ := strings.Cut(agent.Name(), "@")
	usagelog.Append(session.ID, prov, model, resp.Usage)

	result, ok := resp.Choices[0].Message.Content.(string)
	if !ok || strings.TrimSpace(result) == "" {
		return false
	}

	session.OldHistories = nil
	session.SummaryMessage = provider.Message{}
	head := []provider.Message{
		{Role: "user", Content: "以下是先前工具查詢結果的整併資料（回答原始問題所需的需求資料，尚未回覆使用者）：\n\n" + strings.TrimSpace(result)},
	}
	head = append(head, planPair...)
	session.ToolHistories = append(head, tail...)

	return true
}

func groupStartIndices(msgs []provider.Message) []int {
	var idx []int
	for i, m := range msgs {
		if m.Role == "assistant" && len(m.ToolCalls) > 0 {
			idx = append(idx, i)
		}
	}
	return idx
}

func extractUserText(input provider.Message) string {
	switch v := input.Content.(type) {
	case string:
		return v
	case []agentTypes.ContentPart:
		for _, part := range v {
			if part.Type == "text" {
				return part.Text
			}
		}
	}
	return ""
}
