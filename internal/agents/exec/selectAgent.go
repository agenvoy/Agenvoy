package exec

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	go_pkg_filesystem "github.com/pardnchiu/go-pkg/filesystem"

	"github.com/pardnchiu/agenvoy/configs"
	"github.com/pardnchiu/agenvoy/internal/agents/exec/compact"
	"github.com/pardnchiu/agenvoy/internal/agents/exec/fast"
	"github.com/pardnchiu/agenvoy/internal/agents/exec/retryHandler"
	agentTypes "github.com/pardnchiu/agenvoy/internal/agents/types"
	"github.com/pardnchiu/agenvoy/internal/filesystem"
	"github.com/pardnchiu/agenvoy/internal/filesystem/skill"
	"github.com/pardnchiu/agenvoy/internal/session/config"
	configBot "github.com/pardnchiu/agenvoy/internal/session/config/bot"
	provider "github.com/pardnchiu/go-llm-router/core"
)

const (
	DispatcherCallTimeout        = 30 * time.Second
	UnresponsiveProbeInterval    = 30 * time.Second
	UnresponsiveRetryInterval    = 10 * time.Second
	MaxUnresponsiveProbeFailures = 3
	HealthCheckTimeout           = 10 * time.Second
	SendTimeoutRetryInterval     = 15 * time.Second
	MaxSendTimeoutRetries        = 3
)

type AgentConfig struct {
	DefaultModel string                  `json:"default_model"`
	Models       []agentTypes.AgentEntry `json:"models"`
}

func GetAgent() []agentTypes.AgentEntry {
	cfg, err := go_pkg_filesystem.ReadJSON[AgentConfig](filesystem.ConfigPath)
	if err != nil || len(cfg.Models) == 0 {
		return []agentTypes.AgentEntry{}
	}
	for i := range cfg.Models {
		cfg.Models[i].Name = config.NormalizeModel(cfg.Models[i].Name)
	}
	if cfg.DefaultModel == "" {
		cfg.DefaultModel = cfg.Models[0].Name
	} else {
		for i, m := range cfg.Models {
			// * move default model to first be fallback
			if m.Name == cfg.DefaultModel {
				cfg.Models[0], cfg.Models[i] = cfg.Models[i], cfg.Models[0]
				break
			}
		}
	}
	return cfg.Models
}

func SkillHint(s *skill.Skill) string {
	if s == nil {
		return ""
	}
	return strings.TrimSpace(s.Description)
}

