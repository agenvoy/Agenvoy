package exec

import (
	"encoding/json"
	"strings"

	go_llm_router "github.com/pardnchiu/go-llm-router/core"
	go_pkg_utils "github.com/pardnchiu/go-pkg/utils"

	"github.com/pardnchiu/agenvoy/configs"
	agenttypes "github.com/pardnchiu/agenvoy/internal/agents/types"
	"github.com/pardnchiu/agenvoy/internal/filesystem/skill"
)

func assignSkill(session *agenttypes.Session, s *skill.Skill) {
	uuid := go_pkg_utils.UUID()
	raw, _ := json.Marshal(map[string]string{"skill": s.Name})

	tool := go_llm_router.ToolCall{
		ID:       uuid,
		Type:     "function",
		Function: go_llm_router.ToolCallFunction{Name: "run_skill", Arguments: string(raw)},
	}

	// * pretend assistant called this tool to assign skill
	session.ToolHistories = append(session.ToolHistories,
		go_llm_router.Message{
			Role:      "assistant",
			ToolCalls: []go_llm_router.ToolCall{tool},
		},
		go_llm_router.Message{
			Role: "tool",
			Content: strings.NewReplacer(
				"{{.SkillName}}", s.Name,
				"{{.Content}}", renderActivation(s),
			).Replace(strings.TrimSpace(configs.AssignSkill)),
			ToolCallID: uuid,
		},
	)
}
