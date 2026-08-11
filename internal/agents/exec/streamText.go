package exec

import (
	"context"
	"errors"
	"log/slog"
	"regexp"
	"strings"

	"github.com/pardnchiu/agenvoy/configs"
	agentTypes "github.com/pardnchiu/agenvoy/internal/agents/types"
	internalUtils "github.com/pardnchiu/agenvoy/internal/utils"
	provider "github.com/pardnchiu/go-llm-router/core"
)

var (
	partialMarkers  = []string{"<think>", configs.GuardrailSentinel}
	headerRuleRegex = regexp.MustCompile(`^-{3,}$`)
	headerMetaRegex = regexp.MustCompile(`^(?:當前時間|工作目錄|傳送者|當前 chat ID|當前 channel)\s*[:：]`)
)

func streamSend(
	ctx context.Context,
	agent agentTypes.Agent,
	messages []provider.Message,
	tools []provider.Tool,
	reasoning provider.Reasoning,
	mode provider.Mode,
	events chan<- agentTypes.Event,
	shownReasoning *[]string,
) (out *provider.Output, code int, textSent bool, reasonSent bool, err error) {
	if streamer, ok := agent.(provider.StreamAgent); ok {
		stream, streamErr := streamer.SendStream(ctx, messages, tools, reasoning, mode)
		switch {
		case streamErr == nil:
			out, textSent, reasonSent, err = consumeStream(stream, events, shownReasoning)
			return out, streamErrorCode(err), textSent, reasonSent, err
		case !errors.Is(streamErr, provider.ErrStreamUnsupported):
			return nil, streamErrorCode(streamErr), false, false, streamErr
		}
		slog.Debug("stream unsupported, falling back to Send",
			slog.String("name", agent.Name()),
			slog.String("error", streamErr.Error()))
	}

	resp, code, err := agent.Send(ctx, messages, tools, reasoning, mode)
	if err != nil {
		return nil, code, false, false, err
	}
	textSent, reasonSent = replayOutput(resp, events, shownReasoning)
	return resp, code, textSent, reasonSent, nil
}

func streamErrorCode(err error) int {
	var streamError *provider.StreamError
	if errors.As(err, &streamError) {
		return streamError.Code
	}
	return 0
}

func replayOutput(resp *provider.Output, events chan<- agentTypes.Event, shownReasoning *[]string) (textSent bool, reasonSent bool) {
	if resp == nil || len(resp.Choices) == 0 {
		return false, false
	}

	message := resp.Choices[0].Message
	reason := lineEmitter{events: events, kind: agentTypes.EventReasoning, shown: shownReasoning}
	reason.write(message.ReasoningContent)
	reason.close()

	text := lineEmitter{events: events, kind: agentTypes.EventText, emitDelta: true}
	if str, ok := message.Content.(string); ok {
		text.write(str)
	}
	text.close()

	return text.sent, reason.sent
}

func consumeStream(stream <-chan provider.StreamEvent, events chan<- agentTypes.Event, shownReasoning *[]string) (*provider.Output, bool, bool, error) {
	text := lineEmitter{events: events, kind: agentTypes.EventText, emitDelta: true}
	reason := lineEmitter{events: events, kind: agentTypes.EventReasoning, shown: shownReasoning}

	var content, reasoning strings.Builder
	var usage provider.Usage
	var finishReason string
	var streamErr error
	toolIndex := map[int]int{}
	var toolCalls []provider.ToolCall
	frames := 0

	for ev := range stream {
		frames++
		switch ev.Type {
		case provider.StreamEventText:
			reason.close()
			text.write(ev.TextDelta)
			content.WriteString(ev.TextDelta)

		case provider.StreamEventReasoning:
			reason.write(ev.ReasoningDelta)
			reasoning.WriteString(ev.ReasoningDelta)

		case provider.StreamEventToolCall:
			if ev.ToolCall == nil {
				break
			}
			pos, ok := toolIndex[ev.ToolCall.Index]
			if !ok {
				pos = len(toolCalls)
				toolIndex[ev.ToolCall.Index] = pos
				toolCalls = append(toolCalls, provider.ToolCall{Type: "function"})
			}
			call := &toolCalls[pos]
			if ev.ToolCall.ID != "" {
				call.ID = ev.ToolCall.ID
			}
			if ev.ToolCall.Name != "" {
				call.Function.Name = ev.ToolCall.Name
			}
			if ev.ToolCall.ThoughtSignature != "" {
				call.ThoughtSignature = ev.ToolCall.ThoughtSignature
			}
			call.Function.Arguments += ev.ToolCall.Arguments

		case provider.StreamEventUsage:
			if ev.Usage != nil {
				usage = *ev.Usage
			}

		case provider.StreamEventDone:
			if ev.FinishReason != "" {
				finishReason = ev.FinishReason
			}

		case provider.StreamEventError:
			if ev.Err != nil {
				streamErr = ev.Err
			}
		}
	}

	reason.close()
	text.close()

	if streamErr != nil {
		return nil, text.sent, reason.sent, streamErr
	}
	if frames == 0 {
		return nil, false, false, errors.New("no events in stream")
	}

	if finishReason == "" {
		finishReason = "stop"
		if len(toolCalls) > 0 {
			finishReason = "tool_calls"
		}
	}
	out := &provider.Output{
		Choices: []provider.OutputChoices{{
			Message: provider.Message{
				Role:             "assistant",
				Content:          content.String(),
				ReasoningContent: reasoning.String(),
				ToolCalls:        toolCalls,
			},
			FinishReason: finishReason,
		}},
		Usage: usage,
	}
	return out, text.sent, reason.sent, nil
}

