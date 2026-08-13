package historyStore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	go_pkg_filesystem "github.com/pardnchiu/go-pkg/filesystem"
	go_pkg_filesystem_reader "github.com/pardnchiu/go-pkg/filesystem/reader"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
)

const notRecorded = "\n  — but this restore was not recorded (%v), so it cannot itself be undone"

func restore(ctx context.Context, id int64, meta Meta) (string, bool, error) {
	row, content, err := Get(ctx, id)
	if err != nil {
		return "", false, err
	}
	if reason := row.RestoreBlock(); reason != "" {
		return "", false, fmt.Errorf("%s cannot be restored to %s: %s", filepath.Join(row.Dir, row.Name), time.Unix(0, row.ChangedAt).Format(TimeLayout), reason)
	}

	path := filepath.Join(row.Dir, row.Name)
	switch row.Action {
	case ActionCreate:
		return restoreCreate(ctx, row, path, meta)
	case ActionModify:
		return restoreModify(ctx, row, path, content, meta)
	case ActionDelete:
		return restoreDelete(ctx, row, path, meta)
	}
	return "", false, fmt.Errorf("%s has unknown action %q", filepath.Join(row.Dir, row.Name), row.Action)
}

func restoreCreate(ctx context.Context, row Row, path string, meta Meta) (string, bool, error) {
	if !go_pkg_filesystem_reader.Exists(path) {
		return fmt.Sprintf("%s already does not exist, nothing to undo", path), false, nil
	}

	trashPath, err := filesystem.MoveToStoreTemp(path)
	if err != nil {
		return "", false, fmt.Errorf("internal/filesystem: MoveToStoreTemp: %w", err)
	}
	done := fmt.Sprintf("%s did not exist yet at %s, so it is now removed to %s", path, time.Unix(0, row.ChangedAt).Format(TimeLayout), trashPath)
	if err := RecordDelete(ctx, path, trashPath, meta); err != nil {
		return done + fmt.Sprintf(notRecorded, err), true, nil
	}
	return done, true, nil
}

func restoreModify(ctx context.Context, row Row, path string, content []byte, meta Meta) (string, bool, error) {
	change, err := Capture(path)
	if err != nil {
		return "", false, fmt.Errorf("Capture [%s]: %w", path, err)
	}
	if change.hash != "" && change.hash == row.Hash {
		return fmt.Sprintf("%s already holds its version from %s", path, time.Unix(0, row.ChangedAt).Format(TimeLayout)), false, nil
	}

	if row.TrashPath != "" {
		if err := filesystem.CopyPath(row.TrashPath, path); err != nil {
			return "", false, fmt.Errorf("internal/filesystem: CopyPath [%s → %s]: %w", row.TrashPath, path, err)
		}
	} else if err := go_pkg_filesystem.WriteFile(path, string(content), 0644); err != nil {
		return "", false, fmt.Errorf("github.com/pardnchiu/go-pkg/filesystem WriteFile [%s]: %w", path, err)
	}

	done := fmt.Sprintf("%s rolled back to its version from %s (%d bytes)", path, time.Unix(0, row.ChangedAt).Format(TimeLayout), row.Size)
	if err := recordRestored(ctx, change, path, meta); err != nil {
		return done + fmt.Sprintf(notRecorded, err), true, nil
	}
	return done, true, nil
}

func restoreDelete(ctx context.Context, row Row, path string, meta Meta) (string, bool, error) {
	change, err := Capture(path)
	if err != nil {
		return "", false, fmt.Errorf("Capture [%s]: %w", path, err)
	}

	if err := filesystem.CopyPath(row.TrashPath, path); err != nil {
		return "", false, fmt.Errorf("internal/filesystem: CopyPath [%s → %s]: %w", row.TrashPath, path, err)
	}

	done := fmt.Sprintf("%s brought back from %s", path, row.TrashPath)
	if err := recordRestored(ctx, change, path, meta); err != nil {
		return done + fmt.Sprintf(notRecorded, err), true, nil
	}
	return done, true, nil
}

func recordRestored(ctx context.Context, before Change, path string, meta Meta) error {
	if before.action == ActionModify {
		return Record(ctx, before, meta)
	}

	after, err := Capture(path)
	if err != nil {
		return fmt.Errorf("Capture [%s]: %w", path, err)
	}
	if after.action == "" {
		after = Change{dir: filepath.Dir(path), name: filepath.Base(path)}
	}
	after.action = ActionCreate
	return Record(ctx, after, meta)
}

func RestoreTo(ctx context.Context, id int64, meta Meta) (string, error) {
	row, _, err := Get(ctx, id)
	if err != nil {
		return "", err
	}

	path := filepath.Join(row.Dir, row.Name)
	asked := fmt.Sprintf("%s (version %d)", time.Unix(0, row.ChangedAt).Format(TimeLayout), row.ID)

	var nextID int64
	err = conn.Read.QueryRowContext(ctx, `
	SELECT id
	FROM file_history
	WHERE dir = ? AND name = ? AND (changed_at > ? OR (changed_at = ? AND id > ?))
	ORDER BY changed_at ASC, id ASC
	LIMIT 1
	`, row.Dir, row.Name, row.ChangedAt, row.ChangedAt, row.ID).Scan(&nextID)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return fmt.Sprintf("%s is already at its state from %s, the newest one recorded", path, asked), nil
	case err != nil:
		return "", fmt.Errorf("sql.Row Scan [SELECT file_history]: %w", err)
	}

	// The bytes come from the row that replaced this state, but the caller asked
	// for this version and has to read that back — naming the other row's
	// timestamp here reads as having restored the wrong one.
	line, wrote, err := restore(ctx, nextID, meta)
	if err != nil {
		return "", err
	}

	verb := "is already at"
	if wrote {
		verb = "put back to"
	}
	done := fmt.Sprintf("%s %s its state from %s", path, verb, asked)
	if i := strings.Index(line, "\n  —"); i >= 0 {
		done += line[i:]
	}
	return done, nil
}

func Undo(ctx context.Context, f Filter, meta Meta) ([]string, error) {
	list, err := perPath(ctx, f, "ASC")
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, nil
	}

	report := make([]string, 0, len(list))
	for _, row := range list {
		line, _, err := restore(ctx, row.ID, meta)
		if err != nil {
			report = append(report, fmt.Sprintf("%s failed: %v", filepath.Join(row.Dir, row.Name), err))
			continue
		}
		report = append(report, line)
	}
	return report, nil
}
