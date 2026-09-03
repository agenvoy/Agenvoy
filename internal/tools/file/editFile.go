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
		Name:        "edit_file",
		SystemUse:   false,
		AlwaysLoad:  true,
		AlwaysAllow: false,
		Concurrent:  false,
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
					"description": "The file this call acts on — '/abs/path/foo.go', '~/notes.md', 'relative/file.md'. One file per call. Required for write, patch and remove; on restore it narrows a task_id undo to that one file. Blank on write lands in ~/Downloads.",
					"default":     "",
				},
				"content": map[string]any{
					"type":        "string",
					"description": "mode=write: the complete file content, not a diff.",
				},
				"targets": map[string]any{
					"type":        "array",
					"description": "mode=patch: the edits to that file — read_files it first so every old_string matches the current bytes, minus the \"12\\t\" line-number prefix that read_files adds. Each item is {old_string, new_string}: new_string takes the place of old_string, empty deletes it, and to add lines you repeat old_string at the start of new_string. Items apply in the order listed, so sequence overlapping edits yourself.",
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"old_string": map[string]any{
								"type":        "string",
								"description": "Exact text to replace, indentation included — copy it from the file bytes, never with the line-number prefix. Must match once unless replace_all is set; extend it with surrounding lines until it does.",
							},
							"new_string": map[string]any{
								"type":        "string",
								"description": "What old_string becomes. Empty deletes it. To insert, start with old_string unchanged and append the new lines after it.",
							},
							"replace_all": map[string]any{
								"type":        "boolean",
								"description": "Replace every occurrence instead of the single unique one — for renames.",
								"default":     false,
							},
						},
						"required": []string{"old_string", "new_string"},
					},
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
				Version int64         `json:"version"`
				TaskID  string        `json:"task_id"`
			}
			if err := json.Unmarshal(args, &params); err != nil {
				return "", fmt.Errorf("json.Unmarshal: %w", err)
			}

			if err := rejectElided(params.Content, params.Targets); err != nil {
				return "", err
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
				path := strings.TrimSpace(params.Path)
				if path == "" {
					return "", fmt.Errorf("path is required when mode=remove")
				}
				return RemoveToTrash(ctx, e, []string{path}, "edit_file")
			case "restore":
				var narrow []string
				if path := strings.TrimSpace(params.Path); path != "" {
					narrow = []string{path}
				}
				return fileHistory.Restore(ctx, e, params.Version, params.TaskID, narrow)
			}
			return "", fmt.Errorf("unknown mode %q; available: write, patch, remove, restore", mode)
		},
	})
}
