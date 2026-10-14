package model

import (
	"context"
	"log/slog"
	"strings"

	"github.com/pardnchiu/agenvoy/configs"
	agenttypes "github.com/pardnchiu/agenvoy/internal/agents/types"
	configBot "github.com/pardnchiu/agenvoy/internal/session/config/bot"
)

type RetryState struct {
	Agents []agenttypes.Agent
	Round  int
}

func Next(ctx context.Context, sessionID, currentModel string, state *RetryState) (agenttypes.Agent, string) {
	// * if assign, then skip fallback
	if model, _ := configBot.GetModel(sessionID); model != configBot.DefaultModel {
		return nil, ""
	}

	// * check if model is exist
	provider, _, ok := strings.Cut(currentModel, "@")
	if !ok {
		return nil, ""
	}
	provider += "@"

	fallbackAgents := filterOutSameProvider(state.Agents, provider)
	if len(fallbackAgents) == 0 {
		return nil, ""
	}
	state.Agents = fallbackAgents

	for {
		agent, modelName := getFallback(ctx, fallbackAgents)
		if agent != nil {
			return agent, modelName
		}
		state.Round++
		if state.Round > configs.MAX_RETRY_TIMES {
			return nil, ""
		}
		if ctx.Err() != nil {
			return nil, ""
		}
		slog.Warn("[Unavailable] fallback: all agents failed, starting next round",
			slog.Int("round", state.Round+1),
			slog.Int("max", configs.MAX_RETRY_TIMES+1))
	}
}

func filterOutSameProvider(poolAgents []agenttypes.Agent, provider string) (fallbackAgents []agenttypes.Agent) {
	fallbackAgents = make([]agenttypes.Agent, 0, len(poolAgents))
	for _, e := range poolAgents {
		if e == nil || strings.HasPrefix(e.Name(), provider) {
			continue
		}
		fallbackAgents = append(fallbackAgents, e)
	}
	return fallbackAgents
}

func getFallback(ctx context.Context, poolAgents []agenttypes.Agent) (agent agenttypes.Agent, modelName string) {
	for _, e := range poolAgents {
		if e == nil {
			continue
		}
		if CheckAlive(ctx, e, configs.HEALTH_CHECK_TIMEOUT) {
			return e, e.Name()
		}
		if ctx.Err() == nil {
			slog.Debug("[Unavailable] fallback: health check failed",
				slog.String("name", e.Name()),
				slog.Duration("timeout", configs.HEALTH_CHECK_TIMEOUT))
		}
	}
	return nil, ""
}
