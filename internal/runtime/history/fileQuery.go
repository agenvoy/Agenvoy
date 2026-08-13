package historyStore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	go_pkg_filesystem_reader "github.com/pardnchiu/go-pkg/filesystem/reader"
)

var errNotFound = errors.New("no such change")

const TimeLayout = "2006-01-02 15:04:05"

type Row struct {
	ID        int64
	Dir       string
	Name      string
	Action    string
	Hash      string
	Size      int64
	Truncated bool
	TrashPath string
	SessionID string
	TaskID    string
	Tool      string
	ChangedAt int64
}

func (r Row) RestoreBlock() string {
	if r.Action == ActionCreate {
		return ""
	}

	switch {
	case r.TrashPath != "" && !go_pkg_filesystem_reader.Exists(r.TrashPath):
		return fmt.Sprintf("the copy kept at %s is gone", r.TrashPath)
	case r.TrashPath != "":
		return ""
	case r.Truncated:
		return "the file was over 1 MiB and no copy of it could be kept"
	case r.Action == ActionDelete:
		return "no copy location was recorded"
	}
	return ""
}

type Filter struct {
	Path   string
	Dir    string
	TaskID string
	From   int64
	To     int64
	Limit  int
}

func (f Filter) where() ([]string, []any) {
	var where []string
	var args []any

	if f.Path != "" {
		where = append(where, "dir = ? AND name = ?")
		args = append(args, filepath.Dir(f.Path), filepath.Base(f.Path))
	}
	if f.TaskID != "" {
		where = append(where, "task_id = ?")
		args = append(args, f.TaskID)
	}
	if f.From > 0 {
		where = append(where, "changed_at >= ?")
		args = append(args, f.From)
	}
	if f.To > 0 {
		where = append(where, "changed_at < ?")
		args = append(args, f.To)
	}
	return where, args
}

func (f Filter) limitClause() (string, []any) {
	if f.Limit <= 0 {
		return "", nil
	}
	return "\n\tLIMIT ?", []any{f.Limit}
}

func List(ctx context.Context, f Filter) ([]Row, error) {
	if conn == nil {
		return nil, nil
	}

	where, args := f.where()
	query := `
	SELECT id, dir, name, action, hash, size, truncated, trash_path, session_id, task_id, tool, changed_at
	FROM file_history`
	if len(where) > 0 {
		query += "\n\tWHERE " + strings.Join(where, " AND ")
	}
	query += "\n\tORDER BY changed_at DESC, id DESC"
	clause, limitArgs := f.limitClause()
	query += clause
	args = append(args, limitArgs...)

	rows, err := conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("sql.DB QueryContext [SELECT file_history]: %w", err)
	}
	defer rows.Close()

	var list []Row
	for rows.Next() {
		r, err := scan(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sql.Rows Err [SELECT file_history]: %w", err)
	}
	return list, nil
}

func Newest(ctx context.Context, f Filter) ([]Row, error) {
	return perPath(ctx, f, "DESC")
}

func perPath(ctx context.Context, f Filter, order string) ([]Row, error) {
	if conn == nil {
		return nil, nil
	}

	where, args := f.where()
	inner := `
		SELECT id, dir, name, action, hash, size, truncated, trash_path, session_id, task_id, tool, changed_at,
			ROW_NUMBER() OVER (PARTITION BY dir, name ORDER BY changed_at ` + order + `, id ` + order + `) AS pick
		FROM file_history`
	if len(where) > 0 {
		inner += "\n\t\tWHERE " + strings.Join(where, " AND ")
	}

	query := fmt.Sprintf(`
	SELECT id, dir, name, action, hash, size, truncated, trash_path, session_id, task_id, tool, changed_at
	FROM (%s)
	WHERE pick = 1
	ORDER BY changed_at %s, id %s`, inner, order, order)
	clause, limitArgs := f.limitClause()
	query += clause
	args = append(args, limitArgs...)

	rows, err := conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("sql.DB QueryContext [SELECT file_history]: %w", err)
	}
	defer rows.Close()

	var list []Row
	for rows.Next() {
		r, err := scan(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sql.Rows Err [SELECT file_history]: %w", err)
	}
	return list, nil
}

func Get(ctx context.Context, id int64) (Row, []byte, error) {
	if conn == nil {
		return Row{}, nil, errNotFound
	}

	var (
		r         Row
		truncated int
		trashPath sql.NullString
		content   []byte
	)
	err := conn.Read.QueryRowContext(ctx, `
	SELECT id, dir, name, action, hash, size, truncated, trash_path, session_id, task_id, tool, changed_at, content
	FROM file_history
	WHERE id = ?
	`, id).Scan(
		&r.ID, &r.Dir, &r.Name, &r.Action, &r.Hash, &r.Size, &truncated, &trashPath,
		&r.SessionID, &r.TaskID, &r.Tool, &r.ChangedAt, &content,
	)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return Row{}, nil, errNotFound
	case err != nil:
		return Row{}, nil, fmt.Errorf("sql.Row Scan [SELECT file_history]: %w", err)
	}

	r.Truncated = truncated == 1
	r.TrashPath = trashPath.String
	return r, content, nil
}

func scan(rows *sql.Rows) (Row, error) {
	var (
		r         Row
		truncated int
		trashPath sql.NullString
	)
	if err := rows.Scan(
		&r.ID, &r.Dir, &r.Name, &r.Action, &r.Hash, &r.Size, &truncated, &trashPath,
		&r.SessionID, &r.TaskID, &r.Tool, &r.ChangedAt,
	); err != nil {
		return Row{}, fmt.Errorf("sql.Rows Scan [SELECT file_history]: %w", err)
	}

	r.Truncated = truncated == 1
	r.TrashPath = trashPath.String
	return r, nil
}
