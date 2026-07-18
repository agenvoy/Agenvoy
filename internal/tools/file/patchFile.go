package file

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	go_pkg_filesystem "github.com/pardnchiu/go-pkg/filesystem"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
	"github.com/pardnchiu/agenvoy/internal/tools/file/denied"
	toolRegister "github.com/pardnchiu/agenvoy/internal/tools/register"
	toolTypes "github.com/pardnchiu/agenvoy/internal/tools/types"
)

func registPatchFile() {
	toolRegister.Regist(toolRegister.Def{
		Name: "patch_file",
		Description: `
Replace one or more exact string matches inside a file, or insert new lines at a given line number.
Use for targeted edits; write_file for full rewrite; patch_skill for skill files.
Must read_files before patching to get the exact anchor string or line number.
Each target is either a replace (old_string/new_string) or a pure insert (insert_string/row) — never both.
Targets with row are applied from the highest row to the lowest first (so line numbers stay valid against
the original file even when other targets shift lines), then remaining targets apply top to bottom.`,
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "File to edit (e.g. '/abs/path/foo.go', '~/notes.md', 'relative/file.md').",
				},
				"targets": map[string]any{
					"type":        "array",
					"description": "One or more edits. Each is either {old_string, new_string[, replace_all][, row]} or {insert_string, row}.",
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
								"description": "Text to insert as new, independent line(s) at row — not a replacement of that line, not prepended to it. The existing line at row (and everything after) shifts down. Requires row. Cannot combine with old_string/new_string.",
							},
							"row": map[string]any{
								"type":        "integer",
								"description": "1-based line number. With old_string: disambiguates which occurrence to edit when old_string is not unique. With insert_string: the line insert_string is inserted before.",
							},
						},
					},
				},
			},
			"required": []string{
				"path",
				"targets",
			},
		},
		Handler: func(ctx context.Context, e *toolTypes.Executor, args json.RawMessage) (string, error) {
			var params struct {
				Path    string `json:"path"`
				Targets []struct {
					OldString    string `json:"old_string"`
					NewString    string `json:"new_string"`
					ReplaceAll   bool   `json:"replace_all"`
					InsertString string `json:"insert_string"`
					Row          int    `json:"row"`
				} `json:"targets"`
			}
			if err := json.Unmarshal(args, &params); err != nil {
				return "", fmt.Errorf("json.Unmarshal: %w", err)
			}
			if len(params.Targets) == 0 {
				return "", fmt.Errorf("targets is required")
			}

			baseDir := e.WorkDir
			if baseDir == "" {
				baseDir = filesystem.DownloadDir
			}

			absPath, err := go_pkg_filesystem.AbsPath(baseDir, params.Path, go_pkg_filesystem.AbsPathOption{HomeOnly: true})
			if err != nil {
				return "", fmt.Errorf("go_pkg_filesystem.AbsPath: %w", err)
			}
			if absPath == "" {
				return "", fmt.Errorf("path or name is required")
			}

			if parent, ok := denied.Hit(e.SessionID, absPath); ok {
				return "", fmt.Errorf("permission denied: %s is under previously rejected %s; not retried", absPath, parent)
			}

			info, err := os.Stat(absPath)
			if err != nil {
				if denied.IsPermission(err) {
					denied.Register(e.SessionID, absPath)
					return "", fmt.Errorf("permission denied: %s (recorded; further edits under this path will be skipped)", absPath)
				}
				return "", fmt.Errorf("os.Stat: %w", err)
			}
			if info.Size() > maxReadSize {
				return "", fmt.Errorf("file too large (%d bytes, max 1 MB)", info.Size())
			}

			fileContent, err := go_pkg_filesystem.ReadText(absPath)
			if err != nil {
				if denied.IsPermission(err) {
					denied.Register(e.SessionID, absPath)
					return "", fmt.Errorf("permission denied: %s (recorded; further edits under this path will be skipped)", absPath)
				}
				return "", fmt.Errorf("go_pkg_filesystem.ReadText: %w", err)
			}

			order := make([]int, len(params.Targets))
			for i := range order {
				order[i] = i
			}
			sort.SliceStable(order, func(a, b int) bool {
				ra, rb := params.Targets[order[a]].Row, params.Targets[order[b]].Row
				if ra > 0 && rb > 0 {
					return ra > rb
				}
				return ra > 0 && rb == 0
			})

			for _, i := range order {
				t := params.Targets[i]

				switch {
				case t.InsertString != "":
					if t.OldString != "" || t.NewString != "" {
						return "", fmt.Errorf("targets[%d]: insert_string cannot be combined with old_string/new_string", i)
					}
					if t.Row <= 0 {
						return "", fmt.Errorf("targets[%d]: row is required when insert_string is set", i)
					}
					updated, err := insertAtRow(fileContent, t.InsertString, t.Row)
					if err != nil {
						return "", fmt.Errorf("targets[%d]: %w", i, err)
					}
					fileContent = updated

				case t.OldString != "":
					// * not to trim string, avoid user use " " to indicate indent
					old := t.OldString
					new := t.NewString
					if old == new {
						return "", fmt.Errorf("targets[%d]: no edit needed", i)
					}
					if !strings.Contains(fileContent, old) {
						return "", fmt.Errorf("targets[%d]: %s is not found in %s", i, old, absPath)
					}

					search := old
					if new == "" && !strings.HasSuffix(old, "\n") && strings.Contains(fileContent, old+"\n") {
						search = old + "\n"
					}

					switch {
					case t.ReplaceAll:
						fileContent = strings.ReplaceAll(fileContent, search, new)
					case t.Row > 0:
						updated, err := replaceAtRow(fileContent, search, new, t.Row)
						if err != nil {
							return "", fmt.Errorf("targets[%d]: %w", i, err)
						}
						fileContent = updated
					default:
						if n := strings.Count(fileContent, search); n > 1 {
							return "", fmt.Errorf("targets[%d]: %s occurs %d times in %s; set replace_all or specify row to disambiguate", i, old, n, absPath)
						}
						fileContent = strings.Replace(fileContent, search, new, 1)
					}

				default:
					return "", fmt.Errorf("targets[%d]: either old_string or insert_string is required", i)
				}
			}

			if err := go_pkg_filesystem.WriteFile(absPath, fileContent, 0644); err != nil {
				if denied.IsPermission(err) {
					denied.Register(e.SessionID, absPath)
					return "", fmt.Errorf("permission denied: %s (recorded; further edits under this path will be skipped)", absPath)
				}
				return "", fmt.Errorf("go_pkg_filesystem.WriteFile: %w", err)
			}

			filesystem.GitAutoCommitByPath(ctx, filesystem.GitSkills, absPath, false)
			return fmt.Sprintf("successfully updated %s", absPath), nil
		},
	})
}

func replaceAtRow(content, search, new string, row int) (string, error) {
	idx := 0
	for {
		i := strings.Index(content[idx:], search)
		if i < 0 {
			break
		}
		pos := idx + i
		line := strings.Count(content[:pos], "\n") + 1
		if line == row {
			return content[:pos] + new + content[pos+len(search):], nil
		}
		idx = pos + 1
	}
	return "", fmt.Errorf("no match for %q at row %d", search, row)
}

func insertAtRow(content, insert string, row int) (string, error) {
	lines := strings.Split(content, "\n")
	lineCount := len(lines)
	if lineCount > 0 && lines[lineCount-1] == "" {
		lineCount--
	}
	if row < 1 || row > lineCount+1 {
		return "", fmt.Errorf("row %d out of range (file has %d lines)", row, lineCount)
	}

	idx := row - 1
	out := make([]string, 0, len(lines)+1)
	out = append(out, lines[:idx]...)
	out = append(out, strings.Split(insert, "\n")...)
	out = append(out, lines[idx:]...)
	return strings.Join(out, "\n"), nil
}
