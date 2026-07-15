package openaicodex

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	oauthCodex "github.com/pardnchiu/agenvoy/internal/agents/oauth/codex"
)

const prefix = "codex@"

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

	token *oauthCodex.StoredToken
}

func New(model ...string) (*Agent, error) {
	if len(model) == 0 || !strings.HasPrefix(model[0], prefix) {
		return nil, fmt.Errorf("openaicodex.New: model arg required with %q prefix", prefix)
	}
	usedModel := strings.TrimPrefix(model[0], prefix)

	token, err := oauthCodex.Load()
	if err != nil {
		return nil, fmt.Errorf("oauthCodex.Load: %w", err)
	}
	if token == nil {
		return nil, fmt.Errorf("codex token missing; run `agen model add` to authenticate")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	token, err = oauthCodex.EnsureFresh(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("codex token expired and refresh failed: %w; run `agen model add` to re-authenticate", err)
	}

	return &Agent{
		httpClient: newHTTPClient(),
		model:      usedModel,
		token:      token,
	}, nil
}

func (a *Agent) Name() string {
	return prefix + a.model
}

func (a *Agent) authHeader(ctx context.Context) (string, error) {
	token, err := oauthCodex.EnsureFresh(ctx, a.token)
	if err != nil {
		return "", fmt.Errorf("oauthCodex.EnsureFresh: %w", err)
	}
	a.token = token
	return "Bearer " + token.AccessToken, nil
}
