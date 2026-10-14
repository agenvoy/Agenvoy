package exec

import (
	"log/slog"
	"strings"
	"time"

	"github.com/pardnchiu/agenvoy/configs"
	agenttypes "github.com/pardnchiu/agenvoy/internal/agents/types"
	"github.com/pardnchiu/agenvoy/internal/tools/interactive"
	provider "github.com/pardnchiu/go-llm-router/core"
)

const emptyDataReply = "no usable data, retry later, or using other tools."

func emptyRetryExhausted(emptyCount *int, events chan<- agenttypes.Event, sessionID, taskHash, model, reason string, usage *provider.Usage, start time.Time, outputElapsed time.Duration) bool {
	*emptyCount++
	if *emptyCount > configs.MAX_RETRY_TIMES {
		slog.Error("model returned empty response, retries exhausted",
			slog.String("session", sessionID),
			slog.String("name", model),
			slog.String("reason", reason),
			slog.Int("attempts", *emptyCount))
		sendEmptyData(events, sessionID, taskHash, model, usage, start, outputElapsed)
		return true
	}
	return false
}

func sendEmptyData(events chan<- agenttypes.Event, sessionID, taskHash, model string, usage *provider.Usage, start time.Time, outputElapsed time.Duration) {
	sendText(events, emptyDataReply)
	events <- agenttypes.DoneEvent(model, usage, time.Since(start), outputElapsed)
	interactive.FinalizePending(sessionID, taskHash, emptyDataReply)
}

func sendText(events chan<- agenttypes.Event, str string) {
	str = strings.TrimRight(str, "\n")
	if str != "" {
		for line := range strings.SplitSeq(str, "\n") {
			events <- agenttypes.Event{Type: agenttypes.EventText, Text: line}
		}
	}
	events <- agenttypes.Event{Type: agenttypes.EventTextDone}
}

func emitReasoning(events chan<- agenttypes.Event, str string, shown *[]string) {
	cur := normalizeReasoning(str)
	if cur == "" {
		return
	}
	for _, prev := range *shown {
		if strings.Contains(prev, cur) {
			return
		}
	}
	events <- agenttypes.Event{Type: agenttypes.EventReasoning, Text: strings.TrimSpace(str)}
	*shown = append(*shown, cur)
}

func markReasoningShown(str string, shown *[]string) {
	cur := normalizeReasoning(str)
	if cur == "" {
		return
	}
	*shown = append(*shown, cur)
}

func normalizeReasoning(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
