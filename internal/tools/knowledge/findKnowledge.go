package toolKnowledge

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pardnchiu/agenvoy/internal/knowledge"
	toolRegister "github.com/pardnchiu/agenvoy/internal/tools/register"
	toolTypes "github.com/pardnchiu/agenvoy/internal/tools/types"
)

const Name = "find_knowledge"

func init() {
	registFindKnowledge()
}

func registFindKnowledge() {
	toolRegister.Regist(toolRegister.Def{
		Name:        Name,
		SystemUse:   true,
		AlwaysLoad:  true,
		AlwaysAllow: true,
		Concurrent:  true,
		Description: `Notes the operator wrote — house rules, conventions, background this workspace assumes.
mode=search for 我們的規範是什麼 / 這個專案怎麼做 / 有哪些筆記, and before answering anything a local convention would override.
It returns names only, never content — judge which names matter, then mode=read those, one call each, issued together.
Past runs and conversation → chat_history; files on disk → find_files.`,
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"mode": map[string]any{
					"type":        "string",
					"enum":        []string{"search", "read"},
					"description": "search: names of the notes whose name or body matches the keywords, most matched first. read: one whole note, by name. Omitted: name selects read, otherwise search.",
					"default":     "search",
				},
				"keywords": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type": "string",
					},
					"description": "mode=search: terms to look for, matched case-insensitively as substrings; required, a call without them lists nothing. Send several — synonyms and both languages of a term — since a note matching more of them ranks higher.",
				},
				"name": map[string]any{
					"type":        "string",
					"description": "mode=read: the note name, exactly as a search hit spells it.",
				},
				"limit": map[string]any{
					"type":        "integer",
					"description": "mode=search: notes to return, most matched first; never above 20.",
					"default":     knowledge.DefaultLimit,
				},
			},
		},
		Handler: func(_ context.Context, _ *toolTypes.Executor, args json.RawMessage) (string, error) {
			var params struct {
				Mode     string   `json:"mode"`
				Keywords []string `json:"keywords"`
				Name     string   `json:"name"`
				Limit    int      `json:"limit"`
			}
			if len(args) > 0 {
				if err := json.Unmarshal(args, &params); err != nil {
					return "", fmt.Errorf("json.Unmarshal: %w", err)
				}
			}

			mode := strings.ToLower(strings.TrimSpace(params.Mode))
			if mode == "" {
				mode = "search"
				if len(params.Keywords) == 0 && strings.TrimSpace(params.Name) != "" {
					mode = "read"
				}
			}

			switch mode {
			case "read":
				name := strings.TrimSpace(params.Name)
				if name == "" {
					return "", fmt.Errorf("name is required when mode=read")
				}
				record, ok := knowledge.Read(name)
				if !ok {
					return "", fmt.Errorf("no note named %q", name)
				}
				return marshal(record)

			case "search":
				if len(params.Keywords) == 0 {
					return "", fmt.Errorf("keywords is required when mode=search")
				}
				hits := knowledge.Search(params.Keywords, params.Limit)
				if len(hits) == 0 {
					return "no matching note", nil
				}
				return marshal(hits)
			}
			return "", fmt.Errorf("unknown mode %q; available: search, read", mode)
		},
	})
}

func marshal(value any) (string, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("json.Marshal: %w", err)
	}
	return string(raw), nil
}
