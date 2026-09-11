package probe

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	go_pkg_http "github.com/pardnchiu/go-pkg/http"

	provider "github.com/pardnchiu/go-llm-router/core"
	"github.com/pardnchiu/go-llm-router/core/claude"
	"github.com/pardnchiu/go-llm-router/core/cloudflare"
	"github.com/pardnchiu/go-llm-router/core/copilot"
	"github.com/pardnchiu/go-llm-router/core/deepseek"
	"github.com/pardnchiu/go-llm-router/core/gemini"
	"github.com/pardnchiu/go-llm-router/core/grok"
	grokoauth "github.com/pardnchiu/go-llm-router/core/grokOauth"
	"github.com/pardnchiu/go-llm-router/core/mistral"
	"github.com/pardnchiu/go-llm-router/core/nvidia"
	openrouter "github.com/pardnchiu/go-llm-router/core/openRouter"
	"github.com/pardnchiu/go-llm-router/core/openai"
	openaicodex "github.com/pardnchiu/go-llm-router/core/openaiCodex"

	agentKeychain "github.com/pardnchiu/agenvoy/internal/agents/keychain"
)

type listFn func(context.Context, provider.Config) ([]string, error)

func lookup(prov string) listFn {
	filter := provider.ModelFilter{TextOnly: true}
	switch prov {
	case "openai":
		return func(ctx context.Context, cfg provider.Config) ([]string, error) {
			return openai.Models(ctx, cfg, filter)
		}
	case "codex":
		return func(ctx context.Context, cfg provider.Config) ([]string, error) {
			return openaicodex.Models(ctx, cfg, filter)
		}
	case "claude":
		return func(ctx context.Context, cfg provider.Config) ([]string, error) {
			return claude.Models(ctx, cfg, filter)
		}
	case "gemini":
		return func(ctx context.Context, cfg provider.Config) ([]string, error) {
			return gemini.Models(ctx, cfg, filter)
		}
	case "grok":
		return func(ctx context.Context, cfg provider.Config) ([]string, error) {
			return grok.Models(ctx, cfg, filter)
		}
	case "grok-oauth":
		return func(ctx context.Context, cfg provider.Config) ([]string, error) {
			return grokoauth.Models(ctx, cfg, filter)
		}
	case "copilot":
		return func(ctx context.Context, cfg provider.Config) ([]string, error) {
			return copilot.Models(ctx, cfg, filter)
		}
	case "deepseek":
		return func(ctx context.Context, cfg provider.Config) ([]string, error) {
			return deepseek.Models(ctx, cfg, filter)
		}
	case "mistral":
		return func(ctx context.Context, cfg provider.Config) ([]string, error) {
			return mistral.Models(ctx, cfg, filter)
		}
	case "nvidia":
		return func(ctx context.Context, cfg provider.Config) ([]string, error) {
			return nvidia.Models(ctx, cfg, filter)
		}
	case "openrouter":
		return func(ctx context.Context, cfg provider.Config) ([]string, error) {
			return openrouter.Models(ctx, cfg, filter)
		}
	case "cloudflare":
		return func(ctx context.Context, cfg provider.Config) ([]string, error) {
			return cloudflare.Models(ctx, cfg, filter)
		}
	default:
		return nil
	}
}

func Supports(prov string) bool {
	return lookup(prov) != nil
}

func Models(ctx context.Context, name string) ([]string, error) {
	prov := Provider(name)
	fn := lookup(prov)
	if fn == nil {
		if _, ok := agentKeychain.CompatInstance(name); !ok {
			return nil, fmt.Errorf("provider %q does not support model listing", prov)
		}
		fn = func(ctx context.Context, cfg provider.Config) ([]string, error) {
			return CompatModels(ctx, cfg.BaseURL, cfg.APIKey)
		}
	}
	cfg, err := agentKeychain.Config(ctx, name)
	if err != nil {
		return nil, err
	}
	return fn(ctx, cfg)
}

func CompatModels(ctx context.Context, baseURL, apiKey string) ([]string, error) {
	baseURL = strings.TrimRight(baseURL, "/")
	if baseURL == "" {
		return nil, fmt.Errorf("compat base url is not configured")
	}

	var headers map[string]string
	if apiKey != "" {
		headers = map[string]string{"Authorization": "Bearer " + apiKey}
	}

	client := &http.Client{Timeout: 10 * time.Second}
	data, status, err := go_pkg_http.GET[compatModelList](ctx, client, baseURL+"/models", headers)
	if err != nil {
		return nil, fmt.Errorf("go_pkg_http.GET %s/models: %w", baseURL, err)
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("%s/models: http %d", baseURL, status)
	}

	ids := make([]string, 0, len(data.Data))
	for _, one := range data.Data {
		if id := strings.TrimSpace(one.ID); id != "" {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

type compatModelList struct {
	Data []struct {
		ID string `json:"id"`
	} `json:"data"`
}

func Provider(name string) string {
	providerFull, _, _ := strings.Cut(name, "@")
	prov, _, _ := strings.Cut(providerFull, "[")
	return prov
}

func Alive(ctx context.Context, name string, timeout time.Duration) bool {
	probeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	models, err := Models(probeCtx, name)
	return err == nil && len(models) > 0
}
