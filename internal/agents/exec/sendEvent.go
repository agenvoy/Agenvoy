package exec

import (
	"log/slog"
	"strings"
	"time"

	agentTypes "github.com/pardnchiu/agenvoy/internal/agents/types"
	"github.com/pardnchiu/agenvoy/internal/tools/interactive"
	provider "github.com/pardnchiu/go-llm-router/core"
)

const maxEmptyRetry = 3
const emptyDataReply = "no usable data, retry later, or using other tools."

func emptyRetryExhausted(emptyCount *int, events chan<- agentTypes.Event, sessionID, taskHash, model, reason string, usage *provider.Usage, start time.Time) bool {
	*emptyCount++
	if *emptyCount >= maxEmptyRetry {
		slog.Error("model returned empty response, retries exhausted",
			slog.String("session", sessionID),
			slog.String("name", model),
			slog.String("reason", reason),
			slog.Int("attempts", *emptyCount))
		sendEmptyData(events, sessionID, taskHash, model, usage, start)
		return true
	}
	return false
}

func sendEmptyData(events chan<- agentTypes.Event, sessionID, taskHash, model string, usage *provider.Usage, start time.Time) {
	sendText(events, emptyDataReply)
	events <- agentTypes.Event{Type: agentTypes.EventDone, Model: model, Usage: usage, Duration: time.Since(start)}
	interactive.FinalizePending(sessionID, taskHash, emptyDataReply)
}

func sendText(events chan<- agentTypes.Event, str string) {
	str = strings.TrimRight(str, "\n")
	if str != "" {
		for line := range strings.SplitSeq(str, "\n") {
			events <- agentTypes.Event{Type: agentTypes.EventText, Text: line}
		}
	}
	events <- agentTypes.Event{Type: agentTypes.EventTextDone}
}

func emitReasoning(events chan<- agentTypes.Event, str string, shown *[]string) {
	cur := normalizeReasoning(str)
	if cur == "" {
		return
	}
	for _, prev := range *shown {
		if strings.Contains(prev, cur) {
			return
		}
	}
	events <- agentTypes.Event{Type: agentTypes.EventReasoning, Text: strings.TrimSpace(str)}
	*shown = append(*shown, cur)
}

func normalizeReasoning(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
