package openaicodex

import (
	"context"
	"fmt"
	"net/http"
	"time"

	oauthCodex "github.com/pardnchiu/agenvoy/internal/agents/oauth/codex"
	"github.com/pardnchiu/agenvoy/internal/agents/provider"
)

const Prefix = "codex@"

func newHTTPClient() *http.Client {
	base, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		base = &http.Transport{}
	}
	transport := base.Clone()
	transport.ResponseHeaderTimeout = 15 * time.Second
	return &http.Client{
		Timeout:   10 * time.Minute,
		Transport: transport,
	}
}

type Agent struct {
	httpClient *http.Client
	model      string

	token *provider.CodexToken
}

func New(config provider.Config) (*Agent, error) {
	token, ok := config.Token.(*provider.CodexToken)
	if !ok || token == nil {
		return nil, fmt.Errorf("openaicodex.New: Token is required")
	}

	return &Agent{
		httpClient: newHTTPClient(),
		model:      config.Model,
		token:      token,
	}, nil
}

func (a *Agent) Name() string {
	return Prefix + a.model
}

func (a *Agent) authHeader(ctx context.Context) (string, error) {
	token, err := oauthCodex.EnsureFresh(ctx, a.token)
	if err != nil {
		return "", fmt.Errorf("oauthCodex.EnsureFresh: %w", err)
	}
	a.token = token
	return "Bearer " + token.AccessToken, nil
}