type lineEmitter struct {
	events      chan<- agentTypes.Event
	kind        agentTypes.EventType
	emitDelta   bool
	shown       *[]string
	seen        strings.Builder
	raw         strings.Builder
	line        strings.Builder
	inThink     bool
	fence       internalUtils.FenceState
	inHeader    bool
	headerDone  bool
	replaying   bool
	headerBuf   []string
	headerBytes int
	deltaHold   strings.Builder
	sent        bool
	stopped     bool
}

func (e *lineEmitter) write(delta string) {
	if e.stopped || delta == "" {
		return
	}

	e.seen.WriteString(delta)
	if seen := e.seen.String(); isGuardrailRefusal(seen) || summaryLeakMarkerRegex.MatchString(seen) {
		e.stopped = true
		e.raw.Reset()
		e.line.Reset()
		return
	}

	e.raw.WriteString(delta)
	for {
		rest := e.raw.String()

		if e.inThink {
			loc := thinkCloseRegex.FindStringIndex(rest)
			if loc == nil {
				return
			}
			e.setRaw(rest[loc[1]:])
			e.inThink = false
			continue
		}

		if loc := thinkOpenRegex.FindStringIndex(rest); loc != nil {
			e.feed(rest[:loc[0]])
			e.setRaw(rest[loc[1]:])
			e.inThink = true
			continue
		}

		hold := holdLen(rest)
		e.feed(rest[:len(rest)-hold])
		e.setRaw(rest[len(rest)-hold:])
		return
	}
}

func (e *lineEmitter) close() {
	if e.stopped || e.inThink {
		return
	}
	e.feed(e.raw.String())
	e.flushHeader()
	e.setRaw("")
	if tail := e.line.String(); tail != "" {
		e.line.Reset()
		e.emit(tail)
	}
}

func (e *lineEmitter) flushHeader() {
	if !e.inHeader {
		return
	}
	buf := e.headerBuf
	e.inHeader = false
	e.headerDone = true
	e.headerBuf = nil
	e.headerBytes = 0
	e.releaseDelta()
	e.replaying = true
	for _, line := range buf {
		e.emit(line)
	}
	e.replaying = false
}

func (e *lineEmitter) releaseDelta() {
	if !e.emitDelta {
		return
	}
	rest := e.deltaHold.String()
	e.deltaHold.Reset()
	if e.headerBytes >= len(rest) {
		e.headerBytes = 0
		return
	}
	rest = rest[e.headerBytes:]
	e.headerBytes = 0
	if rest != "" {
		e.events <- agentTypes.Event{Type: agentTypes.EventTextDelta, Text: rest}
	}
}

func (e *lineEmitter) feed(text string) {
	if text == "" {
		return
	}

	if e.emitDelta {
		if e.headerDone {
			e.events <- agentTypes.Event{Type: agentTypes.EventTextDelta, Text: text}
		} else {
			e.deltaHold.WriteString(text)
		}
	}

	e.line.WriteString(text)
	for {
		line, tail, ok := strings.Cut(e.line.String(), "\n")
		if !ok {
			return
		}
		e.line.Reset()
		e.line.WriteString(tail)
		e.emit(line)
	}
}

func (e *lineEmitter) emit(line string) {
	if e.header(line) {
		return
	}
	line, ok := e.normalize(line)
	if !ok || e.duplicate(line) {
		return
	}
	e.events <- agentTypes.Event{Type: e.kind, Text: line}
	e.sent = true
}

func (e *lineEmitter) header(line string) bool {
	if e.headerDone {
		return false
	}

	trimmed := strings.TrimSpace(line)
	if !e.inHeader {
		if trimmed == "" {
			e.headerBytes += len(line) + 1
			return true
		}
		if !headerRuleRegex.MatchString(trimmed) {
			e.headerDone = true
			e.releaseDelta()
			return false
		}
		e.inHeader = true
		e.headerBuf = append(e.headerBuf, line)
		e.headerBytes += len(line) + 1
		return true
	}

	if headerRuleRegex.MatchString(trimmed) {
		e.inHeader = false
		e.headerDone = true
		e.headerBuf = nil
		e.headerBytes += len(line) + 1
		e.releaseDelta()
		return true
	}

	if trimmed != "" && !headerMetaRegex.MatchString(trimmed) {
		e.flushHeader()
		return false
	}

	e.headerBuf = append(e.headerBuf, line)
	e.headerBytes += len(line) + 1
	return true
}

func (e *lineEmitter) normalize(line string) (string, bool) {
	out, marker := e.fence.Normalize(line)
	if marker || e.fence.InFence {
		return out, true
	}

	if e.replaying {
		return out, true
	}

	stripped := StripModelResponse(out)
	return stripped, stripped != ""
}

func (e *lineEmitter) duplicate(line string) bool {
	if e.shown == nil {
		return false
	}
	cur := normalizeReasoning(line)
	if cur == "" {
		return false
	}
	for _, prev := range *e.shown {
		if strings.Contains(prev, cur) {
			return true
		}
	}
	*e.shown = append(*e.shown, cur)
	return false
}

func (e *lineEmitter) setRaw(text string) {
	e.raw.Reset()
	e.raw.WriteString(text)
}

func holdLen(str string) int {
	longest := 0
	for _, marker := range partialMarkers {
		size := min(len(marker)-1, len(str))
		for i := size; i > longest; i-- {
			if strings.EqualFold(str[len(str)-i:], marker[:i]) {
				longest = i
				break
			}
		}
	}
	return longest
}
