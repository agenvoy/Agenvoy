package grok

import (
	"context"
	"fmt"
	"strings"

	"github.com/pardnchiu/agenvoy/internal/agents/provider"
	go_pkg_http "github.com/pardnchiu/go-pkg/http"
)

const (
	chatAPI = "https://api.x.ai/v1/chat/completions"
)

func (a *Agent) Send(ctx context.Context, messages []provider.Message, tools []provider.Tool) (*provider.Output, error) {
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

	body := map[string]any{
		"model":    a.model,
		"messages": merged,
		"tools":    tools,
	}
	if provider.SupportTemperature("grok", a.model) {
		body["temperature"] = 0.2
	}
	var reasoning string
	if provider.SupportReasoningEffort("grok", a.model) {
		reasoning = provider.ClampReasoningLevel(provider.GetReasoningLevel(), provider.MaxReasoningLevel("grok", a.model))
		if !provider.ReasoningDisabled(reasoning) {
			body["reasoning_effort"] = reasoning
		}
	}

	out, _, err := go_pkg_http.POST[provider.Output](ctx, a.httpClient, chatAPI, map[string]string{
		"Authorization": "Bearer " + a.apiKey,
		"Content-Type":  "application/json",
	}, body, "json")
	if err != nil {
		return nil, fmt.Errorf("http.POST: %w", err)
	}
	if out.Error != nil {
		return nil, fmt.Errorf("http.POST: %s", out.Error.Message)
	}

	return &out, nil
}
