package actionHistory

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	historyStore "github.com/pardnchiu/agenvoy/internal/runtime/history"
	toolTypes "github.com/pardnchiu/agenvoy/internal/tools/types"
)

const displayLayout = "2006-01-02 15:04"

type record struct {
	Model       string   `json:"model,omitempty"`
	Reasoning   string   `json:"reasoning,omitempty"`
	Objective   string   `json:"objective,omitempty"`
	ToolResults []result `json:"tool_results,omitempty"`
	Todos       []todo   `json:"todos,omitempty"`
	Reply       string   `json:"reply,omitempty"`
}

type result struct {
	Name   string `json:"name"`
	Args   string `json:"args,omitempty"`
	Result string `json:"result"`
}

type todo struct {
	Content string `json:"content"`
	Status  string `json:"status"`
}

type entry struct {
	taskID string
	at     time.Time
	row    historyStore.ActionRecord
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
		r, err := load(item)
		if err != nil {
			return ""
		}
		return r.Objective
	}
	return ""
}

func entriesOf(sessionID string) ([]entry, error) {
	rows, err := historyStore.ListAction(context.Background(), sessionID)
	if err != nil {
		return nil, err
	}

	list := make([]entry, 0, len(rows))
	for _, row := range rows {
		list = append(list, entry{taskID: row.TaskHash, at: row.EndAt, row: row})
	}
	return list, nil
}

func load(item entry) (record, error) {
	raw, err := json.Marshal(item.row)
	if err != nil {
		return record{}, fmt.Errorf("encoding/json Marshal: %w", err)
	}

	var r record
	if err := json.Unmarshal(raw, &r); err != nil {
		return record{}, fmt.Errorf("encoding/json Unmarshal: %w", err)
	}
	return r, nil
}
