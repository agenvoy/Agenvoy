package subagent

import (
	"fmt"
	"strings"

	"github.com/pardnchiu/agenvoy/internal/session"
	go_pkg_utils "github.com/pardnchiu/go-pkg/utils"
)

func listSessions() string {
	list := session.ListNamedSessions()
	if len(list) == 0 {
		return "no reusable named sessions — spawn a temp subagent (name empty)"
	}

	var sb strings.Builder
	for _, s := range list {
		role := strings.Join(strings.Fields(s.Role), " ")
		if role == "" {
			role = "(no role description)"
		}
		fmt.Fprintf(&sb, "- %s — %s\n", s.Name, go_pkg_utils.TruncateString(role, 256))
	}
	return strings.TrimRight(sb.String(), "\n")
}