func SelectAgentNames(ctx context.Context, bot agentTypes.Agent, registry agentTypes.AgentRegistry, userInput string, hasSkill bool, skillHint string, sessionID string) ([]string, map[string]bool) {
	dead := map[string]bool{}

	if sessionID != "" {
		model, _ := configBot.GetModel(sessionID)
		if model != "" && model != configBot.DefaultModel {
			if _, ok := registry.Registry[model]; ok {
				return []string{model}, dead
			}
		}
	}

	tiers := map[string]string{}
	tierLines := "(none set)"
	if cfg, err := config.Load(); err == nil {
		tiers = cfg.ModelTag
		tierLines = config.ModelTagLines(cfg)
	}
	registryOrder := make([]string, 0, len(registry.Entries))
	passOrder := []string{}
	known := make(map[string]struct{}, len(registry.Entries))
	for _, e := range registry.Entries {
		known[e.Name] = struct{}{}
		if tiers[e.Name] == config.ModelTagPass {
			passOrder = append(passOrder, e.Name)
		} else {
			registryOrder = append(registryOrder, e.Name)
		}
	}

	if len(registry.Entries) <= 1 {
		return append(registryOrder, passOrder...), dead
	}

	picked := []string{}
	seen := map[string]bool{}

	bot = retryHandler.Check(bot, registry)

	if bot != nil {
		agentJson, err := json.Marshal(registry.Entries)
		if err == nil {
			userContent := strings.TrimSpace(userInput)
			if hasSkill {
				userContent = "[Run Skill] " + userContent
				if desc := strings.TrimSpace(skillHint); desc != "" {
					userContent += " — " + desc
				}
			}
			messages := []provider.Message{
				{Role: "system", Content: strings.ReplaceAll(strings.TrimSpace(configs.AgentSelector), "{{.ModelTag}}", tierLines)},
				{Role: "user", Content: fmt.Sprintf("Available agents:\n%s\nUser request: %s", string(agentJson), userContent)},
			}
			dispatchCtx := agentTypes.WithSessionID(ctx, sessionID)
			for range len(registry.Entries) {
				if ctx.Err() != nil {
					break
				}
				routingCtx, cancel := context.WithTimeout(dispatchCtx, DispatcherCallTimeout)
				resp, sendCode, sendErr := bot.Send(routingCtx, messages, nil, provider.ReasoningNone, fast.Mode())
				cancel()
				if sendErr == nil {
					retryHandler.Clear(bot.Name())
					if resp != nil && len(resp.Choices) > 0 {
						if content, ok := resp.Choices[0].Message.Content.(string); ok {
							raw := strings.Trim(strings.TrimSpace(content), "\"'` \n")
							if raw != "" && raw != "NONE" {
								for n := range strings.SplitSeq(raw, ",") {
									n = strings.Trim(strings.TrimSpace(n), "\"'`")
									if n == "" || seen[n] {
										continue
									}
									if _, ok := known[n]; !ok {
										continue
									}
									if retryHandler.IsCoolingDown(n) {
										dead[n] = true
										continue
									}
									picked = append(picked, n)
									seen[n] = true
								}
							}
						}
					}
					break
				}
				dead[bot.Name()] = true
				rateLimited := sendCode == 429
				if rateLimited {
					retryHandler.Register(bot.Name())
				}
				next := retryHandler.Check(nil, registry)
				hasNext := next != nil && !dead[next.Name()]
				if ctx.Err() == nil && !rateLimited {
					slog.Debug("dispatcher routing failed",
						slog.String("name", bot.Name()),
						slog.String("error", sendErr.Error()))
				}
				if !hasNext {
					break
				}
				if ctx.Err() == nil && !rateLimited {
					slog.Debug("dispatcher retrying with fallback",
						slog.String("name", next.Name()))
				}
				bot = next
			}
		}
	}

	for _, n := range registryOrder {
		if seen[n] || dead[n] {
			continue
		}
		picked = append(picked, n)
		seen[n] = true
	}
	picked = orderProviders(picked)
	for _, n := range passOrder {
		if !seen[n] && !dead[n] {
			picked = append(picked, n)
		}
	}
	return picked, dead
}

var providerRank = map[string]int{
	"codex":      0,
	"grok-oauth": 0,
	"copilot":    1,
	"openrouter": 3,
}

func providerOrder(name string) int {
	prov, _, _ := strings.Cut(name, "@")
	if rank, ok := providerRank[prov]; ok {
		return rank
	}
	return 2
}

func baseModel(name string) string {
	prov, model, ok := strings.Cut(name, "@")
	if !ok {
		return name
	}
	if prov == "openrouter" {
		if _, rest, found := strings.Cut(model, "/"); found {
			model = rest
		}
	}
	return strings.ToLower(model)
}

func orderProviders(names []string) []string {
	slots := make(map[string][]int, len(names))
	for i, n := range names {
		key := baseModel(n)
		slots[key] = append(slots[key], i)
	}
	out := slices.Clone(names)
	for _, idx := range slots {
		if len(idx) < 2 {
			continue
		}
		group := make([]string, len(idx))
		for i, at := range idx {
			group[i] = names[at]
		}
		slices.SortStableFunc(group, func(a, b string) int { return providerOrder(a) - providerOrder(b) })
		for i, at := range idx {
			out[at] = group[i]
		}
	}
	return out
}

