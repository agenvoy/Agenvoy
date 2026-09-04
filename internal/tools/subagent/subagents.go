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
		Description: `Runs a subtask in its own session (invoke), or looks up a named agent's self id (list).
Naming an agent is an order: 呼叫 X / 請 X / 找 X / call X / ask X → dispatch to X, never answer it yourself.
Also fan out when one lookup repeats across 3+ entities or 2+ source classes.
The leg's report comes back whole — relay it; "已呼叫" is not an answer.
One call per subtask, three at a time. Protocol and model ladder → reasoning_guide(topic=subagent_dispatch).`,
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"mode": map[string]any{
					"type":        "string",
					"enum":        []string{"invoke", "list"},
					"description": "invoke: run the task in a subagent session. list: the sessions whose self id contains `self_id`, with their roles — run it when the user delegates by name, then invoke with the self id it prints. Omitted: task → invoke, otherwise list.",
					"default":     "invoke",
				},
				"task": map[string]any{
					"type":        "string",
					"description": "mode=invoke: the subtask, written to stand on its own — the leg sees none of this conversation. Its result comes back prefixed [subagent · <model> · session=<id> · usage: ...], and that usage line is the leg's whole token cost, to be tallied across every fan-out call when reporting this turn's cost.",
				},
				"self_id": map[string]any{
					"type":        "string",
					"description": "mode=list: required, the delegated name to resolve; only sessions whose self id contains it come back. mode=invoke: the self id of the existing non-temp session to run in, spelled exactly as mode=list prints it — set it verbatim when the user delegates by name, otherwise leave EMPTY. Never invent a descriptive label: an unmatched value resolves to nothing and the run becomes a temp session anyway. Broad parallel fan-out stays anonymous.",
					"default":     "",
				},
				"model": map[string]any{
					"type":        "string",
					"description": "mode=invoke: worker model, used only when the run lands in a temp session — set it whenever `self_id` is empty. A `self_id` that resolves to an existing session ignores this field and runs under that session's own configured model and reasoning. For a temp run always set it; blank spends an extra dispatcher call and over-selects for what is plain collection work. DEFAULT TIER — take the first of these the registry offers and stay here unless an escalation trigger below fires: `*-nano` → `*-mini` → `deepseek-flash` → `claude-haiku` → `*-luna` → `grok`. Fetching, listing, scraping, single-source lookup, format conversion and per-entity fan-out legs stay in this tier however many legs there are — width is not complexity. ESCALATE one rung, and only when the task text itself names the difficulty: cross-checking 3+ sources that can disagree → `*-terra`; the leg must write reasoned prose rather than return gathered facts → `claude-sonnet`/`gemini-pro`; a long tool chain where a wrong early call invalidates everything after → `glm`/`k3`. Wanting `*-sol`/`claude-opus` means the task should have been split — split it instead. `-sol`/`-terra`/`-luna` are rungs, not versions: `gpt-5.6-terra` sits at `*-terra`. Open-weight models under `100b` (`*-20b`, `*-8b`) have unreliable tool-calling — sole candidate only; `*-nano`/`*-mini` are hosted rungs and do not fall under this.",
					"default":     "",
					"enum":        models,
				},
				"reasoning": map[string]any{
					"type":        "string",
					"enum":        reasoningLevels,
					"default":     "low",
					"description": "mode=invoke: thinking depth, used only when the run lands in a temp session; a resolved `self_id` uses that session's own setting. Keep `low` — gathering needs none and depth multiplies across the fan-out. Raise it only for a leg whose own written output must reason rather than gather.",
				},
				"new": map[string]any{
					"type":        "boolean",
					"description": "mode=invoke: start the leg with a clean slate — prior conversation history, summary and past tool records of whatever session it lands in are ignored for this run (nothing is deleted). Omitted: true when no session is named, false when `self_id` resolves to one. Pass false to continue a session's earlier work — including a temp session whose previous leg failed, so the new leg sees what was already gathered.",
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
				if strings.TrimSpace(params.SelfID) == "" {
					return "", fmt.Errorf("self_id is required when mode=list")
				}
				return listSessions(params.SelfID), nil
			}
			return "", fmt.Errorf("unknown mode %q; available: invoke, list", mode)
		},
	})
}
