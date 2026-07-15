package openai

import (
	"context"
	"fmt"

	"github.com/pardnchiu/agenvoy/internal/agents/provider"
	copilotResponse "github.com/pardnchiu/agenvoy/internal/agents/provider/copilot/response"
	agentTypes "github.com/pardnchiu/agenvoy/internal/agents/types"
	usagelog "github.com/pardnchiu/agenvoy/internal/session/usage"
	toolTypes "github.com/pardnchiu/agenvoy/internal/tools/types"
	go_pkg_http "github.com/pardnchiu/go-pkg/http"
)

const (
	chatAPI      = "https://api.openai.com/v1/chat/completions"
	responsesAPI = "https://api.openai.com/v1/responses"
)

func (a *Agent) Send(ctx context.Context, messages []agentTypes.Message, tools []toolTypes.Tool) (*agentTypes.Output, error) {
	headers := map[string]string{
		"Authorization": "Bearer " + a.apiKey,
		"Content-Type":  "application/json",
	}

	if provider.ResponsesAPI("openai", a.model) {
		var instructions string
		nonSystem := make([]agentTypes.Message, 0, len(messages))
		for _, m := range messages {
			if m.Role == "system" {
				if s, ok := m.Content.(string); ok {
					if instructions != "" {
						instructions += "\n"
					}
					instructions += s
				}
				continue
			}
			nonSystem = append(nonSystem, m)
		}

		reasoning := provider.ClampReasoningLevel(provider.GetReasoningLevel(), provider.MaxReasoningLevel("openai", a.model))
		body := map[string]any{
			"model":        a.model,
			"input":        copilotResponse.ConvertInput(nonSystem),
			"tools":        copilotResponse.ConvertTools(tools),
			"instructions": instructions,
			"store":        false,
		}
		if !provider.ReasoningDisabled(reasoning) {
			body["reasoning"] = map[string]any{"effort": reasoning, "summary": "auto"}
		}

		result, _, err := go_pkg_http.POST[copilotResponse.Output](ctx, a.httpClient, responsesAPI, headers, body, "json")
		if err != nil {
			return nil, fmt.Errorf("http.POST: %w", err)
		}
		if result.Error != nil {
			return nil, fmt.Errorf("http.POST: %s", result.Error.Message)
		}
		out := copilotResponse.ConvertOutput(result)
		usagelog.Append(agentTypes.SessionIDFrom(ctx), "openai", a.model, reasoning, out.Usage)
		return &out, nil
	}

	body := map[string]any{
		"model":    a.model,
		"messages": messages,
		"tools":    tools,
	}
	if provider.SupportTemperature("openai", a.model) {
		body["temperature"] = 0.2
	}
	var reasoning string
	if provider.SupportReasoningEffort("openai", a.model) {
		reasoning = provider.ClampReasoningLevel(provider.GetReasoningLevel(), provider.MaxReasoningLevel("openai", a.model))
		if !provider.ReasoningDisabled(reasoning) {
			body["reasoning_effort"] = reasoning
		}
	}
	result, _, err := go_pkg_http.POST[agentTypes.Output](ctx, a.httpClient, chatAPI, headers, body, "json")
	if err != nil {
		return nil, fmt.Errorf("http.POST: %w", err)
	}
	if result.Error != nil {
		return nil, fmt.Errorf("http.POST: %s", result.Error.Message)
	}

	usagelog.Append(agentTypes.SessionIDFrom(ctx), "openai", a.model, reasoning, result.Usage)
	return &result, nil
}
