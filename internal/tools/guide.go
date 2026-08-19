package tools

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
	"tool_generate":     configs.GuideToolGenerate,
	"tool_error":        configs.GuideToolError,
	"rag_web":           configs.GuideRAGWeb,
	"market_analysis":   configs.GuideMarketAnalysis,
	"targeted_read":     configs.GuideTargetedRead,
	"ask_user":          configs.GuideAskUser,
	"subagent_dispatch": configs.GuideSubagentDispatch,
	"write_todo":        configs.GuideWriteTodo,
	"html_render":       configs.GuideHtmlRender,
}

func registReasoningGuide() {
	toolRegister.Regist(toolRegister.Def{
		Name:        "reasoning_guide",
		AlwaysAllow: true,
		AlwaysLoad:  true,
		Concurrent:  true,
		SystemUse:   true,
		Description: `[system-default]
Full rule per topic — call before acting on any match:

- tool_error: any tool call failure — recovery loop, script_*/api_* auto-repair via patch_tool, [RETRY_REQUIRED] handling. Read before retrying, before error_history, before patch_tool.
- tool_generate: request needs live external data (weather, currency, stock, geocoding, translation, ...) and no api_*/script_*/ext_* covers it — search_tools found nothing, or an existing one fails. Carries the build contract (naming, description rules, tool.json/script.py format, execution flow), then write_tool → test_tool (script only) → call it. Hard gate: fetching the answer directly via send_http_request or run_command curl/python3 is PROHIBITED even with a known endpoint — fetch_page is for docs, the data fetch lives in script.py. Never say "tool not available" — build one.
- rag_web: non-smalltalk info query (people, orgs, facts, current events, prices, time-sensitive) — RAG and live web fire in parallel every time both are available; carries the source-citation rule and the fallback when one side is missing.
- market_analysis: stock/ETF/market analysis — assess macro, regional, industry, asset-specific layers, never single region.
- targeted_read: file question needs only specific symbols/sections/keywords — search first, narrow read_files over whole-file read.
- ask_user: missing target, vague scope, unclear spec, ambiguous time, scheduling without content, non-unique tool choice — resolve intent first.
- subagent_dispatch: the same lookup repeating across 3+ entities, a lookup spanning 2+ source classes, a set just discovered that now needs per-entity work, a named session ("call X"/"呼叫 X"), or a reusable single subtask — read before any subagent(mode=invoke).
- write_todo: analysis/research task or complex multi-step task, no active Skill — decide checklist before write_todo.
- html_render: producing a standalone HTML deliverable (report, dashboard, chart, map, 3D view) — settle libraries, breakpoints and visual direction before writing the file.`,
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"topic": map[string]any{
					"type":        "string",
					"enum":        []string{"tool_generate", "tool_error", "rag_web", "market_analysis", "targeted_read", "ask_user", "subagent_dispatch", "write_todo", "html_render"},
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
				return "", fmt.Errorf("unknown topic %q; available: tool_generate, tool_error, rag_web, market_analysis, targeted_read, ask_user, subagent_dispatch, write_todo, html_render", topic)
			}
			return guide, nil
		},
	})
}
