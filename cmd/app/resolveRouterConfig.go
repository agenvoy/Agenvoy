package main

import (
	"context"
	"fmt"
	"strings"

	oauthCodex "github.com/pardnchiu/agenvoy/internal/agents/oauth/codex"
	oauthCopilot "github.com/pardnchiu/agenvoy/internal/agents/oauth/copilot"
	oauthGrok "github.com/pardnchiu/agenvoy/internal/agents/oauth/grok"
	"github.com/pardnchiu/agenvoy/internal/agents/router"
	sessionConfig "github.com/pardnchiu/agenvoy/internal/session/config"
	"github.com/pardnchiu/go-pkg/filesystem/keychain"
)

func resolveRouterConfig(ctx context.Context, name string) (router.Config, error) {
	providerFull, _, _ := strings.Cut(name, "@")
	prov, _, _ := strings.Cut(providerFull, "[")

	switch prov {
	case "claude":
		apiKey := keychain.Get("CLAUDE_API_KEY")
		if apiKey == "" {
			return router.Config{}, fmt.Errorf("keychain.Get: CLAUDE_API_KEY is required")
		}
		return router.Config{Name: name, APIKey: apiKey}, nil

	case "openai":
		apiKey := keychain.Get("OPENAI_API_KEY")
		if apiKey == "" {
			return router.Config{}, fmt.Errorf("keychain.Get: OPENAI_API_KEY is required")
		}
		return router.Config{Name: name, APIKey: apiKey}, nil

	case "gemini":
		apiKey := keychain.Get("GEMINI_API_KEY")
		if apiKey == "" {
			return router.Config{}, fmt.Errorf("keychain.Get: GEMINI_API_KEY is required")
		}
		return router.Config{Name: name, APIKey: apiKey}, nil

	case "grok":
		apiKey := keychain.Get("GROK_API_KEY")
		if apiKey == "" {
			return router.Config{}, fmt.Errorf("keychain.Get: GROK_API_KEY is required")
		}
		return router.Config{Name: name, APIKey: apiKey}, nil

	case "deepseek":
		apiKey := keychain.Get("DEEPSEEK_API_KEY")
		if apiKey == "" {
			return router.Config{}, fmt.Errorf("keychain.Get: DEEPSEEK_API_KEY is required")
		}
		return router.Config{Name: name, APIKey: apiKey}, nil

	case "nvidia":
		apiKey := keychain.Get("NVIDIA_API_KEY")
		if apiKey == "" {
			return router.Config{}, fmt.Errorf("keychain.Get: NVIDIA_API_KEY is required")
		}
		return router.Config{Name: name, APIKey: apiKey}, nil

	case "openrouter":
		apiKey := keychain.Get("OPENROUTER_API_KEY")
		if apiKey == "" {
			return router.Config{}, fmt.Errorf("keychain.Get: OPENROUTER_API_KEY is required")
		}
		return router.Config{Name: name, APIKey: apiKey}, nil

	case "cloudflare":
		apiKey := keychain.Get("CLOUDFLARE_API_KEY")
		if apiKey == "" {
			return router.Config{}, fmt.Errorf("keychain.Get: CLOUDFLARE_API_KEY is required")
		}
		accountID := keychain.Get("CLOUDFLARE_ACCOUNT_ID")
		if accountID == "" {
			return router.Config{}, fmt.Errorf("keychain.Get: CLOUDFLARE_ACCOUNT_ID is required")
		}
		return router.Config{
			Name:      name,
			APIKey:    apiKey,
			AccountID: accountID,
			GatewayID: keychain.Get("CLOUDFLARE_GATEWAY_ID"),
		}, nil

	case "compat":
		instanceName := ""
		if start := strings.Index(name, "["); start != -1 {
			if end := strings.Index(name, "]"); end > start {
				instanceName = strings.ToUpper(name[start+1 : end])
			}
		}
		apiKeyEnvKey := "COMPAT_API_KEY"
		if instanceName != "" {
			apiKeyEnvKey = "COMPAT_" + instanceName + "_API_KEY"
		}
		return router.Config{
			Name:    name,
			APIKey:  keychain.Get(apiKeyEnvKey),
			BaseURL: sessionConfig.GetCompatURL(instanceName),
		}, nil

	case "copilot":
		token, err := oauthCopilot.Load()
		if err != nil {
			return router.Config{}, fmt.Errorf("oauthCopilot.Load: %w", err)
		}
		if token == nil {
			return router.Config{}, fmt.Errorf("copilot token missing; run `agen model add` to authenticate")
		}
		return router.Config{Name: name, Token: token}, nil

	case "codex":
		token, err := oauthCodex.Load()
		if err != nil {
			return router.Config{}, fmt.Errorf("oauthCodex.Load: %w", err)
		}
		if token == nil {
			return router.Config{}, fmt.Errorf("codex token missing; run `agen model add` to authenticate")
		}
		token, err = oauthCodex.EnsureFresh(ctx, token)
		if err != nil {
			return router.Config{}, fmt.Errorf("codex token expired and refresh failed: %w; run `agen model add` to re-authenticate", err)
		}
		return router.Config{Name: name, Token: token}, nil

	case "grok-oauth":
		token, err := oauthGrok.Load()
		if err != nil {
			return router.Config{}, fmt.Errorf("oauthGrok.Load: %w", err)
		}
		if token == nil {
			return router.Config{}, fmt.Errorf("grok-oauth token missing; run `agen model add` to authenticate")
		}
		token, err = oauthGrok.EnsureFresh(ctx, token)
		if err != nil {
			return router.Config{}, fmt.Errorf("grok-oauth token expired and refresh failed: %w; run `agen model add` to re-authenticate", err)
		}
		return router.Config{Name: name, Token: token}, nil

	default:
		return router.Config{}, fmt.Errorf("resolveRouterConfig: unknown provider %q in %q", prov, name)
	}
}
