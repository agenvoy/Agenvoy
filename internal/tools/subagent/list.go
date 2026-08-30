package subagent

import (
	"fmt"
	"strings"

	"github.com/pardnchiu/agenvoy/internal/session"
	go_pkg_utils "github.com/pardnchiu/go-pkg/utils"
)

func listSessions() string {
	list := session.ListSessions()
	if len(list) == 0 {
		return "no session carries a self id — spawn a temp subagent (name empty)"
	}

	var sb strings.Builder
	for _, s := range list {
		role := strings.Join(strings.Fields(s.Role), " ")
		if role == "" {
			role = "(no role description)"
		}
		label := s.SelfID
		if s.Name != "" && s.Name != s.SelfID {
			label = fmt.Sprintf("%s (%s)", s.SelfID, s.Name)
		}
		fmt.Fprintf(&sb, "- %s — %s\n", label, go_pkg_utils.TruncateString(role, 256))
	}
	return strings.TrimRight(sb.String(), "\n")
}
