package subagent

import (
	"context"
	"encoding/json"
	"fmt"
	provider "github.com/pardnchiu/go-llm-router/core"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/pardnchiu/agenvoy/internal/agents/exec"
	"github.com/pardnchiu/agenvoy/internal/filesystem"
	"github.com/pardnchiu/agenvoy/internal/session"
	toolRegister "github.com/pardnchiu/agenvoy/internal/tools/register"
	toolTypes "github.com/pardnchiu/agenvoy/internal/tools/types"
)

var reasoningLevels = func() []string {
	out := make([]string, 0, int(provider.ReasoningMax)+1)
	for r := provider.ReasoningNone; r <= provider.ReasoningMax; r++ {
		out = append(out, r.String())
	}
	return out
}()

func registInvokeSubagent() {
	models := []string{}
	for _, m := range exec.GetAgent() {
		if m.Name != "" {
			models = append(models, m.Name)
		}
	}

	toolRegister.Regist(toolRegister.Def{
		Name:        "invoke_subagent",
		AlwaysAllow: true,
		Concurrent:  true,
		Timeout:     time.Duration(filesystem.MaxSubagentTimeoutMin) * time.Minute,
		Description: "Spawn a subagent in its own session. For a SINGLE delegated subtask, first `list_subagent_sessions` — if a listed role fits the task, `ask_user` whether to route there; on yes set `name` to that session's name, on no leave `name` EMPTY (temp). Set `name` verbatim also when the user explicitly delegates to a session (呼叫/請/找/call/ask/let X do Y — X is that name). Otherwise leave `name` EMPTY — never invent a descriptive label (e.g. 'market-news-24h'); an unmatched name resolves to nothing and the run becomes a temp session regardless. Broad PARALLEL fan-out skips this check and stays anonymous (name empty). One call per distinct subtask — never duplicate the same task. Result is prefixed `[subagent · <model> · session=<id> · usage: in=X out=Y cached=Z]` — that usage line is this subagent's total token cost (summed across its own tool-call loop); tally it across all fan-out calls when reporting your own turn's cost.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"task": map[string]any{
					"type":        "string",
					"description": "Self-contained task description for the subagent.",
				},
				"name": map[string]any{
					"type":        "string",
					"description": "Name of ANY existing (non-temp) session to reuse, matching its bot.md frontmatter `name`. Leave EMPTY for a fresh/anonymous subtask — never invent a descriptive label here; a non-matching name is ignored and the subtask runs as an unlabeled temp session. Resolves to its session_id; takes precedence over session_id when both are set.",
					"default":     "",
				},
				"session_id": map[string]any{
					"type":        "string",
					"description": "Persistent session id to thread multi-turn subagent calls (e.g. 'researcher', 'dispatcher-2'). Blank uses an ephemeral temp session. Ignored when name resolves successfully.",
					"default":     "",
				},
				"model": map[string]any{
					"type":        "string",
					"description": "Worker model — always set it; blank spends an extra dispatcher call and over-selects for what is plain collection work. Walk fastest-first, take the first that can do the subtask: `gpt-oss-120b` → `deepseek-flash` → `grok` → `*-luna` → `claude-haiku` → `gemini-flash` → `*-terra` → `claude-sonnet` → `deepseek-pro` → `gemini-pro` → `glm` → `k3` → `*-sol` → `claude-opus`. Collection lands in the first half; pass `*-terra` only to cross-verify many sources through a long tool loop. Needing `*-sol`/`claude-opus` means the task should have been split further — take one, or anything below `gpt-oss-120b` where tool-calling turns unreliable, only as the sole candidate. `-sol`/`-terra`/`-luna` are rungs, not versions: `gpt-5.6-terra` sits at `*-terra`.",
					"default":     "",
					"enum":        models,
				},
				"reasoning": map[string]any{
					"type":        "string",
					"enum":        reasoningLevels,
					"default":     "low",
					"description": "Thinking depth. Keep `low` — gathering needs none and depth multiplies across the fan-out. Raise it only for a leg whose own written output must reason rather than gather.",
				},
				"system_prompt": map[string]any{
					"type":        "string",
					"description": "Extra role or constraints appended to the subagent's system prompt.",
					"default":     "",
				},
				"exclude_tools": map[string]any{
					"type":        "array",
					"items":       map[string]any{"type": "string"},
					"description": "Extra tool names to exclude on top of the always-excluded set (invoke_subagent, list_subagent_sessions). The default set cannot be overridden.",
					"default":     []string{},
				},
			},
			"required": []string{
				"task",
			},
		},
		Handler: func(ctx context.Context, e *toolTypes.Executor, args json.RawMessage) (string, error) {
			var params struct {
				Task         string   `json:"task"`
				Name         string   `json:"name,omitempty"`
				SessionID    string   `json:"session_id,omitempty"`
				Model        string   `json:"model,omitempty"`
				Reasoning    string   `json:"reasoning,omitempty"`
				SystemPrompt string   `json:"system_prompt,omitempty"`
				ExcludeTools []string `json:"exclude_tools,omitempty"`
			}
			if err := json.Unmarshal(args, &params); err != nil {
				return "", fmt.Errorf("json.Unmarshal: %w", err)
			}

			task := strings.TrimSpace(params.Task)
			if task == "" {
				return "", fmt.Errorf("task is required")
			}

			sessionID := strings.TrimSpace(params.SessionID)
			if name := strings.TrimSpace(params.Name); name != "" {
				if resolved := session.GetSessionID(name); resolved != "" {
					sessionID = resolved
				}
			}

			model := strings.TrimSpace(params.Model)
			if model != "" && !slices.Contains(models, model) {
				slog.Warn("invalid model, fallback to auto-select",
					slog.String("session", sessionID))
				model = ""
			}

			reasoning := strings.TrimSpace(params.Reasoning)
			if _, ok := provider.ParseReasoning(reasoning); !ok {
				reasoning = provider.ReasoningLow.String()
			}

			systemPrompt := strings.TrimSpace(params.SystemPrompt)

			excludeTools := params.ExcludeTools
			if excludeTools == nil {
				excludeTools = []string{}
			}

			return exec.ExecWithSubagent(ctx, task, sessionID, model, reasoning, systemPrompt, excludeTools, e.SessionID)
		},
	})
}
