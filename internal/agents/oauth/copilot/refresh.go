package oauthCopilot

import (
	"context"
	"fmt"
	"net/http"
	"time"

	go_pkg_http "github.com/pardnchiu/go-pkg/http"

	"github.com/pardnchiu/agenvoy/internal/agents/provider"
)

const (
	copilotTokenAPI = "https://api.github.com/copilot_internal/v2/token"
)

func EnsureFreshSession(ctx context.Context, token *provider.CopilotToken, refresh *provider.CopilotRefreshToken) (*provider.CopilotRefreshToken, error) {
	if refresh != nil && time.Now().Unix() < refresh.ExpiresAt-60 {
		return refresh, nil
	}

	next, code, err := go_pkg_http.GET[provider.CopilotRefreshToken](ctx, nil, copilotTokenAPI, map[string]string{
		"Authorization":  "token " + token.AccessToken,
		"Accept":         "application/json",
		"Editor-Version": "vscode/1.95.0",
	})
	if err != nil {
		return nil, fmt.Errorf("http.GET: %w", err)
	}
	if code == http.StatusUnauthorized {
		return nil, fmt.Errorf("copilot token expired; run `agen model add` to re-authenticate")
	}
	if code == http.StatusForbidden || code == http.StatusNotFound {
		return nil, fmt.Errorf("http.GET: token refresh failed")
	}
	if code != http.StatusOK {
		return nil, fmt.Errorf("http.GET: %d", code)
	}

	return &next, nil
}
