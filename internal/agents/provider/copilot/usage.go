package copilot

import (
	"context"
	"fmt"
	"net/http"
	"time"

	go_pkg_http "github.com/pardnchiu/go-pkg/http"

	"github.com/pardnchiu/agenvoy/internal/agents/provider"
)

const userAPI = "https://api.github.com/copilot_internal/user"

type quotaSnapshot struct {
	Entitlement      int     `json:"entitlement"`
	Remaining        float64 `json:"remaining"`
	PercentRemaining float64 `json:"percent_remaining"`
}

type userResponse struct {
	QuotaSnapshots struct {
		PremiumInteractions quotaSnapshot `json:"premium_interactions"`
		Chat                quotaSnapshot `json:"chat"`
	} `json:"quota_snapshots"`
}

func Usage(ctx context.Context, config provider.Config) (float64, error) {
	token, ok := config.Token.(*provider.CopilotToken)
	if !ok || token == nil {
		return 0, fmt.Errorf("Usage: Token is required")
	}

	client := &http.Client{Timeout: 10 * time.Second}
	data, status, err := go_pkg_http.GET[userResponse](ctx, client, userAPI, map[string]string{
		"Authorization":  "token " + token.AccessToken,
		"Editor-Version": "vscode/1.96.2",
	})
	if err != nil {
		return 0, fmt.Errorf("go_pkg_http.GET: %w", err)
	}
	if status != http.StatusOK {
		return 0, fmt.Errorf("go_pkg_http.GET: http %d", status)
	}

	snap := data.QuotaSnapshots.Chat
	if snap.Entitlement == 0 {
		snap = data.QuotaSnapshots.PremiumInteractions
	}
	return snap.PercentRemaining, nil
}
