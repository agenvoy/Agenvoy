package compat

import (
	"context"
	"fmt"

	"github.com/pardnchiu/agenvoy/internal/agents/provider"
	go_pkg_http "github.com/pardnchiu/go-pkg/http"
)

func (a *Agent) Send(ctx context.Context, messages []provider.Message, tools []provider.Tool, reasoning string) (*provider.Output, int, error) {
	chatAPI := a.baseURL + "/chat/completions"

	headers := map[string]string{
		"Content-Type": "application/json",
	}
	if a.apiKey != "" {
		headers["Authorization"] = "Bearer " + a.apiKey
	}

	result, code, err := go_pkg_http.POST[provider.Output](ctx, a.httpClient, chatAPI, headers, map[string]any{
		"model":       a.model,
		"messages":    messages,
		"temperature": 0.2,
		"tools":       tools,
	}, "json")
	if err != nil {
		return nil, code, err
	}
	if result.Error != nil {
		return nil, code, fmt.Errorf("%s", result.Error.Message)
	}
	return &result, code, nil
}
