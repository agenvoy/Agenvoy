package openrouter

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/pardnchiu/go-pkg/filesystem/keychain"

	"github.com/pardnchiu/agenvoy/internal/agents/provider"
)

type Agent struct {
	httpClient *http.Client
	model      string
	apiKey     string
}

const prefix = "openrouter@"

func New(model ...string) (*Agent, error) {
	if len(model) == 0 || !strings.HasPrefix(model[0], prefix) {
		return nil, fmt.Errorf("openrouter.New: model arg required with %q prefix", prefix)
	}
	usedModel := strings.TrimPrefix(model[0], prefix)

	apiKey := keychain.Get(keychainKey)
	if apiKey == "" {
		return nil, fmt.Errorf("keychain.Get: %s is required", keychainKey)
	}

	return &Agent{
		httpClient: provider.NewHTTPClient(),
		model:      usedModel,
		apiKey:     apiKey,
	}, nil
}

func (a *Agent) Name() string {
	return a.model
}
