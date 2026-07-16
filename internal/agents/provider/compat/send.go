package compat

import (
	"context"
	"fmt"

	"github.com/pardnchiu/agenvoy/internal/agents/provider"
	toolTypes "github.com/pardnchiu/agenvoy/internal/tools/types"
	go_pkg_http "github.com/pardnchiu/go-pkg/http"
)

func (a *Agent) Send(ctx context.Context, messages []provider.Message, tools []toolTypes.Tool) (*provider.Output, error) {
	chatAPI := a.baseURL + "/chat/completions"

	headers := map[string]string{
		"Content-Type": "application/json",
	}
	if a.apiKey != "" {
		headers["Authorization"] = "Bearer " + a.apiKey
	}

	result, _, err := go_pkg_http.POST[provider.Output](ctx, a.httpClient, chatAPI, headers, map[string]any{
		"model":       a.model,
		"messages":    messages,
		"temperature": 0.2,
		"tools":       tools,
	}, "json")
	if err != nil {
		return nil, fmt.Errorf("http.POST: %w", err)
	}
	if result.Error != nil {
		return nil, fmt.Errorf("http.POST: %s", result.Error.Message)
	}

	return &result, nil
}
