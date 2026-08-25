package subagent

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	provider "github.com/pardnchiu/go-llm-router/core"

	"github.com/pardnchiu/agenvoy/internal/agents/exec"
	"github.com/pardnchiu/agenvoy/internal/session"
	toolTypes "github.com/pardnchiu/agenvoy/internal/tools/types"
)

var reasoningLevels = func() []string {
	out := make([]string, 0, int(provider.ReasoningMax)+1)
	for r := provider.ReasoningNone; r <= provider.ReasoningMax; r++ {
		out = append(out, r.String())
	}
	return out
}()

type invokeParams struct {
	Mode         string   `json:"mode,omitempty"`
	Task         string   `json:"task"`
	Name         string   `json:"name,omitempty"`
	SessionID    string   `json:"session_id,omitempty"`
	Model        string   `json:"model,omitempty"`
	Reasoning    string   `json:"reasoning,omitempty"`
	SystemPrompt string   `json:"system_prompt,omitempty"`
	ExcludeTools []string `json:"exclude_tools,omitempty"`
}

func invokeSubagent(ctx context.Context, e *toolTypes.Executor, params invokeParams, models []string) (string, error) {
	task := strings.TrimSpace(params.Task)
	if task == "" {
		return "", fmt.Errorf("task is required when mode=invoke")
	}

	sessionID := strings.TrimSpace(params.SessionID)
	if name := strings.TrimSpace(params.Name); name != "" {
		if resolved := session.GetSessionID(name); resolved != "" {
			sessionID = resolved
		}
	}

	model := strings.TrimSpace(params.Model)
	if model != "" && !slices.Contains(models, model) {
		slog.Debug("invalid model, fallback to auto-select",
			slog.String("session", sessionID))
		model = ""
	}

	reasoning := strings.TrimSpace(params.Reasoning)
	if _, ok := provider.ParseReasoning(reasoning); !ok {
		reasoning = provider.ReasoningLow.String()
	}

	excludeTools := params.ExcludeTools
	if excludeTools == nil {
		excludeTools = []string{}
	}

	return exec.ExecWithSubagent(ctx, task, sessionID, model, reasoning,
		strings.TrimSpace(params.SystemPrompt), excludeTools, e.SessionID)
}
