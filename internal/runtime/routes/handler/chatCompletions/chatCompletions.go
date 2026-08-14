package chatCompletions

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pardnchiu/go-pkg/utils"

	agentTypes "github.com/pardnchiu/agenvoy/internal/agents/types"
	provider "github.com/pardnchiu/go-llm-router/core"
)

var (
	zedWorkDirRegex = regexp.MustCompile("(?m)^- `(/[^`]+)`")
)

type Request struct {
	Model           string             `json:"model"`
	Messages        []provider.Message `json:"messages"`
	Stream          bool               `json:"stream"`
	ReasoningEffort string             `json:"reasoning_effort"`
	Tools           []provider.Tool    `json:"tools"`
	workDir         string             `json:"-"`
	systemPrompts   []provider.Message `json:"-"`
	agentMode       bool               `json:"-"`
}

const AgentHeader = "X-Agenvoy-Agent"

func ChatCompletions() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req Request
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
				"message": err.Error(),
				"type":    "invalid_request_error",
			}})
			return
		}

		switch strings.ToLower(strings.TrimSpace(c.GetHeader(AgentHeader))) {
		case "1", "true", "yes":
			req.agentMode = true
		}

		normalizeContent(req.Messages)

		if _, ok := provider.ParseReasoning(req.ReasoningEffort); !ok {
			req.ReasoningEffort = "medium"
		}

		workDir := extractWorkDirFromZed(req.Messages)
		if workDir != "" {
			req.workDir = workDir
		}

		systemPrompts := make([]provider.Message, 0)
		messages := make([]provider.Message, 0, len(req.Messages))
		for _, msg := range req.Messages {
			if msg.Role == "system" {
				systemPrompts = append(systemPrompts, msg)
				continue
			}
			messages = append(messages, msg)
		}
		req.systemPrompts = systemPrompts
		req.Messages = messages

		input := ""
		for i := len(req.Messages) - 1; i >= 0; i-- {
			if req.Messages[i].Role != "user" {
				continue
			}
			if str, ok := req.Messages[i].Content.(string); ok {
				input = str
				break
			}
		}

		if strings.TrimSpace(input) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
				"message": "no user message",
				"type":    "invalid_request_error",
			}})
			return
		}

		events := make(chan agentTypes.Event, 64)
		ctx := c.Request.Context()

		go func() {
			defer close(events)
			run(ctx, req, input, events)
		}()

		id := "chatcmpl-" + utils.UUID()
		created := time.Now().Unix()
		if req.Stream {
			stream(c, id, created, req.Model, events)
		} else {
			collect(c, id, created, req.Model, events)
		}
	}
}

func normalizeContent(messages []provider.Message) {
	for i := range messages {
		raw, ok := messages[i].Content.([]any)
		if !ok {
			continue
		}
		parts := make([]provider.ContentPart, 0, len(raw))
		allText := true
		var textBuf strings.Builder
		for _, item := range raw {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			typeStr, _ := m["type"].(string)
			text, _ := m["text"].(string)
			switch typeStr {
			case "text", "input_text", "output_text":
				if textBuf.Len() > 0 {
					textBuf.WriteByte('\n')
				}
				textBuf.WriteString(text)
				parts = append(parts, provider.ContentPart{Type: "text", Text: text})
			case "image_url":
				url, detail := extractImageURL(m["image_url"])
				parts = append(parts, provider.ContentPart{
					Type:     "image_url",
					ImageURL: &provider.ImageURL{URL: url, Detail: detail},
				})
				allText = false
			case "input_image":
				url, _ := m["image_url"].(string)
				parts = append(parts, provider.ContentPart{
					Type:     "image_url",
					ImageURL: &provider.ImageURL{URL: url, Detail: "auto"},
				})
				allText = false
			}
		}
		if allText {
			messages[i].Content = textBuf.String()
		} else {
			messages[i].Content = parts
		}
	}
}

func extractImageURL(v any) (url, detail string) {
	detail = "auto"
	switch t := v.(type) {
	case string:
		url = t
	case map[string]any:
		if u, ok := t["url"].(string); ok {
			url = u
		}
		if d, ok := t["detail"].(string); ok && d != "" {
			detail = d
		}
	}
	return
}

func extractWorkDirFromZed(messages []provider.Message) string {
	for _, msg := range messages {
		if msg.Role != "system" {
			continue
		}

		str, ok := msg.Content.(string)
		if !ok {
			continue
		}
		if matches := zedWorkDirRegex.FindStringSubmatch(str); len(matches) > 1 {
			return matches[1]
		}
	}
	return ""
}

func collect(c *gin.Context, id string, created int64, model string, events <-chan agentTypes.Event) {
	var textBuf strings.Builder
	var usage provider.Usage
	var streamErr error
	var clientCalls []provider.ToolCall
	for ev := range events {
		switch ev.Type {
		case agentTypes.EventText:
			textBuf.WriteString(ev.Text)

		case agentTypes.EventClientToolCall:
			clientCalls = append(clientCalls, ev.ClientToolCalls...)

		case agentTypes.EventDone:
			if ev.Usage != nil {
				usage = *ev.Usage
			}

		case agentTypes.EventError:
			streamErr = ev.Err
		}
	}
	if streamErr != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{"message": streamErr.Error(), "type": "upstream_error"}})
		return
	}

	message := gin.H{"role": "assistant", "content": textBuf.String()}
	finish := "stop"
	if len(clientCalls) > 0 {
		message = gin.H{"role": "assistant", "content": nil, "tool_calls": toolCallPayload(clientCalls)}
		if text := strings.TrimSpace(textBuf.String()); text != "" {
			message["content"] = text
		}
		finish = "tool_calls"
	}

	c.JSON(http.StatusOK, gin.H{
		"id":      id,
		"object":  "chat.completion",
		"created": created,
		"model":   model,
		"choices": []gin.H{{
			"index":         0,
			"message":       message,
			"finish_reason": finish,
		}},
		"usage": gin.H{
			"prompt_tokens":     usage.Input + usage.CacheRead,
			"completion_tokens": usage.Output,
			"total_tokens":      usage.Input + usage.CacheRead + usage.Output,
			"prompt_tokens_details": gin.H{
				"cached_tokens": usage.CacheRead,
			},
		},
	})
}

func toolCallPayload(calls []provider.ToolCall) []gin.H {
	out := make([]gin.H, 0, len(calls))
	for i, call := range calls {
		id := call.ID
		if id == "" {
			id = fmt.Sprintf("call_%s", utils.UUID())
		}
		kind := call.Type
		if kind == "" {
			kind = "function"
		}
		out = append(out, gin.H{
			"index": i,
			"id":    id,
			"type":  kind,
			"function": gin.H{
				"name":      call.Function.Name,
				"arguments": call.Function.Arguments,
			},
		})
	}
	return out
}
