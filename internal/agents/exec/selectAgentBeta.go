package exec

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/pardnchiu/go-pkg/filesystem/keychain"
	go_pkg_http "github.com/pardnchiu/go-pkg/http"

	"github.com/pardnchiu/agenvoy/internal/runtime/torii"
	"github.com/pardnchiu/agenvoy/internal/session/config"
	"github.com/pardnchiu/agenvoy/internal/session/history"
)

const (
	typesafeEndpoint   = "https://api.typesafe.ai/v1/systemone"
	betaContextTurns   = 4
	betaContextMaxRune = 2048
	betaNamedNone      = "none"
	betaTopicSame      = "same"
	betaLastModelKey   = "lastModel:"
)

var betaWorkReasoning = map[string]string{
	"code":     "xhigh",
	"chat":     "none",
	"fetch":    "low",
	"research": "high",
	"work":     "medium",
}

var betaWorkCriteria = map[string]any{
	"code": map[string]any{
		"what":    betaWorkWhat("code"),
		"not_for": "Explaining, reviewing or planning code without changing it.",
		"examples": []string{
			"這段有 race condition 嗎？幫我修掉",
			"Add a retry to the upload handler and write a test for it",
			"幫我把這個函式改成非同步並補上錯誤處理",
		},
	},
	"chat": map[string]any{
		"what":    betaWorkWhat("chat"),
		"not_for": "Anything that needs tools, research or more than a few sentences.",
		"examples": []string{
			"早安",
			"Translate this paragraph into Japanese",
			"謝謝",
		},
	},
	"fetch": map[string]any{
		"what":    betaWorkWhat("fetch"),
		"not_for": "Comparing, explaining or drawing conclusions from the data.",
		"examples": []string{
			"查一下現在的天氣",
			"What is this ticker trading at right now?",
			"列出這個資料夾的檔案",
		},
	},
	"research": map[string]any{
		"what":    betaWorkWhat("research"),
		"not_for": "Fetching one value and returning it as-is, or a task that needs no investigation.",
		"examples": []string{
			"比較這三家雲端供應商的價格與限制",
			"Research how this regulation affects the products we ship",
			"分析這家公司近四季財報的趨勢",
		},
	},
	"work": map[string]any{
		"what":    betaWorkWhat("work"),
		"not_for": "A request that merely sounds important or long but is really one of the other options.",
		"examples": []string{
			"Review this design doc and list the risks",
			"幫我規劃下週的發版流程",
			"把這段會議紀錄整理成待辦清單",
		},
	},
}

func betaWorkWhat(key string) string {
	i := slices.IndexFunc(config.WorkKinds, func(k config.WorkKind) bool { return k.Key == key })
	if i < 0 {
		return ""
	}
	return config.WorkKinds[i].What
}

type betaAnswer struct {
	Answers map[string]struct {
		Choice string `json:"choice"`
	} `json:"answers"`
}

func selectAgentBeta(ctx context.Context, candidates, passNames []string, tiers map[string]string, request, sessionID string) ([]string, string, error) {
	key := strings.TrimSpace(keychain.Get(config.TypesafeKey))
	if key == "" {
		return nil, "", fmt.Errorf("missing key: %s", config.TypesafeKey)
	}
	if len(candidates) == 0 {
		return candidates, "", nil
	}

	named := map[string]any{betaNamedNone: "The request does not ask to use a specific model."}
	for _, name := range slices.Concat(candidates, passNames) {
		named[name] = "The request asks to use " + name + " (fuzzy match on provider, family or version)."
	}

	turns := betaContext(sessionID)
	questions := map[string]any{
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
	}

	var previous, previousReasoning string
	if len(turns) > 0 {
		previous, previousReasoning = betaPrevious(ctx, sessionID, candidates)
	}
	if previous != "" {
		questions["topic"] = map[string]any{
			"type": "choice",
			"instructions": map[string]any{
				"question": "Does `request` continue the subject of the last user message in `context`?",
				"focus":    "Compare only with the last user message in `context` and its reply; earlier turns do not count. Follow-ups, corrections, refinements and next steps on that subject are same, even when the kind of work changes.",
			},
			"criteria": map[string]any{
				betaTopicSame: "`request` follows up on, refers to or builds on the last user message in `context` and its reply.",
				"new":         "`request` changes to a different subject than the last user message in `context`, even if an earlier turn discussed it.",
			},
		}
	}

	body := map[string]any{
		"model": "jev-latest",
		"state": map[string]any{
			"context": turns,
			"request": request,
		},
		"questions": questions,
	}

	routingCtx, cancel := context.WithTimeout(ctx, DispatcherCallTimeout)
	defer cancel()
	result, _, err := go_pkg_http.POST[betaAnswer](routingCtx, nil, typesafeEndpoint, map[string]string{
		"Authorization": "Bearer " + key,
	}, body, "json")
	if err != nil {
		return nil, "", fmt.Errorf("go_pkg_http.POST: %w", err)
	}

	work := result.Answers["work"].Choice
	order, ok := config.WorkTiers(work)
	if !ok {
		return nil, "", fmt.Errorf("invalid work choice: %q", work)
	}

	list := make([]string, 0, len(candidates)+1)
	if choice := result.Answers["named"].Choice; choice != betaNamedNone && named[choice] != nil {
		list = append(list, choice)
	}
	level := betaWorkReasoning[work]
	if previous != "" && result.Answers["topic"].Choice == betaTopicSame {
		if len(list) == 0 || list[0] == previous {
			level = cmp.Or(previousReasoning, level)
		}
		if !slices.Contains(list, previous) {
			list = append(list, previous)
		}
	}

	rank := func(name string) int {
		tier, family := config.NameTier(name)
		i := slices.Index(order, cmp.Or(tiers[name], tier))
		if i < 0 {
			i = len(order)
		}
		return i*100 + family
	}
	ranked := slices.Clone(candidates)
	slices.SortStableFunc(ranked, func(a, b string) int { return cmp.Compare(rank(a), rank(b)) })
	for _, name := range ranked {
		if !slices.Contains(list, name) {
			list = append(list, name)
		}
	}
	return list, level, nil
}

func betaLastModelTTL(name string) int64 {
	prov, _, _ := strings.Cut(name, "@")
	switch prov {
	case "openai", "codex":
		return 30 * 60
	case "gemini":
		return 60 * 60
	}
	return 5 * 60
}

func betaPrevious(ctx context.Context, sessionID string, candidates []string) (model, reasoning string) {
	if sessionID == "" {
		return "", ""
	}
	entry, ok := torii.DB(torii.DBToolCache).Get(ctx, betaLastModelKey+sessionID)
	if !ok {
		return "", ""
	}
	value := entry.Value()
	i := strings.LastIndex(value, "/")
	if i < 0 || !slices.Contains(candidates, value[:i]) {
		return "", ""
	}
	return value[:i], value[i+1:]
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
