package variant

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	toolRegister "github.com/pardnchiu/agenvoy/internal/tools/register"
	toolTypes "github.com/pardnchiu/agenvoy/internal/tools/types"
)

func registEditSkill() {
	toolRegister.Regist(toolRegister.Def{
		Name:        "edit_skill",
		AlwaysAllow: true,
		Description: `The files under the skills directory: create or rewrite one (write), replace an exact string inside one (patch), trash a whole skill (remove).
Use for 改 skill / 新增 skill / 這個 skill 要調整, and for write_skill / patch_skill / remove_skill.
Running a skill → run_skill; ordinary files → edit_file; building a new skill from scratch → the skill-creator skill.`,
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"mode": map[string]any{
					"type":        "string",
					"enum":        []string{"write", "patch", "remove"},
					"description": "write: a first version or a full rewrite. patch: a targeted edit. remove: trashes the whole skill directory, recoverable from .Trash. Omitted: content → write, old_string → patch, name alone → remove.",
				},
				"path": map[string]any{
					"type":        "string",
					"description": "mode=write / mode=patch: relative path under the skills dir (e.g. 'my-skill/SKILL.md', 'my-skill/scripts/01.md').",
				},
				"content": map[string]any{
					"type":        "string",
					"description": "mode=write: the complete file content, not a diff.",
				},
				"old_string": map[string]any{
					"type":        "string",
					"description": "mode=patch: exact string to replace, including indentation. Must be unique unless replace_all is true.",
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
				"name": map[string]any{
					"type":        "string",
					"description": "mode=remove: skill directory name, single segment (e.g. 'my-skill').",
				},
			},
		},
		Handler: func(ctx context.Context, e *toolTypes.Executor, args json.RawMessage) (string, error) {
			var params struct {
				Mode       string `json:"mode"`
				Path       string `json:"path"`
				Content    string `json:"content"`
				OldString  string `json:"old_string"`
				NewString  string `json:"new_string"`
				ReplaceAll bool   `json:"replace_all"`
				Name       string `json:"name"`
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
				case strings.TrimSpace(params.Name) != "":
					mode = "remove"
				default:
					return "", fmt.Errorf("mode is required; available: write, patch, remove")
				}
			}

			switch mode {
			case "write":
				return writeSkillFile(ctx, e, params.Path, params.Content)
			case "patch":
				return patchSkillFile(ctx, e, params.Path, params.OldString, params.NewString, params.ReplaceAll)
			case "remove":
				return removeSkillDir(ctx, e, params.Name)
			}
			return "", fmt.Errorf("unknown mode %q; available: write, patch, remove", mode)
		},
	})
}
