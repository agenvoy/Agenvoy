package deepseek

import (
	"context"
	"fmt"
	"strings"

	"github.com/pardnchiu/agenvoy/internal/agents/provider"
	go_pkg_http "github.com/pardnchiu/go-pkg/http"
)

const (
	chatAPI = "https://api.deepseek.com/v1/chat/completions"
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

	for i := range merged {
		if merged[i].Role == "assistant" && merged[i].ReasoningContent == "" {
			merged[i].ReasoningContent = "(reasoning omitted)"
		}
	}

	body := map[string]any{
		"model":    a.model,
		"messages": merged,
		"tools":    tools,
	}
	if provider.SupportTemperature("deepseek", a.model) {
		body["temperature"] = 0.2
	}

	result, _, err := go_pkg_http.POST[provider.Output](ctx, a.httpClient, chatAPI, map[string]string{
		"Authorization": "Bearer " + a.apiKey,
		"Content-Type":  "application/json",
	}, body, "json")
	if err != nil {
		return nil, fmt.Errorf("http.POST: %w", err)
	}
	if result.Error != nil {
		return nil, fmt.Errorf("http.POST: %s", result.Error.Message)
	}

	return &result, nil
}
