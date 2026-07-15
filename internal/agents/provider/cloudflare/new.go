package cloudflare

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
	accountID  string
	gatewayID  string
}

const (
	prefix = "cloudflare@"
)

func New(model ...string) (*Agent, error) {
	if len(model) == 0 || !strings.HasPrefix(model[0], prefix) {
		return nil, fmt.Errorf("cloudflare.New: model arg required with %q prefix", prefix)
	}
	usedModel := strings.TrimPrefix(model[0], prefix)

	apiKey := keychain.Get("CLOUDFLARE_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("keychain.Get: CLOUDFLARE_API_KEY is required")
	}

	accountID := keychain.Get("CLOUDFLARE_ACCOUNT_ID")
	if accountID == "" {
		return nil, fmt.Errorf("keychain.Get: CLOUDFLARE_ACCOUNT_ID is required")
	}

	gatewayID := keychain.Get("CLOUDFLARE_GATEWAY_ID")
	if gatewayID == "" {
		gatewayID = "default"
	}

	return &Agent{
		httpClient: provider.NewHTTPClient(),
		model:      usedModel,
		apiKey:     apiKey,
		accountID:  accountID,
		gatewayID:  gatewayID,
	}, nil
}

func (a *Agent) Name() string {
	return a.model
}
