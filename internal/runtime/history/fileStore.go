package historyStore

import (
	"context"
	"fmt"
	"time"
)

const keepPerPath = 24

func insert(ctx context.Context, c Change, meta Meta) error {
	var content any
	if len(c.content) > 0 {
		content = c.content
	}
	var trashPath any
	if c.trashPath != "" {
		trashPath = c.trashPath
	}
	truncated := 0
	if c.truncated {
		truncated = 1
	}

	if _, err := conn.ExecContext(ctx, `
	INSERT INTO file_history
	(dir, name, action, content, hash, size, truncated, trash_path, session_id, task_id, tool, changed_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(dir, name, changed_at, task_id, session_id) DO UPDATE SET
		action     = excluded.action,
		content    = excluded.content,
		hash       = excluded.hash,
		size       = excluded.size,
		truncated  = excluded.truncated,
		trash_path = excluded.trash_path,
		tool       = excluded.tool
	`,
		c.dir, c.name, c.action, content, c.hash, c.size, truncated, trashPath,
		meta.SessionID, meta.TaskID, meta.Tool, time.Now().Truncate(time.Second).UnixNano(),
	); err != nil {
		return fmt.Errorf("sql.DB ExecContext [INSERT file_history]: %w", err)
	}

	return prune(ctx, c.dir, c.name)
}

func prune(ctx context.Context, dir, name string) error {
	if _, err := conn.ExecContext(ctx, `
	DELETE FROM file_history
	WHERE id IN (
		SELECT id FROM file_history
		WHERE dir = ? AND name = ?
		ORDER BY changed_at DESC, id DESC
		LIMIT -1 OFFSET ?
	)
	`, dir, name, keepPerPath); err != nil {
		return fmt.Errorf("sql.DB ExecContext [DELETE file_history]: %w", err)
	}
	return nil
}

func latestModifyHash(ctx context.Context, dir, name string) string {
	var action, hash string
	conn.Read.QueryRowContext(ctx, `
	SELECT action, hash
	FROM file_history
	WHERE dir = ? AND name = ?
	ORDER BY changed_at DESC, id DESC
	LIMIT 1
	`, dir, name).Scan(&action, &hash)

	if action != ActionModify {
		return ""
	}
	return hash
}
