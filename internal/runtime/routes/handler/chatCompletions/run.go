package chatCompletions

import (
	"context"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/pardnchiu/agenvoy/internal/agents"
	"github.com/pardnchiu/agenvoy/internal/agents/exec"
	agentTypes "github.com/pardnchiu/agenvoy/internal/agents/types"
	"github.com/pardnchiu/agenvoy/internal/runtime"
	"github.com/pardnchiu/agenvoy/internal/tools"
	provider "github.com/pardnchiu/go-llm-router/core"
)

func run(ctx context.Context, req Request, userContent string, events chan<- agentTypes.Event) {
	scanner := agents.Scanner()
	if scanner != nil {
		scanner.Scan()
	}

	trimContent := strings.TrimSpace(userContent)
	if trimContent != "" {
		events <- agentTypes.Event{Type: agentTypes.EventUserInput, Text: trimContent}
	}

	events <- agentTypes.Event{Type: agentTypes.EventAgentSelect}

	var agent agentTypes.Agent
	var fallbacks []agentTypes.Agent
	registry := agents.Registry()
	switch {
	case req.Model == "" || req.Model == "auto":
		primary, rest, err := exec.ResolveAgent(ctx, agents.DispatcherBot(), registry, trimContent, false, "", "")
		if err != nil {
			events <- agentTypes.Event{Type: agentTypes.EventError, Err: err}
			return
		}
		agent = primary
		fallbacks = rest

	default:
		a, ok := registry.Registry[req.Model]
		if !ok {
			events <- agentTypes.Event{Type: agentTypes.EventError, Err: fmt.Errorf("model %q not found", req.Model)}
			return
		}
		agent = a
	}
	events <- agentTypes.Event{Type: agentTypes.EventAgentResult, Text: strings.TrimSpace(agent.Name())}

	workDir := req.workDir
	if workDir == "" {
		workDir, _ = os.UserHomeDir()
	}

	excludeTools := []string{"*"}
	if req.agentMode {
		excludeTools = tools.TUIOnlyTools
	}

	data := exec.ExecuteMeta{
		Agent:          agent,
		FallbackAgents: fallbacks,
		WorkDir:        workDir,
		Content:        trimContent,
		Reasoning:      req.ReasoningEffort,
		ExcludeTools:   excludeTools,
		ExcludeSkills:  tools.TUIOnlySkills,
		ClientTools:    req.Tools,
		AllowAll:       true,
	}

	session := buildStatelessSession(req, trimContent, workDir, scanner, data.ExcludeSkills, data.ModelName())

	if err := exec.Execute(ctx, data, session, events, true); err != nil {
		events <- agentTypes.Event{Type: agentTypes.EventError, Err: err}
	}
}

func buildStatelessSession(req Request, userInput, workDir string, scanner *runtime.SkillScanner, excludeSkills []string, model string) *agentTypes.AgentSession {
	var systemPrompts []provider.Message
	if req.agentMode {
		systemPrompts = exec.BuildChatCompletionsSystemPrompts(workDir, scanner, excludeSkills, model)
	}
	systemPrompts = append(systemPrompts, req.systemPrompts...)

	lastUserIdx := -1
	for i, one := range slices.Backward(req.Messages) {
		if one.Role == "user" {
			lastUserIdx = i
			break
		}
	}
	var oldHistories []provider.Message
	if lastUserIdx > 0 {
		oldHistories = append(oldHistories, req.Messages[:lastUserIdx]...)
	}

	return &agentTypes.AgentSession{
		SystemPrompts: systemPrompts,
		OldHistories:  oldHistories,
		Histories:     append([]provider.Message{}, oldHistories...),
		ToolHistories: []provider.Message{},
		Tools:         []provider.Message{},
		UserInput:     provider.Message{Role: "user", Content: userInput},
		Stateless:     true,
	}
}
