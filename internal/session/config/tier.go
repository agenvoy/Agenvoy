package config

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	"github.com/pardnchiu/agenvoy/configs"
)

type WorkKind struct {
	Key   string
	What  string
	Tiers []string
}

var WorkKinds = []WorkKind{
	{
		Key:   "code",
		What:  "Writing, fixing, debugging or testing code; a request that asks outright for depth or precision; a Skill that builds or tests code, or creates, installs, migrates or probes something.",
		Tiers: []string{"S", "A", "B", "C"},
	},
	{
		Key:   "research",
		What:  "Research, analysis, comparison and reports: gathering from several sources or data points, then synthesizing findings and drawing conclusions.",
		Tiers: []string{"S", "A", "B", "C"},
	},
	{
		Key:   "work",
		What:  "General tasks: planning, reviewing, drafting, editing or organizing content the user already has, and anything that fits none of the other options.",
		Tiers: []string{"A", "S", "B", "C"},
	},
	{
		Key:   "chat",
		What:  "Greeting, small talk, a short factual answer, or translation.",
		Tiers: []string{"B", "C", "A", "S"},
	},
	{
		Key:   "fetch",
		What:  "Calling tools to fetch data and returning it without judgement; a Skill with fixed input and a deterministic transform.",
		Tiers: []string{"C", "B", "A", "S"},
	},
}

func WorkTiers(key string) ([]string, bool) {
	i := slices.IndexFunc(WorkKinds, func(k WorkKind) bool { return k.Key == key })
	if i < 0 {
		return nil, false
	}
	return WorkKinds[i].Tiers, true
}

func NameTier(name string) (string, int) {
	model := strings.ToLower(name[strings.Index(name, "@")+1:])
	model = model[strings.LastIndex(model, "/")+1:]

	hasPrefix := func(prefixes ...string) bool {
		return slices.ContainsFunc(prefixes, func(p string) bool { return strings.HasPrefix(model, p) })
	}
	family := func(prefix, tag string) bool {
		return strings.HasPrefix(model, prefix) && strings.Contains(model, tag)
	}

	switch {
	case strings.Contains(model, "-mini"), strings.Contains(model, "-nano"), family("gemini-", "-flash-lite"),
		hasPrefix("gemma", "gpt-oss", "qwen", "llama"):
		return "C", 30
	case hasPrefix("claude-fable"):
		return "S", 0
	case hasPrefix("claude-opus"):
		return "S", 1
	case family("gpt-", "-astra"):
		return "S", 2
	case family("gpt-", "-sol"):
		return "A", 10
	case hasPrefix("grok-"):
		var version float64
		fmt.Sscanf(strings.TrimPrefix(model, "grok-"), "%f", &version)
		if version >= 4.5 {
			return "A", 11
		}
		return "B", 23
	case hasPrefix("claude-sonnet"):
		return "A", 12
	case family("gpt-", "-terra"):
		return "A", 13
	case family("gemini-", "-pro"):
		return "A", 14
	case hasPrefix("deepseek-pro"):
		return "A", 15
	case hasPrefix("glm"):
		return "A", 16
	case hasPrefix("kimi"):
		return "A", 17
	case hasPrefix("claude-haiku"):
		return "B", 20
	case family("gpt-", "-luna"):
		return "B", 21
	case family("gemini-", "-flash"):
		return "B", 22
	case hasPrefix("deepseek"):
		return "B", 24
	}
	return "A", 18
}

func ModelTier(tags map[string]string, name string) string {
	if tag := tags[name]; tag != "" {
		return tag
	}
	tier, _ := NameTier(name)
	return tier
}

var providerRank = map[string]int{
	"codex":      0,
	"grok-oauth": 0,
	"copilot":    1,
	"openrouter": 3,
}

func ProviderOrder(name string) int {
	prov, _, _ := strings.Cut(name, "@")
	if rank, ok := providerRank[prov]; ok {
		return rank
	}
	return 2
}

func ModelSelection(cfg *Config) string {
	groups := make(map[string][]string, len(ModelTags))
	for _, m := range cfg.Models {
		tier := ModelTier(cfg.ModelTag, m.Name)
		groups[tier] = append(groups[tier], m.Name)
	}

	tierLines := make([]string, 0, len(ModelTags))
	for _, tier := range ModelTags {
		names := groups[tier]
		if len(names) == 0 {
			continue
		}
		slices.SortStableFunc(names, func(a, b string) int {
			_, fa := NameTier(a)
			_, fb := NameTier(b)
			return cmp.Or(cmp.Compare(fa, fb), cmp.Compare(ProviderOrder(a), ProviderOrder(b)))
		})
		tierLines = append(tierLines, fmt.Sprintf("- %s: %s", tier, strings.Join(names, ", ")))
	}
	modelTier := "(no models registered)"
	if len(tierLines) > 0 {
		modelTier = strings.Join(tierLines, "\n")
	}

	workLines := make([]string, 0, len(WorkKinds))
	for _, k := range WorkKinds {
		workLines = append(workLines, fmt.Sprintf("- %s (%s): %s", k.Key, strings.Join(k.Tiers, " > "), k.What))
	}

	return strings.NewReplacer(
		"{{.ModelTier}}", modelTier,
		"{{.WorkTiers}}", strings.Join(workLines, "\n"),
	).Replace(strings.TrimSpace(configs.ModelSelection))
}
