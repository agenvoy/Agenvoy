package exec

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	agentTypes "github.com/pardnchiu/agenvoy/internal/agents/types"
)

type RateLimit struct {
	Agent    string
	ResetsAt int64
	Body     string
}

var (
	cooldownMap      sync.Map
	providerPriority = map[string]int{
		"codex":      0,
		"grok-oauth": 1,
		"grok":       2,
		"openrouter": 3,
		"deepseek":   4,
		"copilot":    5,
		"claude":     6,
		"gemini":     7,
		"nvidia":     8,
		"openai":     9,
		"compat":     10,
	}
)

func (e *RateLimit) Error() string {
	return fmt.Sprintf("HTTP 429: rate limit until %d: %s", e.ResetsAt, e.Body)
}

func isRateLimit(err error) *RateLimit {
	var rateLimit *RateLimit
	if errors.As(err, &rateLimit) {
		return rateLimit
	}
	return nil
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
	if bot != nil && !isCoolingDown(bot.Name()) {
		return bot
	}
	var best agentTypes.Agent
	bestPri := len(providerPriority) + 1
	for _, e := range registry.Entries {
		if isCoolingDown(e.Name) {
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
