package cloudflare

import (
	"context"
	"fmt"
	"strings"

	agentTypes "github.com/pardnchiu/agenvoy/internal/agents/types"
	usagelog "github.com/pardnchiu/agenvoy/internal/session/usage"
	toolTypes "github.com/pardnchiu/agenvoy/internal/tools/types"
	go_pkg_http "github.com/pardnchiu/go-pkg/http"
)

const (
	runAPI = "https://api.cloudflare.com/client/v4/accounts/"
)

func (a *Agent) endpoint() string {
	return runAPI + a.accountID + "/ai/run"
}

func flattenContent(c any) string {
	switch v := c.(type) {
	case string:
		return v
	case []any:
		var buf strings.Builder
		for _, item := range v {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			t, _ := m["type"].(string)
			switch t {
			case "text", "input_text", "output_text":
				text, _ := m["text"].(string)
				if buf.Len() > 0 {
					buf.WriteByte('\n')
				}
				buf.WriteString(text)
			}
		}
		return buf.String()
	}
	return fmt.Sprintf("%v", c)
}

func (a *Agent) Send(ctx context.Context, messages []agentTypes.Message, tools []toolTypes.Tool) (*agentTypes.Output, error) {
	var merged []agentTypes.Message
	var systemParts []string
	for _, m := range messages {
		if m.Role == "system" {
			s := flattenContent(m.Content)
			if s != "" {
				systemParts = append(systemParts, s)
			}
		} else {
			merged = append(merged, agentTypes.Message{
				Role:       m.Role,
				Content:    flattenContent(m.Content),
				ToolCalls:  m.ToolCalls,
				ToolCallID: m.ToolCallID,
			})
		}
	}
	if len(systemParts) > 0 {
		merged = append([]agentTypes.Message{{Role: "system", Content: strings.Join(systemParts, "\n\n")}}, merged...)
	}

	input := map[string]any{
		"messages":   merged,
		"max_tokens": 4096,
	}
	if len(tools) > 0 {
		input["tools"] = tools
	}

	headers := map[string]string{
		"Authorization":      "Bearer " + a.apiKey,
		"Content-Type":       "application/json",
		"cf-aig-gateway-id": a.gatewayID,
	}

	resp, _, err := go_pkg_http.POST[response](ctx, a.httpClient, a.endpoint(), headers, map[string]any{
		"model": a.model,
		"input": input,
	}, "json")
	if err != nil {
		return nil, fmt.Errorf("http.POST: %w", err)
	}
	if len(resp.Errors) > 0 {
		return nil, fmt.Errorf("http.POST: %s", resp.Errors[0].Message)
	}

	usagelog.Append(agentTypes.SessionIDFrom(ctx), "cloudflare", a.model, "", resp.Result.Usage)
	return &resp.Result, nil
}
