package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const protocolVersion = "2024-11-05"

type Client interface {
	List(ctx context.Context) ([]Tool, error)
	Call(ctx context.Context, name string, args map[string]any) (string, error)
	Close() error
}

type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`
}

func newClient(ctx context.Context, name string, cfg ServerConfig) (Client, error) {
	expanded := cfg.Expand()
	switch {
	case expanded.IsHTTP():
		return &HttpClient{
			url:        strings.TrimSpace(cfg.URL),
			headers:    cfg.Headers,
			httpClient: &http.Client{Timeout: 60 * time.Second},
		}, nil
	case expanded.IsStdio():
		return newStdioClient(ctx, expanded)
	default:
		return nil, fmt.Errorf("server %q has neither command nor url", name)
	}
}

type Result struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	IsError bool `json:"isError"`
}

func extractText(raw json.RawMessage) (string, error) {
	if len(raw) == 0 {
		return "", nil
	}

	var result Result
	if err := json.Unmarshal(raw, &result); err != nil {
		return string(raw), nil
	}

	var sb strings.Builder
	for _, c := range result.Content {
		if c.Type == "text" || c.Text != "" {
			if sb.Len() > 0 {
				sb.WriteString("\n")
			}
			sb.WriteString(c.Text)
		}
	}
	if result.IsError {
		if sb.Len() == 0 {
			return "", fmt.Errorf("tool error")
		}
		return "", fmt.Errorf("tool error: %s", sb.String())
	}
	if sb.Len() == 0 {
		return string(raw), nil
	}
	return sb.String(), nil
}
