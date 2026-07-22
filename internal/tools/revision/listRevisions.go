package revision

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
	toolRegister "github.com/pardnchiu/agenvoy/internal/tools/register"
	toolTypes "github.com/pardnchiu/agenvoy/internal/tools/types"
)

func matchTarget(tag string) (filesystem.GitTarget, error) {
	switch tag {
	case "skills":
		return filesystem.GitSkills, nil
	case "tools":
		return filesystem.GitTools, nil
	default:
		return 0, fmt.Errorf("tag must be 'skills' or 'tools', got %q", tag)
	}
}

func registListRevisions() {
	toolRegister.Regist(toolRegister.Def{
		Name:        "list_revisions",
		AlwaysAllow: true,
		Concurrent:  true,
		Description: `
List revision history of Agenvoy's own skill or tool storage.
Scope is limited to ~/.config/agenvoy — never the user's project repository.
Use to find a revision before restore_revision, or to verify auto-commits landed.`,
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"tag": map[string]any{
					"type":        "string",
					"enum":        []string{"skills", "tools"},
					"description": "Target storage. 'skills' = ~/.config/agenvoy/skills, 'tools' = ~/.config/agenvoy/tools.",
				},
				"limit": map[string]any{
					"type":        "integer",
					"description": "Number of revisions to return. Clamped to [1, 50] (e.g. 10).",
					"default":     20,
				},
			},
			"required": []string{"tag"},
		},
		Handler: func(ctx context.Context, _ *toolTypes.Executor, args json.RawMessage) (string, error) {
			var params struct {
				Tag   string `json:"tag"`
				Limit int    `json:"limit"`
			}
			if err := json.Unmarshal(args, &params); err != nil {
				return "", fmt.Errorf("encoding/json: Unmarshal: %w", err)
			}

			target, err := matchTarget(params.Tag)
			if err != nil {
				return "", err
			}

			if params.Limit > 50 {
				params.Limit = 50
			}

			result, err := filesystem.GitLog(ctx, target, params.Limit)
			if err != nil {
				return "", fmt.Errorf("internal/filesystem: GitLog [%s]: %w", params.Tag, err)
			}
			if result == "" {
				return "no commits yet", nil
			}
			return result, nil
		},
	})
}
