package knowledge

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrNotFound = errors.New("knowledge not found")
	ErrExists   = errors.New("knowledge already exists")
	ErrWrite    = errors.New("knowledge write failed")
)

type Record struct {
	Name      string `json:"name"`
	Content   string `json:"content"`
	UpdatedAt int64  `json:"updated_at"`
}

const MaxNameRunes = 32

func Name(name, content string) (string, error) {
	if strings.TrimSpace(name) == "" {
		name = firstLine(content)
	}
	key, err := Key(name)
	if err != nil {
		return "", err
	}
	if len([]rune(key)) > MaxNameRunes {
		return "", fmt.Errorf("name cannot exceed %d characters", MaxNameRunes)
	}
	return key, nil
}

func firstLine(content string) string {
	for line := range strings.SplitSeq(content, "\n") {
		line = strings.Map(func(r rune) rune {
			if strings.ContainsRune(`/\*?[]`, r) {
				return -1
			}
			return r
		}, line)
		line = strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(line), "#. \t"))
		if line == "" {
			continue
		}
		if list := []rune(line); len(list) > MaxNameRunes {
			line = strings.TrimSpace(string(list[:MaxNameRunes]))
		}
		return line
	}
	return ""
}

func Key(name string) (string, error) {
	name = strings.TrimSpace(name)
	name = strings.TrimSuffix(name, ".md")

	switch {
	case name == "":
		return "", fmt.Errorf("name is required")
	case strings.ContainsAny(name, `/\`):
		return "", fmt.Errorf("name cannot contain a path separator")
	case name == "." || name == "..":
		return "", fmt.Errorf("name cannot be a path segment")
	case strings.HasPrefix(name, "."):
		return "", fmt.Errorf("name cannot start with a dot")
	case strings.ContainsAny(name, "*?[]"):
		return "", fmt.Errorf("name cannot contain a glob character")
	}
	return name, nil
}

func Read(name string) (Record, bool) {
	if conn == nil {
		return Record{}, false
	}

	record := Record{Name: name}
	if err := conn.Read.QueryRow(`
	SELECT content, updated_at
	FROM knowledge
	WHERE name = ?
	`, name).Scan(&record.Content, &record.UpdatedAt); err != nil {
		return Record{}, false
	}
	return record, true
}

func Write(name, content string) error {
	if conn == nil {
		return fmt.Errorf("internal/knowledge: New has not run")
	}

	_, err := conn.Exec(`
	INSERT INTO knowledge (name, content, updated_at)
	VALUES (?, ?, ?)
	ON CONFLICT(name)
	DO UPDATE SET content = excluded.content, updated_at = excluded.updated_at
	`, name, content, time.Now().Unix())
	return err
}

func Delete(name string) bool {
	if conn == nil {
		return false
	}

	result, err := conn.Exec(`DELETE FROM knowledge WHERE name = ?`, name)
	if err != nil {
		return false
	}
	affected, err := result.RowsAffected()
	return err == nil && affected > 0
}

func List() []Record {
	if conn == nil {
		return nil
	}

	rows, err := conn.Read.Query(`
	SELECT name, content, updated_at
	FROM knowledge
	ORDER BY name
	`)
	if err != nil {
		return nil
	}
	defer rows.Close()

	out := []Record{}
	for rows.Next() {
		var record Record
		if err := rows.Scan(&record.Name, &record.Content, &record.UpdatedAt); err != nil {
			return nil
		}
		out = append(out, record)
	}
	if rows.Err() != nil {
		return nil
	}
	return out
}

func Create(name, content string) (string, error) {
	key, err := Name(name, content)
	if err != nil {
		return "", err
	}
	if _, exists := Read(key); exists {
		return "", ErrExists
	}
	if err := Write(key, content); err != nil {
		return "", fmt.Errorf("%w: %w", ErrWrite, err)
	}
	return key, nil
}

func Update(name, rename, content string) (string, error) {
	key, err := Key(name)
	if err != nil {
		return "", err
	}
	if _, exists := Read(key); !exists {
		return "", ErrNotFound
	}

	target := key
	if strings.TrimSpace(rename) != "" {
		if target, err = Name(rename, content); err != nil {
			return "", err
		}
		if target != key {
			if _, exists := Read(target); exists {
				return "", ErrExists
			}
		}
	}

	if err := Write(target, content); err != nil {
		return "", fmt.Errorf("%w: %w", ErrWrite, err)
	}
	if target != key {
		Delete(key)
	}
	return target, nil
}
