package claudeCode

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	provider "github.com/pardnchiu/go-llm-router/core"
	go_pkg_filesystem "github.com/pardnchiu/go-pkg/filesystem"
	go_pkg_http "github.com/pardnchiu/go-pkg/http"
	go_pkg_utils "github.com/pardnchiu/go-pkg/utils"
)

const (
	usageAPI          = "https://api.anthropic.com/api/oauth/usage"
	modelsAPI         = "https://api.anthropic.com/v1/models?limit=100"
	usageAPITimeout   = 5 * time.Second
	credentialService = "Claude Code-credentials"
)

var usedPattern = regexp.MustCompile(`(?m)^Current (?:session|week \(all models\)): (\d+(?:\.\d+)?)% used`)

var usageClient = &http.Client{Timeout: usageAPITimeout}

type oauthCredential struct {
	ClaudeAiOauth struct {
		AccessToken string `json:"accessToken"`
		ExpiresAt   int64  `json:"expiresAt"`
	} `json:"claudeAiOauth"`
}

type modelsResponse struct {
	Data []struct {
		ID string `json:"id"`
	} `json:"data"`
	HasMore bool   `json:"has_more"`
	LastID  string `json:"last_id"`
}

type usageWindow struct {
	Utilization *float64 `json:"utilization"`
}

type usageResponse struct {
	FiveHour *usageWindow `json:"five_hour"`
	SevenDay *usageWindow `json:"seven_day"`
}

func Usage(ctx context.Context, _ provider.Config) (float64, error) {
	if err := CheckBinary(); err != nil {
		return 0, err
	}
	value, err := apiUsage(ctx)
	if err == nil {
		return value, nil
	}
	slog.Debug("claudeCode.apiUsage", slog.String("error", err.Error()))
	return cliUsage(ctx)
}

func accessToken(ctx context.Context) (string, error) {
	var raw string
	if runtime.GOOS == "darwin" {
		out, err := exec.CommandContext(ctx, "security", "find-generic-password", "-s", credentialService, "-w").Output()
		if err != nil {
			return "", fmt.Errorf("security find-generic-password %q: %w", credentialService, err)
		}
		raw = string(out)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("os.UserHomeDir: %w", err)
		}
		text, err := go_pkg_filesystem.ReadText(filepath.Join(home, ".claude", ".credentials.json"))
		if err != nil {
			return "", fmt.Errorf("go_pkg_filesystem.ReadText: %w", err)
		}
		raw = text
	}

	var cred oauthCredential
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &cred); err != nil {
		return "", fmt.Errorf("json.Unmarshal credential: %w", err)
	}
	token := cred.ClaudeAiOauth.AccessToken
	if token == "" {
		return "", fmt.Errorf("claude credential has no access token")
	}
	if exp := cred.ClaudeAiOauth.ExpiresAt; exp > 0 && time.Now().UnixMilli() >= exp {
		return "", fmt.Errorf("claude access token expired")
	}
	return token, nil
}

func freshToken(ctx context.Context) (string, error) {
	token, err := accessToken(ctx)
	if err == nil {
		return token, nil
	}
	if _, cliErr := cliUsage(ctx); cliErr != nil {
		return "", fmt.Errorf("%w; refresh through claude /usage: %v", err, cliErr)
	}
	return accessToken(ctx)
}

func oauthHeaders(token string) map[string]string {
	return map[string]string{
		"Authorization":     "Bearer " + token,
		"anthropic-beta":    "oauth-2025-04-20",
		"anthropic-version": "2023-06-01",
	}
}

func listModels(ctx context.Context) ([]string, error) {
	token, err := freshToken(ctx)
	if err != nil {
		return nil, err
	}

	var list []string
	api := modelsAPI
	for {
		resp, status, err := go_pkg_http.GET[modelsResponse](ctx, usageClient, api, oauthHeaders(token))
		if err != nil {
			return nil, fmt.Errorf("go_pkg_http.GET: %w", err)
		}
		if status != http.StatusOK {
			return nil, fmt.Errorf("claude models api: http %d", status)
		}
		for _, m := range resp.Data {
			if id := strings.TrimSpace(m.ID); id != "" {
				list = append(list, id)
			}
		}
		if !resp.HasMore || resp.LastID == "" {
			break
		}
		api = modelsAPI + "&after_id=" + url.QueryEscape(resp.LastID)
	}
	if len(list) == 0 {
		return nil, fmt.Errorf("claude models api returned no models")
	}
	return list, nil
}

func apiUsage(ctx context.Context) (float64, error) {
	token, err := accessToken(ctx)
	if err != nil {
		return 0, err
	}
	resp, status, err := go_pkg_http.GET[usageResponse](ctx, usageClient, usageAPI, oauthHeaders(token))
	if err != nil {
		return 0, fmt.Errorf("go_pkg_http.GET: %w", err)
	}
	if status != http.StatusOK {
		return 0, fmt.Errorf("claude usage api: http %d", status)
	}

	used, found := 0.0, false
	for _, w := range []*usageWindow{resp.FiveHour, resp.SevenDay} {
		if w != nil && w.Utilization != nil {
			used, found = max(used, *w.Utilization), true
		}
	}
	if !found {
		return 0, fmt.Errorf("claude usage api: no five_hour or seven_day utilization")
	}
	return 100 - used, nil
}

func cliUsage(ctx context.Context) (float64, error) {
	cmd := exec.CommandContext(ctx, "claude", "-p", "/usage",
		"--safe-mode", "--tools", "", "--strict-mcp-config", "--no-session-persistence")
	cmd.Dir = os.TempDir()
	stderr := &limitedBuffer{}
	cmd.Stderr = stderr
	raw, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("claude /usage: %w: %s", err, stderr.String())
	}

	text := string(raw)
	matches := usedPattern.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		return 0, fmt.Errorf("claude /usage: no subscription limits in output: %s", go_pkg_utils.TruncateString(strings.TrimSpace(text), 200))
	}
	used := 0.0
	for _, m := range matches {
		value, err := strconv.ParseFloat(m[1], 64)
		if err != nil {
			return 0, fmt.Errorf("strconv.ParseFloat %q: %w", m[1], err)
		}
		used = max(used, value)
	}
	return 100 - used, nil
}
