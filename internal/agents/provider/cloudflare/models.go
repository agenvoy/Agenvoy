package cloudflare

import (
	"context"
	"fmt"
	"net/http"
	"time"

	go_pkg_http "github.com/pardnchiu/go-pkg/http"

	"github.com/pardnchiu/agenvoy/internal/agents/provider"
)

func Models(ctx context.Context, config provider.Config) ([]string, error) {
	if config.APIKey == "" || config.AccountID == "" {
		return nil, fmt.Errorf("Models: APIKey and AccountID are required")
	}

	endpoint := runAPI + config.AccountID + "/ai/models/search"
	client := &http.Client{Timeout: 10 * time.Second}
	data, status, err := go_pkg_http.GET[provider.CloudFlareModels](ctx, client, endpoint, map[string]string{
		"Authorization": "Bearer " + config.APIKey,
	})
	if err != nil {
		return nil, fmt.Errorf("go_pkg_http.GET: %w", err)
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("go_pkg_http.GET: http %d", status)
	}

	ids := make([]string, 0, len(data.Result))
	for _, m := range data.Result {
		if m.Name != "" && m.Task.Name == "Text Generation" {
			ids = append(ids, m.Name)
		}
	}
	return ids, nil
}
