package grokoauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/pardnchiu/agenvoy/internal/agents/provider"
)

const (
	usageAPI            = "https://cli-chat-proxy.grok.com/v1/billing?format=credits"
	xaiTokenAuth        = "xai-grok-cli"
	xaiClientVersion    = "1.0.0"
	xaiClientModeHeader = "interactive"
)

type usageResponse struct {
	Config struct {
		CreditUsagePercent float64 `json:"creditUsagePercent"`
	} `json:"config"`
}

func Usage(ctx context.Context, config provider.Config) (float64, error) {
	if config.APIKey == "" {
		return 0, fmt.Errorf("Usage: APIKey is required")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, usageAPI, nil)
	if err != nil {
		return 0, fmt.Errorf("http.NewRequestWithContext: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+config.APIKey)
	req.Header.Set("X-XAI-Token-Auth", xaiTokenAuth)
	req.Header.Set("x-grok-client-version", xaiClientVersion)
	req.Header.Set("x-grok-client-mode", xaiClientModeHeader)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("http.Do: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<10))
		return 0, fmt.Errorf("grok usage http %d: %s", resp.StatusCode, raw)
	}

	var usage usageResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(&usage); err != nil {
		return 0, fmt.Errorf("json.Decode: %w", err)
	}

	return 100 - usage.Config.CreditUsagePercent, nil
}
