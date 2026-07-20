package subagent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pardnchiu/agenvoy/internal/session"
	toolRegister "github.com/pardnchiu/agenvoy/internal/tools/register"
	toolTypes "github.com/pardnchiu/agenvoy/internal/tools/types"
	go_pkg_utils "github.com/pardnchiu/go-pkg/utils"
)

func registListSubagent() {
	toolRegister.Regist(toolRegister.Def{
		Name:        "list_subagent_sessions",
		AlwaysAllow: true,
		Concurrent:  true,
		Description: "List existing named (non-temp) sessions you can reuse as a subagent, each with its role. Call this BEFORE spawning a subagent for a single delegated task: if a listed role fits the task, `ask_user` whether to route to that session — on yes call `invoke_subagent(name=<that name>, ...)`, on no spawn a temp (name empty). When nothing is returned, no named session exists — spawn a temp directly. Not for broad parallel fan-out, which stays anonymous.",
		Parameters: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
		Handler: func(_ context.Context, _ *toolTypes.Executor, _ json.RawMessage) (string, error) {
			list := session.ListNamedSessions()
			if len(list) == 0 {
				return "no reusable named sessions — spawn a temp subagent (name empty)", nil
			}
			var sb strings.Builder
			for _, s := range list {
				role := strings.Join(strings.Fields(s.Role), " ")
				if role == "" {
					role = "(no role description)"
				}
				fmt.Fprintf(&sb, "- %s — %s\n", s.Name, go_pkg_utils.TruncateString(role, 256))
			}
			return strings.TrimRight(sb.String(), "\n"), nil
		},
	})
}
