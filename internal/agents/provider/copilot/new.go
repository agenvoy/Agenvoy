package copilot

import (
	"context"
	"fmt"
	"net/http"

	oauthCopilot "github.com/pardnchiu/agenvoy/internal/agents/oauth/copilot"
	"github.com/pardnchiu/agenvoy/internal/agents/provider"
)

type Agent struct {
	httpClient *http.Client
	model      string
	Token      *provider.CopilotToken
	Refresh    *provider.CopilotRefreshToken
}

const (
	Prefix = "copilot@"
)

func New(config provider.Config) (*Agent, error) {
	token, ok := config.Token.(*provider.CopilotToken)
	if !ok || token == nil {
		return nil, fmt.Errorf("copilot.New: Token is required")
	}

	return &Agent{
		httpClient: provider.NewHTTPClient(),
		model:      config.Model,
		Token:      token,
	}, nil
}

func (a *Agent) Name() string {
	return Prefix + a.model
}

func (a *Agent) authHeader(ctx context.Context) (string, error) {
	refresh, err := oauthCopilot.EnsureFreshSession(ctx, a.Token, a.Refresh)
	if err != nil {
		return "", fmt.Errorf("oauth.EnsureFreshSession: %w", err)
	}
	a.Refresh = refresh
	return "Bearer " + refresh.Token, nil
}
