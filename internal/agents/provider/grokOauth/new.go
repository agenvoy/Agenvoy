package grokoauth

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	oauthGrok "github.com/pardnchiu/agenvoy/internal/agents/oauth/grok"
)

const prefix = "grok-oauth@"

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
	workDir    string

	token *oauthGrok.StoredToken
}

func New(model ...string) (*Agent, error) {
	if len(model) == 0 || !strings.HasPrefix(model[0], prefix) {
		return nil, fmt.Errorf("grokoauth.New: model arg required with %q prefix", prefix)
	}
	usedModel := strings.TrimPrefix(model[0], prefix)

	workDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("os.Getwd: %w", err)
	}

	token, err := oauthGrok.Load()
	if err != nil {
		return nil, fmt.Errorf("oauthGrokOauth.Load: %w", err)
	}
	if token == nil {
		return nil, fmt.Errorf("grok-oauth token missing; run `agen model add` to authenticate")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	token, err = oauthGrok.EnsureFresh(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("grok-oauth token expired and refresh failed: %w; run `agen model add` to re-authenticate", err)
	}

	return &Agent{
		httpClient: newHTTPClient(),
		model:      usedModel,
		workDir:    workDir,
		token:      token,
	}, nil
}

func (a *Agent) Name() string {
	return prefix + a.model
}

func (a *Agent) authHeader(ctx context.Context) (string, error) {
	token, err := oauthGrok.EnsureFresh(ctx, a.token)
	if err != nil {
		return "", fmt.Errorf("oauthGrokOauth.EnsureFresh: %w", err)
	}
	a.token = token
	return "Bearer " + token.AccessToken, nil
}
