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
		Description: `
Every change to a file on disk: create or wholesale replace one (write), edit regions of an existing one (patch), take one out of the way (remove), put a recorded version back (restore).
patch is the default for anything that already exists — re-sending a whole file to change part of it throws away text that was already correct, and leaves the same edit to be made again. Read the file first: anchors must match its current bytes.
remove never deletes: the file moves to the store temp dir and stays restorable, the same path run_command rm takes.
Replaces write_file, patch_file and restore_file. Skill files go to edit_skill, tool definitions to edit_tool.`,
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"mode": map[string]any{
					"type":        "string",
					"enum":        []string{"write", "patch", "remove", "restore"},
					"description": "write: full content. patch: edit regions of an existing file. remove: move files out of the way, restorable. restore: put a recorded version back. remove and restore are never inferred — name them explicitly.",
				},
				"path": map[string]any{
					"type":        "string",
					"description": "mode=write / mode=patch: the file (e.g. '/abs/path/foo.go', '~/notes.md', 'relative/file.md'). Exports with no path given belong in ~/Downloads or ~/.config/agenvoy/download/.",
					"default":     "",
				},
				"content": map[string]any{
					"type":        "string",
					"description": "mode=write: the complete file content.",
				},
				"targets": map[string]any{
					"type":        "array",
					"description": "mode=patch: one or more edits to that file. Each is either {old_string, new_string[, replace_all][, row]} or {insert_string, row}, never both. Targets carrying row apply highest row first, so line numbers stay valid against the original file even when other targets shift lines; the remaining targets then apply top to bottom against each other's output — order overlapping old_string targets accordingly.",
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"old_string": map[string]any{
								"type":        "string",
								"description": "Exact string to replace, including indentation. Must be unique unless replace_all is true or row is given. Omit when using insert_string.",
							},
							"new_string": map[string]any{
								"type":        "string",
								"description": "Replacement string. Empty string deletes old_string. Combine with row to delete only the occurrence on that line, leaving other occurrences of old_string untouched. Ignored when insert_string is set.",
							},
							"replace_all": map[string]any{
								"type":        "boolean",
								"description": "If true, replace all occurrences (e.g. when renaming a variable). Defaults to false.",
								"default":     false,
							},
							"insert_string": map[string]any{
								"type":        "string",
								"description": "Text to insert as new, independent line(s) at row — not a replacement of that line, not prepended to it. The existing line at row (and everything after) shifts down. Requires row. Cannot combine with old_string.",
							},
							"row": map[string]any{
								"type":        "integer",
								"description": "1-based line number. With old_string: disambiguates which occurrence to edit when old_string is not unique. With insert_string: the line insert_string is inserted before.",
							},
						},
					},
				},
				"paths": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type": "string",
					},
					"description": "mode=remove: the files to move out of the way. mode=restore: narrows a task_id undo to these files only.",
				},
				"version": map[string]any{
					"type":        "integer",
					"description": "mode=restore: the version to go back to, from a file_history row. Enough on its own — it names both the file and the state.",
				},
				"task_id": map[string]any{
					"type":        "string",
					"description": "mode=restore: undo a whole task instead, when no version is given: every file it touched, back to before it ran. 'current' for the task running now.",
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
