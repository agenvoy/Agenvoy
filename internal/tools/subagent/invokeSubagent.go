package subagent

import (
	"context"
	"encoding/json"
	"fmt"
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
		Description: "Spawn a subagent in its own session. For a SINGLE delegated subtask, first `list_subagent_sessions` — if a listed role fits the task, `ask_user` whether to route there; on yes set `name` to that session's name, on no leave `name` EMPTY (temp). Set `name` verbatim also when the user explicitly delegates to a session (呼叫/請/找/call/ask/let X do Y — X is that name). Otherwise leave `name` EMPTY — never invent a descriptive label (e.g. 'market-news-24h'); an unmatched name resolves to nothing and the run becomes a temp session regardless. Broad PARALLEL fan-out skips this check and stays anonymous (name empty). One call per distinct subtask — never duplicate the same task.",
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
					"description": "Worker model name. Leave blank for dispatcher auto-select.",
					"default":     "",
					"enum":        models,
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
		Handler: func(ctx context.Context, _ *toolTypes.Executor, args json.RawMessage) (string, error) {
			var params struct {
				Task         string   `json:"task"`
				Name         string   `json:"name,omitempty"`
				SessionID    string   `json:"session_id,omitempty"`
				Model        string   `json:"model,omitempty"`
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

			systemPrompt := strings.TrimSpace(params.SystemPrompt)

			excludeTools := params.ExcludeTools
			if excludeTools == nil {
				excludeTools = []string{}
			}

			return exec.ExecWithSubagent(ctx, task, sessionID, model, systemPrompt, excludeTools)
		},
	})
}
