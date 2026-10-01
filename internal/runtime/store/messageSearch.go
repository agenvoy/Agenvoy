package historyStore

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/pardnchiu/agenvoy/configs"
)

type Result struct {
	Timestamp int64
	Role      string
	Content   string
	Sender    string
}

const ftsMinRunes = 3

func escapeLike(keyword string) string {
	return strings.NewReplacer(`\`, `\\`, "%", `\%`, "_", `\_`).Replace(keyword)
}

func Search(sessionID, keyword, timeRange string, limit int) ([]Result, error) {
	if conn == nil {
		return nil, nil
	}

	jsonStart := GetStartAt(sessionID)

	match := `AND m.id IN (SELECT rowid FROM messages_fts5 WHERE messages_fts5 MATCH ?)`
	term := fmt.Sprintf(`"%s"`, strings.ReplaceAll(keyword, `"`, `""`))
	if len([]rune(keyword)) < ftsMinRunes {
		match = `AND m.content LIKE ? ESCAPE '\'`
		term = "%" + escapeLike(keyword) + "%"
	}

	var after int64
	if d, ok := configs.TIME_RANGES[timeRange]; ok {
		after = time.Now().Add(-d).UnixNano()
	}

	before := int64(math.MaxInt64)
	if jsonStart > 0 {
		before = jsonStart
	}

	rows, err := conn.Query(`
	SELECT m.send_at, m.role, m.content, m.sender
	FROM messages m
	WHERE m.session_id = ?
	`+match+`
	AND m.send_at >= ?
	AND m.send_at < ?
	ORDER BY m.send_at DESC
	LIMIT ?
	`, sessionID, term, after, before, limit)
	if err != nil {
		return nil, fmt.Errorf("sql.DB Query [SELECT messages]: %w", err)
	}
	defer rows.Close()

	var list []Result
	for rows.Next() {
		var result Result
		if err := rows.Scan(&result.Timestamp, &result.Role, &result.Content, &result.Sender); err != nil {
			continue
		}
		list = append(list, result)
	}
	return list, rows.Err()
}
