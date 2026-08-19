package chatCompletions

import (
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	go_pkg_utils "github.com/pardnchiu/go-pkg/utils"

	agentTypes "github.com/pardnchiu/agenvoy/internal/agents/types"
	toolRegister "github.com/pardnchiu/agenvoy/internal/tools/register"
	internalUtils "github.com/pardnchiu/agenvoy/internal/utils"
	provider "github.com/pardnchiu/go-llm-router/core"
)

var (
	codeBlovkRegex    = regexp.MustCompile("(?s)```[^\n]*\n.*?```")
	multiNewlineRegex = regexp.MustCompile(`\n{3,}`)
)

func stream(c *gin.Context, id string, created int64, model string, events <-chan agentTypes.Event) {
	w := c.Writer
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	flusher, ok := w.(http.Flusher)
	if !ok {
		return
	}

	writeChunk := func(choices []gin.H, extra gin.H) bool {
		chunk := gin.H{
			"id":      id,
			"object":  "chat.completion.chunk",
			"created": created,
			"model":   model,
			"choices": choices,
		}
		maps.Copy(chunk, extra)
		raw, err := json.Marshal(chunk)
		if err != nil {
			return false
		}
		if _, err := fmt.Fprintf(w, "data: %s\n\n", raw); err != nil {
			return false
		}
		flusher.Flush()
		return true
	}

	if !writeChunk([]gin.H{{"index": 0, "delta": gin.H{"role": "assistant", "content": ""}, "finish_reason": nil}}, nil) {
		return
	}

	ctx := c.Request.Context()
	var usage provider.Usage
	var streamErr error
	var clientCalls []provider.ToolCall

	emitContent := func(str string) bool {
		if str == "" {
			return true
		}
		return writeChunk([]gin.H{{"index": 0, "delta": gin.H{"content": normalizeMarkdown(str)}, "finish_reason": nil}}, nil)
	}
	emitReasoningLine := func(line string) bool {
		if line == "" {
			return true
		}
		return writeChunk([]gin.H{{"index": 0, "delta": gin.H{"reasoning_content": line + "\n\n"}, "finish_reason": nil}}, nil)
	}

	var lastReasoning string
	emitDedup := func(line string) bool {
		if line == "" || line == lastReasoning {
			return true
		}
		lastReasoning = line
		return emitReasoningLine(line)
	}
	formatToolLine := func(name, args string) string {
		arg := internalUtils.FormatToolEvent(name, args)
		if arg == "" {
			return name
		}
		return name + "  " + go_pkg_utils.TruncateString(arg, 128)
	}

	for ev := range events {
		select {
		case <-ctx.Done():
			return
		default:
		}
		switch ev.Type {
		case agentTypes.EventAgentResult:
			if t := strings.TrimSpace(ev.Text); t != "" && !emitDedup("▸ "+t) {
				return
			}
		case agentTypes.EventSkillResult:
			if t := strings.TrimSpace(ev.Text); t != "" && !emitDedup("▸ skill: "+t) {
				return
			}
		case agentTypes.EventToolCall:
			if ev.ToolName == "" || toolRegister.IsSystemUse(ev.ToolName) {
				break
			}
			if !emitDedup("▸ " + formatToolLine(ev.ToolName, ev.ToolArgs)) {
				return
			}
		case agentTypes.EventToolSkipped:
			if ev.ToolName != "" && !emitDedup("▸ skipped: "+ev.ToolName) {
				return
			}
		case agentTypes.EventExecError:
			if t := strings.TrimSpace(ev.Text); t != "" && !emitReasoningLine("⚠ "+t) {
				return
			}
		case agentTypes.EventReasoning:
			if t := strings.TrimSpace(ev.Text); t != "" && !emitReasoningLine(t) {
				return
			}
		case agentTypes.EventTextDelta:
			if !emitContent(ev.Text) {
				return
			}
		case agentTypes.EventClientToolCall:
			clientCalls = append(clientCalls, ev.ClientToolCalls...)
		case agentTypes.EventDone:
			if ev.Usage != nil {
				usage = *ev.Usage
			}
		case agentTypes.EventError:
			if ev.Err != nil {
				streamErr = ev.Err
			}
		}
	}

	switch {
	case streamErr != nil:
		errChunk := []gin.H{{"index": 0, "delta": gin.H{"content": "\n[error] " + streamErr.Error()}, "finish_reason": "stop"}}
		writeChunk(errChunk, nil)
	case len(clientCalls) > 0:
		writeChunk([]gin.H{{"index": 0, "delta": gin.H{"tool_calls": toolCallPayload(clientCalls)}}}, nil)
		writeChunk([]gin.H{{"index": 0, "delta": gin.H{}, "finish_reason": "tool_calls"}}, nil)
	default:
		writeChunk([]gin.H{{"index": 0, "delta": gin.H{}, "finish_reason": "stop"}}, nil)
	}

	promptTokens := usage.Input + usage.CacheRead
	writeChunk([]gin.H{}, gin.H{"usage": gin.H{
		"prompt_tokens":     promptTokens,
		"completion_tokens": usage.Output,
		"total_tokens":      promptTokens + usage.Output,
		"prompt_tokens_details": gin.H{
			"cached_tokens": usage.CacheRead,
		},
	}})

	if _, err := fmt.Fprintf(w, "data: [DONE]\n\n"); err == nil {
		flusher.Flush()
	}
}

func normalizeMarkdown(str string) string {
	var blocks []string
	str = codeBlovkRegex.ReplaceAllStringFunc(str, func(m string) string {
		blocks = append(blocks, m)
		return fmt.Sprintf("\x00CB%d\x00", len(blocks)-1)
	})

	str = strings.ReplaceAll(str, "\n", "\n\n")
	str = multiNewlineRegex.ReplaceAllString(str, "\n\n")

	for i, b := range blocks {
		str = strings.Replace(str, fmt.Sprintf("\x00CB%d\x00", i), b, 1)
	}
	return str
}
