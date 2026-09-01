package knowledge

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/pardnchiu/agenvoy/internal/runtime/torii"
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
	entry, ok := torii.Remote(torii.DBKnowledge).Get(context.Background(), name)
	if !ok {
		return Record{}, false
	}
	var record Record
	if err := json.Unmarshal([]byte(entry.Value()), &record); err != nil {
		return Record{}, false
	}
	record.Name = name
	return record, true
}

func Write(name, content string) error {
	raw, err := json.Marshal(Record{Name: name, Content: content, UpdatedAt: time.Now().Unix()})
	if err != nil {
		return fmt.Errorf("json.Marshal: %w", err)
	}
	return torii.Remote(torii.DBKnowledge).Set(context.Background(), name, string(raw), nil)
}

func Delete(name string) bool {
	return torii.Remote(torii.DBKnowledge).Del(context.Background(), name) > 0
}

func List() []Record {
	entries := torii.Remote(torii.DBKnowledge).Scan(context.Background(), "*", torii.ScanOption{})

	out := make([]Record, 0, len(entries))
	for _, entry := range entries {
		var record Record
		if err := json.Unmarshal([]byte(entry.Value()), &record); err != nil {
			continue
		}
		record.Name = entry.Key
		out = append(out, record)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}
