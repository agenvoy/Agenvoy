package claude

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/pardnchiu/agenvoy/internal/agents/provider"
)

type Agent struct {
	httpClient *http.Client
	model      string
	apiKey     string
}

const (
	Prefix = "claude@"
)

func New(config provider.Config) (*Agent, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("claude.New: APIKey is required")
	}

	return &Agent{
		httpClient: provider.NewHTTPClient(),
		model:      config.Model,
		apiKey:     config.APIKey,
	}, nil
}

func (a *Agent) Name() string {
	return a.model
}

func (a *Agent) maxOutputTokens() int {
	switch {
	case strings.HasPrefix(a.model, "claude-opus-4-6"),
		strings.HasPrefix(a.model, "claude-opus-4-7"):
		return 128000
	case strings.HasPrefix(a.model, "claude-opus-4-1-"),
		strings.HasPrefix(a.model, "claude-opus-4-2"):
		return 32000
	}
	return 64000
}
