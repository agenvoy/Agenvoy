package chatCompletions

import (
	"cmp"
	"context"
	"os"
	"slices"
	"strings"

	"github.com/pardnchiu/agenvoy/internal/agents"
	"github.com/pardnchiu/agenvoy/internal/agents/exec"
	execModel "github.com/pardnchiu/agenvoy/internal/agents/exec/model"
	agentTypes "github.com/pardnchiu/agenvoy/internal/agents/types"
	configBot "github.com/pardnchiu/agenvoy/internal/session/config/bot"
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

	model := configBot.DefaultModel
	if trimModel := strings.TrimSpace(req.Model); trimModel != "" {
		model = trimModel
	}
	agent, fallbacks, reasoning, err := execModel.Route(ctx, model, trimContent, "", "")
	if err != nil {
		events <- agentTypes.Event{Type: agentTypes.EventError, Err: err}
		return
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
		Task:           trimContent,
		Reasoning:      cmp.Or(req.ReasoningEffort, reasoning, "medium"),
		ExcludeTools:   excludeTools,
		ExcludeSkills:  tools.TUIOnlySkills,
		ClientTools:    req.Tools,
		AllowAll:       true,
	}

	session := buildStatelessSession(req, trimContent, workDir, data.Agent.Name())

	if err := exec.Execute(ctx, data, session, events, true); err != nil {
		events <- agentTypes.Event{Type: agentTypes.EventError, Err: err}
	}
}

func buildStatelessSession(req Request, userInput, workDir string, model string) *agentTypes.AgentSession {
	var systemPrompts []provider.Message
	if req.agentMode {
		systemPrompts = exec.BuildChatCompletionsSystemPrompts(workDir, model)
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
		SystemPrompts:    systemPrompts,
		ContextHistories: oldHistories,
		ToolHistories:    []provider.Message{},
		Tools:            []provider.Message{},
		Message:          provider.Message{Role: "user", Content: userInput},
		Stateless:        true,
	}
}
