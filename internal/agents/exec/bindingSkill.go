package exec

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	agentTypes "github.com/pardnchiu/agenvoy/internal/agents/types"
	"github.com/pardnchiu/agenvoy/internal/filesystem/skill"
	provider "github.com/pardnchiu/go-llm-router/core"
)

func assignBindingSkill(session *agentTypes.AgentSession, s *skill.Skill) {
	id := "skill-assign-" + newID("skill", s.Name)
	argsJSON, _ := json.Marshal(map[string]string{"skill": s.Name})
	call := provider.ToolCall{
		ID:   id,
		Type: "function",
	}
	call.Function.Name = "run_skill"
	call.Function.Arguments = string(argsJSON)

	session.ToolHistories = append(session.ToolHistories,
		provider.Message{
			Role:      "assistant",
			ToolCalls: []provider.ToolCall{call},
		},
		provider.Message{
			Role:       "tool",
			Content:    renderActivation(s),
			ToolCallID: id,
		},
	)

	bindingHeader := fmt.Sprintf(
		"## BINDING SKILL EXECUTION — /%s\n\nThe user invoked /%s. Execute the procedure below by making the tool calls SKILL.md prescribes, in order.\n\n### How to obey\n\n- **When SKILL.md says «ask_user», invoke the `ask_user` tool** with JSON arguments matching the template SKILL.md gives. Writing a text question and waiting for a chat reply is NOT the same action and does not satisfy the step.\n- **The text following `/%s` is the user's INPUT to gather around, not a set of pre-filled answers.** Even if it looks complete, your next action is still `ask_user` to verify direction. Treat it like a topic, not a finished spec.\n- **After one tool call's result arrives, immediately make the next tool call SKILL.md prescribes**, in the same turn. Do not insert text like «下一步要不要繼續» between steps — the user already authorized the full procedure by typing `/%s`.\n- **Tool calls beat chat text.** If you find yourself writing instructions to the user («再丟一句…», «直接回我…»), stop and make the corresponding tool call instead.\n\n### Quick self-check before each turn\n\n1. What does SKILL.md say the next step is? (e.g. «呼叫 ask_user 問三維度之一»)\n2. Have I made that exact tool call in this turn? If no → make it now. If yes and result is back → make the step-after's tool call.\n\n---\n\n",
		s.Name, s.Name, s.Name, s.Name,
	)
	session.SystemPrompts = append(session.SystemPrompts, provider.Message{
		Role:    "system",
		Content: bindingHeader + renderActivation(s),
	})
}

func newID(parts ...string) string {
	h := sha256.Sum256([]byte(strings.Join(parts, "|") + fmt.Sprint(time.Now().UnixNano())))
	return hex.EncodeToString(h[:])[:8]
}
