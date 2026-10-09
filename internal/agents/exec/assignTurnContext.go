package exec

import (
	"strings"

	provider "github.com/pardnchiu/go-llm-router/core"

	agentTypes "github.com/pardnchiu/agenvoy/internal/agents/types"
	"github.com/pardnchiu/agenvoy/internal/runtime"
)

func assignTurnContext(session *agentTypes.AgentSession, workDir string, allowAll bool, scanner *runtime.SkillScanner, excludeSkills []string, withSkills bool, toolNames []string) {
	var parts []string
	if !session.Stateless {
		parts = append(parts,
			"Work directory: `"+workDir+"` is authoritative this turn; ignore earlier ones in history. `run_command` already starts there — `cd` only to reach another directory: `run_command argv=[\"cd\", \"<path>\"]`.",
			buildPermissionModeSection(allowAll),
		)
	}
	if len(toolNames) > 0 {
		parts = append(parts, "Tools (call through run_tool): "+strings.Join(toolNames, ", "))
	}
	if withSkills {
		if list := skillListBlock(scanner, excludeSkills); list != "" {
			parts = append(parts, "Available skills:\n\n"+list)
		}
	}
	if len(parts) == 0 {
		return
	}
	session.TurnContext = provider.Message{
		Role:    "user",
		Content: strings.Join(parts, "\n\n"),
	}
}
