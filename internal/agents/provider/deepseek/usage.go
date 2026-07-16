package deepseek

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	go_pkg_http "github.com/pardnchiu/go-pkg/http"

	"github.com/pardnchiu/agenvoy/internal/agents/provider"
)

const balanceAPI = "https://api.deepseek.com/user/balance"

type balanceResponse struct {
	IsAvailable  bool `json:"is_available"`
	BalanceInfos []struct {
		Currency     string `json:"currency"`
		TotalBalance string `json:"total_balance"`
	} `json:"balance_infos"`
}

func Usage(ctx context.Context, config provider.Config) (float64, error) {
	if config.APIKey == "" {
		return 0, fmt.Errorf("Usage: APIKey is required")
	}

	client := &http.Client{Timeout: 10 * time.Second}
	data, status, err := go_pkg_http.GET[balanceResponse](ctx, client, balanceAPI, map[string]string{
		"Authorization": "Bearer " + config.APIKey,
	})
	if err != nil {
		return 0, fmt.Errorf("go_pkg_http.GET: %w", err)
	}
	if status != http.StatusOK {
		return 0, fmt.Errorf("go_pkg_http.GET: http %d", status)
	}
	if len(data.BalanceInfos) == 0 {
		return 0, fmt.Errorf("no balance_infos returned")
	}

	balance, err := strconv.ParseFloat(data.BalanceInfos[0].TotalBalance, 64)
	if err != nil {
		return 0, fmt.Errorf("strconv.ParseFloat: %w", err)
	}
	return balance, nil
}
