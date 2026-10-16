package toolSearcher

import (
	"context"
	"encoding/json"

	toolregister "github.com/pardnchiu/agenvoy/internal/tools/register"
	tooltypes "github.com/pardnchiu/agenvoy/internal/tools/types"
	utilsmsg "github.com/pardnchiu/agenvoy/internal/utils/msg"
)

func registRunTool() {
	toolregister.Regist(toolregister.Def{
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
		Handler: func(_ context.Context, _ *tooltypes.Executor, _ json.RawMessage) (string, error) {
			return "", utilsmsg.New(utilsmsg.Internal, "run_tool", "resolved by the exec loop and cannot be dispatched directly")
		},
	})
}
