package app

import (
	"context"
	"log/slog"

	"github.com/pardnchiu/agenvoy/internal/agents/exec"
	agentKeychain "github.com/pardnchiu/agenvoy/internal/agents/keychain"
	agentTypes "github.com/pardnchiu/agenvoy/internal/agents/types"
	"github.com/pardnchiu/agenvoy/internal/session/config"
	provider "github.com/pardnchiu/go-llm-router/core"
	"github.com/pardnchiu/go-llm-router/core/router"
)

type resolvedAgent struct {
	name string
}

func (a *resolvedAgent) Name() string {
	return a.name
}

func (a *resolvedAgent) build(ctx context.Context) (agentTypes.Agent, error) {
	cfg, err := routerConfig(ctx, a.name)
	if err != nil {
		return nil, err
	}
	return router.New(cfg)
}

func (a *resolvedAgent) Send(ctx context.Context, messages []provider.Message, toolDefs []provider.Tool, reasoning provider.Reasoning, mode provider.Mode) (*provider.Output, int, error) {
	inner, err := a.build(ctx)
	if err != nil {
		return nil, 0, err
	}
	return inner.Send(ctx, messages, toolDefs, reasoning, mode)
}

func (a *resolvedAgent) SendStream(ctx context.Context, messages []provider.Message, toolDefs []provider.Tool, reasoning provider.Reasoning, mode provider.Mode) (<-chan provider.StreamEvent, error) {
	inner, err := a.build(ctx)
	if err != nil {
		return nil, err
	}
	streamer, ok := inner.(provider.StreamAgent)
	if !ok {
		return nil, provider.ErrStreamUnsupported
	}
	return streamer.SendStream(ctx, messages, toolDefs, reasoning, mode)
}

func routerConfig(ctx context.Context, name string) (router.Config, error) {
	cfg, err := agentKeychain.Config(ctx, name)
	if err != nil {
		return router.Config{}, err
	}
	return router.Config{
		Name:      name,
		APIKey:    cfg.APIKey,
		Token:     cfg.Token,
		AccountID: cfg.AccountID,
		GatewayID: cfg.GatewayID,
		BaseURL:   cfg.BaseURL,
	}, nil
}

func NewAgentRegistry() agentTypes.AgentRegistry {
	agentEntries := exec.GetAgent()
	registry := agentTypes.AgentRegistry{
		Registry: make(map[string]agentTypes.Agent, len(agentEntries)),
		Entries:  make([]agentTypes.AgentEntry, 0, len(agentEntries)),
	}
	for _, e := range agentEntries {
		a := &resolvedAgent{name: e.Name}
		if _, err := a.build(context.Background()); err != nil {
			slog.Warn("failed to initialize",
				slog.String("name", e.Name),
				slog.String("error", err.Error()))
			continue
		}
		registry.Registry[e.Name] = a
		registry.Entries = append(registry.Entries, e)
		if registry.Fallback == nil {
			registry.Fallback = a
		}
	}

	return registry
}

func SelectDispatcher(registry agentTypes.AgentRegistry) agentTypes.Agent {
	if cfg, err := config.Load(); err == nil && cfg.DispatcherModel != "" {
		if a, ok := registry.Registry[cfg.DispatcherModel]; ok {
			return a
		}
	}
	return registry.Fallback
}

func SelectSummary(registry agentTypes.AgentRegistry) agentTypes.Agent {
	if cfg, err := config.Load(); err == nil && cfg.SummaryModel != "" {
		if a, ok := registry.Registry[cfg.SummaryModel]; ok {
			return a
		}
	}
	return nil
}

func RefreshHost() (agentTypes.Agent, agentTypes.Agent, agentTypes.AgentRegistry) {
	registry := NewAgentRegistry()
	return SelectDispatcher(registry), SelectSummary(registry), registry
}
