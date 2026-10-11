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

var topicNames = []string{"tool_generate", "tool_error", "web_search", "market_analysis", "targeted_read", "ask_user", "subagent_dispatch", "write_todo", "html_render", "office"}

var topicGuides = map[string]string{
	"tool_generate":     configs.GuideToolGenerate,
	"tool_error":        configs.GuideToolError,
	"web_search":        configs.GuideWebSearch,
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
		Description: `[system-default] Full rules for the listed topics; call before acting on any match, every matching topic in one call:
- tool_error: a tool call just failed — before retrying, error_history or edit_tool(mode=patch).
- tool_generate: live external data (weather, FX, stock, geocoding, translation...) no api_*/script_*/ext_* covers, or the one found fails. A capability called again is built, never fetched via http_request / run_command curl; a one-off check calls the endpoint and reports.
- web_search: non-smalltalk info query (people, orgs, facts, current events, prices).
- market_analysis: stock / ETF / market analysis or an entry decision.
- targeted_read: a file question needing only some symbols, sections or keywords.
- ask_user: missing target, vague scope or spec, ambiguous time, scheduling without content, more than one fitting tool.
- subagent_dispatch: one lookup across 3+ entities or 2+ source classes, a discovered set needing per-entity work, a named session ("call X" / "呼叫 X"), a reusable subtask — before subagents(mode=invoke).
- write_todo: analysis / research / evaluation, or work needing more than one round of tool calls — before the first tool call; an active Skill follows the Skill Execution Rules.
- html_render: an HTML deliverable, before writing any markup.
- office: creating or modifying .docx / .xlsx / .pptx, before writing the file.`,
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"topics": map[string]any{
					"type":        "array",
					"items":       map[string]any{"type": "string", "enum": topicNames},
					"minItems":    1,
					"description": "Every topic this turn needs, in one call.",
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
