package guide

import (
	"context"
	"encoding/json"

	"github.com/pardnchiu/agenvoy/configs"
	toolRegister "github.com/pardnchiu/agenvoy/internal/tools/register"
	toolTypes "github.com/pardnchiu/agenvoy/internal/tools/types"
)

func registToolErrorGuide() {
	toolRegister.Regist(toolRegister.Def{
		Name:        "tool_error_guide",
		AlwaysAllow: true,
		AlwaysLoad:  true,
		Concurrent:  true,
		Description: "[system-default] Read FIRST on any tool call failure — before retry, search_error_history, patch_tool.\n\n" + configs.ToolErrorGuide,
		Parameters: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
		Handler: func(_ context.Context, _ *toolTypes.Executor, _ json.RawMessage) (string, error) {
			return configs.ToolErrorGuide, nil
		},
	})
}
