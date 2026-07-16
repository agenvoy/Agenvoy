package openrouter

import (
	"context"
	"fmt"
	"strings"

	"github.com/pardnchiu/agenvoy/internal/agents/provider"
	go_pkg_http "github.com/pardnchiu/go-pkg/http"
)

const (
	chatAPI = "https://openrouter.ai/api/v1/chat/completions"
)

func (a *Agent) Send(ctx context.Context, messages []provider.Message, tools []provider.Tool, reasoning string) (*provider.Output, error) {
	var merged []provider.Message
	var systemParts []string
	for _, m := range messages {
		if m.Role == "system" {
			if s, ok := m.Content.(string); ok && s != "" {
				systemParts = append(systemParts, s)
			}
		} else {
			merged = append(merged, m)
		}
	}
	if len(systemParts) > 0 {
		merged = append([]provider.Message{{Role: "system", Content: strings.Join(systemParts, "\n\n")}}, merged...)
	}

	effort := provider.ClampReasoningLevel(reasoning, provider.MaxReasoningLevel("openrouter", a.model))
	body := map[string]any{
		"model":       a.model,
		"messages":    merged,
		"temperature": 0.2,
		"tools":       tools,
	}
	if !provider.ReasoningDisabled(effort) {
		body["reasoning"] = map[string]any{"effort": effort}
	}
	result, _, err := go_pkg_http.POST[orOutput](ctx, a.httpClient, chatAPI, map[string]string{
		"Authorization": "Bearer " + a.apiKey,
		"Content-Type":  "application/json",
		"HTTP-Referer":  "https://github.com/pardnchiu/agenvoy",
		"X-Title":       "Agenvoy",
	}, body, "json")
	if err != nil {
		return nil, fmt.Errorf("http.POST: %w", err)
	}
	if result.Error != nil {
		return nil, fmt.Errorf("http.POST: %s", result.Error.Message)
	}

	out := result.toOutput()
	return out, nil
}

type orOutput struct {
	Choices []struct {
		Message struct {
			Role             string `json:"role"`
			Content          any    `json:"content"`
			Reasoning        string `json:"reasoning"`
			ReasoningDetails []struct {
				Type    string `json:"type"`
				Text    string `json:"text"`
				Summary string `json:"summary"`
			} `json:"reasoning_details"`
			ToolCalls []provider.ToolCall `json:"tool_calls"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage provider.Usage `json:"usage"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (o *orOutput) toOutput() *provider.Output {
	out := &provider.Output{Usage: o.Usage}
	for _, c := range o.Choices {
		reasoning := c.Message.Reasoning
		if reasoning == "" {
			var sb strings.Builder
			for _, d := range c.Message.ReasoningDetails {
				seg := d.Text
				if seg == "" {
					seg = d.Summary
				}
				if seg == "" {
					continue
				}
				if sb.Len() > 0 {
					sb.WriteByte('\n')
				}
				sb.WriteString(seg)
			}
			reasoning = sb.String()
		}
		out.Choices = append(out.Choices, provider.OutputChoices{
			Message: provider.Message{
				Role:             c.Message.Role,
				Content:          c.Message.Content,
				ReasoningContent: reasoning,
				ToolCalls:        c.Message.ToolCalls,
			},
			FinishReason: c.FinishReason,
		})
	}
	return out
}
