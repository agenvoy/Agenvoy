package copilot

import (
	"context"
	"fmt"

	"github.com/pardnchiu/agenvoy/internal/agents/provider"
	copilotResponse "github.com/pardnchiu/agenvoy/internal/agents/provider/copilot/response"
	go_pkg_http "github.com/pardnchiu/go-pkg/http"
)

const (
	chatAPI      = "https://api.githubcopilot.com/chat/completions"
	responsesAPI = "https://api.githubcopilot.com/responses"
)

func (a *Agent) Send(ctx context.Context, messages []provider.Message, tools []provider.Tool, reasoning string) (*provider.Output, error) {
	auth, err := a.authHeader(ctx)
	if err != nil {
		return nil, fmt.Errorf("a.authHeader: %w", err)
	}

	headers := map[string]string{
		"Authorization":  auth,
		"Editor-Version": "vscode/1.95.0",
	}

	if provider.ResponsesAPI("copilot", a.model) {
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

		effort := provider.ClampReasoningLevel(reasoning, provider.MaxReasoningLevel("copilot", a.model))
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

		result, _, err := go_pkg_http.POST[copilotResponse.Output](ctx, a.httpClient, responsesAPI, headers, body, "json")
		if err != nil {
			return nil, fmt.Errorf("http.POST: %w", err)
		}
		if result.Error != nil {
			return nil, fmt.Errorf("http.POST: %s", result.Error.Message)
		}
		out := copilotResponse.ConvertOutput(result)
		return &out, nil
	}

	body := map[string]any{
		"model":    a.model,
		"messages": messages,
		"tools":    tools,
	}
	if provider.SupportTemperature("copilot", a.model) {
		body["temperature"] = 0.2
	}
	if provider.SupportReasoningEffort("copilot", a.model) {
		effort := provider.ClampReasoningLevel(reasoning, provider.MaxReasoningLevel("copilot", a.model))
		if !provider.ReasoningDisabled(effort) {
			body["reasoning_effort"] = effort
		}
	}

	result, _, err := go_pkg_http.POST[provider.Output](ctx, a.httpClient, chatAPI, headers, body, "json")
	if err != nil {
		return nil, fmt.Errorf("http.POST: %w", err)
	}
	if result.Error != nil {
		return nil, fmt.Errorf("http.POST: %s", result.Error.Message)
	}

	return &result, nil
}
