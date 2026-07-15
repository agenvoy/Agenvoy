package grok

import (
	"fmt"
	"net/http"

	"github.com/pardnchiu/agenvoy/internal/agents/provider"
)

type Agent struct {
	httpClient *http.Client
	model      string
	apiKey     string
}

const (
	Prefix = "grok@"
)

func New(config provider.Config) (*Agent, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("grok.New: APIKey is required")
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
