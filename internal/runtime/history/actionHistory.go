package historyStore

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

const actionKeepDays = 3

type ActionRecord struct {
	TaskHash    string          `json:"-"`
	EndAt       time.Time       `json:"-"`
	Model       string          `json:"model,omitempty"`
	Reasoning   string          `json:"reasoning,omitempty"`
	Objective   string          `json:"objective,omitempty"`
	ToolResults json.RawMessage `json:"tool_results,omitempty"`
	Todos       json.RawMessage `json:"todos,omitempty"`
	Reply       string          `json:"reply,omitempty"`
}

func rawText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	return string(raw)
}

func textRaw(text string) json.RawMessage {
	if text == "" {
		return nil
	}
	return json.RawMessage(text)
}

func WriteAction(ctx context.Context, sessionID string, r ActionRecord) error {
	if conn == nil {
		return fmt.Errorf("internal/runtime/history: New has not run")
	}
	if sessionID == "" || r.TaskHash == "" {
		return fmt.Errorf("session_id and task_hash are required")
	}

	if _, err := conn.ExecContext(ctx, `
	INSERT INTO action_history
	(session_id, task_hash, end_at, model, reasoning, objective, tool_results, todos, reply)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(session_id, task_hash) DO UPDATE SET
		end_at        = excluded.end_at,
		model         = excluded.model,
		reasoning     = excluded.reasoning,
		objective     = excluded.objective,
		tool_results  = excluded.tool_results,
		todos         = excluded.todos,
		reply         = excluded.reply`,
		sessionID, r.TaskHash, r.EndAt.UnixNano(), r.Model, r.Reasoning, r.Objective,
		rawText(r.ToolResults), rawText(r.Todos), r.Reply); err != nil {
		return fmt.Errorf("sql.DB ExecContext [INSERT action_history]: %w", err)
	}
	return nil
}

const actionColumns = `task_hash, end_at, model, reasoning, objective, tool_results, todos, reply`

func scanAction(rows interface {
	Scan(dest ...any) error
}) (ActionRecord, error) {
	var (
		one              ActionRecord
		endAt            int64
		toolResults, tds string
	)
	if err := rows.Scan(&one.TaskHash, &endAt, &one.Model, &one.Reasoning, &one.Objective,
		&toolResults, &tds, &one.Reply); err != nil {
		return ActionRecord{}, fmt.Errorf("sql.Rows Scan [SELECT action_history]: %w", err)
	}

	one.EndAt = time.Unix(0, endAt)
	one.ToolResults = textRaw(toolResults)
	one.Todos = textRaw(tds)
	return one, nil
}

func ListAction(ctx context.Context, sessionID string) ([]ActionRecord, error) {
	if conn == nil || sessionID == "" {
		return nil, nil
	}

	rows, err := conn.QueryContext(ctx, `
	SELECT `+actionColumns+`
	FROM action_history
	WHERE session_id = ?
	ORDER BY end_at DESC, id DESC`, sessionID)
	if err != nil {
		return nil, fmt.Errorf("sql.DB QueryContext [SELECT action_history]: %w", err)
	}
	defer rows.Close()

	var list []ActionRecord
	for rows.Next() {
		one, err := scanAction(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, one)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sql.Rows Err [SELECT action_history]: %w", err)
	}
	return list, nil
}

func ReadAction(ctx context.Context, sessionID, taskHash string) (ActionRecord, bool, error) {
	if conn == nil || sessionID == "" || taskHash == "" {
		return ActionRecord{}, false, nil
	}

	rows, err := conn.QueryContext(ctx, `
	SELECT `+actionColumns+`
	FROM action_history
	WHERE session_id = ? AND task_hash = ?`, sessionID, taskHash)
	if err != nil {
		return ActionRecord{}, false, fmt.Errorf("sql.DB QueryContext [SELECT action_history]: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		return ActionRecord{}, false, rows.Err()
	}
	one, err := scanAction(rows)
	if err != nil {
		return ActionRecord{}, false, err
	}
	return one, true, nil
}

func ClearAction(ctx context.Context, sessionID string) error {
	if conn == nil || sessionID == "" {
		return nil
	}

	if _, err := conn.ExecContext(ctx,
		`DELETE FROM action_history WHERE session_id = ?`, sessionID); err != nil {
		return fmt.Errorf("sql.DB ExecContext [DELETE action_history]: %w", err)
	}
	return nil
}

func PruneAction(ctx context.Context) error {
	if conn == nil {
		return nil
	}

	cutoff := time.Now().AddDate(0, 0, -actionKeepDays).UnixNano()
	if _, err := conn.ExecContext(ctx,
		`DELETE FROM action_history WHERE end_at < ?`, cutoff); err != nil {
		return fmt.Errorf("sql.DB ExecContext [DELETE action_history]: %w", err)
	}
	return nil
}
