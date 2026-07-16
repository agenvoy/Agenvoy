package exec

import (
	"strings"
	"sync"
	"time"

	"github.com/pardnchiu/go-llm-router/core"
	agentTypes "github.com/pardnchiu/agenvoy/internal/agents/types"
)

var (
	cooldownMap      sync.Map
	providerPriority = map[string]int{
		"codex":      0,
		"copilot":    1,
		"grok-oauth": 2,
		"grok":       3,
		"openrouter": 4,
		"deepseek":   5,
		"claude":     6,
		"gemini":     7,
		"nvidia":     8,
		"openai":     9,
		"compat":     10,
	}
)

func registerCooldown(agentName string) {
	cooldownMap.Store(agentName, time.Now().Add(provider.RateLimitCooldown).Unix())
}

func clearCooldown(agentName string) {
	cooldownMap.Delete(agentName)
}

func isCoolingDown(agentName string) bool {
	v, ok := cooldownMap.Load(agentName)
	if !ok {
		return false
	}
	resetsAt := v.(int64)
	if time.Now().Unix() >= resetsAt {
		cooldownMap.Delete(agentName)
		return false
	}
	return true
}

func checkCooldown(bot agentTypes.Agent, registry agentTypes.AgentRegistry) agentTypes.Agent {
	// * only one model, skip cooldown
	if len(registry.Entries) <= 1 {
		if bot != nil {
			return bot
		}
		if len(registry.Entries) == 1 {
			return registry.Registry[registry.Entries[0].Name]
		}
		return nil
	}

	if bot != nil && !isCoolingDown(bot.Name()) {
		return bot
	}

	var excludePrefix string
	if bot != nil {
		if p, _, ok := strings.Cut(bot.Name(), "@"); ok {
			excludePrefix = p + "@"
		}
	}

	if best := bestCandidate(registry, excludePrefix, true); best != nil {
		return best
	}
	// * no healthy, skip cooldown
	if best := bestCandidate(registry, excludePrefix, false); best != nil {
		return best
	}
	return bot
}

func bestCandidate(registry agentTypes.AgentRegistry, excludePrefix string, respectCooldown bool) agentTypes.Agent {
	var best agentTypes.Agent
	bestPri := len(providerPriority) + 1
	for _, e := range registry.Entries {
		if excludePrefix != "" && strings.HasPrefix(e.Name, excludePrefix) {
			continue
		}
		if respectCooldown && isCoolingDown(e.Name) {
			continue
		}
		providor, _, _ := strings.Cut(e.Name, "@")
		pri, ok := providerPriority[providor]
		if !ok {
			pri = len(providerPriority)
		}
		if pri < bestPri {
			bestPri = pri
			best = registry.Registry[e.Name]
		}
	}
	return best
}
