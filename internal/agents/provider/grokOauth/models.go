package grokoauth

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	go_pkg_http "github.com/pardnchiu/go-pkg/http"

	"github.com/pardnchiu/agenvoy/internal/agents/provider"
)

const (
	modelsAPI = "https://api.x.ai/v1/models"
)

func Models(ctx context.Context, config provider.Config) ([]string, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("Models: APIKey is required")
	}

	client := &http.Client{Timeout: 10 * time.Second}
	data, status, err := go_pkg_http.GET[provider.Models](ctx, client, modelsAPI, map[string]string{
		"Authorization": "Bearer " + config.APIKey,
	})
	if err != nil {
		return nil, fmt.Errorf("go_pkg_http.GET: %w", err)
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("go_pkg_http.GET: http %d", status)
	}

	ids := make([]string, 0, len(data.Data))
	for _, m := range data.Data {
		if id := strings.TrimSpace(m.ID); id != "" {
			ids = append(ids, id)
		}
	}
	return ids, nil
}
