package app

import (
	"context"
	"log/slog"

	go_llm_router "github.com/pardnchiu/go-llm-router/core"
	"github.com/pardnchiu/go-llm-router/core/claudeCode"
	"github.com/pardnchiu/go-llm-router/core/router"

	"github.com/pardnchiu/agenvoy/configs"
	execmodel "github.com/pardnchiu/agenvoy/internal/agents/exec/model"
	agentkeychain "github.com/pardnchiu/agenvoy/internal/agents/keychain"
	agenttypes "github.com/pardnchiu/agenvoy/internal/agents/types"
	"github.com/pardnchiu/agenvoy/internal/filesystem"
	historystore "github.com/pardnchiu/agenvoy/internal/runtime/store"
	"github.com/pardnchiu/agenvoy/internal/session/config"
)

type resolvedAgent struct {
	name string
}

func (a *resolvedAgent) Name() string {
	return a.name
}

func (a *resolvedAgent) build(ctx context.Context) (agenttypes.Agent, error) {
	cfg, err := routerConfig(ctx, a.name)
	if err != nil {
		return nil, err
	}
	return router.New(cfg)
}

func (a *resolvedAgent) Send(ctx context.Context, messages []go_llm_router.Message, toolDefs []go_llm_router.Tool, reasoning go_llm_router.Reasoning, mode go_llm_router.Mode) (*go_llm_router.Output, int, error) {
	inner, err := a.build(ctx)
	if err != nil {
		return nil, 0, err
	}
	if claudeCode.Is(a.name) {
		ctx = claudeCodeContext(ctx)
	}
	return inner.Send(ctx, messages, toolDefs, reasoning, mode)
}

func (a *resolvedAgent) SendStream(ctx context.Context, messages []go_llm_router.Message, toolDefs []go_llm_router.Tool, reasoning go_llm_router.Reasoning, mode go_llm_router.Mode) (<-chan go_llm_router.StreamEvent, error) {
	inner, err := a.build(ctx)
	if err != nil {
		return nil, err
	}
	streamer, ok := inner.(go_llm_router.StreamAgent)
	if !ok {
		return nil, go_llm_router.ErrStreamUnsupported
	}
	return streamer.SendStream(ctx, messages, toolDefs, reasoning, mode)
}

func routerConfig(ctx context.Context, name string) (router.Config, error) {
	cfg, err := agentkeychain.Config(ctx, name)
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

		EnableClaude: agenttypes.EnableClaudeCode,
		StateDir:     filesystem.SessionsDir,
	}, nil
}

func claudeCodeContext(ctx context.Context) context.Context {
	sessionID := agenttypes.SessionIDFrom(ctx)
	if sessionID == "" {
		return ctx
	}
	ctx = go_llm_router.WithSessionID(ctx, sessionID)
	row, ok, err := historystore.ReadSession(ctx, sessionID)
	if err != nil || !ok || row.Model == "" || row.Model == historystore.DefaultModel || row.Reasoning == configs.REASONING_AUTO {
		return ctx
	}
	return go_llm_router.WithCacheTTL(ctx, "1h")
}

func NewAgentRegistry() agenttypes.ModelSet {
	agentEntries := execmodel.List()
	registry := agenttypes.ModelSet{
		Registry: make(map[string]agenttypes.Agent, len(agentEntries)),
		Order:    make([]agenttypes.ModelEntry, 0, len(agentEntries)),
	}
	skipClaudeCode := !agenttypes.ClaudeCodeEnabled()
	for _, e := range agentEntries {
		if skipClaudeCode && claudeCode.Is(e.Name) {
			continue
		}
		a := &resolvedAgent{name: e.Name}
		if _, err := a.build(context.Background()); err != nil {
			slog.Warn("failed to initialize",
				slog.String("name", e.Name),
				slog.String("error", err.Error()))
			continue
		}
		registry.Registry[e.Name] = a
		registry.Order = append(registry.Order, e)
		if registry.Fallback == nil {
			registry.Fallback = a
		}
	}

	return registry
}

func SelectDispatcher(registry agenttypes.ModelSet) agenttypes.Agent {
	if cfg, err := config.Load(); err == nil && cfg.DispatcherModel != "" {
		if a, ok := registry.Registry[cfg.DispatcherModel]; ok {
			return a
		}
	}
	return registry.Fallback
}

func SelectSummary(registry agenttypes.ModelSet) agenttypes.Agent {
	if cfg, err := config.Load(); err == nil && cfg.SummaryModel != "" {
		if a, ok := registry.Registry[cfg.SummaryModel]; ok {
			return a
		}
	}
	return nil
}

func RefreshHost() (agenttypes.Agent, agenttypes.Agent, agenttypes.ModelSet) {
	registry := NewAgentRegistry()
	return SelectDispatcher(registry), SelectSummary(registry), registry
}
