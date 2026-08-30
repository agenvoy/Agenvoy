package historyStore

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

const (
	DefaultModel     = "auto"
	DefaultReasoning = "medium"
)

type SessionRow struct {
	SessionID string
	SelfID    string
	Name      string
	Model     string
	Reasoning string
	Rule      string
	ChatID    string
	GuildID   string
	ChannelID string
	UserID    string
}

const sessionColumns = `session_id, self_id, name, model, reasoning, rule, chat_id, guild_id, channel_id, user_id`

const SelfIDLimit = 32

var selfIDRegex = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

func ValidSelfID(selfID string) error {
	if selfID == "" {
		return nil
	}
	if len([]rune(selfID)) > SelfIDLimit {
		return fmt.Errorf("self_id is limited to %d characters", SelfIDLimit)
	}
	if !selfIDRegex.MatchString(selfID) {
		return fmt.Errorf("self_id allows A-Z a-z 0-9 _ - only")
	}
	return nil
}

var ErrDuplicateSelfID = errors.New("self_id is already used by another session")

func isSelfIDConflict(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed: session.self_id")
}

func scanSession(rows interface {
	Scan(dest ...any) error
}) (SessionRow, error) {
	var one SessionRow
	if err := rows.Scan(&one.SessionID, &one.SelfID, &one.Name, &one.Model, &one.Reasoning, &one.Rule,
		&one.ChatID, &one.GuildID, &one.ChannelID, &one.UserID); err != nil {
		return SessionRow{}, fmt.Errorf("sql.Rows Scan [SELECT session]: %w", err)
	}
	return one, nil
}

func WriteSession(ctx context.Context, r SessionRow) error {
	if conn == nil {
		return fmt.Errorf("internal/runtime/history: New has not run")
	}
	if r.SessionID == "" {
		return fmt.Errorf("session_id is required")
	}
	if r.Model == "" {
		r.Model = DefaultModel
	}
	if r.Reasoning == "" {
		r.Reasoning = DefaultReasoning
	}
	if err := ValidSelfID(r.SelfID); err != nil {
		return err
	}

	if _, err := conn.ExecContext(ctx, `
	INSERT INTO session (`+sessionColumns+`)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(session_id) DO UPDATE SET
		self_id    = excluded.self_id,
		name       = excluded.name,
		model      = excluded.model,
		reasoning  = excluded.reasoning,
		rule       = excluded.rule,
		chat_id    = excluded.chat_id,
		guild_id   = excluded.guild_id,
		channel_id = excluded.channel_id,
		user_id    = excluded.user_id`,
		r.SessionID, r.SelfID, r.Name, r.Model, r.Reasoning, r.Rule,
		r.ChatID, r.GuildID, r.ChannelID, r.UserID); err != nil {
		if isSelfIDConflict(err) {
			return fmt.Errorf("%w: %s", ErrDuplicateSelfID, r.SelfID)
		}
		return fmt.Errorf("sql.DB ExecContext [INSERT session]: %w", err)
	}
	return nil
}

func ReadSession(ctx context.Context, sessionID string) (SessionRow, bool, error) {
	if conn == nil || sessionID == "" {
		return SessionRow{}, false, nil
	}

	rows, err := conn.QueryContext(ctx,
		`SELECT `+sessionColumns+` FROM session WHERE session_id = ?`, sessionID)
	if err != nil {
		return SessionRow{}, false, fmt.Errorf("sql.DB QueryContext [SELECT session]: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		return SessionRow{}, false, rows.Err()
	}
	one, err := scanSession(rows)
	if err != nil {
		return SessionRow{}, false, err
	}
	return one, true, nil
}

func ListSessionRows(ctx context.Context) (map[string]SessionRow, error) {
	if conn == nil {
		return nil, nil
	}

	rows, err := conn.QueryContext(ctx, `SELECT `+sessionColumns+` FROM session`)
	if err != nil {
		return nil, fmt.Errorf("sql.DB QueryContext [SELECT session]: %w", err)
	}
	defer rows.Close()

	dic := map[string]SessionRow{}
	for rows.Next() {
		one, err := scanSession(rows)
		if err != nil {
			return nil, err
		}
		dic[one.SessionID] = one
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sql.Rows Err [SELECT session]: %w", err)
	}
	return dic, nil
}

func FindSessionByChat(ctx context.Context, chatID string) string {
	return findSessionBy(ctx, "chat_id", chatID)
}

func FindSessionByChannel(ctx context.Context, channelID string) string {
	return findSessionBy(ctx, "channel_id", channelID)
}

func FindSessionBySelfID(ctx context.Context, selfID string) string {
	return findSessionBy(ctx, "self_id", selfID)
}

func findSessionBy(ctx context.Context, column, value string) string {
	if conn == nil || value == "" {
		return ""
	}

	rows, err := conn.QueryContext(ctx,
		`SELECT session_id FROM session WHERE `+column+` = ? LIMIT 1`, value)
	if err != nil {
		return ""
	}
	defer rows.Close()

	if !rows.Next() {
		return ""
	}
	var sessionID string
	if err := rows.Scan(&sessionID); err != nil {
		return ""
	}
	return sessionID
}

func DeleteSession(ctx context.Context, sessionID string) error {
	if conn == nil || sessionID == "" {
		return nil
	}

	if _, err := conn.ExecContext(ctx, `DELETE FROM session WHERE session_id = ?`, sessionID); err != nil {
		return fmt.Errorf("sql.DB ExecContext [DELETE session]: %w", err)
	}
	return nil
}
