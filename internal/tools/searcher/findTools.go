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
		Description: `
The tool registry itself: what exists, and pulling a tool's schema in so it can be called.
mode=search when a capability isn't loaded — keywords, or 'select:<name>' to activate by exact name.
mode=list to see what is available, name plus one line each, without loading any schema.
Prefer unmarked tools (mcp__* > script_* > api_*) over [system-default] for the same intent.
Replaces search_tools and list_tools: an instruction naming either means this tool — search_tools → mode=search, list_tools → mode=list.`,
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"mode": map[string]any{
					"type":        "string",
					"enum":        []string{"search", "list"},
					"description": "search: match the registry and inject the schemas, needs query. list: names and one-line descriptions only. Omitted: query selects search, otherwise list.",
					"default":     "search",
				},
				"query": map[string]any{
					"type":        "string",
					"description": `mode=search: keywords (all must match), or "select:<name>,<name>" for exact activation.`,
				},
				"mcp": map[string]any{
					"type":        "boolean",
					"description": "mode=list: when true, only MCP-exposed tools — builtin + script_/api_/ext_ prefixed.",
					"default":     false,
				},
				"system": map[string]any{
					"type":        "boolean",
					"description": "mode=list: when true, also list system tools used for internal bookkeeping.",
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
