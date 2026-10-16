package historyStore

import (
	"context"
	"errors"
	"regexp"
	"strings"

	utilsmsg "github.com/pardnchiu/agenvoy/internal/utils/msg"
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
	Role      string
	ChatID    string
	GuildID   string
	ChannelID string
	UserID    string
}

// ! the rule column is kept for older builds and will deprecate in the future
const (
	sessionInsertColumns = `session_id, self_id, name, model, reasoning, role, rule, chat_id, guild_id, channel_id, user_id`
	sessionSelectColumns = `session_id, self_id, name, model, reasoning, COALESCE(NULLIF(role, ''), rule), chat_id, guild_id, channel_id, user_id`
)

const SelfIDLimit = 32

var selfIDRegex = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

func NormalizeSelfID(selfID string) string {
	return strings.ToLower(strings.TrimSpace(selfID))
}

func ValidSelfID(selfID string) error {
	if selfID == "" {
		return nil
	}
	if len([]rune(selfID)) > SelfIDLimit {
		return utilsmsg.New(utilsmsg.Limit, "self_id", "limited to %d characters", SelfIDLimit)
	}
	if !selfIDRegex.MatchString(selfID) {
		return utilsmsg.New(utilsmsg.Invalid, "self_id", "allows A-Z a-z 0-9 _ - only")
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
	if err := rows.Scan(&one.SessionID, &one.SelfID, &one.Name, &one.Model, &one.Reasoning, &one.Role,
		&one.ChatID, &one.GuildID, &one.ChannelID, &one.UserID); err != nil {
		return SessionRow{}, utilsmsg.Wrap(utilsmsg.Internal, "sql.Rows Scan", "[SELECT session]", err)
	}
	return one, nil
}

func WriteSession(ctx context.Context, r SessionRow) error {
	if conn == nil {
		return utilsmsg.New(utilsmsg.Internal, "internal/runtime/store", "New has not run")
	}
	if r.SessionID == "" {
		return utilsmsg.New(utilsmsg.Invalid, "session_id", "required")
	}
	if r.Model == "" {
		r.Model = DefaultModel
	}
	if r.Reasoning == "" {
		r.Reasoning = DefaultReasoning
	}
	r.SelfID = NormalizeSelfID(r.SelfID)
	if err := ValidSelfID(r.SelfID); err != nil {
		return err
	}

	if _, err := conn.ExecContext(ctx, `
	INSERT INTO session (`+sessionInsertColumns+`)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(session_id) DO UPDATE SET
		self_id    = excluded.self_id,
		name       = excluded.name,
		model      = excluded.model,
		reasoning  = excluded.reasoning,
		role       = excluded.role,
		rule       = excluded.rule,
		chat_id    = excluded.chat_id,
		guild_id   = excluded.guild_id,
		channel_id = excluded.channel_id,
		user_id    = excluded.user_id`,
		r.SessionID, r.SelfID, r.Name, r.Model, r.Reasoning, r.Role, r.Role,
		r.ChatID, r.GuildID, r.ChannelID, r.UserID); err != nil {
		if isSelfIDConflict(err) {
			return utilsmsg.New(utilsmsg.Conflict, "session", "%w: %s", ErrDuplicateSelfID, r.SelfID)
		}
		return utilsmsg.Wrap(utilsmsg.Internal, "sql.DB ExecContext", "[INSERT session]", err)
	}
	return nil
}

func ReadSession(ctx context.Context, sessionID string) (SessionRow, bool, error) {
	if conn == nil || sessionID == "" {
		return SessionRow{}, false, nil
	}

	rows, err := conn.QueryContext(ctx,
		`SELECT `+sessionSelectColumns+` FROM session WHERE session_id = ?`, sessionID)
	if err != nil {
		return SessionRow{}, false, utilsmsg.Wrap(utilsmsg.Internal, "sql.DB QueryContext", "[SELECT session]", err)
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

	rows, err := conn.QueryContext(ctx, `SELECT `+sessionSelectColumns+` FROM session`)
	if err != nil {
		return nil, utilsmsg.Wrap(utilsmsg.Internal, "sql.DB QueryContext", "[SELECT session]", err)
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
		return nil, utilsmsg.Wrap(utilsmsg.Internal, "sql.Rows Err", "[SELECT session]", err)
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
	return findSessionBy(ctx, "self_id", NormalizeSelfID(selfID))
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
		return utilsmsg.Wrap(utilsmsg.Internal, "sql.DB ExecContext", "[DELETE session]", err)
	}
	return nil
}
