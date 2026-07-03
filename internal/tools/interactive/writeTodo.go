package interactive

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	agentTypes "github.com/pardnchiu/agenvoy/internal/agents/types"
	"github.com/pardnchiu/agenvoy/internal/filesystem"
	toolRegister "github.com/pardnchiu/agenvoy/internal/tools/register"
	toolTypes "github.com/pardnchiu/agenvoy/internal/tools/types"
	go_pkg_filesystem "github.com/pardnchiu/go-pkg/filesystem"
)

type todoInput struct {
	Content    string `json:"content"`
	Status     string `json:"status"`
	ActiveForm string `json:"active_form,omitempty"`
}

func registWriteTodo() {
	toolRegister.Regist(toolRegister.Def{
		Name:        "write_todo",
		AlwaysAllow: true,
		AlwaysLoad:  true,
		Concurrent:  false,
		Description: "Maintain a live task checklist that the user watches update in real time. Call it at the START of any task with 3+ distinct steps, or when the user gives a multi-part / plan-then-execute request — write the full plan as todos, then call again after each step to flip its status. Pass the ENTIRE list every time (state is replaced, not merged). Exactly one item may be `in_progress`; mark a step `completed` only once it is truly done, then set the next one `in_progress` in the same call. Skip for single-step tasks, smalltalk, or anything one tool call resolves. This tool only records progress; it never executes the steps.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"todos": map[string]any{
					"type":        "array",
					"description": "The complete ordered checklist, replacing any prior list. First item(s) done, exactly one in progress, the rest pending.",
					"minItems":    1,
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"content": map[string]any{
								"type":        "string",
								"description": "Imperative step title (e.g. 'Add lang toggle to nav'). Short, concrete, verifiable.",
							},
							"status": map[string]any{
								"type":        "string",
								"enum":        []string{agentTypes.TodoPending, agentTypes.TodoInProgress, agentTypes.TodoCompleted},
								"description": "pending = not started, in_progress = working now (only one allowed), completed = done.",
							},
							"active_form": map[string]any{
								"type":        "string",
								"description": "Present-continuous label shown while this step runs (e.g. 'Adding lang toggle'). Optional; falls back to content.",
							},
						},
						"required": []string{"content", "status"},
					},
				},
			},
			"required": []string{"todos"},
		},
		Handler: func(ctx context.Context, e *toolTypes.Executor, args json.RawMessage) (string, error) {
			var params struct {
				Todos []todoInput `json:"todos"`
			}
			if err := json.Unmarshal(args, &params); err != nil {
				return "", fmt.Errorf("json.Unmarshal: %w", err)
			}
			if len(params.Todos) == 0 {
				return "", fmt.Errorf("todos must contain at least one item")
			}

			todos := make([]agentTypes.TodoItem, 0, len(params.Todos))
			var done, doing, pending int
			for i, in := range params.Todos {
				content := strings.TrimSpace(in.Content)
				if content == "" {
					return "", fmt.Errorf("todo #%d has empty content", i+1)
				}
				status := strings.TrimSpace(in.Status)
				switch status {
				case agentTypes.TodoCompleted:
					done++
				case agentTypes.TodoInProgress:
					doing++
				case agentTypes.TodoPending, "":
					status = agentTypes.TodoPending
					pending++
				default:
					return "", fmt.Errorf("todo #%d has invalid status %q (want pending / in_progress / completed)", i+1, in.Status)
				}
				todos = append(todos, agentTypes.TodoItem{
					Content:    content,
					Status:     status,
					ActiveForm: strings.TrimSpace(in.ActiveForm),
				})
			}
			if doing > 1 {
				return "", fmt.Errorf("only one todo may be in_progress at a time, got %d", doing)
			}

			sessionID, taskHash := "", ""
			if e != nil {
				sessionID, taskHash = e.SessionID, e.PendingTask
			}
			if taskHash != "" {
				if err := WriteTodos(sessionID, taskHash, todos); err != nil {
					return "", fmt.Errorf("WriteTodos: %w", err)
				}
			}

			return fmt.Sprintf("checklist saved: %d step(s) — %d done, %d in progress, %d pending", len(todos), done, doing, pending), nil
		},
	})
}

func WriteTodos(sessionID, taskHash string, todos []agentTypes.TodoItem) error {
	if taskHash == "" {
		return nil
	}
	pendingMu.Lock()
	defer pendingMu.Unlock()

	meta, err := go_pkg_filesystem.ReadJSON[pendingMeta](filesystem.PendingMetaPath(sessionID, taskHash))
	if err != nil {
		meta = pendingMeta{}
	}
	meta.Todos = todos
	if writeErr := writePending(sessionID, taskHash, &meta); writeErr != nil {
		return fmt.Errorf("writePending: %w", writeErr)
	}
	return nil
}

func LoadTodos(sessionID, taskHash string) []agentTypes.TodoItem {
	if taskHash == "" {
		return nil
	}
	meta, err := go_pkg_filesystem.ReadJSON[pendingMeta](filesystem.PendingMetaPath(sessionID, taskHash))
	if err != nil {
		return nil
	}
	return meta.Todos
}
