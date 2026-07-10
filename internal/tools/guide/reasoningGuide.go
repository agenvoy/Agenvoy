package guide

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pardnchiu/agenvoy/configs"
	toolRegister "github.com/pardnchiu/agenvoy/internal/tools/register"
	toolTypes "github.com/pardnchiu/agenvoy/internal/tools/types"
)

var topicGuides = map[string]string{
	"rag_web":             configs.ReasoningRAGWeb,
	"market_analysis":     configs.ReasoningMarketAnalysis,
	"targeted_read":       configs.ReasoningTargetedRead,
	"ask_user":            configs.ReasoningAskUser,
	"subagent_delegation": configs.ReasoningSubagentDelegation,
	"write_todo":          configs.ReasoningWriteTodo,
}

func registReasoningGuide() {
	toolRegister.Regist(toolRegister.Def{
		Name:        "reasoning_guide",
		AlwaysAllow: true,
		AlwaysLoad:  true,
		Concurrent:  true,
		Description: `[system-default] Full reasoning rule per topic — call before acting on any match:

- rag_web: non-smalltalk info query (people, orgs, facts, current events, prices, time-sensitive) — ground in search_rag + live web, never training knowledge alone.
- market_analysis: stock/ETF/market analysis — assess macro, regional, industry, asset-specific layers, never single region.
- targeted_read: file question needs only specific symbols/sections/keywords — search first, narrow read_file over whole-file read.
- ask_user: missing target, vague scope, unclear spec, ambiguous time, scheduling without content, non-unique tool choice — resolve intent first.
- subagent_delegation: named session ("call X"/"呼叫 X"), reusable single subtask, or broad multi-source/cross-entity request — decide delegation before invoke_subagent.
- write_todo: analysis/research task or complex multi-step task, no active Skill — decide checklist before write_todo.`,
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"topic": map[string]any{
					"type":        "string",
					"enum":        []string{"rag_web", "market_analysis", "targeted_read", "ask_user", "subagent_delegation", "write_todo"},
					"description": "Which Reasoning Rules topic to fetch.",
				},
			},
			"required": []string{"topic"},
		},
		Handler: func(_ context.Context, _ *toolTypes.Executor, args json.RawMessage) (string, error) {
			var params struct {
				Topic string `json:"topic"`
			}
			if err := json.Unmarshal(args, &params); err != nil {
				return "", fmt.Errorf("json.Unmarshal: %w", err)
			}
			topic := strings.TrimSpace(params.Topic)
			guide, ok := topicGuides[topic]
			if !ok {
				return "", fmt.Errorf("unknown topic %q; available: rag_web, market_analysis, targeted_read, ask_user, subagent_delegation, write_todo", topic)
			}
			return guide, nil
		},
	})
}
