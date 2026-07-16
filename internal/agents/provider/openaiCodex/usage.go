package openaicodex

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/pardnchiu/agenvoy/internal/agents/provider"
)

const usageAPI = "https://chatgpt.com/backend-api/wham/usage"

type usageWindow struct {
	UsedPercent float64 `json:"used_percent"`
}

type usageResponse struct {
	RateLimit struct {
		PrimaryWindow   usageWindow `json:"primary_window"`
		SecondaryWindow usageWindow `json:"secondary_window"`
	} `json:"rate_limit"`
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
	if config.AccountID != "" {
		req.Header.Set("ChatGPT-Account-Id", config.AccountID)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("http.Do: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<10))
		return 0, fmt.Errorf("codex usage http %d: %s", resp.StatusCode, raw)
	}

	var usage usageResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(&usage); err != nil {
		return 0, fmt.Errorf("json.Decode: %w", err)
	}

	used := usage.RateLimit.PrimaryWindow.UsedPercent
	if usage.RateLimit.SecondaryWindow.UsedPercent > used {
		used = usage.RateLimit.SecondaryWindow.UsedPercent
	}
	return 100 - used, nil
}
