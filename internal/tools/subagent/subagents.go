package subagent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/pardnchiu/agenvoy/internal/agents/exec"
	"github.com/pardnchiu/agenvoy/internal/filesystem"
	toolRegister "github.com/pardnchiu/agenvoy/internal/tools/register"
	toolTypes "github.com/pardnchiu/agenvoy/internal/tools/types"
)

func Register() {
	registSubagents()
}

func registSubagents() {
	models := []string{}
	for _, m := range exec.GetAgent() {
		if m.Name != "" {
			models = append(models, m.Name)
		}
	}

	toolRegister.Regist(toolRegister.Def{
		Name:        "subagents",
		SystemUse:   false,
		AlwaysLoad:  false,
		AlwaysAllow: true,
		Concurrent:  true,
		Timeout:     time.Duration(filesystem.MaxSubagentTimeoutMin) * time.Minute,
		Description: `Runs a subtask in its own session (invoke), or lists the named sessions available to run it (list).
Use when the same lookup repeats across 3+ entities, spans 2+ source classes, or the user delegates by name (呼叫 X / 請 X / 找 X).
One call per distinct subtask, at most three at a time — a fourth queues while its own timeout runs. The full dispatch protocol and the model ladder → reasoning_guide(topic=subagent_dispatch).`,
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"mode": map[string]any{
					"type":        "string",
					"enum":        []string{"invoke", "list"},
					"description": "invoke: run the task in a subagent session. list: the reusable named sessions and their roles — run it first for a single delegated subtask, then ask_user whether to route there. Omitted: task → invoke, otherwise list.",
					"default":     "invoke",
				},
				"task": map[string]any{
					"type":        "string",
					"description": "mode=invoke: the subtask, written to stand on its own — the leg sees none of this conversation. Its result comes back prefixed [subagent · <model> · session=<id> · usage: …], and that usage line is the leg's whole token cost, to be tallied across every fan-out call when reporting this turn's cost.",
				},
				"name": map[string]any{
					"type":        "string",
					"description": "mode=invoke: an existing non-temp session to reuse, matching its `bot.json` `name` — set it verbatim when the user delegates by name, otherwise leave EMPTY. Never invent a descriptive label: an unmatched name resolves to nothing and the run becomes a temp session anyway. Broad parallel fan-out stays anonymous. Takes precedence over session_id.",
					"default":     "",
				},
				"session_id": map[string]any{
					"type":        "string",
					"description": "mode=invoke: persistent session id to thread multi-turn subagent calls (e.g. 'researcher', 'dispatcher-2'). Blank uses an ephemeral temp session. Ignored when name resolves successfully.",
					"default":     "",
				},
				"model": map[string]any{
					"type":        "string",
					"description": "mode=invoke: worker model, used only when the run lands in a temp session — set it whenever `name` is empty. A `name` that resolves to an existing session ignores this field and runs under that session's own configured model and reasoning. For a temp run always set it; blank spends an extra dispatcher call and over-selects for what is plain collection work. DEFAULT TIER — take the first of these the registry offers and stay here unless an escalation trigger below fires: `gpt-oss-120b` → `*-nano` → `*-mini` → `deepseek-flash` → `claude-haiku` → `*-luna` → `grok`. Fetching, listing, scraping, single-source lookup, format conversion and per-entity fan-out legs stay in this tier however many legs there are — width is not complexity. ESCALATE one rung, and only when the task text itself names the difficulty: cross-checking 3+ sources that can disagree → `*-terra`; the leg must write reasoned prose rather than return gathered facts → `claude-sonnet`/`gemini-pro`; a long tool chain where a wrong early call invalidates everything after → `glm`/`k3`. Wanting `*-sol`/`claude-opus` means the task should have been split — split it instead. `-sol`/`-terra`/`-luna` are rungs, not versions: `gpt-5.6-terra` sits at `*-terra`. Open-weight models under `100b` (`*-20b`, `*-8b`) have unreliable tool-calling — sole candidate only; `*-nano`/`*-mini` are hosted rungs and do not fall under this.",
					"default":     "",
					"enum":        models,
				},
				"reasoning": map[string]any{
					"type":        "string",
					"enum":        reasoningLevels,
					"default":     "low",
					"description": "mode=invoke: thinking depth, used only when the run lands in a temp session; a resolved `name` uses that session's own setting. Keep `low` — gathering needs none and depth multiplies across the fan-out. Raise it only for a leg whose own written output must reason rather than gather.",
				},
				"system_prompt": map[string]any{
					"type":        "string",
					"description": "mode=invoke: extra role or constraints appended to the subagent's system prompt.",
					"default":     "",
				},
				"exclude_tools": map[string]any{
					"type":        "array",
					"items":       map[string]any{"type": "string"},
					"description": "mode=invoke: extra tool names to exclude on top of the always-excluded set (subagent, write_file, patch_file). The default set cannot be overridden.",
					"default":     []string{},
				},
			},
		},
		Handler: func(ctx context.Context, e *toolTypes.Executor, args json.RawMessage) (string, error) {
			var params invokeParams
			if len(args) > 0 {
				if err := json.Unmarshal(args, &params); err != nil {
					return "", fmt.Errorf("json.Unmarshal: %w", err)
				}
			}

			mode := strings.TrimSpace(params.Mode)
			if mode == "" {
				mode = "list"
				if strings.TrimSpace(params.Task) != "" {
					mode = "invoke"
				}
			}

			switch mode {
			case "invoke":
				return invokeSubagent(ctx, e, params, models)
			case "list":
				return listSessions(), nil
			}
			return "", fmt.Errorf("unknown mode %q; available: invoke, list", mode)
		},
	})
}
