package image

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync"

	provider "github.com/pardnchiu/go-llm-router/core"
	"github.com/pardnchiu/go-llm-router/core/gemini"
	"github.com/pardnchiu/go-llm-router/core/grok"
	grokOauth "github.com/pardnchiu/go-llm-router/core/grokOauth"
	openrouter "github.com/pardnchiu/go-llm-router/core/openRouter"
	"github.com/pardnchiu/go-llm-router/core/openai"
	"github.com/pardnchiu/go-llm-router/core/router"

	agentKeychain "github.com/pardnchiu/agenvoy/internal/agents/keychain"
	"github.com/pardnchiu/agenvoy/internal/runtime/torii"
	"github.com/pardnchiu/agenvoy/internal/session/config"
)

var Providers = []string{"openai", "codex", "grok", "grok-oauth", "gemini", "openrouter"}

const Off = ""

func Selected() string {
	cfg, err := config.Load()
	if err != nil || cfg == nil {
		return Off
	}
	name := strings.TrimSpace(cfg.ImageGenerator)
	if name == "off" {
		return Off
	}
	return name
}

func Enabled() bool {
	return Selected() != Off
}

const modelsCacheTTL = 15 * 60

func modelsCacheKey(name string) string {
	return "provider:models:image:" + name
}

func DropModelsCache(ctx context.Context, name string) {
	torii.DropKeys(ctx, modelsCacheKey(name))
}

func Available(ctx context.Context) []string {
	filter := provider.ModelFilter{ImageOnly: true}
	found := make([][]string, len(Providers))

	var wg sync.WaitGroup
	for i, name := range Providers {
		wg.Go(func() {
			cfg, err := agentKeychain.Config(ctx, name+"@")
			if err != nil {
				return
			}
			if name == "codex" {
				found[i] = []string{name}
				return
			}

			base := provider.Config{APIKey: cfg.APIKey, AccountID: cfg.AccountID}
			models, err := torii.CachedList(ctx, modelsCacheKey(name), modelsCacheTTL, func() ([]string, error) {
				switch name {
				case "openai":
					return openai.Models(ctx, base, filter)
				case "grok":
					return grok.Models(ctx, base, filter)
				case "grok-oauth":
					return grokOauth.Models(ctx, base, filter)
				case "gemini":
					return gemini.Models(ctx, base, filter)
				case "openrouter":
					return openrouter.Models(ctx, base, filter)
				}
				return nil, nil
			})
			if err != nil {
				slog.Debug("image.Available",
					slog.String("provider", name),
					slog.String("error", err.Error()))
				return
			}
			for _, model := range models {
				found[i] = append(found[i], name+"@"+model)
			}
		})
	}
	wg.Wait()

	list := []string{}
	for _, models := range found {
		list = append(list, models...)
	}
	return list
}

func Prune(ctx context.Context) {
	name := Selected()
	if name == Off {
		return
	}
	if !strings.Contains(name, "@") {
		if _, err := agentKeychain.Config(ctx, name+"@"); err == nil {
			return
		}
	} else if slices.Contains(Available(ctx), name) {
		return
	}

	cfg, err := config.Load()
	if err != nil || cfg == nil {
		return
	}
	cfg.ImageGenerator = Off
	if err := config.Save(cfg); err != nil {
		slog.Warn("image.Prune config.Save",
			slog.String("provider", name),
			slog.String("error", err.Error()))
	}
}

func agent(ctx context.Context) (provider.ImageAgent, string, error) {
	name := Selected()
	if name == Off {
		return nil, "", fmt.Errorf("image generation is off; set it in Config → Model → Setting Models")
	}
	full := name
	if !strings.Contains(full, "@") {
		full += "@"
	}
	cfg, err := agentKeychain.Config(ctx, full)
	if err != nil {
		return nil, "", fmt.Errorf("%s is not configured: %w", name, err)
	}
	built, err := router.New(router.Config{
		Name:      full,
		APIKey:    cfg.APIKey,
		Token:     cfg.Token,
		BaseURL:   cfg.BaseURL,
		AccountID: cfg.AccountID,
		GatewayID: cfg.GatewayID,
	})
	if err != nil {
		return nil, "", fmt.Errorf("router.New [%s]: %w", full, err)
	}
	img, ok := built.(provider.ImageAgent)
	if !ok {
		return nil, "", fmt.Errorf("%s cannot generate images", name)
	}
	return img, full, nil
}
