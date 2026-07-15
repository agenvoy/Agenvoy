package keychain

import (
	"context"
	"fmt"
	"strings"

	oauthCodex "github.com/pardnchiu/agenvoy/internal/agents/oauth/codex"
	oauthCopilot "github.com/pardnchiu/agenvoy/internal/agents/oauth/copilot"
	oauthGrok "github.com/pardnchiu/agenvoy/internal/agents/oauth/grok"
	"github.com/pardnchiu/agenvoy/internal/agents/provider"
	sessionConfig "github.com/pardnchiu/agenvoy/internal/session/config"
	go_pkg_keychain "github.com/pardnchiu/go-pkg/filesystem/keychain"
)

func Config(ctx context.Context, name string) (provider.Config, error) {
	providerFull, _, _ := strings.Cut(name, "@")
	prov, _, _ := strings.Cut(providerFull, "[")

	switch prov {
	case "claude":
		apiKey := go_pkg_keychain.Get("CLAUDE_API_KEY")
		if apiKey == "" {
			return provider.Config{}, fmt.Errorf("keychain.Get: CLAUDE_API_KEY is required")
		}
		return provider.Config{APIKey: apiKey}, nil

	case "openai":
		apiKey := go_pkg_keychain.Get("OPENAI_API_KEY")
		if apiKey == "" {
			return provider.Config{}, fmt.Errorf("keychain.Get: OPENAI_API_KEY is required")
		}
		return provider.Config{APIKey: apiKey}, nil

	case "gemini":
		apiKey := go_pkg_keychain.Get("GEMINI_API_KEY")
		if apiKey == "" {
			return provider.Config{}, fmt.Errorf("keychain.Get: GEMINI_API_KEY is required")
		}
		return provider.Config{APIKey: apiKey}, nil

	case "grok":
		apiKey := go_pkg_keychain.Get("GROK_API_KEY")
		if apiKey == "" {
			return provider.Config{}, fmt.Errorf("keychain.Get: GROK_API_KEY is required")
		}
		return provider.Config{APIKey: apiKey}, nil

	case "deepseek":
		apiKey := go_pkg_keychain.Get("DEEPSEEK_API_KEY")
		if apiKey == "" {
			return provider.Config{}, fmt.Errorf("keychain.Get: DEEPSEEK_API_KEY is required")
		}
		return provider.Config{APIKey: apiKey}, nil

	case "nvidia":
		apiKey := go_pkg_keychain.Get("NVIDIA_API_KEY")
		if apiKey == "" {
			return provider.Config{}, fmt.Errorf("keychain.Get: NVIDIA_API_KEY is required")
		}
		return provider.Config{APIKey: apiKey}, nil

	case "openrouter":
		apiKey := go_pkg_keychain.Get("OPENROUTER_API_KEY")
		if apiKey == "" {
			return provider.Config{}, fmt.Errorf("keychain.Get: OPENROUTER_API_KEY is required")
		}
		return provider.Config{APIKey: apiKey}, nil

	case "cloudflare":
		apiKey := go_pkg_keychain.Get("CLOUDFLARE_API_KEY")
		if apiKey == "" {
			return provider.Config{}, fmt.Errorf("keychain.Get: CLOUDFLARE_API_KEY is required")
		}
		accountID := go_pkg_keychain.Get("CLOUDFLARE_ACCOUNT_ID")
		if accountID == "" {
			return provider.Config{}, fmt.Errorf("keychain.Get: CLOUDFLARE_ACCOUNT_ID is required")
		}
		return provider.Config{
			APIKey:    apiKey,
			AccountID: accountID,
			GatewayID: go_pkg_keychain.Get("CLOUDFLARE_GATEWAY_ID"),
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
		return provider.Config{
			APIKey:  go_pkg_keychain.Get(apiKeyEnvKey),
			BaseURL: sessionConfig.GetCompatURL(instanceName),
		}, nil

	case "copilot":
		token, err := oauthCopilot.Load()
		if err != nil {
			return provider.Config{}, fmt.Errorf("oauthCopilot.Load: %w", err)
		}
		if token == nil {
			return provider.Config{}, fmt.Errorf("copilot token missing; run `agen model add` to authenticate")
		}
		refresh, err := oauthCopilot.EnsureFreshSession(ctx, token, nil)
		if err != nil {
			return provider.Config{}, fmt.Errorf("oauthCopilot.EnsureFreshSession: %w", err)
		}
		return provider.Config{APIKey: refresh.Token, Token: token}, nil

	case "codex":
		token, err := oauthCodex.Load()
		if err != nil {
			return provider.Config{}, fmt.Errorf("oauthCodex.Load: %w", err)
		}
		if token == nil {
			return provider.Config{}, fmt.Errorf("codex token missing; run `agen model add` to authenticate")
		}
		token, err = oauthCodex.EnsureFresh(ctx, token)
		if err != nil {
			return provider.Config{}, fmt.Errorf("codex token expired and refresh failed: %w; run `agen model add` to re-authenticate", err)
		}
		return provider.Config{APIKey: token.AccessToken, AccountID: token.AccountID, Token: token}, nil

	case "grok-oauth":
		token, err := oauthGrok.Load()
		if err != nil {
			return provider.Config{}, fmt.Errorf("oauthGrok.Load: %w", err)
		}
		if token == nil {
			return provider.Config{}, fmt.Errorf("grok-oauth token missing; run `agen model add` to authenticate")
		}
		token, err = oauthGrok.EnsureFresh(ctx, token)
		if err != nil {
			return provider.Config{}, fmt.Errorf("grok-oauth token expired and refresh failed: %w; run `agen model add` to re-authenticate", err)
		}
		return provider.Config{APIKey: token.AccessToken, Token: token}, nil

	default:
		return provider.Config{}, fmt.Errorf("credential.Resolve: unknown provider %q in %q", prov, name)
	}
}
