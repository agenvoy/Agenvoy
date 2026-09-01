package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	toolRegister "github.com/pardnchiu/agenvoy/internal/tools/register"
	toolTypes "github.com/pardnchiu/agenvoy/internal/tools/types"
)

func registSchedules() {
	toolRegister.Regist(toolRegister.Def{
		Name:        "schedules",
		SystemUse:   false,
		AlwaysLoad:  false,
		AlwaysAllow: true,
		Concurrent:  false,
		Description: `Scheduled runs bound to a scheduler skill: what is queued (list), moving one to a new time (patch), cancelling one (remove).
Use for 有哪些排程 / 改時間 / 取消排程 / 那個定時任務還在嗎.
Creating a schedule → the scheduler-skill-creator skill, never mode=write. A dry-run request → find the name with mode=list, read its SKILL.md and run the steps now; never answer "run /sched-X in the TUI".`,
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"mode": map[string]any{
					"type":        "string",
					"enum":        []string{"list", "patch", "remove", "write"},
					"description": "list: what this session has queued. patch: move an entry to a new time. remove: cancel it and trash its skill. write: the internal binding step of scheduler-skill-creator — its skill_name carries a hash only that flow produces, so a hand-made one always fails. Omitted: list.",
					"default":     "list",
				},
				"target": map[string]any{
					"type":        "string",
					"enum":        []string{"task", "cron", "all"},
					"description": "Which kind of entry. task: one-shot fire time. cron: recurring 5-field expression. all: mode=list only.",
					"default":     "all",
				},
				"skill_name": map[string]any{
					"type":        "string",
					"description": "mode=write / mode=patch / mode=remove: scheduler skill full name including its hash suffix (e.g. 'meeting-reminder-a3f9b2c1'), no 'scheduler-' prefix. Required.",
				},
				"time": map[string]any{
					"type":        "string",
					"description": "mode=write / mode=patch: required. target=task: '+5m' / '+1h30m' (relative), '15:04' (today clock), '2006-01-02 15:04' (local datetime), or RFC3339. target=cron: 5-field expression '{min} {hour} {dom} {mon} {dow}', a descriptor (@yearly @annually @monthly @weekly @daily @midnight @hourly), or '@every {duration}' (minimum 30s).",
				},
			},
		},
		Handler: func(ctx context.Context, e *toolTypes.Executor, args json.RawMessage) (string, error) {
			var params struct {
				Mode      string `json:"mode"`
				Target    string `json:"target"`
				SkillName string `json:"skill_name"`
				Time      string `json:"time"`
			}
			if len(args) > 0 {
				if err := json.Unmarshal(args, &params); err != nil {
					return "", fmt.Errorf("json.Unmarshal: %w", err)
				}
			}

			mode := strings.ToLower(strings.TrimSpace(params.Mode))
			if mode == "" {
				mode = "list"
			}
			target := strings.ToLower(strings.TrimSpace(params.Target))
			skill := strings.TrimSpace(params.SkillName)

			if mode == "list" {
				if target == "" {
					target = "all"
				}
				if target != "task" && target != "cron" && target != "all" {
					return "", fmt.Errorf("target must be 'task', 'cron', or 'all' (got %q)", target)
				}
				return listSchedule(e, target)
			}

			if skill == "" {
				return "", fmt.Errorf("skill_name is required when mode=%s", mode)
			}
			if target != "task" && target != "cron" {
				return "", fmt.Errorf("target must be 'task' or 'cron' when mode=%s (got %q)", mode, target)
			}

			switch mode {
			case "write":
				if strings.TrimSpace(params.Time) == "" {
					return "", fmt.Errorf("time is required when mode=write")
				}
				return writeSchedule(e, target, params.Time, skill)
			case "patch":
				if strings.TrimSpace(params.Time) == "" {
					return "", fmt.Errorf("time is required when mode=patch")
				}
				return patchSchedule(target, params.Time, skill)
			case "remove":
				return removeSchedule(ctx, e, target, skill)
			}
			return "", fmt.Errorf("unknown mode %q; available: list, patch, remove, write", mode)
		},
	})
}
