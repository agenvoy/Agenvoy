package historyStore

import (
	"fmt"
	"strings"

	provider "github.com/pardnchiu/go-llm-router/core"
)

type Message struct {
	SendAt  int64
	Role    string
	Content string
	Sender  string
}

func Write(sessionID string, list []Message) error {
	if conn == nil || len(list) == 0 {
		return nil
	}

	tx, err := conn.Write.Begin()
	if err != nil {
		return fmt.Errorf("sql.Tx Begin: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
	INSERT INTO messages (session_id, send_at, role, content, sender)
	VALUES (?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("sql.Tx Prepare [INSERT messages]: %w", err)
	}
	defer stmt.Close()

	var lastTS int64
	for _, row := range list {
		if strings.TrimSpace(row.Content) == "" {
			continue
		}

		ts := row.SendAt
		if ts == 0 && lastTS > 0 {
			ts = lastTS + 1
		}
		if ts > lastTS {
			lastTS = ts
		}

		if _, err := stmt.Exec(sessionID, ts, row.Role, row.Content, row.Sender); err != nil {
			return fmt.Errorf("sql.Stmt Exec: %w", err)
		}
	}

	return tx.Commit()
}

func ExtractContent(content any) string {
	switch val := content.(type) {
	case string:
		return val

	case []provider.ContentPart:
		var parts []string
		for _, part := range val {
			if part.Type == "text" && part.Text != "" {
				parts = append(parts, part.Text)
			}
		}
		return strings.Join(parts, "\n")

	case []any:
		var parts []string
		for _, item := range val {
			dic, ok := item.(map[string]any)
			if !ok {
				continue
			}
			if text, _ := dic["type"].(string); text == "text" {
				if text, _ := dic["text"].(string); text != "" {
					parts = append(parts, text)
				}
			}
		}
		return strings.Join(parts, "\n")

	default:
		if content == nil {
			return ""
		}
		return fmt.Sprint(content)
	}
}

func Clear(sessionID string) error {
	if conn == nil {
		return nil
	}

	if _, err := conn.Exec(`
	DELETE FROM messages
	WHERE session_id = ?
	`, sessionID); err != nil {
		return fmt.Errorf("sql.DB Exec [DELETE messages]: %w", err)
	}

	if _, err := conn.Exec(`
	DELETE FROM message_meta
	WHERE session_id = ?
	`, sessionID); err != nil {
		return fmt.Errorf("sql.DB Exec [DELETE message_meta]: %w", err)
	}
	return nil
}
