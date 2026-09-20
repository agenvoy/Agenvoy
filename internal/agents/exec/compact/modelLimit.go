package compact

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	go_pkg_http "github.com/pardnchiu/go-pkg/http"
)

const (
	limitAPI     = "https://llm-io.agenvoy.com/"
	limitTTL     = time.Hour
	limitTimeout = 5 * time.Second
)

type modelLimit struct {
	In  int `json:"in"`
	Out int `json:"out"`
}

var (
	limitMu  sync.RWMutex
	limitDic map[string]map[string]modelLimit
	limitAt  time.Time
)

var vendorAlias = map[string]string{
	"google":    "gemini",
	"x-ai":      "grok",
	"mistralai": "mistral",
	"anthropic": "claude",
}

var providerVendor = map[string]string{
	"openai":     "openai",
	"codex":      "openai",
	"claude":     "claude",
	"gemini":     "gemini",
	"grok":       "grok",
	"grok-oauth": "grok",
	"mistral":    "mistral",
	"deepseek":   "deepseek",
}

var vendorInModel = map[string]bool{
	"nvidia":     true,
	"openrouter": true,
}

func limitPair(modelName string) (string, string) {
	prefix, model, ok := strings.Cut(strings.TrimSpace(modelName), "@")
	if !ok || model == "" {
		return "", ""
	}

	if vendorInModel[prefix] {
		vendor, rest, ok := strings.Cut(model, "/")
		if !ok || rest == "" {
			return "", ""
		}
		vendor = strings.TrimPrefix(vendor, "~")
		if alias, ok := vendorAlias[vendor]; ok {
			vendor = alias
		}
		return vendor, rest
	}

	vendor, ok := providerVendor[prefix]
	if !ok {
		return "", ""
	}
	return vendor, model
}

func Warm(ctx context.Context) {
	limitMu.RLock()
	fresh := limitDic != nil && time.Since(limitAt) < limitTTL
	limitMu.RUnlock()
	if fresh {
		return
	}

	client := &http.Client{Timeout: limitTimeout}
	dic, status, err := go_pkg_http.GET[map[string]map[string]modelLimit](ctx, client, limitAPI, nil)
	if err != nil {
		slog.Debug("compact.Warm", slog.String("error", err.Error()))
		return
	}
	if status != http.StatusOK || len(dic) == 0 {
		slog.Debug("compact.Warm",
			slog.Int("status", status),
			slog.Int("vendors", len(dic)))
		return
	}

	limitMu.Lock()
	limitDic = dic
	limitAt = time.Now()
	limitMu.Unlock()
}

func Window(ctx context.Context, modelName string) (int, int, bool) {
	Warm(ctx)

	vendor, model := limitPair(modelName)
	if vendor == "" || model == "" {
		return 0, 0, false
	}

	limitMu.RLock()
	defer limitMu.RUnlock()
	limit, ok := limitDic[vendor][model]
	if !ok || (limit.In <= 0 && limit.Out <= 0) {
		return 0, 0, false
	}
	return limit.In, limit.Out, true
}

func lookupLimit(modelName string) (int, bool) {
	vendor, model := limitPair(modelName)
	if vendor == "" || model == "" {
		return 0, false
	}

	limitMu.RLock()
	defer limitMu.RUnlock()
	limit, ok := limitDic[vendor][model]
	if !ok || limit.In <= 0 {
		return 0, false
	}
	return limit.In, true
}
