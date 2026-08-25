package image

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	provider "github.com/pardnchiu/go-llm-router/core"
	"github.com/pardnchiu/go-llm-router/core/router"

	"github.com/pardnchiu/agenvoy/internal/agents"
	agentKeychain "github.com/pardnchiu/agenvoy/internal/agents/keychain"
	"github.com/pardnchiu/agenvoy/internal/session/config"
)

var Providers = []string{"openai", "codex", "grok", "grok-oauth", "gemini"}

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

func Available(ctx context.Context) []string {
	list := []string{}
	for _, name := range Providers {
		if _, err := agentKeychain.Config(ctx, name+"@"); err == nil {
			list = append(list, name)
		}
	}
	return list
}

func Prune(ctx context.Context) {
	name := Selected()
	if name == Off {
		return
	}
	if slices.Contains(Available(ctx), name) {
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
	prefix := name + "@"
	for registered, a := range agents.Registry().Registry {
		if !strings.HasPrefix(registered, prefix) {
			continue
		}
		if img, ok := a.(provider.ImageAgent); ok {
			return img, registered, nil
		}
	}

	full := prefix
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
