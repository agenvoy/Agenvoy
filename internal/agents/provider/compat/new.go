package compat

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/pardnchiu/agenvoy/internal/agents/provider"
)

type Agent struct {
	httpClient *http.Client
	model      string
	baseURL    string
	apiKey     string
}

const (
	defaultBaseURL = "http://localhost:11434/v1"
)

func New(config provider.Config) (*Agent, error) {
	if config.Model == "" {
		return nil, fmt.Errorf("compat.New: Model is required")
	}

	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	baseURL = strings.TrimRight(baseURL, "/")

	return &Agent{
		httpClient: &http.Client{Timeout: 10 * time.Minute},
		model:      config.Model,
		baseURL:    baseURL,
		apiKey:     config.APIKey,
	}, nil
}

func (a *Agent) Name() string {
	return a.model
}
