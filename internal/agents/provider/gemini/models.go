package gemini

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
	modelsAPI = "https://generativelanguage.googleapis.com/v1beta/models"
)

func Models(ctx context.Context, config provider.Config) ([]string, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("Models: APIKey is required")
	}

	client := &http.Client{Timeout: 10 * time.Second}
	data, status, err := go_pkg_http.GET[provider.GeminiModels](ctx, client, modelsAPI+"?key="+config.APIKey, nil)
	if err != nil {
		return nil, fmt.Errorf("go_pkg_http.GET: %w", err)
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("go_pkg_http.GET: http %d", status)
	}

	ids := make([]string, 0, len(data.Models))
	for _, m := range data.Models {
		if name := strings.TrimPrefix(m.Name, "models/"); name != "" {
			ids = append(ids, name)
		}
	}
	return ids, nil
}
