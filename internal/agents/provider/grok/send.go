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

func (a *Agent) Send(ctx context.Context, messages []provider.Message, tools []provider.Tool, reasoning string) (*provider.Output, int, error) {
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
	if provider.SupportReasoningEffort("grok", a.model) {
		effort := provider.ClampReasoningLevel(reasoning, provider.MaxReasoningLevel("grok", a.model))
		if !provider.ReasoningDisabled(effort) {
			body["reasoning_effort"] = effort
		}
	}

	out, code, err := go_pkg_http.POST[provider.Output](ctx, a.httpClient, chatAPI, map[string]string{
		"Authorization": "Bearer " + a.apiKey,
		"Content-Type":  "application/json",
	}, body, "json")
	if err != nil {
		return nil, code, err
	}
	if out.Error != nil {
		return nil, code, fmt.Errorf("%s", out.Error.Message)
	}
	return &out, code, nil
}
