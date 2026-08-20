package file

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	go_pkg_filesystem "github.com/pardnchiu/go-pkg/filesystem"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
	historyStore "github.com/pardnchiu/agenvoy/internal/runtime/history"
	toolTypes "github.com/pardnchiu/agenvoy/internal/tools/types"
)

func RemoveToTrash(ctx context.Context, e *toolTypes.Executor, paths []string, tool string) (string, error) {
	var moved, failed, unrecorded []string
	for _, one := range paths {
		if err := ctx.Err(); err != nil {
			return "", fmt.Errorf("remove cancelled: %w", err)
		}
		if strings.HasPrefix(one, "-") {
			continue
		}

		src, err := go_pkg_filesystem.AbsPath(e.WorkDir, one, go_pkg_filesystem.AbsPathOption{HomeOnly: true})
		if err != nil {
			failed = append(failed, fmt.Sprintf("%s (%v)", one, err))
			continue
		}

		dst, err := filesystem.MoveToStoreTemp(src)
		if err != nil {
			failed = append(failed, fmt.Sprintf("%s (%v)", one, err))
			continue
		}

		moved = append(moved, src)
		if err := historyStore.RecordDelete(ctx, src, dst, historyStore.Meta{SessionID: e.SessionID, TaskID: e.PendingTask, Tool: tool}); err != nil {
			slog.Warn("historyStore.RecordDelete",
				slog.String("path", src),
				slog.String("error", err.Error()))
			unrecorded = append(unrecorded, fmt.Sprintf("%s now at %s (%v)", src, dst, err))
		}
	}

	switch {
	case len(moved) == 0 && len(failed) == 0:
		return "", fmt.Errorf("paths is required when mode=remove")
	case len(moved) == 0:
		return "", fmt.Errorf("removed nothing: %s", strings.Join(failed, "; "))
	}

	report := fmt.Sprintf("moved to %s: %s", filesystem.StoreTempDir, strings.Join(moved, ", "))
	if len(failed) > 0 {
		report += "\nnot removed: " + strings.Join(failed, "; ")
	}
	if len(unrecorded) > 0 {
		report += "\nnot recorded, so edit_file(mode=restore) will not find these: " + strings.Join(unrecorded, "; ")
	}
	return report, nil
}
