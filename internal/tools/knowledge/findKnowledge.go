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

func init() {
	registFindKnowledge()
}

func registFindKnowledge() {
	toolRegister.Regist(toolRegister.Def{
		Name:        "find_knowledge",
		AlwaysAllow: true,
		Concurrent:  true,
		Description: `Notes the operator wrote — house rules, conventions, background this workspace assumes.
mode=search for 我們的規範是什麼 / 這個專案怎麼做, and before answering anything a local convention would override; mode=list for 有哪些筆記.
Both return names only, never content — judge which names matter, then mode=read those, one call each, issued together.
Past runs and conversation → chat_history; files on disk → find_files.`,
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"mode": map[string]any{
					"type":        "string",
					"enum":        []string{"search", "list", "read"},
					"description": "search: names of the notes matching the keywords, most matched first. list: the name of every note. read: one whole note, by name. Omitted: keywords selects search, name selects read, otherwise list.",
					"default":     "search",
				},
				"keywords": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type": "string",
					},
					"description": "mode=search: terms to look for, matched case-insensitively as substrings. Send several — synonyms and both languages of a term — since a note matching more of them ranks higher.",
				},
				"name": map[string]any{
					"type":        "string",
					"description": "mode=read: the note name, exactly as list or a search hit spells it.",
				},
				"limit": map[string]any{
					"type":        "integer",
					"description": "mode=search: notes to return, most matched first; never above 20. Ignored when mode=list.",
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
				switch {
				case len(params.Keywords) > 0:
					mode = "search"
				case strings.TrimSpace(params.Name) != "":
					mode = "read"
				default:
					mode = "list"
				}
			}

			switch mode {
			case "list":
				records := knowledge.List()
				if len(records) == 0 {
					return "no note", nil
				}
				names := make([]string, 0, len(records))
				for _, record := range records {
					names = append(names, record.Name)
				}
				return marshal(names)
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
			return "", fmt.Errorf("unknown mode %q; available: search, list, read", mode)
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
