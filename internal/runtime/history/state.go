package historyStore

import (
	"context"
	"fmt"
)

type StateRow struct {
	SessionID string
	State     string
	InAction  int
}

func WriteState(ctx context.Context, r StateRow) error {
	if conn == nil {
		return fmt.Errorf("internal/runtime/history: New has not run")
	}
	if r.SessionID == "" {
		return fmt.Errorf("session_id is required")
	}

	if _, err := conn.ExecContext(ctx, `
	INSERT INTO state (session_id, state, in_action)
	VALUES (?, ?, ?)
	ON CONFLICT(session_id) DO UPDATE SET
		state = excluded.state,
		in_action = excluded.in_action`,
		r.SessionID, r.State, r.InAction); err != nil {
		return fmt.Errorf("sql.DB ExecContext [INSERT state]: %w", err)
	}
	return nil
}

func ReadState(ctx context.Context, sessionID string) (StateRow, bool, error) {
	if conn == nil || sessionID == "" {
		return StateRow{}, false, nil
	}

	rows, err := conn.QueryContext(ctx,
		`SELECT session_id, state, in_action FROM state WHERE session_id = ?`, sessionID)
	if err != nil {
		return StateRow{}, false, fmt.Errorf("sql.DB QueryContext [SELECT state]: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		return StateRow{}, false, rows.Err()
	}

	var one StateRow
	if err := rows.Scan(&one.SessionID, &one.State, &one.InAction); err != nil {
		return StateRow{}, false, fmt.Errorf("sql.Rows Scan [SELECT state]: %w", err)
	}
	return one, true, nil
}

func ListStateRows(ctx context.Context) (map[string]StateRow, error) {
	if conn == nil {
		return nil, nil
	}

	rows, err := conn.QueryContext(ctx, `SELECT session_id, state, in_action FROM state`)
	if err != nil {
		return nil, fmt.Errorf("sql.DB QueryContext [SELECT state]: %w", err)
	}
	defer rows.Close()

	dic := map[string]StateRow{}
	for rows.Next() {
		var one StateRow
		if err := rows.Scan(&one.SessionID, &one.State, &one.InAction); err != nil {
			return nil, fmt.Errorf("sql.Rows Scan [SELECT state]: %w", err)
		}
		dic[one.SessionID] = one
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sql.Rows Err [SELECT state]: %w", err)
	}
	return dic, nil
}

func EnterState(ctx context.Context, sessionID, online string) error {
	if conn == nil || sessionID == "" {
		return nil
	}

	if _, err := conn.ExecContext(ctx, `
	INSERT INTO state (session_id, state, in_action)
	VALUES (?, ?, 1)
	ON CONFLICT(session_id) DO UPDATE SET
		in_action = in_action + 1,
		state     = ?`, sessionID, online, online); err != nil {
		return fmt.Errorf("sql.DB ExecContext [INSERT state enter]: %w", err)
	}
	return nil
}

func LeaveState(ctx context.Context, sessionID, online, idle string) error {
	if conn == nil || sessionID == "" {
		return nil
	}

	if _, err := conn.ExecContext(ctx, `
	UPDATE state SET
		in_action = MAX(in_action - 1, 0),
		state     = CASE WHEN in_action - 1 > 0 THEN ? ELSE ? END
	WHERE session_id = ?`, online, idle, sessionID); err != nil {
		return fmt.Errorf("sql.DB ExecContext [UPDATE state leave]: %w", err)
	}
	return nil
}

func ResetState(ctx context.Context, idle string) error {
	if conn == nil {
		return nil
	}

	if _, err := conn.ExecContext(ctx,
		`UPDATE state SET state = ?, in_action = 0 WHERE state <> ? OR in_action <> 0`,
		idle, idle); err != nil {
		return fmt.Errorf("sql.DB ExecContext [UPDATE state]: %w", err)
	}
	return nil
}

func DeleteState(ctx context.Context, sessionID string) error {
	if conn == nil || sessionID == "" {
		return nil
	}

	if _, err := conn.ExecContext(ctx, `DELETE FROM state WHERE session_id = ?`, sessionID); err != nil {
		return fmt.Errorf("sql.DB ExecContext [DELETE state]: %w", err)
	}
	return nil
}
