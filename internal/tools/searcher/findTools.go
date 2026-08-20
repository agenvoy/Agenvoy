package toolSearcher

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	toolRegister "github.com/pardnchiu/agenvoy/internal/tools/register"
	toolTypes "github.com/pardnchiu/agenvoy/internal/tools/types"
)

const systemDefaultMarker = "[system-default]"

type Tool struct {
	Name          string `json:"name"`
	Description   string `json:"description"`
	SystemDefault bool   `json:"system_default,omitempty"`
}

func registFindTools() {
	toolRegister.Regist(toolRegister.Def{
		Name:        "find_tools",
		AlwaysAllow: true,
		AlwaysLoad:  true,
		Concurrent:  true,
		SystemUse:   true,
		Description: `The tool registry: what exists (list), and pulling a tool's schema in so it can be called (search).
Use for 有哪些工具 / 找工具 / 有沒有可以…的工具, and for search_tools / list_tools.
A capability that seems missing comes from here before anything is built. Building one → edit_tool; running a script tool → test_tool.`,
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"mode": map[string]any{
					"type":        "string",
					"enum":        []string{"search", "list"},
					"description": "search: match the registry and inject the schemas. list: names and one-line descriptions only. Omitted: query → search, otherwise list.",
					"default":     "search",
				},
				"query": map[string]any{
					"type":        "string",
					"description": `mode=search: keywords, all of which must match, or "select:<name>,<name>" to activate by exact name. Prefer unmarked tools (mcp__* > script_* > api_*) over [system-default] for the same intent.`,
				},
				"mcp": map[string]any{
					"type":        "boolean",
					"description": "mode=list: only MCP-exposed tools — builtin + script_/api_/ext_ prefixed.",
					"default":     false,
				},
				"system": map[string]any{
					"type":        "boolean",
					"description": "mode=list: also list the system tools used for internal bookkeeping.",
					"default":     false,
				},
			},
		},
		Handler: func(_ context.Context, e *toolTypes.Executor, args json.RawMessage) (string, error) {
			var params struct {
				Mode   string `json:"mode"`
				Query  string `json:"query"`
				MCP    bool   `json:"mcp"`
				System bool   `json:"system"`
			}
			if len(args) > 0 {
				if err := json.Unmarshal(args, &params); err != nil {
					return "", fmt.Errorf("json Unmarshal: %w", err)
				}
			}

			params.Mode = strings.TrimSpace(params.Mode)
			params.Query = strings.TrimSpace(params.Query)
			if params.Mode == "" {
				params.Mode = "list"
				if params.Query != "" {
					params.Mode = "search"
				}
			}

			switch params.Mode {
			case "search":
				if params.Query == "" {
					return "", fmt.Errorf("query is required when mode=search")
				}
				return searchTools(e, params.Query)
			case "list":
				return listTools(e, params.MCP, params.System)
			}
			return "", fmt.Errorf("unknown mode %q; available: search, list", params.Mode)
		},
	})
}
