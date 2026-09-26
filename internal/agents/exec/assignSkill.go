package exec

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pardnchiu/agenvoy/configs"
	agentTypes "github.com/pardnchiu/agenvoy/internal/agents/types"
	"github.com/pardnchiu/agenvoy/internal/filesystem/skill"
	provider "github.com/pardnchiu/go-llm-router/core"
	go_pkg_utils "github.com/pardnchiu/go-pkg/utils"
)

func assignSkill(session *agentTypes.AgentSession, s *skill.Skill) {
	uuid := go_pkg_utils.UUID()
	raw, _ := json.Marshal(map[string]string{"skill": s.Name})

	tool := provider.ToolCall{ID: uuid, Type: "function"}
	// TODO: gp append type ToolCallFunction to go-llm-router
	tool.Function.Name = "run_skill"
	tool.Function.Arguments = string(raw)

	// * pretend assistant already called this tool, record tool call in history
	session.ToolHistories = append(session.ToolHistories,
		provider.Message{
			Role:      "assistant",
			ToolCalls: []provider.ToolCall{tool},
		},
		provider.Message{
			Role:       "tool",
			Content:    fmt.Sprintf("skill %s is loaded; its steps, execution rules and the built-in tool list are in the BINDING SKILL system message.", s.Name),
			ToolCallID: uuid,
		},
	)

	session.SystemPrompts = append(session.SystemPrompts, provider.Message{
		Role: "system",
		Content: strings.NewReplacer(
			"{{.SkillName}}", s.Name,
			"{{.Content}}", renderActivation(s),
		).Replace(strings.TrimSpace(configs.AssignSkill)),
	})
}
