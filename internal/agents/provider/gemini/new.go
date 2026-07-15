package gemini

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/pardnchiu/agenvoy/internal/agents/provider"
)

type Agent struct {
	httpClient *http.Client
	model      string
	apiKey     string
	cacheMu    sync.Mutex
	cacheStore map[string]*geminiCacheEntry
}

const (
	Prefix = "gemini@"
)

func New(config provider.Config) (*Agent, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("gemini.New: APIKey is required")
	}

	return &Agent{
		httpClient: provider.NewHTTPClient(),
		model:      config.Model,
		apiKey:     config.APIKey,
		cacheStore: make(map[string]*geminiCacheEntry),
	}, nil
}

func (a *Agent) Name() string {
	return a.model
}
