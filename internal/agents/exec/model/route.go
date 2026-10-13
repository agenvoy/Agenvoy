package model

import (
	"context"
	"fmt"

	"github.com/pardnchiu/agenvoy/configs"
	"github.com/pardnchiu/agenvoy/internal/agents"
	agentTypes "github.com/pardnchiu/agenvoy/internal/agents/types"
	"github.com/pardnchiu/agenvoy/internal/session/config"
	configBot "github.com/pardnchiu/agenvoy/internal/session/config/bot"
)

func Route(ctx context.Context, assignModel, requestTask, skillDesc, sessionID string) (agentTypes.Agent, []agentTypes.Agent, string, error) {
	// * if assign exist, then use it
	modelSet := agents.ModelSet()
	req := SelectMeta{
		Agent:            agents.DispatcherBot(),
		ModelSet:         modelSet,
		Task:             requestTask,
		SkillDescription: skillDesc,
		SessionID:        sessionID,
	}
	if assignModel != configBot.DefaultModel {
		agent, ok := modelSet.Registry[assignModel]
		if !ok || agent == nil {
			return nil, nil, "", fmt.Errorf("[Not found] model: %q", assignModel)
		}

		// * check reasoning, if auto, then get fit
		_, reasoning := configBot.GetModel(sessionID)
		if reasoning == configs.REASONING_AUTO {
			cfg := loadConfig()
			_, reasoning, _ = runSelect(ctx, req, cfg, []string{assignModel}, map[string]bool{})
		}
		return agent, []agentTypes.Agent{}, reasoning, nil
	}

	// * if no assign, then get fit
	models, deadMap, reasoning := selectModel(ctx, req)
	var agent agentTypes.Agent
	for _, e := range models {
		if deadMap[e] {
			continue
		}
		if a, ok := modelSet.Registry[e]; ok && a != nil {
			agent, assignModel = a, e
			break
		}
	}

	// * get fallback
	cfg := loadConfig()
	fallbackAgents := make([]agentTypes.Agent, 0, len(modelSet.Order))
	for _, e := range modelSet.Order {
		if e.Name == assignModel || deadMap[e.Name] || cfg.TagMap[e.Name] == config.ModelTagPass {
			continue
		}
		if a, ok := modelSet.Registry[e.Name]; ok && a != nil {
			fallbackAgents = append(fallbackAgents, a)
		}
	}

	if agent == nil {
		if len(fallbackAgents) == 0 {
			return nil, nil, "", fmt.Errorf("[Unavailable] model: none available (dead: %d)", len(deadMap))
		}
		agent, fallbackAgents = fallbackAgents[0], fallbackAgents[1:]
	}
	return agent, fallbackAgents, reasoning, nil
}
