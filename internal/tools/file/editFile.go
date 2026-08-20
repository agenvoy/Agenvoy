package file

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	fileHistory "github.com/pardnchiu/agenvoy/internal/tools/history/file"
	toolRegister "github.com/pardnchiu/agenvoy/internal/tools/register"
	toolTypes "github.com/pardnchiu/agenvoy/internal/tools/types"
)

func registEditFile() {
	toolRegister.Regist(toolRegister.Def{
		Name: "edit_file",
		Description: `Every change to a file on disk: create or replace (write), edit regions (patch), move aside (remove), put a recorded version back (restore).
Use for 寫檔 / 改這一段 / 刪掉這個檔 / 還原 / 改回上一版, and for write_file / patch_file / remove_file / restore_file / delete.
Skill files → edit_skill; tool definitions → edit_tool; past versions → file_history.`,
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"mode": map[string]any{
					"type":        "string",
					"enum":        []string{"write", "patch", "remove", "restore"},
					"description": "write: a first version or a deliberate full replacement. patch: everything else — a whole-file rewrite throws away text that was already correct. remove: moves the file aside, still restorable. restore: put a recorded version back. Omitted: content → write, targets → patch; remove and restore are never inferred.",
				},
				"path": map[string]any{
					"type":        "string",
					"description": "mode=write / mode=patch: the file — '/abs/path/foo.go', '~/notes.md', 'relative/file.md'. Blank on write lands in ~/Downloads.",
					"default":     "",
				},
				"content": map[string]any{
					"type":        "string",
					"description": "mode=write: the complete file content, not a diff.",
				},
				"targets": map[string]any{
					"type":        "array",
					"description": "mode=patch: the edits to that file — read_files it first so every anchor matches the current bytes. Each item is {old_string, new_string[, replace_all][, row]} or {insert_string, row}, never both. Items carrying row apply highest-row first so line numbers stay valid against the original; the rest then apply in listed order, so sequence overlapping old_string items yourself.",
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"old_string": map[string]any{
								"type":        "string",
								"description": "Exact text to replace, indentation included. Must be unique unless replace_all or row disambiguates. Omit with insert_string.",
							},
							"new_string": map[string]any{
								"type":        "string",
								"description": "Replacement text; empty deletes old_string. With row, deletes only that line's occurrence. Ignored when insert_string is set.",
							},
							"replace_all": map[string]any{
								"type":        "boolean",
								"description": "Replace every occurrence instead of the single unique one — for renames.",
								"default":     false,
							},
							"insert_string": map[string]any{
								"type":        "string",
								"description": "New line(s) inserted before row — the existing line shifts down, nothing is replaced. Requires row; cannot combine with old_string.",
							},
							"row": map[string]any{
								"type":        "integer",
								"description": "1-based line number. With old_string: which occurrence to edit. With insert_string: the line to insert before.",
							},
						},
					},
				},
				"paths": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type": "string",
					},
					"description": "mode=remove: the files to move aside. mode=restore: limits a task_id undo to these files.",
				},
				"version": map[string]any{
					"type":        "integer",
					"description": "mode=restore: version id from a file_history row — enough on its own, it names both the file and the state.",
				},
				"task_id": map[string]any{
					"type":        "string",
					"description": "mode=restore: undo a whole task instead — every file it touched, back to before it ran. 'current' = the task running now. Either this or version is required.",
				},
			},
		},
		Handler: func(ctx context.Context, e *toolTypes.Executor, args json.RawMessage) (string, error) {
			var params struct {
				Mode    string        `json:"mode"`
				Path    string        `json:"path"`
				Content string        `json:"content"`
				Targets []patchTarget `json:"targets"`
				Paths   []string      `json:"paths"`
				Version int64         `json:"version"`
				TaskID  string        `json:"task_id"`
			}
			if err := json.Unmarshal(args, &params); err != nil {
				return "", fmt.Errorf("json.Unmarshal: %w", err)
			}

			mode := strings.ToLower(strings.TrimSpace(params.Mode))
			if mode == "" {
				switch {
				case len(params.Targets) > 0:
					mode = "patch"
				case params.Content != "":
					mode = "write"
				default:
					return "", fmt.Errorf("mode is required; available: write, patch, remove, restore")
				}
			}

			switch mode {
			case "write":
				return writeFileContent(ctx, e, params.Path, params.Content)
			case "patch":
				return patchFileTargets(ctx, e, params.Path, params.Targets)
			case "remove":
				paths := params.Paths
				if len(paths) == 0 && strings.TrimSpace(params.Path) != "" {
					paths = []string{params.Path}
				}
				return RemoveToTrash(ctx, e, paths, "edit_file")
			case "restore":
				return fileHistory.Restore(ctx, e, params.Version, params.TaskID, params.Paths)
			}
			return "", fmt.Errorf("unknown mode %q; available: write, patch, remove, restore", mode)
		},
	})
}
