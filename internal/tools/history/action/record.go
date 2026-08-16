package actionHistory

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	go_pkg_filesystem "github.com/pardnchiu/go-pkg/filesystem"
	go_pkg_filesystem_reader "github.com/pardnchiu/go-pkg/filesystem/reader"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
	toolTypes "github.com/pardnchiu/agenvoy/internal/tools/types"
)

const (
	nameLayout    = "2006-01-02-15-04"
	displayLayout = "2006-01-02 15:04"
)

type record struct {
	Model        string    `json:"model,omitempty"`
	Reasoning    string    `json:"reasoning,omitempty"`
	Objective    string    `json:"objective,omitempty"`
	Completed    []string  `json:"completed,omitempty"`
	NextSteps    []string  `json:"next_steps,omitempty"`
	Answer       []answer  `json:"answer,omitempty"`
	ToolAttempts []attempt `json:"tool_attempts,omitempty"`
	ToolResults  []result  `json:"tool_results,omitempty"`
	Todos        []todo    `json:"todos,omitempty"`
	Reply        string    `json:"reply,omitempty"`
}

type answer struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

type attempt struct {
	Name string `json:"name"`
	Args string `json:"args"`
}

type result struct {
	Name   string `json:"name"`
	ID     string `json:"id"`
	Result string `json:"result"`
}

type todo struct {
	Content    string `json:"content"`
	Status     string `json:"status"`
	ActiveForm string `json:"active_form,omitempty"`
}

type entry struct {
	path   string
	taskID string
	at     time.Time
}

func entries(e *toolTypes.Executor) ([]entry, error) {
	if e == nil || e.SessionID == "" {
		return nil, fmt.Errorf("no session: task history is recorded per session")
	}
	return entriesOf(e.SessionID)
}

func Objective(sessionID, taskID string) string {
	if sessionID == "" || taskID == "" {
		return ""
	}

	list, err := entriesOf(sessionID)
	if err != nil {
		return ""
	}
	for _, item := range list {
		if item.taskID != taskID {
			continue
		}
		r, err := load(item.path)
		if err != nil {
			return ""
		}
		return r.Objective
	}
	return ""
}

func entriesOf(sessionID string) ([]entry, error) {
	dir := filesystem.TaskHistoryDir(sessionID)
	if !go_pkg_filesystem_reader.IsDir(dir) {
		return nil, nil
	}

	names, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("os.ReadDir [%s]: %w", dir, err)
	}

	var list []entry
	for _, name := range names {
		if name.IsDir() || !strings.HasSuffix(name.Name(), ".json") {
			continue
		}

		trimmed := strings.TrimSuffix(name.Name(), ".json")
		if len(trimmed) <= len(nameLayout)+1 {
			continue
		}
		at, err := time.ParseInLocation(nameLayout, trimmed[:len(nameLayout)], time.Local)
		if err != nil {
			continue
		}

		list = append(list, entry{
			path:   filepath.Join(dir, name.Name()),
			taskID: trimmed[len(nameLayout)+1:],
			at:     at,
		})
	}

	slices.SortFunc(list, func(a, b entry) int {
		return strings.Compare(b.path, a.path)
	})
	return list, nil
}

func load(path string) (record, error) {
	r, err := go_pkg_filesystem.ReadJSON[record](path)
	if err != nil {
		return record{}, fmt.Errorf("github.com/pardnchiu/go-pkg/filesystem ReadJSON [%s]: %w", path, err)
	}
	return r, nil
}
