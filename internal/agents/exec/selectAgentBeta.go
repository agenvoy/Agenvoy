package exec

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/pardnchiu/go-pkg/filesystem/keychain"
	go_pkg_http "github.com/pardnchiu/go-pkg/http"

	"github.com/pardnchiu/agenvoy/internal/session/config"
	"github.com/pardnchiu/agenvoy/internal/session/history"
)

const (
	typesafeEndpoint   = "https://api.typesafe.ai/v1/systemone"
	betaContextTurns   = 6
	betaContextMaxRune = 2000
	betaNamedNone      = "none"
)

var betaWorkTiers = map[string][]string{
	"code":  {"S", "A", "B", "C"},
	"chat":  {"B", "C", "A", "S"},
	"fetch": {"C", "B", "A", "S"},
	"work":  {"S", "A", "B", "C"},
}

var betaWorkCriteria = map[string]any{
	"code": map[string]any{
		"what":    "Writing, fixing, debugging or testing code; a request that asks outright for depth or precision (詳細分析, 深入, 精確, in-depth, rigorous); a Skill that builds or tests code, or creates, installs, migrates or probes something.",
		"not_for": "Explaining, reviewing or planning code without changing it.",
		"examples": []string{
			"這段有 race condition 嗎？幫我修掉",
			"Add a retry to the upload handler and write a test for it",
			"詳細分析這份財報的每一項數字",
		},
	},
	"chat": map[string]any{
		"what":    "Greeting, small talk, a short factual answer, or translation.",
		"not_for": "Anything that needs tools, research or more than a few sentences.",
		"examples": []string{
			"早安",
			"Translate this paragraph into Japanese",
			"謝謝，這樣就可以了",
		},
	},
	"fetch": map[string]any{
		"what":    "Calling tools to fetch data and returning it without judgement; a Skill with fixed input and a deterministic transform.",
		"not_for": "Comparing, explaining or drawing conclusions from the data.",
		"examples": []string{
			"查一下台北現在天氣",
			"What is AAPL trading at right now?",
			"列出這個資料夾的檔案",
		},
	},
	"work": map[string]any{
		"what":    "Reports, analysis, review, planning, research, and anything that fits none of the other options.",
		"not_for": "A request that merely sounds important or long but is really one of the other options.",
		"examples": []string{
			"比較這三家雲端供應商的價格與限制",
			"Review this design doc and list the risks",
			"幫我規劃下週的發版流程",
		},
	},
}

type betaAnswer struct {
	Answers map[string]struct {
		Choice string `json:"choice"`
	} `json:"answers"`
}

func selectAgentBeta(ctx context.Context, candidates, passNames []string, tiers map[string]string, request, sessionID string) ([]string, error) {
	key := strings.TrimSpace(keychain.Get(config.TypesafeKey))
	if key == "" {
		return nil, fmt.Errorf("missing key: %s", config.TypesafeKey)
	}
	if len(candidates) == 0 {
		return candidates, nil
	}

	named := map[string]any{betaNamedNone: "The request does not ask to use a specific model."}
	for _, name := range slices.Concat(candidates, passNames) {
		named[name] = "The request asks to use " + name + " (fuzzy match on provider, family or version)."
	}

	body := map[string]any{
		"model": "jev-latest",
		"state": map[string]any{
			"context": betaContext(sessionID),
			"request": request,
		},
		"questions": map[string]any{
			"work": map[string]any{
				"type": "choice",
				"instructions": map[string]any{
					"question": "What kind of work does `request` ask for?",
					"focus":    "Classify `request`; use `context` only to resolve references such as 'this' or 'continue'. A request starting with [Run Skill] runs a Skill: classify the work the Skill does, not its name.",
				},
				"criteria": betaWorkCriteria,
			},
			"named": map[string]any{
				"type": "choice",
				"instructions": map[string]any{
					"question": "Does `request` explicitly ask to use a specific model, such as \"use/with <name>\" or 指定/用 <名稱>?",
					"focus":    "Only an explicit instruction about which model to use counts; a model merely mentioned as a topic is none.",
				},
				"criteria": named,
			},
		},
	}

	routingCtx, cancel := context.WithTimeout(ctx, DispatcherCallTimeout)
	defer cancel()
	result, _, err := go_pkg_http.POST[betaAnswer](routingCtx, nil, typesafeEndpoint, map[string]string{
		"Authorization": "Bearer " + key,
	}, body, "json")
	if err != nil {
		return nil, fmt.Errorf("go_pkg_http.POST: %w", err)
	}

	work := result.Answers["work"].Choice
	order, ok := betaWorkTiers[work]
	if !ok {
		return nil, fmt.Errorf("invalid work choice: %q", work)
	}

	list := make([]string, 0, len(candidates)+1)
	if choice := result.Answers["named"].Choice; choice != betaNamedNone && named[choice] != nil {
		list = append(list, choice)
	}

	rank := func(name string) int {
		if i := slices.Index(order, cmp.Or(tiers[name], betaNameTier(name))); i >= 0 {
			return i
		}
		return len(order)
	}
	ranked := slices.Clone(candidates)
	slices.SortStableFunc(ranked, func(a, b string) int { return cmp.Compare(rank(a), rank(b)) })
	for _, name := range ranked {
		if !slices.Contains(list, name) {
			list = append(list, name)
		}
	}
	return list, nil
}

func betaNameTier(name string) string {
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
		return "C"
	case hasPrefix("claude-fable", "claude-opus"), family("gpt-", "-astra"), family("gpt-", "-sol"):
		return "S"
	case strings.HasPrefix(model, "grok-"):
		var version float64
		fmt.Sscanf(strings.TrimPrefix(model, "grok-"), "%f", &version)
		if version >= 4.5 {
			return "S"
		}
		return "B"
	case hasPrefix("claude-sonnet", "deepseek-pro", "glm", "kimi"), family("gpt-", "-terra"), family("gemini-", "-pro"):
		return "A"
	case hasPrefix("claude-haiku", "deepseek"), family("gpt-", "-luna"), family("gemini-", "-flash"):
		return "B"
	}
	return "A"
}

func betaContext(sessionID string) []map[string]string {
	var records []history.Record
	if sessionID != "" {
		_, records = history.Get(sessionID)
	}

	list := []map[string]string{}
	for _, r := range slices.Backward(records) {
		if len(list) >= betaContextTurns {
			break
		}
		if r.Role != "user" && r.Role != "assistant" {
			continue
		}
		text := strings.TrimSpace(history.StripPrefix(r.Text()))
		if text == "" {
			continue
		}
		if runes := []rune(text); len(runes) > betaContextMaxRune {
			text = string(runes[:betaContextMaxRune]) + "..."
		}
		list = append(list, map[string]string{"role": r.Role, "content": text})
	}
	slices.Reverse(list)
	return list
}
