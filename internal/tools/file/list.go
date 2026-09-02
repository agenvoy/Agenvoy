package file

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/pardnchiu/agenvoy/internal/tools/file/boundary"
	toolTypes "github.com/pardnchiu/agenvoy/internal/tools/types"
	go_pkg_filesystem_reader "github.com/pardnchiu/go-pkg/filesystem/reader"
)

func listBatch(ctx context.Context, e *toolTypes.Executor, queries []findQuery) (string, error) {
	out := make(map[string]any, len(queries))
	budget := newSizeBudget()
	for _, q := range queries {
		files, err := listOne(ctx, e, q.Dir, q.Recursive)
		if err != nil {
			out[q.Dir] = "error: " + err.Error()
			continue
		}
		out[q.Dir] = budget.take(files)
		if err := ctx.Err(); err != nil {
			return "", err
		}
	}

	raw, err := json.Marshal(out)
	if err != nil {
		return "", fmt.Errorf("json.Marshal: %w", err)
	}
	return string(raw) + budget.notice(), nil
}

func listOne(ctx context.Context, e *toolTypes.Executor, dir string, recursive bool) ([]go_pkg_filesystem_reader.File, error) {
	dir = strings.TrimSpace(dir)
	absPath, err := boundary.Resolve(e.SessionID, e.WorkDir, dir)
	if err != nil {
		return nil, fmt.Errorf("boundary.Resolve: %w", err)
	}

	if file, err := os.Open(absPath); err != nil {
		return nil, fmt.Errorf("os.Open: %w", err)
	} else {
		file.Close()
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if recursive {
		files, err := go_pkg_filesystem_reader.WalkFiles(absPath, go_pkg_filesystem_reader.ListOption{
			SkipExcluded:      true,
			SkipDenied:        true,
			IgnoreWalkError:   true,
			IncludeNonRegular: true,
		})
		if err != nil {
			return nil, fmt.Errorf("github.com/pardnchiu/go-pkg/filesystem/reader: WalkFiles: %w", err)
		}
		return files, nil
	}

	files, err := go_pkg_filesystem_reader.ListAll(absPath, go_pkg_filesystem_reader.ListOption{SkipExcluded: true})
	if err != nil {
		return nil, fmt.Errorf("github.com/pardnchiu/go-pkg/filesystem/reader: ListAll: %w", err)
	}
	return files, nil
}
