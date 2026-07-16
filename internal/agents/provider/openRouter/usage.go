package openrouter

import (
	"context"
	"fmt"
	"net/http"
	"time"

	go_pkg_http "github.com/pardnchiu/go-pkg/http"

	"github.com/pardnchiu/agenvoy/internal/agents/provider"
)

const creditsAPI = "https://openrouter.ai/api/v1/credits"

type creditsResponse struct {
	Data struct {
		TotalCredits float64 `json:"total_credits"`
		TotalUsage   float64 `json:"total_usage"`
	} `json:"data"`
}

func Usage(ctx context.Context, config provider.Config) (float64, error) {
	if config.APIKey == "" {
		return 0, fmt.Errorf("Usage: APIKey is required")
	}

	client := &http.Client{Timeout: 10 * time.Second}
	data, status, err := go_pkg_http.GET[creditsResponse](ctx, client, creditsAPI, map[string]string{
		"Authorization": "Bearer " + config.APIKey,
	})
	if err != nil {
		return 0, fmt.Errorf("go_pkg_http.GET: %w", err)
	}
	if status != http.StatusOK {
		return 0, fmt.Errorf("go_pkg_http.GET: http %d", status)
	}

	return data.Data.TotalCredits - data.Data.TotalUsage, nil
}
