package deepseek

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
	Prefix = "deepseek@"
)

func New(config provider.Config) (*Agent, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("deepseek.New: APIKey is required")
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
