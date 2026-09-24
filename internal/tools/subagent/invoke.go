package subagent

import (
	"context"
	"fmt"
	"slices"
	"strings"

	provider "github.com/pardnchiu/go-llm-router/core"

	"github.com/pardnchiu/agenvoy/internal/agents"
	"github.com/pardnchiu/agenvoy/internal/agents/exec"
	"github.com/pardnchiu/agenvoy/internal/session"
	"github.com/pardnchiu/agenvoy/internal/session/config"
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
	SelfID       string   `json:"self_id,omitempty"`
	Model        string   `json:"model,omitempty"`
	Reasoning    string   `json:"reasoning,omitempty"`
	SystemPrompt string   `json:"system_prompt,omitempty"`
	New          *bool    `json:"new,omitempty"`
	ExcludeTools []string `json:"exclude_tools,omitempty"`
}

func invokeSubagent(ctx context.Context, e *toolTypes.Executor, params invokeParams) (string, error) {
	task := strings.TrimSpace(params.Task)
	if task == "" {
		return "", fmt.Errorf("task is required when mode=invoke")
	}

	sessionID := ""
	if selfID := strings.TrimSpace(params.SelfID); selfID != "" {
		sessionID = session.GetSessionIDBySelfID(selfID)
	}

	model := strings.TrimSpace(params.Model)
	if model != "" {
		if err := checkLegModel(model); err != nil {
			return "", err
		}
	}

	reasoning := strings.TrimSpace(params.Reasoning)
	if _, ok := provider.ParseReasoning(reasoning); !ok {
		reasoning = provider.ReasoningLow.String()
	}

	excludeTools := params.ExcludeTools
	if excludeTools == nil {
		excludeTools = []string{}
	}

	ignoreHistory := sessionID == ""
	if params.New != nil {
		ignoreHistory = *params.New
	}

	return exec.ExecWithSubagent(ctx, task, sessionID, model, reasoning,
		strings.TrimSpace(params.SystemPrompt), excludeTools, e.SessionID, ignoreHistory)
}

func checkLegModel(model string) error {
	registry := agents.Registry()
	tags := map[string]string{}
	if cfg, err := config.Load(); err == nil {
		tags = cfg.ModelTag
	}

	allowed := make([]string, 0, len(registry.Entries))
	for _, e := range registry.Entries {
		if tags[e.Name] != config.ModelTagPass {
			allowed = append(allowed, e.Name)
		}
	}
	if slices.Contains(allowed, model) {
		return nil
	}

	reason := "is not registered"
	if _, ok := registry.Registry[model]; ok {
		reason = "is pass tier and cannot run a subagent leg"
	}
	return fmt.Errorf("model %q %s; available: %s", model, reason, strings.Join(allowed, ", "))
}
