package note

import (
	"fmt"
	"time"
)

type Record struct {
	Name      string `json:"name"`
	Content   string `json:"content"`
	UpdatedAt int64  `json:"updated_at"`
}

func Read(name string) (Record, bool) {
	if conn == nil {
		return Record{}, false
	}

	record := Record{Name: name}
	if err := conn.Read.QueryRow(`
	SELECT content, updated_at
	FROM note
	WHERE name = ?
	`, name).Scan(&record.Content, &record.UpdatedAt); err != nil {
		return Record{}, false
	}
	return record, true
}

func Write(name, content string) error {
	if conn == nil {
		return fmt.Errorf("internal/note: New has not run")
	}

	_, err := conn.Exec(`
	INSERT INTO note (name, content, updated_at)
	VALUES (?, ?, ?)
	ON CONFLICT(name)
	DO UPDATE SET content = excluded.content, updated_at = excluded.updated_at
	`, name, content, time.Now().Unix())
	return err
}

func Exists() bool {
	if conn == nil {
		return false
	}

	var exists bool
	if err := conn.Read.QueryRow(`SELECT EXISTS(SELECT 1 FROM note)`).Scan(&exists); err != nil {
		return false
	}
	return exists
}
