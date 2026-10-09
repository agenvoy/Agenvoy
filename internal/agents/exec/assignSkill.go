package exec

import (
	"encoding/json"
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

	tool := provider.ToolCall{
		ID:       uuid,
		Type:     "function",
		Function: provider.ToolCallFunction{Name: "run_skill", Arguments: string(raw)},
	}

	// * pretend assistant called this tool to assign skill
	session.ToolHistories = append(session.ToolHistories,
		provider.Message{
			Role:      "assistant",
			ToolCalls: []provider.ToolCall{tool},
		},
		provider.Message{
			Role: "tool",
			Content: strings.NewReplacer(
				"{{.SkillName}}", s.Name,
				"{{.Content}}", renderActivation(s),
			).Replace(strings.TrimSpace(configs.AssignSkill)),
			ToolCallID: uuid,
		},
	)
}
