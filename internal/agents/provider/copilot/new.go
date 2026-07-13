package copilot

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	oauthCopilot "github.com/pardnchiu/agenvoy/internal/agents/oauth/copilot"
	"github.com/pardnchiu/agenvoy/internal/agents/provider"
)

type Agent struct {
	httpClient *http.Client
	model      string
	Token      *oauthCopilot.Token
	Refresh    *oauthCopilot.RefreshToken
	workDir    string
}

const (
	prefix = "copilot@"
)

func New(model ...string) (*Agent, error) {
	if len(model) == 0 || !strings.HasPrefix(model[0], prefix) {
		return nil, fmt.Errorf("copilot.New: model arg required with %q prefix", prefix)
	}
	usedModel := strings.TrimPrefix(model[0], prefix)

	workDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("os.Getwd: %w", err)
	}

	token, err := oauthCopilot.Load()
	if err != nil {
		return nil, fmt.Errorf("oauth.Load: %w", err)
	}
	if token == nil {
		return nil, fmt.Errorf("copilot token missing; run `agen model add` to authenticate")
	}

	return &Agent{
		httpClient: provider.NewHTTPClient(),
		model:      usedModel,
		workDir:    workDir,
		Token:      token,
	}, nil
}

func (a *Agent) Name() string {
	return prefix + a.model
}

func (a *Agent) authHeader(ctx context.Context) (string, error) {
	refresh, err := oauthCopilot.EnsureFreshSession(ctx, a.Token, a.Refresh)
	if err != nil {
		return "", fmt.Errorf("oauth.EnsureFreshSession: %w", err)
	}
	a.Refresh = refresh
	return "Bearer " + refresh.Token, nil
}
