package compact

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	oauthCopilot "github.com/pardnchiu/go-llm-router/core/oauth/copilot"
	go_pkg_http "github.com/pardnchiu/go-pkg/http"
)

const (
	copilotModelsAPI  = "https://api.githubcopilot.com/models"
	copilotEditor     = "vscode/1.95.0"
	copilotLimitTTL   = 24 * time.Hour
	copilotAPITimeout = 10 * time.Second
)

type copilotModels struct {
	Data []struct {
		ID           string `json:"id"`
		Capabilities struct {
			Type   string `json:"type"`
			Limits struct {
				MaxContextWindowTokens int `json:"max_context_window_tokens"`
			} `json:"limits"`
		} `json:"capabilities"`
	} `json:"data"`
}

var (
	copilotMu     sync.RWMutex
	copilotDic    map[string]int
	copilotAt     time.Time
	copilotFlying bool
)

func copilotLimit(model string) (int, bool) {
	copilotMu.RLock()
	defer copilotMu.RUnlock()
	in, ok := copilotDic[model]
	return in, ok
}

func WarmCopilot() {
	copilotMu.Lock()
	if copilotFlying || (copilotDic != nil && time.Since(copilotAt) < copilotLimitTTL) {
		copilotMu.Unlock()
		return
	}
	copilotFlying = true
	copilotMu.Unlock()

	go func() {
		defer func() {
			copilotMu.Lock()
			copilotFlying = false
			copilotMu.Unlock()
		}()

		dic, err := fetchCopilotLimit()
		if err != nil {
			slog.Debug("compact.WarmCopilot", slog.String("error", err.Error()))
			return
		}
		copilotMu.Lock()
		copilotDic = dic
		copilotAt = time.Now()
		copilotMu.Unlock()
	}()
}

func fetchCopilotLimit() (map[string]int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), copilotAPITimeout)
	defer cancel()

	token, err := oauthCopilot.Load()
	if err != nil {
		return nil, err
	}
	if token == nil {
		return nil, fmt.Errorf("copilot token missing; run `agen model add` to authenticate")
	}
	session, err := oauthCopilot.EnsureFreshSession(ctx, token, nil)
	if err != nil {
		return nil, err
	}

	data, status, err := go_pkg_http.GET[copilotModels](ctx, &http.Client{Timeout: copilotAPITimeout}, copilotModelsAPI, map[string]string{
		"Authorization":  "Bearer " + session.Token,
		"Editor-Version": copilotEditor,
	})
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("http %d", status)
	}

	dic := make(map[string]int, len(data.Data))
	for _, one := range data.Data {
		id := strings.TrimSpace(one.ID)
		limits := one.Capabilities.Limits
		if id == "" || one.Capabilities.Type != "chat" || limits.MaxContextWindowTokens <= 0 {
			continue
		}
		dic[id] = limits.MaxContextWindowTokens
	}
	if len(dic) == 0 {
		return nil, fmt.Errorf("no chat model reported max_context_window_tokens")
	}
	return dic, nil
}
