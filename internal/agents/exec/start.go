package exec

import (
	"context"
	"fmt"
	"slices"
	"strings"

	go_pkg_utils "github.com/pardnchiu/go-pkg/utils"

	"github.com/pardnchiu/agenvoy/configs"
	"github.com/pardnchiu/agenvoy/internal/agents"
	agentTypes "github.com/pardnchiu/agenvoy/internal/agents/types"
	"github.com/pardnchiu/agenvoy/internal/runtime"
	sessionLog "github.com/pardnchiu/agenvoy/internal/session/log"
	"github.com/pardnchiu/agenvoy/internal/tools"
)

func Prepare(data ExecuteMeta) ExecuteMeta {
	if !data.TUI {
		data.ExcludeTools = append(append([]string{}, tools.TUIOnlyTools...), data.ExcludeTools...)
		data.ExcludeSkills = append(append([]string{}, tools.TUIOnlySkills...), data.ExcludeSkills...)
	}

	scanner := data.SkillScanner
	if scanner == nil {
		scanner = agents.Scanner()
	}
	if scanner != nil {
		scanner.Scan()
		if data.Skill == nil && data.SkillName == "" {
			if matched, effective := runtime.MatchSkill(scanner, data.Content, data.ExcludeSkills...); matched != nil {
				data.Skill = matched
				data.Content = strings.TrimSpace(effective)
				data.Input = data.Content
			}
		}
	}
	return data
}

func Start(ctx context.Context, data ExecuteMeta, events chan<- agentTypes.Event) error {
	sessionID := strings.TrimSpace(data.SessionID)
	if sessionID == "" {
		return fmt.Errorf("data.SessionID is required")
	}

	windowHash := agentTypes.WindowHash(ctx)
	taskHash := data.PendingTask
	if taskHash == "" {
		taskHash = go_pkg_utils.UUID()
		data.TaskHash = taskHash
	}
	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)
	registerCancel(taskHash, cancel)
	defer unregisterCancel(taskHash)

	if name := data.SkillName; name != "" && data.Skill == nil {
		if slices.Contains(data.ExcludeSkills, name) {
			return fmt.Errorf("skill %q is not available here", name)
		}
		scanner := data.SkillScanner
		if scanner == nil {
			scanner = agents.Scanner()
		}
		if scanner != nil {
			data.Skill = scanner.Lookup(name)
		}
		if data.Skill == nil {
			return fmt.Errorf("skill %q not found", name)
		}
	}

	if data.Skill != nil {
		skillResult := agentTypes.Event{Type: agentTypes.EventSkillResult, Text: strings.TrimSpace(data.Skill.Name), TaskHash: taskHash, WindowHash: windowHash}
		events <- skillResult
		sessionLog.Record(sessionID, skillResult)
	}

	if input := strings.TrimSpace(data.Input); input != "" {
		events <- agentTypes.Event{Type: agentTypes.EventUserInput, Text: input, TaskHash: taskHash, WindowHash: windowHash}
		sessionLog.Append(sessionID, windowHash, input)
	}

	events <- agentTypes.Event{Type: agentTypes.EventAgentSelect, TaskHash: taskHash, WindowHash: windowHash}

	agent, fallbacks, reasoning, err := ResolveAgent(ctx, data.Model, data.Content, data.Skill != nil, SkillHint(data.Skill), sessionID)
	if err != nil {
		return fmt.Errorf("ResolveAgent: %w", err)
	}
	if data.Reasoning == "" || data.Reasoning == configs.REASONING_AUTO {
		data.Reasoning = reasoning
	}
	agentName := strings.TrimSpace(agent.Name())
	agentResult := agentTypes.Event{
		Type:       agentTypes.EventAgentResult,
		Text:       agentName,
		Reasoning:  resolveReasoning(sessionID, data.Reasoning).String(),
		TaskHash:   taskHash,
		WindowHash: windowHash,
	}
	events <- agentResult
	sessionLog.Record(sessionID, agentResult)

	data.Agent = agent
	data.FallbackAgents = fallbacks
	session, err := GetSession(ctx, data)
	if err != nil {
		return fmt.Errorf("GetSession: %w", err)
	}
	return Execute(ctx, data, session, events, data.AllowAll)
}
