package variant

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	toolRegister "github.com/pardnchiu/agenvoy/internal/tools/register"
	toolTypes "github.com/pardnchiu/agenvoy/internal/tools/types"
)

func registEditTool() {
	toolRegister.Regist(toolRegister.Def{
		Name:        "edit_tool",
		AlwaysAllow: true,
		Description: `
Edit the tools themselves: create or overwrite a definition file (write), fix an exact string inside one (patch), trash an obsolete one (remove).
Replaces write_tool, patch_tool and remove_tool: an instruction naming any of those means this tool.
The Capability Gap flow is write → test_tool (script only) → call the new tool. Finding or activating an existing tool is find_tools.`,
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"mode": map[string]any{
					"type":        "string",
					"enum":        []string{"write", "patch", "remove"},
					"description": "write: full file content. patch: exact-string fix. remove: trash the tool directory — remove is never inferred, name it explicitly.",
				},
				"name": map[string]any{
					"type":        "string",
					"description": "Snake_case name without the 'script_' prefix; the runtime adds it (e.g. 'ip_geolocation_lookup').",
				},
				"tag": map[string]any{
					"type":        "string",
					"enum":        []string{"json", "script", "api"},
					"description": "mode=write / mode=patch: which file. json = tool.json (schema), script = script.py (runtime), api = <name>.json (API tool definition).",
				},
				"content": map[string]any{
					"type":        "string",
					"description": "mode=write: the complete file content, not a diff.",
				},
				"old_string": map[string]any{
					"type":        "string",
					"description": "mode=patch: exact string to replace. Must be unique in the target file.",
				},
				"new_string": map[string]any{
					"type":        "string",
					"description": "mode=patch: replacement string. Empty string deletes old_string.",
				},
				"replace_all": map[string]any{
					"type":        "boolean",
					"description": "mode=patch: replace every occurrence instead of the single unique one.",
					"default":     false,
				},
			},
			"required": []string{"name"},
		},
		Handler: func(ctx context.Context, e *toolTypes.Executor, args json.RawMessage) (string, error) {
			var params struct {
				Mode       string `json:"mode"`
				Name       string `json:"name"`
				Tag        string `json:"tag"`
				Content    string `json:"content"`
				OldString  string `json:"old_string"`
				NewString  string `json:"new_string"`
				ReplaceAll bool   `json:"replace_all"`
			}
			if err := json.Unmarshal(args, &params); err != nil {
				return "", fmt.Errorf("encoding/json: Unmarshal: %w", err)
			}

			mode := strings.ToLower(strings.TrimSpace(params.Mode))
			if mode == "" {
				switch {
				case params.Content != "":
					mode = "write"
				case params.OldString != "":
					mode = "patch"
				default:
					return "", fmt.Errorf("mode is required; available: write, patch, remove")
				}
			}

			switch mode {
			case "write":
				return writeToolFile(ctx, e, params.Name, params.Tag, params.Content)
			case "patch":
				return patchToolFile(ctx, e, params.Name, params.Tag, params.OldString, params.NewString, params.ReplaceAll)
			case "remove":
				return removeToolDir(ctx, e, params.Name)
			}
			return "", fmt.Errorf("unknown mode %q; available: write, patch, remove", mode)
		},
	})
}