func SelectAgent(ctx context.Context, bot agentTypes.Agent, registry agentTypes.AgentRegistry, userInput string, hasSkill bool, skillHint string, sessionID string) agentTypes.Agent {
	names, dead := SelectAgentNames(ctx, bot, registry, userInput, hasSkill, skillHint, sessionID)
	for _, n := range names {
		if dead[n] {
			continue
		}
		if a, ok := registry.Registry[n]; ok && a != nil {
			return a
		}
	}
	return registry.Fallback
}

const maxFallbackRounds = 3

func nextAgent(ctx context.Context, sessionID, currentModel string, fallbacks *[]agentTypes.Agent, allAgents []agentTypes.Agent, round *int, inputTokens int) (agentTypes.Agent, string) {
	if model, _ := configBot.GetModel(sessionID); model != "" && model != configBot.DefaultModel {
		return nil, ""
	}

	prefix, _, ok := strings.Cut(currentModel, "@")
	if !ok {
		return nil, ""
	}
	prefix += "@"

	others := filterOutPrefix(allAgents, prefix)
	if len(others) == 0 {
		return nil, ""
	}
	others = filterByWindow(others, inputTokens)
	*fallbacks = filterByWindow(filterOutPrefix(*fallbacks, prefix), inputTokens)

	for {
		agent, name := pickHealthyFallback(ctx, fallbacks)
		if agent != nil {
			return agent, name
		}
		*round++
		if *round >= maxFallbackRounds {
			return nil, ""
		}
		if ctx.Err() != nil {
			return nil, ""
		}
		slog.Warn("all agents failed, starting retry round",
			slog.Int("round", *round+1),
			slog.Int("max", maxFallbackRounds))
		rebuilt := make([]agentTypes.Agent, len(others))
		copy(rebuilt, others)
		*fallbacks = rebuilt
	}
}

func filterByWindow(agents []agentTypes.Agent, inputTokens int) []agentTypes.Agent {
	if inputTokens <= 0 {
		return agents
	}

	out := make([]agentTypes.Agent, 0, len(agents))
	for _, a := range agents {
		if a == nil || compact.CheckThreshold(a.Name()) < inputTokens {
			continue
		}
		out = append(out, a)
	}
	if len(out) == 0 {
		return agents
	}
	return out
}

func filterOutPrefix(agents []agentTypes.Agent, prefix string) []agentTypes.Agent {
	out := make([]agentTypes.Agent, 0, len(agents))
	for _, a := range agents {
		if a == nil || strings.HasPrefix(a.Name(), prefix) {
			continue
		}
		out = append(out, a)
	}
	return out
}

func pickHealthyFallback(ctx context.Context, fallbacks *[]agentTypes.Agent) (agentTypes.Agent, string) {
	for len(*fallbacks) > 0 {
		cand := (*fallbacks)[0]
		*fallbacks = (*fallbacks)[1:]
		if cand == nil {
			continue
		}
		if checkAgentResponsive(ctx, cand, HealthCheckTimeout) {
			return cand, cand.Name()
		}
		if ctx.Err() == nil {
			slog.Debug("fallback health check failed",
				slog.String("name", cand.Name()),
				slog.Duration("timeout", HealthCheckTimeout))
		}
	}
	return nil, ""
}

func ResolveAgent(ctx context.Context, bot agentTypes.Agent, registry agentTypes.AgentRegistry, userInput string, hasSkill bool, skillHint string, sessionID string) (agentTypes.Agent, []agentTypes.Agent, error) {
	names, dead := SelectAgentNames(ctx, bot, registry, userInput, hasSkill, skillHint, sessionID)
	if len(names) == 0 {
		return nil, nil, fmt.Errorf("no agents available")
	}
	candidates := make([]agentTypes.Agent, 0, len(names))
	for _, n := range names {
		if dead[n] {
			continue
		}
		if a, ok := registry.Registry[n]; ok && a != nil {
			candidates = append(candidates, a)
		}
	}
	if len(candidates) == 0 {
		return nil, nil, fmt.Errorf("no resolvable agents from %d names (dead: %d)", len(names), len(dead))
	}
	return candidates[0], candidates[1:], nil
}
