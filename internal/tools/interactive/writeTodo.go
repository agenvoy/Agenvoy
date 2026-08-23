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
		SystemUse:   true,
		AlwaysLoad:  true,
		AlwaysAllow: true,
		Concurrent:  false,
		Description: `A task checklist the user watches update in real time.
Call it the moment work turns multi-step (3+ steps), at the start or halfway through — N fanned-out searches or subagents count as N steps, and 分析 / 研究 / 調查 / 比較 / 彙整 / 週報 / 盤前 always get a plan.
It records progress and never executes anything. Single-step work, smalltalk, or anything one tool call resolves → skip it.`,
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"todos": map[string]any{
					"type":        "array",
					"description": "The whole ordered checklist every call — state is replaced, not merged. Exactly one item in_progress; when a step is genuinely done, flip it completed and set the next in_progress in the same call. While a plan runs the step set stays fixed: only status advances, never reword, reorder, split, merge, add or drop — a step that turns out wrong or missing is told to the user first. A fully completed plan followed by a new objective starts a fresh list, which is expected rather than a change.",
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
