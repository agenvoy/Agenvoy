package main

import (
	"context"
	"log/slog"

	"github.com/pardnchiu/agenvoy/internal/agents/exec"
	agentKeychain "github.com/pardnchiu/agenvoy/internal/agents/keychain"
	"github.com/pardnchiu/agenvoy/internal/agents/router"
	agentTypes "github.com/pardnchiu/agenvoy/internal/agents/types"
	"github.com/pardnchiu/agenvoy/internal/session/config"
)

func modelConfig(ctx context.Context, name string) (router.Config, error) {
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

func buildAgentRegistry() agentTypes.AgentRegistry {
	agentEntries := exec.GetAgent()
	registry := agentTypes.AgentRegistry{
		Registry: make(map[string]agentTypes.Agent, len(agentEntries)),
		Entries:  make([]agentTypes.AgentEntry, 0, len(agentEntries)),
	}
	for _, e := range agentEntries {
		cfg, err := modelConfig(context.Background(), e.Name)
		if err != nil {
			slog.Warn("failed to resolve config",
				slog.String("name", e.Name),
				slog.String("error", err.Error()))
			continue
		}
		a, err := router.New(cfg)
		if err != nil {
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

func dispatcherSelector(registry agentTypes.AgentRegistry) agentTypes.Agent {
	if cfg, err := config.Load(); err == nil && cfg.DispatcherModel != "" {
		if a, ok := registry.Registry[cfg.DispatcherModel]; ok {
			return a
		}
	}
	return registry.Fallback
}

func summarySelector(registry agentTypes.AgentRegistry) agentTypes.Agent {
	if cfg, err := config.Load(); err == nil && cfg.SummaryModel != "" {
		if a, ok := registry.Registry[cfg.SummaryModel]; ok {
			return a
		}
	}
	return nil
}

func refreshHost() (agentTypes.Agent, agentTypes.Agent, agentTypes.AgentRegistry) {
	registry := buildAgentRegistry()
	return dispatcherSelector(registry), summarySelector(registry), registry
}
