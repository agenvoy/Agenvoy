package errorHistory

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/pardnchiu/agenvoy/internal/agents/exec/memory"
	"github.com/pardnchiu/agenvoy/internal/runtime/torii"
	toolRegister "github.com/pardnchiu/agenvoy/internal/tools/register"
	toolTypes "github.com/pardnchiu/agenvoy/internal/tools/types"
)

const (
	defaultSearchLimit = 4
	maxSearchLimit     = 16
)

func init() {
	registErrorHistory()
}

func registErrorHistory() {
	toolRegister.Regist(toolRegister.Def{
		Name:        "error_history",
		SystemUse:   true,
		AlwaysLoad:  false,
		AlwaysAllow: true,
		Concurrent:  true,
		Description: `Tool failures kept across sessions: what broke, why, what was done about it, and whether that worked.
Search it before a second retry when no error hint was injected, read one by hash when a tool answers "no data: {hash}", write once a non-trivial fix is confirmed or a strategy is confirmed dead.
A past run's own steps → chat_history; the full recovery loop → reasoning_guide(topic=tool_error).`,
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"mode": map[string]any{
					"type":        "string",
					"enum":        []string{"search", "read", "write"},
					"description": "search: past records by keyword — resolved means apply it, failed or abandoned means avoid it. read: one record by hash. write: persist this error. Omitted: hash → read, outcome → write, otherwise search.",
					"default":     "search",
				},
				"keyword": map[string]any{
					"type":        "string",
					"description": "mode=search: tool name, error symptom, or parameter trait.",
				},
				"limit": map[string]any{
					"type":        "integer",
					"description": "mode=search: max records to return. Never above 16.",
					"default":     defaultSearchLimit,
				},
				"hash": map[string]any{
					"type":        "string",
					"description": "mode=read: error hash, 8-char hex (e.g. 'a1b2c3d4').",
				},
				"tool_name": map[string]any{
					"type":        "string",
					"description": "mode=write: the tool that produced the error (e.g. 'fetch_page').",
				},
				"keywords": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type": "string",
					},
					"description": "mode=write: lookup keywords — tool name, error type, parameter traits. Be specific.",
				},
				"symptom": map[string]any{
					"type":        "string",
					"description": "mode=write: observed behavior — what the tool returned or failed on.",
				},
				"cause": map[string]any{
					"type":        "string",
					"description": "mode=write: root cause, once confirmed.",
					"default":     "",
				},
				"action": map[string]any{
					"type":        "string",
					"description": "mode=write: what was done (e.g. 'retried with English keyword', 'fell back to search_web').",
				},
				"outcome": map[string]any{
					"type":        "string",
					"enum":        []string{"resolved", "failed", "abandoned"},
					"description": "mode=write: resolved = the fix worked; failed = strategy confirmed non-working; abandoned = 3+ approaches tried.",
				},
			},
		},
		Handler: func(ctx context.Context, e *toolTypes.Executor, args json.RawMessage) (string, error) {
			var params struct {
				Mode     string   `json:"mode"`
				Keyword  string   `json:"keyword"`
				Limit    int      `json:"limit"`
				Hash     string   `json:"hash"`
				ToolName string   `json:"tool_name"`
				Keywords []string `json:"keywords"`
				Symptom  string   `json:"symptom"`
				Cause    string   `json:"cause"`
				Action   string   `json:"action"`
				Outcome  string   `json:"outcome"`
			}
			if len(args) > 0 {
				if err := json.Unmarshal(args, &params); err != nil {
					return "", fmt.Errorf("json.Unmarshal: %w", err)
				}
			}

			params.Mode = strings.TrimSpace(params.Mode)
			params.Hash = strings.TrimSpace(params.Hash)
			params.Outcome = strings.TrimSpace(params.Outcome)
			if params.Mode == "" {
				params.Mode = "search"
				switch {
				case params.Hash != "":
					params.Mode = "read"
				case params.Outcome != "":
					params.Mode = "write"
				}
			}

			switch params.Mode {
			case "search":
				keyword := strings.TrimSpace(params.Keyword)
				if keyword == "" {
					return "", fmt.Errorf("keyword is required when mode=search")
				}
				limit := params.Limit
				if limit <= 0 {
					limit = defaultSearchLimit
				}
				return memory.Search(ctx, "", keyword, min(limit, maxSearchLimit)), nil

			case "read":
				if params.Hash == "" {
					return "", fmt.Errorf("hash is required when mode=read")
				}
				return readRecord(params.Hash), nil

			case "write":
				record := memory.Record{
					ToolName: strings.TrimSpace(params.ToolName),
					Keywords: params.Keywords,
					Symptom:  strings.TrimSpace(params.Symptom),
					Cause:    strings.TrimSpace(params.Cause),
					Action:   strings.TrimSpace(params.Action),
					Outcome:  params.Outcome,
				}
				if err := requireRecord(record); err != nil {
					return "", err
				}
				saveCtx := context.WithoutCancel(ctx)
				go func() {
					if _, err := memory.Save(saveCtx, e.SessionID, record); err != nil {
						slog.Warn("error_history: memory.Save",
							slog.String("tool", record.ToolName),
							slog.String("error", err.Error()))
					}
				}()
				return "recorded", nil
			}
			return "", fmt.Errorf("unknown mode %q; available: search, read, write", params.Mode)
		},
	})
}

func requireRecord(r memory.Record) error {
	switch {
	case r.ToolName == "":
		return fmt.Errorf("tool_name is required when mode=write")
	case len(r.Keywords) == 0:
		return fmt.Errorf("keywords is required when mode=write")
	case r.Symptom == "":
		return fmt.Errorf("symptom is required when mode=write")
	case r.Action == "":
		return fmt.Errorf("action is required when mode=write")
	case r.Outcome == "":
		return fmt.Errorf("outcome is required when mode=write")
	}
	return nil
}

func readRecord(hash string) string {
	db := torii.DB(torii.DBErrorMemory)
	for _, key := range db.Keys("*") {
		entry, ok := db.Get(key)
		if !ok {
			continue
		}
		var rec memory.Record
		if err := json.Unmarshal([]byte(entry.Value()), &rec); err != nil {
			continue
		}
		if rec.ID == hash {
			return entry.Value()
		}
	}
	return "not found"
}
