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
	registSubagent()
}

func registSubagent() {
	models := []string{}
	for _, m := range exec.GetAgent() {
		if m.Name != "" {
			models = append(models, m.Name)
		}
	}

	toolRegister.Regist(toolRegister.Def{
		Name:        "subagent",
		AlwaysAllow: true,
		Concurrent:  true,
		Timeout:     time.Duration(filesystem.MaxSubagentTimeoutMin) * time.Minute,
		Description: "Run a subtask in its own session (mode=invoke), or list the named sessions available to run it (mode=list). For a SINGLE delegated subtask, `mode=list` first — if a listed role fits the task, `ask_user` whether to route there; on yes set `name` to that session's name, on no leave `name` EMPTY (temp). Set `name` verbatim also when the user explicitly delegates to a session (呼叫/請/找/call/ask/let X do Y — X is that name). Otherwise leave `name` EMPTY — never invent a descriptive label (e.g. 'market-news-24h'); an unmatched name resolves to nothing and the run becomes a temp session regardless. Broad PARALLEL fan-out skips the list check and stays anonymous (name empty). One call per distinct subtask — never duplicate the same task. At most 3 legs run concurrently; a 4th queues while its own timeout runs, so dispatch wide fan-outs in batches of 3. Result is prefixed `[subagent · <model> · session=<id> · usage: in=X out=Y cached=Z]` — that usage line is this subagent's total token cost (summed across its own tool-call loop); tally it across all fan-out calls when reporting your own turn's cost.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"mode": map[string]any{
					"type":        "string",
					"enum":        []string{"invoke", "list"},
					"description": "invoke: run the task in a subagent session, needs task. list: reusable named sessions with their roles, no other field. Omitted: task selects invoke, otherwise list.",
					"default":     "invoke",
				},
				"task": map[string]any{
					"type":        "string",
					"description": "mode=invoke: self-contained task description for the subagent.",
				},
				"name": map[string]any{
					"type":        "string",
					"description": "mode=invoke: name of ANY existing (non-temp) session to reuse, matching its bot.md frontmatter `name`. Leave EMPTY for a fresh/anonymous subtask — never invent a descriptive label here; a non-matching name is ignored and the subtask runs as an unlabeled temp session. Resolves to its session_id; takes precedence over session_id when both are set.",
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
