package guide

import (
	"context"
	"encoding/json"

	"github.com/pardnchiu/agenvoy/configs"
	toolRegister "github.com/pardnchiu/agenvoy/internal/tools/register"
	toolTypes "github.com/pardnchiu/agenvoy/internal/tools/types"
)

func registSubagentDispatchGuide() {
	toolRegister.Regist(toolRegister.Def{
		Name:        "subagent_dispatch_guide",
		AlwaysAllow: true,
		AlwaysLoad:  true,
		Concurrent:  true,
		Description: `Full planner-mode protocol for subagent fan-out (decomposition, parallel dispatch, multi-source mandate, synthesis rules). Call before parallel invoke_subagent calls for broad-analysis/multi-source fan-out — not preloaded in system prompt.`,
		Parameters: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
		Handler: func(_ context.Context, _ *toolTypes.Executor, _ json.RawMessage) (string, error) {
			return configs.SubagentDispatchGuide, nil
		},
	})
}
