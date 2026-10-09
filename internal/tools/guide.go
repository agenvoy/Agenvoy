package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pardnchiu/agenvoy/configs"
	"github.com/pardnchiu/agenvoy/internal/session/config"
	toolRegister "github.com/pardnchiu/agenvoy/internal/tools/register"
	toolTypes "github.com/pardnchiu/agenvoy/internal/tools/types"
)

var topicNames = []string{"tool_generate", "tool_error", "rag_web", "market_analysis", "targeted_read", "ask_user", "subagent_dispatch", "write_todo", "html_render", "office"}

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
	"office":            configs.GuideOffice,
}

func registReasoningGuide() {
	toolRegister.Regist(toolRegister.Def{
		Name:        "reasoning_guide",
		SystemUse:   true,
		AlwaysAllow: true,
		Concurrent:  true,
		Description: `[system-default]
Full rule per topic — call before acting on any match, listing every topic that matches in one call:

- tool_error: read it after a tool call has come back failed, never before one has — recovery loop, script_*/api_* auto-repair via edit_tool(mode=patch), [RETRY_REQUIRED] handling. Read before retrying, before error_history, before edit_tool(mode=patch).
- tool_generate: request needs live external data (weather, currency, stock, geocoding, translation, ...) and no api_*/script_*/ext_* covers it — find_tools(mode=search) found nothing, or an existing one fails. Carries the build contract (naming, description rules, tool.json/script.py format, execution flow), then edit_tool(mode=write) → test_tool (script only) → call it. Hard gate, decided by what the turn leaves behind: a capability that will be called again — fetching it directly via http_request or run_command curl/python3 is PROHIBITED even with a known endpoint, fetch_page is for docs, the data fetch lives in script.py. A finding that answers this turn and is then done leaves no tool: call the endpoint as many times as the check needs and report what came back. A missing tool is built, never reported as a limitation.
- rag_web: non-smalltalk info query (people, orgs, facts, current events, prices, time-sensitive) — RAG and live web fire in parallel every time both are available; carries the source-citation rule and the fallback when one side is missing.
- market_analysis: stock/ETF/market analysis — assess the macro, regional, industry and asset-specific layers, not one alone.
- targeted_read: file question needs only specific symbols/sections/keywords — search first, narrow read_files over whole-file read.
- ask_user: missing target, vague scope, unclear spec, ambiguous time, scheduling without content, non-unique tool choice — resolve intent first.
- subagent_dispatch: the same lookup repeating across 3+ entities, a lookup spanning 2+ source classes, a set just discovered that now needs per-entity work, a named session ("call X"/"呼叫 X"), or a reusable single subtask — read before any subagents(mode=invoke).
- write_todo: analysis/research task, or complex multi-step task (request names three or more steps) — decide checklist before write_todo. Active Skill → its checklist follows the Skill Execution Rules (3+ steps) instead.
- html_render: producing an HTML deliverable (report, dashboard, chart, map, 3D view) — the gallery of worked examples to start from, the QuickUI rendering every page must go through, which libraries are allowed, breakpoints and visual direction, all before writing anything.
- office: creating or modifying a .docx / .xlsx / .pptx — package and registration rules that keep Word/Excel/PowerPoint and Pages/Numbers/Keynote from rejecting the file, Markdown-free text, and the check to run before delivering. Read before writing the file.`,
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"topics": map[string]any{
					"type":        "array",
					"items":       map[string]any{"type": "string", "enum": topicNames},
					"minItems":    1,
					"description": "Every Reasoning Rules topic this turn needs, in one call — a second call costs a whole round trip.",
				},
			},
			"required": []string{"topics"},
		},
		Handler: func(_ context.Context, _ *toolTypes.Executor, args json.RawMessage) (string, error) {
			var params struct {
				Topics []string `json:"topics"`
			}
			if err := json.Unmarshal(args, &params); err != nil {
				return "", fmt.Errorf("json.Unmarshal: %w", err)
			}

			selection := ""
			guides := make([]string, 0, len(params.Topics))
			seen := make(map[string]bool, len(params.Topics))
			for _, one := range params.Topics {
				topic := strings.TrimSpace(one)
				if topic == "" || seen[topic] {
					continue
				}
				guide, ok := topicGuides[topic]
				if !ok {
					return "", fmt.Errorf("unknown topic %q; available: %s", topic, strings.Join(topicNames, ", "))
				}
				if strings.Contains(guide, "{{.ModelSelection}}") {
					if selection == "" {
						selection = config.SubagentModelSelection(&config.Config{})
						if cfg, err := config.Load(); err == nil {
							selection = config.SubagentModelSelection(cfg)
						}
					}
					guide = strings.ReplaceAll(guide, "{{.ModelSelection}}", selection)
				}
				seen[topic] = true
				guides = append(guides, "## "+topic+"\n\n"+strings.TrimSpace(guide))
			}
			if len(guides) == 0 {
				return "", fmt.Errorf("topics is required; available: %s", strings.Join(topicNames, ", "))
			}
			return strings.Join(guides, "\n\n"), nil
		},
	})
}
