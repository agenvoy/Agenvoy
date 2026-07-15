package router

import (
	"fmt"
	"strings"

	"github.com/pardnchiu/agenvoy/internal/agents/provider"
	"github.com/pardnchiu/agenvoy/internal/agents/provider/claude"
	"github.com/pardnchiu/agenvoy/internal/agents/provider/cloudflare"
	"github.com/pardnchiu/agenvoy/internal/agents/provider/compat"
	"github.com/pardnchiu/agenvoy/internal/agents/provider/copilot"
	"github.com/pardnchiu/agenvoy/internal/agents/provider/deepseek"
	"github.com/pardnchiu/agenvoy/internal/agents/provider/gemini"
	"github.com/pardnchiu/agenvoy/internal/agents/provider/grok"
	grokoauth "github.com/pardnchiu/agenvoy/internal/agents/provider/grokOauth"
	"github.com/pardnchiu/agenvoy/internal/agents/provider/nvidia"
	openrouter "github.com/pardnchiu/agenvoy/internal/agents/provider/openRouter"
	"github.com/pardnchiu/agenvoy/internal/agents/provider/openai"
	openaicodex "github.com/pardnchiu/agenvoy/internal/agents/provider/openaiCodex"
	agentTypes "github.com/pardnchiu/agenvoy/internal/agents/types"
)

type Config struct {
	Name    string
	APIKey  string
	Token   any
	BaseURL string

	AccountID string
	GatewayID string
}

var newFn = map[string]func(config Config) (agentTypes.Agent, error){
	"claude": func(config Config) (agentTypes.Agent, error) {
		return claude.New(provider.Config{Model: strings.TrimPrefix(config.Name, claude.Prefix), APIKey: config.APIKey})
	},
	"openai": func(config Config) (agentTypes.Agent, error) {
		return openai.New(provider.Config{Model: strings.TrimPrefix(config.Name, openai.Prefix), APIKey: config.APIKey})
	},
	"gemini": func(config Config) (agentTypes.Agent, error) {
		return gemini.New(provider.Config{Model: strings.TrimPrefix(config.Name, gemini.Prefix), APIKey: config.APIKey})
	},
	"grok": func(config Config) (agentTypes.Agent, error) {
		return grok.New(provider.Config{Model: strings.TrimPrefix(config.Name, grok.Prefix), APIKey: config.APIKey})
	},
	"deepseek": func(config Config) (agentTypes.Agent, error) {
		return deepseek.New(provider.Config{Model: strings.TrimPrefix(config.Name, deepseek.Prefix), APIKey: config.APIKey})
	},
	"nvidia": func(config Config) (agentTypes.Agent, error) {
		return nvidia.New(provider.Config{Model: strings.TrimPrefix(config.Name, nvidia.Prefix), APIKey: config.APIKey})
	},
	"openrouter": func(config Config) (agentTypes.Agent, error) {
		return openrouter.New(provider.Config{Model: strings.TrimPrefix(config.Name, openrouter.Prefix), APIKey: config.APIKey})
	},
	"cloudflare": func(config Config) (agentTypes.Agent, error) {
		return cloudflare.New(provider.Config{
			Model:     strings.TrimPrefix(config.Name, cloudflare.Prefix),
			APIKey:    config.APIKey,
			AccountID: config.AccountID,
			GatewayID: config.GatewayID,
		})
	},
	"compat": func(config Config) (agentTypes.Agent, error) {
		_, model, _ := strings.Cut(config.Name, "@")
		return compat.New(provider.Config{
			Model:   model,
			APIKey:  config.APIKey,
			BaseURL: config.BaseURL,
		})
	},
	"copilot": func(config Config) (agentTypes.Agent, error) {
		return copilot.New(provider.Config{Model: strings.TrimPrefix(config.Name, copilot.Prefix), Token: config.Token})
	},
	"codex": func(config Config) (agentTypes.Agent, error) {
		return openaicodex.New(provider.Config{Model: strings.TrimPrefix(config.Name, openaicodex.Prefix), Token: config.Token})
	},
	"grok-oauth": func(config Config) (agentTypes.Agent, error) {
		return grokoauth.New(provider.Config{Model: strings.TrimPrefix(config.Name, grokoauth.Prefix), Token: config.Token})
	},
}

func New(config Config) (agentTypes.Agent, error) {
	providerFull, _, _ := strings.Cut(config.Name, "@")
	prov, _, _ := strings.Cut(providerFull, "[")
	fn, ok := newFn[prov]
	if !ok {
		return nil, fmt.Errorf("router.New: unknown provider %q in %q", prov, config.Name)
	}
	return fn(config)
}
