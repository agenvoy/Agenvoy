package cooldown

import (
	"strings"
	"sync"
	"time"

	agentTypes "github.com/pardnchiu/agenvoy/internal/agents/types"
)

const rateLimitCooldown = 30 * time.Minute

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
		"mistral":    9,
		"openai":     10,
		"compat":     11,
	}
)

func Register(agentName string) {
	cooldownMap.Store(agentName, time.Now().Add(rateLimitCooldown).Unix())
}

func Clear(agentName string) {
	cooldownMap.Delete(agentName)
}

func IsCoolingDown(agentName string) bool {
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

func Check(bot agentTypes.Agent, registry agentTypes.AgentRegistry) agentTypes.Agent {
	if len(registry.Entries) <= 1 {
		if bot != nil {
			return bot
		}
		if len(registry.Entries) == 1 {
			return registry.Registry[registry.Entries[0].Name]
		}
		return nil
	}

	if bot != nil && !IsCoolingDown(bot.Name()) {
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
		if respectCooldown && IsCoolingDown(e.Name) {
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
