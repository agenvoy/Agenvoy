package openai

import (
	"context"
	"fmt"

	"github.com/pardnchiu/agenvoy/internal/agents/provider"
	copilotResponse "github.com/pardnchiu/agenvoy/internal/agents/provider/copilot/response"
	go_pkg_http "github.com/pardnchiu/go-pkg/http"
)

const (
	chatAPI      = "https://api.openai.com/v1/chat/completions"
	responsesAPI = "https://api.openai.com/v1/responses"
)

func (a *Agent) Send(ctx context.Context, messages []provider.Message, tools []provider.Tool, reasoning string) (*provider.Output, int, error) {
	headers := map[string]string{
		"Authorization": "Bearer " + a.apiKey,
		"Content-Type":  "application/json",
	}

	if provider.ResponsesAPI("openai", a.model) {
		var instructions string
		nonSystem := make([]provider.Message, 0, len(messages))
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

		effort := provider.ClampReasoningLevel(reasoning, provider.MaxReasoningLevel("openai", a.model))
		body := map[string]any{
			"model":        a.model,
			"input":        copilotResponse.ConvertInput(nonSystem),
			"tools":        copilotResponse.ConvertTools(tools),
			"instructions": instructions,
			"store":        false,
		}
		if !provider.ReasoningDisabled(effort) {
			body["reasoning"] = map[string]any{"effort": effort, "summary": "auto"}
		}

		result, code, err := go_pkg_http.POST[copilotResponse.Output](ctx, a.httpClient, responsesAPI, headers, body, "json")
		if err != nil {
			return nil, code, err
		}
		if result.Error != nil {
			return nil, code, fmt.Errorf("%s", result.Error.Message)
		}

		out := copilotResponse.ConvertOutput(result)
		return &out, code, nil
	}

	body := map[string]any{
		"model":    a.model,
		"messages": messages,
		"tools":    tools,
	}
	if provider.SupportTemperature("openai", a.model) {
		body["temperature"] = 0.2
	}
	if provider.SupportReasoningEffort("openai", a.model) {
		effort := provider.ClampReasoningLevel(reasoning, provider.MaxReasoningLevel("openai", a.model))
		if !provider.ReasoningDisabled(effort) {
			body["reasoning_effort"] = effort
		}
	}
	result, code, err := go_pkg_http.POST[provider.Output](ctx, a.httpClient, chatAPI, headers, body, "json")
	if err != nil {
		return nil, code, err
	}
	if result.Error != nil {
		return nil, code, fmt.Errorf("http.POST: %s", result.Error.Message)
	}

	return &result, code, nil
}
