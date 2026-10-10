package toolSearcher

import (
	"context"
	"encoding/json"
	"fmt"

	toolRegister "github.com/pardnchiu/agenvoy/internal/tools/register"
	toolTypes "github.com/pardnchiu/agenvoy/internal/tools/types"
)

func registRunTool() {
	toolRegister.Regist(toolRegister.Def{
		Name:      "run_tool",
		SystemUse: true,
		Description: `Runs a tool by name from the Tools list given before the user input, with args matching its schema.
Every tool other than find_tools goes through here; the list holds names only.
Before the first call to a name, fetch its schema with find_tools(query="select:NAME[,NAME]"), several names at once. Keyword search only when no listed name fits.`,
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name": map[string]any{
					"type":        "string",
					"description": "Exact tool name from the Tools list.",
				},
				"args": map[string]any{
					"type":        "object",
					"description": "Arguments matching that tool's schema; {} when it takes none.",
					"default":     map[string]any{},
				},
			},
			"required": []string{"name"},
		},
		Handler: func(_ context.Context, _ *toolTypes.Executor, _ json.RawMessage) (string, error) {
			return "", fmt.Errorf("run_tool is resolved by the exec loop and cannot be dispatched directly")
		},
	})
}
